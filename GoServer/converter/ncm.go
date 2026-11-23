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

	_ "unlock-music.dev/cli/algo/kwm"
	_ "unlock-music.dev/cli/algo/ncm"
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
	".ncm": true, ".kgg": true, ".kgm": true, ".vpr": true, ".kgma": true, ".kgm.flac": true, ".vpr.flac": true,
	".qmc0": true, ".qmc3": true, ".qmcogg": true, ".qmcflac": true,
	".kwm": true, ".xm": true, ".tm": true, ".ximalaya": true,
}

// IsEncryptedFormat checks if the file extension is a known encrypted format
func IsEncryptedFormat(ext string) bool {
	return supportedEncryptedExts[strings.ToLower(ext)]
}

// EncryptedExtension also preserves KuGou's compound Android suffixes.
func EncryptedExtension(filename string) string {
	name := strings.ToLower(filename)
	for _, suffix := range []string{".kgm.flac", ".vpr.flac"} {
		if strings.HasSuffix(name, suffix) {
			return suffix
		}
	}
	if ext := filepath.Ext(name); IsEncryptedFormat(ext) {
		return ext
	}
	return ""
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
	decoder, err := validatedDecoder(src, filename, logger, nil)
	if err != nil {
		return nil, nil, nil, err
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
	return decryptToFile(ctx, src, filename, outputPath, logger, nil)
}

func decryptToFile(_ context.Context, src io.ReadSeeker, filename, outputPath string, logger *zap.Logger, kugouKey func(string) string) (
	meta *DecryptedMeta,
	_ []byte,
	err error,
) {
	decoder, err := validatedDecoder(src, filename, logger, kugouKey)
	if err != nil {
		return nil, nil, err
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

func validatedDecoder(src io.ReadSeeker, filename string, logger *zap.Logger, kugouKey func(string) string) (common.Decoder, error) {
	ext := EncryptedExtension(filename)
	factories := common.GetDecoder(filename, ext != "")
	if len(factories) == 0 {
		return nil, fmt.Errorf("unsupported format: %s", filename)
	}
	factory := factories[0]
	for _, candidate := range factories {
		if candidate.Suffix == ext {
			factory = candidate
			break
		}
	}
	var decoder common.Decoder
	switch factory.Suffix {
	case ".kgg", ".kgm", ".kgma", ".vpr", ".kgm.flac", ".vpr.flac":
		var err error
		decoder, err = kugouV5(src, kugouKey)
		if err != nil {
			return nil, fmt.Errorf("validate: %w", err)
		}
	}
	if decoder == nil {
		decoder = factory.Create(&common.DecoderParams{Reader: src, Extension: factory.Suffix, FilePath: filename, Logger: logger})
	}
	if err := decoder.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return decoder, nil
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
