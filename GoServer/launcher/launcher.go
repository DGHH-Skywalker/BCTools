package launcher

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// IsBackendRunning checks whether something is listening on the given localhost port.
func IsBackendRunning(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// WaitForPortFree waits until the localhost port is free or the timeout expires.
func WaitForPortFree(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !IsBackendRunning(port) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// ResolveBinary locates a binary in the bin directory (embedded by build) or
// the executable directory. We deliberately do NOT fall back to PATH so the
// app always uses the project's own minimal ffmpeg/ffprobe and never silently
// picks up a full system ffmpeg.
func ResolveBinary(name, binDir, exeDir string) string {
	candidates := []string{
		filepath.Join(binDir, name),
		filepath.Join(exeDir, name),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	_ = exec.LookPath // keep import; explicit no-PATH-fallback by design
	return ""
}

// SafeGo starts a goroutine with panic recovery and logging.
func SafeGo(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("goroutine %s recovered from panic: %v", name, r)
			}
		}()
		fn()
	}()
}
