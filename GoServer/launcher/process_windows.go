//go:build windows

package launcher

import (
	"fmt"
	"golang.org/x/sys/windows"
	"path/filepath"
	"strings"
	"time"
)

type windowsProcess struct{ handle windows.Handle }

func openRuntimeProcess(info RuntimeInfo) (runtimeProcess, error) {
	if info.PID <= 0 || info.Executable == "" {
		return nil, fmt.Errorf("旧实例身份信息不完整")
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(info.PID))
	if err != nil {
		return nil, err
	}
	buffer := make([]uint16, 32768)
	length := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(h, 0, &buffer[0], &length); err != nil {
		windows.CloseHandle(h)
		return nil, err
	}
	if !strings.EqualFold(filepath.Clean(windows.UTF16ToString(buffer[:length])), filepath.Clean(info.Executable)) {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("运行实例路径不匹配，已停止切换")
	}
	return &windowsProcess{handle: h}, nil
}
func (p *windowsProcess) Close() error { return windows.CloseHandle(p.handle) }
func (p *windowsProcess) Wait(timeout time.Duration) bool {
	result, err := windows.WaitForSingleObject(p.handle, uint32(timeout/time.Millisecond))
	return err == nil && result == windows.WAIT_OBJECT_0
}
func (p *windowsProcess) Terminate() error { return windows.TerminateProcess(p.handle, 0) }
