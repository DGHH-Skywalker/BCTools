//go:build windows

package launcher

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

func pipeName(dir string) string {
	dir, _ = filepath.Abs(dir)
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	return fmt.Sprintf(`\\.\pipe\BCTools-%x`, sha256.Sum256([]byte(strings.ToLower(dir))))
}

func Claim(dir, mode string, commands chan<- string) (io.Closer, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(dir, ".instance.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	err = windows.LockFileEx(windows.Handle(lock.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{})
	if err != nil {
		lock.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, nil
		}
		return nil, err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		lock.Close()
		return nil, err
	}
	listener, err := winio.ListenPipe(pipeName(dir), &winio.PipeConfig{
		SecurityDescriptor: fmt.Sprintf("D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;%s)", user.User.Sid),
	})
	if err != nil {
		lock.Close()
		return nil, err
	}
	go serveControl(listener, mode, commands)
	return &ownership{Listener: listener, lock: lock}, nil
}

func SendControl(dir, command string) error {
	timeout := 300 * time.Millisecond
	conn, err := winio.DialPipe(pipeName(dir), &timeout)
	if err != nil {
		return err
	}
	return sendControl(conn, command)
}
