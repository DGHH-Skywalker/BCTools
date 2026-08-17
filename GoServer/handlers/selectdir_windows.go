//go:build windows

package handlers

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// 历史原因：本文件原本走 SHBrowseForFolderW（Win95 时代的旧 API），在 1080p /
// 2K 显示器上字号不随系统缩放、控件糊——就是用户说的「SD 卡导出工具的文件夹
// 选择控件太糊」。
//
// 改为 IFileOpenDialog + FOS_PICKFOLDERS（Vista+ 通用文件夹选择器）：
//   - 自动跟随系统 DPI / per-monitor v2
//   - 跟资源管理器「选择文件夹」对话框视觉一致
//   - 仍是「选择文件」对话框家族，所以走 IFileOpenDialog 而不是 IFileDialog
// 父窗口仍是当前前台窗口（getForegroundWindow），与原行为对齐。
var (
	modole32          = windows.NewLazySystemDLL("ole32.dll")
	procCoInit        = modole32.NewProc("CoInitializeEx")
	procCoUninit      = modole32.NewProc("CoUninitialize")
	procCoCreate      = modole32.NewProc("CoCreateInstance")
	procCoTaskMemFree = modole32.NewProc("CoTaskMemFree")

	moduser32               = windows.NewLazySystemDLL("user32.dll")
	procGetForegroundWindow = moduser32.NewProc("GetForegroundWindow")
)

// IID 与 CLSID（GUID 字节序按 Windows little-endian 排）
var (
	CLSID_FileOpenDialog = windows.GUID{
		Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE,
		Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x66, 0x21, 0x55},
	}
	IID_IFileOpenDialog = windows.GUID{
		Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768,
		Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x6E},
	}
)

// FOS_PICKFOLDERS 让「打开」对话框变成纯文件夹选择器。
const FOS_PICKFOLDERS = 0x00000020

// FOS_FORCEFILESYSTEM 强制只显示文件系统项；不显示「库」「This PC」等虚拟节点。
const FOS_FORCEFILESYSTEM = 0x00000040

// SIGDN_FILESYSPATH 让 IShellItem::GetDisplayName 返回文件系统绝对路径。
const SIGDN_FILESYSPATH = 0x80058000

// HRESULT 成功码 / 用户取消
const (
	S_OK              = 0
	HRESULT_CANCELLED = 0x800704C7
	HRESULT_ABORT     = 0x80004004
)

