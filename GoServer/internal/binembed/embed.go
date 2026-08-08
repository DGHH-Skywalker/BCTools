package binembed

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

//go:embed bin/*
var binFS embed.FS

// Extract copies the embedded ffmpeg.exe / ffprobe.exe into targetDir if they
// are missing or have a different size. This keeps the app as a single .exe
// while still letting converter spawn the helper binaries from a normal path.
func Extract(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create bin dir: %w", err)
	}

	for _, name := range []string{"ffmpeg.exe", "ffprobe.exe"} {
		if err := extractFile(name, targetDir); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(name, targetDir string) error {
	src, err := binFS.Open(path.Join("bin", name))
	if err != nil {
		// Binary not embedded; caller should fall back to existing search.
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open embedded %s: %w", name, err)
	}
	defer src.Close()

	embeddedInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat embedded %s: %w", name, err)
	}

	dstPath := filepath.Join(targetDir, name)
	if existing, err := os.Stat(dstPath); err == nil {
		if existing.Size() == embeddedInfo.Size() {
			// Already up to date; avoid unnecessary write that might look
			// suspicious to antivirus heuristics.
			return nil
		}
	}

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("create %s: %w", dstPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}
	return nil
}
