//go:build !windows

package launcher

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func AcquireLock(key string) (io.Closer, error) {
	f, err := os.OpenFile(filepath.Join(os.TempDir(), fmt.Sprintf("bctools-%d-%x.lock", os.Getuid(), sha256.Sum256([]byte(key)))), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(45 * time.Second)
	for {
		if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			return f, nil
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, fmt.Errorf("等待另一实例退出超时：%w", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
