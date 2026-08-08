package converter

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// ProcessFile converts the input audio to MP3. Encrypted formats are decrypted first.
func (c *FFMpegConverter) ProcessFile(inputPath, outputPath string) (ProbeResult, error) {
	ext := filepath.Ext(inputPath)
	if IsEncryptedFormat(ext) {
		return c.processEncrypted(inputPath, outputPath)
	}
	result, err := c.Probe(inputPath)
	if err != nil {
		result = ProbeResult{}
	}
	if err := c.ConvertToMP3(inputPath, outputPath); err != nil {
		return ProbeResult{}, fmt.Errorf("process file: %w", err)
	}
	return result, nil
}

// StashFile extracts metadata without re-encoding. MP3 is stored as-is at outputPath.
// Encrypted files are decrypted and converted to MP3 at outputPath.
func (c *FFMpegConverter) StashFile(inputPath, outputPath string) (ProbeResult, error) {
	ext := filepath.Ext(inputPath)
	if IsEncryptedFormat(ext) {
		return c.processEncrypted(inputPath, outputPath)
	}
	result, err := c.Probe(inputPath)
	if err != nil {
		result = ProbeResult{}
	}
	if inputPath != outputPath {
		if err := CopyFile(inputPath, outputPath); err != nil {
			return ProbeResult{}, fmt.Errorf("stash copy: %w", err)
		}
	}
	return result, nil
}

func (c *FFMpegConverter) processEncrypted(inputPath, outputPath string) (ProbeResult, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	logger := zap.NewNop()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Stream decrypted audio to a temp file instead of loading it into memory.
	tmpFile, err := os.CreateTemp(filepath.Dir(outputPath), "decrypted-*.tmp")
	if err != nil {
		return ProbeResult{}, fmt.Errorf("temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	type decryptResult struct {
		meta *DecryptedMeta
		err  error
	}
	decryptDone := make(chan decryptResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("decrypt goroutine recovered from panic: %v", r)
				decryptDone <- decryptResult{err: fmt.Errorf("decrypt panic: %v", r)}
			}
		}()
		m, _, e := DecryptToFile(ctx, f, filepath.Base(inputPath), tmpPath, logger)
		decryptDone <- decryptResult{m, e}
	}()

	var meta *DecryptedMeta
	select {
	case res := <-decryptDone:
		meta, err = res.meta, res.err
	case <-ctx.Done():
		// Force unblock the decoder by closing the underlying file.
		f.Close()
		return ProbeResult{}, fmt.Errorf("decrypt timeout")
	}
	if err != nil {
		return ProbeResult{}, fmt.Errorf("decrypt: %w", err)
	}

	result := ProbeResult{}
	if meta != nil && meta.Title != "" {
		result.Title = CleanMetadata(meta.Title)
		if len(meta.Artists) > 0 {
			result.Artist = CleanMetadata(meta.Artists[0])
		}
	} else {
		if pr, err := c.Probe(tmpPath); err == nil {
			result = pr
		}
	}

	// If stash was requested for an encrypted file, convert it to MP3 anyway
	// because the decrypted raw audio may not be MP3.
	if outputPath == inputPath {
		mp3Path := inputPath + ".mp3"
		if err := c.ConvertToMP3(tmpPath, mp3Path); err != nil {
			return ProbeResult{}, fmt.Errorf("convert: %w", err)
		}
		return result, nil
	}

	if err := c.ConvertToMP3(tmpPath, outputPath); err != nil {
		return ProbeResult{}, fmt.Errorf("convert: %w", err)
	}
	return result, nil
}

// CopyFile copies src to dst, creating directories as needed.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		os.Remove(dst)
	}
	return err
}