// selectDirectory opens a modern Windows folder picker (IFileOpenDialog with
// FOS_PICKFOLDERS). Compared to the legacy SHBrowseForFolderW, this dialog
// respects per-monitor DPI scaling, uses the system font, and visually
// matches Explorer's "Select Folder" dialog.
//
// Returns an empty string if the user cancels.
func selectDirectory(title string) (string, error) {
	// COM apartment 必须在线程上初始化；STA 是 IFileOpenDialog 必需的。
	// 第二个参数 2 = COINIT_APARTMENTTHREADED；1 = S_FALSE 表示已初始化过，可忽略。
	hr, _, _ := procCoInit.Call(0, 2)
	if hr != S_OK && hr != 1 {
		return "", fmt.Errorf("CoInitializeEx failed: 0x%x", hr)
	}
	defer procCoUninit.Call()

	// 创建 IFileOpenDialog
	var pUnk uintptr
	hr, _, _ = procCoCreate.Call(
		uintptr(unsafe.Pointer(&CLSID_FileOpenDialog)),
		0, 0,
		uintptr(unsafe.Pointer(&IID_IFileOpenDialog)),
		uintptr(unsafe.Pointer(&pUnk)),
	)
	if hr != S_OK || pUnk == 0 {
		return "", fmt.Errorf("CoCreateInstance(FileOpenDialog) failed: 0x%x", hr)
	}
	defer releaseIUnknown(pUnk)

	// vtable 偏移参考 shobjidl.h（按 IUnknown → IModalWindow → IFileDialog 顺序累计）：
	//   IUnknown:        0=QueryInterface, 1=AddRef, 2=Release
	//   IModalWindow:    3=Show
	//   IFileDialog:     4=SetFileTypes, 5=SetFileTypeIndex, 6=GetFileTypeIndex,
	//                   7=Advise, 8=Unadvise, 9=SetOptions, 10=GetOptions,
	//                   11=SetDefaultFolder, 12=SetFolder, 13=GetFolder,
	//                   14=GetCurrentSelection, 15=SetTitle, 16=SetOkButtonLabel,
	//                   17=SetFileName, 18=GetFileName, 19=SetFileNameLabel,
	//                   20=SetControlLabel, 21=GetControlState, 22=SetControlState,
	//                   23=GetControlItemText, 24=RemoveAllControlItems,
	//                   25=SetFolder ..., 26=GetFolder ..., 27=GetCurrentSelection ...,
	//                   28=SetFileTypes ...  32=GetResult
	//
	// 严格 IFileDialog 顺序中 GetResult 实际是偏移 27。我们需要：SetOptions (9)、
	// SetTitle (17)、Show (3)、GetResult (27)。

	// SetOptions(pUnk, FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM)
	if callVtbl(pUnk, 9, 0, uintptr(FOS_PICKFOLDERS|FOS_FORCEFILESYSTEM)) != S_OK {
		return "", fmt.Errorf("SetOptions failed")
	}

	// SetTitle(pUnk, title)
	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return "", fmt.Errorf("title encoding: %w", err)
	}
	if callVtbl(pUnk, 17, 0, uintptr(unsafe.Pointer(titlePtr))) != S_OK {
		return "", fmt.Errorf("SetTitle failed")
	}

	// Show(pUnk, hwndOwner)
	owner, _, _ := procGetForegroundWindow.Call()
	hrShow := callVtbl(pUnk, 3, 0, owner)
	if hrShow != S_OK {
		// 用户取消：0x800704C7 = ERROR_CANCELLED，0x80004004 = E_ABORT
		if hrShow == HRESULT_CANCELLED || hrShow == HRESULT_ABORT {
			return "", nil
		}
		return "", fmt.Errorf("Show failed: 0x%x", hrShow)
	}

	// GetResult(pUnk, &ppItem)
	var pItem uintptr
	if callVtbl(pUnk, 27, 0, uintptr(unsafe.Pointer(&pItem))) != S_OK || pItem == 0 {
		return "", fmt.Errorf("GetResult failed")
	}
	defer releaseIUnknown(pItem)

	// IShellItem vtable：0=QI, 1=AddRef, 2=Release, 3=BindToHandler, 4=GetParent, 5=GetDisplayName, 6=GetAttributes
	var namePtr *uint16
	if callVtbl(pItem, 5, 0, uintptr(SIGDN_FILESYSPATH), uintptr(unsafe.Pointer(&namePtr))) != S_OK || namePtr == nil {
		return "", fmt.Errorf("GetDisplayName failed")
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(namePtr)))

	return windows.UTF16PtrToString(namePtr), nil
}

// callVtbl 调 p 的 vtable 偏移 idx 处的成员函数；retIdx 是「返回值位于栈中的第几号槽」，
// 因为 Go syscall 没法直接拿 syscall.SyscallN 的真实返回值，所以用 retIdx 把 hr 找出
// 来（vtable 调用的第一个返回值固定在 rax 里，但 Go 把 rax 放到 retIdx 索引位置）。
//
// 我们只关心 HRESULT，所以 retIdx=0 即可。
func callVtbl(p uintptr, idx uintptr, retIdx uintptr, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(p))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + idx*unsafe.Sizeof(uintptr(0))))
	all := append([]uintptr{p}, args...)
	r, _, _ := syscall.SyscallN(fn, all...)
	_ = retIdx
	return r
}

// releaseIUnknown 走 vtable 偏移 2（Release）来释放 COM 对象。
// IUnknown 的三方法布局在所有 COM 接口里都是固定 [QI, AddRef, Release]。
func releaseIUnknown(p uintptr) {
	if p == 0 {
		return
	}
	vtbl := *(*uintptr)(unsafe.Pointer(p))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + 2*unsafe.Sizeof(uintptr(0))))
	syscall.SyscallN(fn, p)
}
