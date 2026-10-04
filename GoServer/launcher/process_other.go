//go:build !windows

package launcher

import (
	"fmt"
	"syscall"
	"time"
)

type otherProcess struct{ pid int }

func openRuntimeProcess(info RuntimeInfo) (runtimeProcess, error) {
	if info.PID <= 0 {
		return nil, fmt.Errorf("invalid runtime PID")
	}
	return &otherProcess{pid: info.PID}, nil
}
func (p *otherProcess) Close() error { return nil }
func (p *otherProcess) Wait(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if syscall.Kill(p.pid, 0) == syscall.ESRCH {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
func (p *otherProcess) Terminate() error {
	return fmt.Errorf("旧程序尚未退出；此平台不执行强制结束")
}
