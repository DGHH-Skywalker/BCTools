//go:build windows

package handlers

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modshell32               = windows.NewLazySystemDLL("shell32.dll")
	procSHBrowseForFolderW   = modshell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = modshell32.NewProc("SHGetPathFromIDListW")
	modole32                 = windows.NewLazySystemDLL("ole32.dll")
	procCoTaskMemFree        = modole32.NewProc("CoTaskMemFree")
	moduser32                = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow  = moduser32.NewProc("GetForegroundWindow")
	procSetWindowPos         = moduser32.NewProc("SetWindowPos")
)

const (
	BIF_RETURNONLYFSDIRS  = 0x00000001
	BIF_DONTGOBELOWDOMAIN = 0x00000002
	BIF_NEWDIALOGSTYLE    = 0x00000040
	BIF_SHAREABLE         = 0x00008000
	MAX_PATH              = 260

	HWND_TOPMOST       = -1
	SWP_NOMOVE         = 0x0002
	SWP_NOSIZE         = 0x0001
	SWP_SHOWWINDOW     = 0x0040
	BFFM_INITIALIZED   = 1
	BFFM_SETSELECTIONW = 0x0400 + 103
)

type browseInfoW struct {
	HwndOwner      windows.HWND
	PIDLRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

// selectDirectory opens a Windows folder browser dialog using SHBrowseForFolderW.
// The dialog is made topmost and owned by the current foreground window so it
// appears in front of the browser / main application window. Returns an empty
// string if the user cancels.
func selectDirectory(title string) (string, error) {
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
		// S_FALSE (already initialized) is acceptable
		if errno, ok := err.(syscall.Errno); !ok || errno != 1 {
			return "", fmt.Errorf("CoInitializeEx failed: %w", err)
		}
	}
	defer windows.CoUninitialize()

	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return "", fmt.Errorf("title encoding: %w", err)
	}

	displayName := make([]uint16, MAX_PATH)

	owner := getForegroundWindow()
	callback := windows.NewCallback(browseCallbackProc)

	bi := browseInfoW{
		HwndOwner:      owner,
		pszDisplayName: &displayName[0],
		lpszTitle:      titlePtr,
		ulFlags:        BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE | BIF_DONTGOBELOWDOMAIN | BIF_SHAREABLE,
		lpfn:           callback,
	}

	ret, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if ret == 0 {
		return "", nil
	}
	defer procCoTaskMemFree.Call(ret)

	pathBuf := make([]uint16, MAX_PATH)
	ok, _, _ := procSHGetPathFromIDListW.Call(ret, uintptr(unsafe.Pointer(&pathBuf[0])))
	if ok == 0 {
		return "", fmt.Errorf("SHGetPathFromIDListW failed")
	}

	return syscall.UTF16ToString(pathBuf), nil
}

func getForegroundWindow() windows.HWND {
	hwnd, _, _ := procGetForegroundWindow.Call()
	return windows.HWND(hwnd)
}

func browseCallbackProc(hwnd windows.HWND, msg uint32, lParam, lpData uintptr) uintptr {
	if msg == BFFM_INITIALIZED {
		// Ensure the folder dialog stays on top of all other windows.
		// HWND_TOPMOST is -1 as a signed HWND value.
		topmost := uintptr(^uint(0))
		procSetWindowPos.Call(
			uintptr(hwnd),
			topmost,
			0, 0, 0, 0,
			SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW,
		)
	}
	return 0
}
