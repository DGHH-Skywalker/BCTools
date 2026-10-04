//go:build windows

package launcher

import (
	"crypto/sha256"
	"fmt"
	"io"
	"runtime"
	"sync"

	"golang.org/x/sys/windows"
)

type processLock struct {
	release chan struct{}
	done    chan struct{}
	once    sync.Once
}

func (l *processLock) Close() error {
	l.once.Do(func() { close(l.release); <-l.done })
	return nil
}

// Mutex ownership is thread-bound. Keep its acquisition and release on a
// dedicated OS thread; Windows reports abandoned owners after a process crash.
func AcquireLock(key string) (io.Closer, error) {
	name, _ := windows.UTF16PtrFromString(fmt.Sprintf("Global\\BCTools-%x", sha256.Sum256([]byte(key))))
	l := &processLock{release: make(chan struct{}), done: make(chan struct{})}
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(l.done)
		h, err := windows.CreateMutex(nil, false, name)
		// CreateMutex returns a valid handle together with ERROR_ALREADY_EXISTS.
		// Existing launchers must wait on that handle, rather than fail startup.
		if err != nil && err != windows.ERROR_ALREADY_EXISTS {
			if h != 0 {
				windows.CloseHandle(h)
			}
			ready <- err
			return
		}
		defer windows.CloseHandle(h)
		result, err := windows.WaitForSingleObject(h, 45000)
		if err != nil || (result != windows.WAIT_OBJECT_0 && result != windows.WAIT_ABANDONED) {
			ready <- fmt.Errorf("等待另一实例完成安全退出超时：%v", err)
			return
		}
		ready <- nil
		<-l.release
		windows.ReleaseMutex(h)
	}()
	if err := <-ready; err != nil {
		return nil, err
	}
	return l, nil
}
