//go:build !windows

package launcher

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func controlPath(dir string) string {
	dir, _ = filepath.Abs(dir)
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	hash := sha256.Sum256([]byte(dir))
	return filepath.Join(os.TempDir(), fmt.Sprintf("bctools-%d-%x.sock", os.Getuid(), hash[:16]))
}

func Claim(dir, mode string, commands chan<- string) (io.Closer, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(dir, ".instance.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, nil
		}
		return nil, err
	}
	name := controlPath(dir)
	_ = os.Remove(name)
	listener, err := net.Listen("unix", name)
	if err != nil {
		lock.Close()
		return nil, err
	}
	if err := os.Chmod(name, 0600); err != nil {
		listener.Close()
		lock.Close()
		return nil, err
	}
	go serveControl(listener, mode, commands)
	return &ownership{Listener: listener, lock: lock}, nil
}

func SendControl(dir, command string) error {
	conn, err := net.DialTimeout("unix", controlPath(dir), 300*time.Millisecond)
	if err != nil {
		return err
	}
	return sendControl(conn, command)
}
