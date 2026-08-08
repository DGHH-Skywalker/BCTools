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
)

const (
	BIF_RETURNONLYFSDIRS  = 0x00000001
	BIF_DONTGOBELOWDOMAIN = 0x00000002
	BIF_NEWDIALOGSTYLE    = 0x00000040
	BIF_SHAREABLE         = 0x00008000
	MAX_PATH              = 260
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
// Returns an empty string if the user cancels.
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

	bi := browseInfoW{
		HwndOwner:      0,
		pszDisplayName: &displayName[0],
		lpszTitle:      titlePtr,
		ulFlags:        BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE | BIF_DONTGOBELOWDOMAIN | BIF_SHAREABLE,
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
