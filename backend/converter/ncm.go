package converter

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"unlock-music.dev/cli/algo/common"

	_ "unlock-music.dev/cli/algo/kgm"
	_ "unlock-music.dev/cli/algo/kwm"
	_ "unlock-music.dev/cli/algo/ncm"
	_ "unlock-music.dev/cli/algo/qmc"
	_ "unlock-music.dev/cli/algo/tm"
	_ "unlock-music.dev/cli/algo/xiami"
	_ "unlock-music.dev/cli/algo/ximalaya"
)

// DecryptedMeta holds metadata from decrypted audio files
type DecryptedMeta struct {
	Title   string
	Artists []string
	Album   string
}

var supportedEncryptedExts = map[string]bool{
	".ncm": true, ".kgm": true, ".vpr": true, ".kgma": true,
	".qmc0": true, ".qmc3": true, ".qmcogg": true, ".qmcflac": true,
	".kwm": true, ".xm": true, ".tm": true, ".ximalaya": true,
}

// IsEncryptedFormat checks if the file extension is a known encrypted format
func IsEncryptedFormat(ext string) bool {
	return supportedEncryptedExts[strings.ToLower(ext)]
}

// Decrypt decrypts an encrypted audio file and returns the raw audio data + metadata.
// Prefer DecryptToFile for large files to avoid loading the whole audio into memory.
// Cover art extraction is skipped to avoid potential network hangs.
func Decrypt(ctx context.Context, src io.ReadSeeker, filename string, logger *zap.Logger) (
	audioData []byte,
	meta *DecryptedMeta,
	_ []byte,
	err error,
) {
	ext := filepath.Ext(filename)
	factories := common.GetDecoder(ext, false)
	if len(factories) == 0 {
		return nil, nil, nil, fmt.Errorf("unsupported format: %s", ext)
	}

	params := &common.DecoderParams{
		Reader:    src,
		Extension: ext,
		FilePath:  filename,
		Logger:    logger,
	}

	decoder := factories[0].Create(params)
	if err := decoder.Validate(); err != nil {
		return nil, nil, nil, fmt.Errorf("validate: %w", err)
	}

	audioData, err = io.ReadAll(decoder)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decrypt read: %w", err)
	}

	meta = extractMeta(decoder)
	return audioData, meta, nil, nil
}

// DecryptToFile decrypts an encrypted audio file and writes the decoded audio to outputPath.
// Streaming avoids loading the entire decrypted audio into memory.
// Cover art extraction is skipped to avoid potential network hangs.
func DecryptToFile(ctx context.Context, src io.ReadSeeker, filename, outputPath string, logger *zap.Logger) (
	meta *DecryptedMeta,
	_ []byte,
	err error,
) {
	ext := filepath.Ext(filename)
	factories := common.GetDecoder(ext, false)
	if len(factories) == 0 {
		return nil, nil, fmt.Errorf("unsupported format: %s", ext)
	}

	params := &common.DecoderParams{
		Reader:    src,
		Extension: ext,
		FilePath:  filename,
		Logger:    logger,
	}

	decoder := factories[0].Create(params)
	if err := decoder.Validate(); err != nil {
		return nil, nil, fmt.Errorf("validate: %w", err)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return nil, nil, fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	// Extract metadata before streaming (decoders cache header metadata)
	meta = extractMeta(decoder)

	if _, err := io.Copy(out, decoder); err != nil {
		os.Remove(outputPath)
		return nil, nil, fmt.Errorf("decrypt copy: %w", err)
	}
	if err := out.Sync(); err != nil {
		os.Remove(outputPath)
		return nil, nil, fmt.Errorf("sync output: %w", err)
	}

	return meta, nil, nil
}

func extractMeta(decoder common.Decoder) *DecryptedMeta {
	getter, ok := decoder.(common.AudioMetaGetter)
	if !ok {
		return nil
	}
	m, err := getter.GetAudioMeta(context.Background())
	if err != nil || m == nil {
		return nil
	}
	return &DecryptedMeta{
		Title:   m.GetTitle(),
		Artists: m.GetArtists(),
		Album:   m.GetAlbum(),
	}
}
