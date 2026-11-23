package converter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"unlock-music.dev/cli/algo/common"
	"unlock-music.dev/cli/algo/kgm"
	"unlock-music.dev/cli/algo/qmc"
)

var ErrKugouKeyMissing = errors.New("Kugou decryption key is missing")

// kugouV5 uses the shared key index for KGG / KGM v5. Legacy KGM still uses
// the upstream decoder. Validate all lengths before using them.
func kugouV5(src io.ReadSeeker, lookup func(string) string) (common.Decoder, error) {
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	var prefix [24]byte
	if _, err := io.ReadFull(src, prefix[:]); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint32(prefix[20:24]) != 5 {
		_, err := src.Seek(0, io.SeekStart)
		return nil, err
	}
	switch [16]byte(prefix[:16]) {
	case [16]byte{0x7c, 0xd5, 0x32, 0xeb, 0x86, 0x02, 0x7f, 0x4b, 0xa8, 0xaf, 0xa6, 0x8e, 0x0f, 0xff, 0x99, 0x14},
		[16]byte{0x05, 0x28, 0xbc, 0x96, 0xe9, 0xe4, 0x5a, 0x43, 0x91, 0xaa, 0xbd, 0xd0, 0x7a, 0xf5, 0x36, 0x31}:
	default:
		return nil, kgm.ErrKgmMagicHeader
	}
	const hashOffset = 0x48
	if _, err := src.Seek(hashOffset-4, io.SeekStart); err != nil {
		return nil, err
	}
	var size uint32
	if err := binary.Read(src, binary.LittleEndian, &size); err != nil {
		return nil, err
	}
	audioOffset := binary.LittleEndian.Uint32(prefix[16:20])
	if size == 0 || size > 256 || audioOffset < hashOffset+size {
		return nil, errors.New("invalid KGM audio header")
	}
	var hash [256]byte
	if _, err := io.ReadFull(src, hash[:size]); err != nil {
		return nil, err
	}
	end, err := src.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if int64(audioOffset) >= end {
		return nil, errors.New("invalid KGM audio offset")
	}
	if lookup == nil {
		return nil, ErrKugouKeyMissing
	}
	key := lookup(string(hash[:size]))
	if key == "" {
		return nil, ErrKugouKeyMissing
	}
	cipher, err := qmc.NewQmcCipherDecoderFromEKey([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("invalid Kugou database key: %w", err)
	}
	return &kugouAudio{src: src, cipher: cipher, start: int64(audioOffset)}, nil
}

type kugouAudio struct {
	src    io.ReadSeeker
	cipher common.StreamDecoder
	start  int64
	offset int
}

func (d *kugouAudio) Validate() error {
	d.offset = 0
	_, err := d.src.Seek(d.start, io.SeekStart)
	return err
}

func (d *kugouAudio) Read(p []byte) (int, error) {
	n, err := d.src.Read(p)
	if n > 0 {
		d.cipher.Decrypt(p[:n], d.offset)
		d.offset += n
	}
	return n, err
}
