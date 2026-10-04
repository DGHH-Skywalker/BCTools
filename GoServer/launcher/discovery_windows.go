//go:build windows

package launcher

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func executableVersion(path string) (string, error) {
	lib := windows.NewLazySystemDLL("version.dll")
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	size, _, _ := lib.NewProc("GetFileVersionInfoSizeW").Call(uintptr(unsafe.Pointer(p)), 0)
	if size == 0 || size > 4*1024*1024 {
		return "", fmt.Errorf("missing version information")
	}
	buf := make([]byte, size)
	loaded, _, _ := lib.NewProc("GetFileVersionInfoW").Call(uintptr(unsafe.Pointer(p)), 0, size, uintptr(unsafe.Pointer(&buf[0])))
	if loaded == 0 {
		return "", fmt.Errorf("unreadable version information")
	}
	query := func(key string) (unsafe.Pointer, uint32, bool) {
		k, _ := windows.UTF16PtrFromString(key)
		var value unsafe.Pointer
		var length uint32
		ok, _, _ := lib.NewProc("VerQueryValueW").Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(k)), uintptr(unsafe.Pointer(&value)), uintptr(unsafe.Pointer(&length)))
		return value, length, ok != 0
	}
	translation, n, ok := query("\\VarFileInfo\\Translation")
	if !ok || n < 4 {
		return "", fmt.Errorf("missing product identity")
	}
	langs := unsafe.Slice((*uint16)(translation), int(n)/2)
	valid := false
	for i := 0; i+1 < len(langs); i += 2 {
		read := func(field string) string {
			value, count, ok := query(fmt.Sprintf("\\StringFileInfo\\%04x%04x\\%s", langs[i], langs[i+1], field))
			if !ok || count == 0 {
				return ""
			}
			return windows.UTF16ToString(unsafe.Slice((*uint16)(value), int(count)))
		}
		if read("ProductName") == "小播点歌工具" && read("InternalName") == "broadcast-tool" {
			valid = true
			break
		}
	}
	if !valid {
		return "", fmt.Errorf("not a BCTools application")
	}
	value, length, ok := query("\\")
	if !ok || length < 52 {
		return "", fmt.Errorf("missing fixed version")
	}
	info := unsafe.Slice((*uint32)(value), 13)
	if info[0] != 0xfeef04bd {
		return "", fmt.Errorf("invalid version signature")
	}
	return fmt.Sprintf("%d.%d.%d.%d", info[4]>>16, info[4]&65535, info[5]>>16, info[5]&65535), nil
}
