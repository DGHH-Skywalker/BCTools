//go:build windows

package handlers

import (
	"fmt"
	"log"
	"runtime"
	"syscall"
	"time"
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
//
// 5.6.1 关键修复（5.6.0 那个窗口看不到的根因）：
//
// 上一版只看到「线程没锁」一个症状，加了 runtime.LockOSThread 之后用户依然
// 看不到窗口。深挖发现是两个独立问题叠加：
//
// (1) vtable 偏移错误
//     IUnknown → IModalWindow → IFileDialog → IFileOpenDialog 的继承链中，
//     IModalWindow 只增加 Show（offset 3），IFileDialog 从 offset 4 开始。
//     旧代码一度把 IModalWindow 误当成继承自 IOleWindow，导致 Show、SetOptions、
//     SetTitle、GetResult 全部调错方法——例如 Show 调到 GetWindow，SetOptions
//     调到 SetFolder，对话框实际上没有被创建，所以「窗口没显示」。
//
// (2) Win10 Foreground Window Restriction
//     即使路径对了，浏览器抢焦点时 bctools.exe 是后台进程，Win10 起新模态
//     窗口拿不到前台；用 GetForegroundWindow 拿到的几乎总是浏览器 HWND（跨
//     进程），UIPI 还会二次拦截。
//     解决：NULL owner（独立 top-level，不再被 UIPI 卡）+ AllowSetForegroundWindow
//     + Show 之后 QueryInterface 出 IOleWindow 拿 HWND，强制 HWND_TOPMOST 抢一次
//     焦点，再降回 NOTOPMOST 避免一直压顶。
var (
	modole32          = windows.NewLazySystemDLL("ole32.dll")
	procCoInit        = modole32.NewProc("CoInitializeEx")
	procCoUninit      = modole32.NewProc("CoUninitialize")
	procCoCreate      = modole32.NewProc("CoCreateInstance")
	procCoTaskMemFree = modole32.NewProc("CoTaskMemFree")

	moduser32 = windows.NewLazySystemDLL("user32.dll")
	procAllowSetForegroundWindow = moduser32.NewProc("AllowSetForegroundWindow")
	procSetForegroundWindow      = moduser32.NewProc("SetForegroundWindow")
	procSetWindowPos             = moduser32.NewProc("SetWindowPos")
	procGetSystemMetrics         = moduser32.NewProc("GetSystemMetrics")
	procGetWindowRect            = moduser32.NewProc("GetWindowRect")

	modkernel32           = windows.NewLazySystemDLL("kernel32.dll")
	procGetCurrentProcessId = modkernel32.NewProc("GetCurrentProcessId")
)

// IID / CLSID（GUID 字节序按 Windows little-endian 排）。
var (
	CLSID_FileOpenDialog = windows.GUID{
		Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE,
		Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x66, 0x21, 0x55},
	}
	IID_IFileOpenDialog = windows.GUID{
		Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768,
		Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x6E},
	}
	// IOleWindow 用于 Show 成功后拿对话框 HWND 强制置顶。
	IID_IOleWindow = windows.GUID{
		Data1: 0x00000114, Data2: 0x0000, Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
)

// FOS_* / SIGDN_* 常量。
const (
	FOS_PICKFOLDERS     = 0x00000020
	FOS_FORCEFILESYSTEM = 0x00000040
	SIGDN_FILESYSPATH   = 0x80058000
)

// HRESULT 通用值。
const (
	S_OK              = 0
	HRESULT_CANCELLED = 0x800704C7
	HRESULT_ABORT     = 0x80004004
)

// COM 创建上下文。
const (
	CLSCTX_INPROC_SERVER = 1
)

// SetWindowPos flag（只取用到的几个；HWND_* 走常量字面值）。
const (
	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040
)

// SetWindowPos 第二个参数 hWndInsertAfter 的特殊值（winuser.h）。
//   HWND_TOP        = 0
//   HWND_BOTTOM     = 1
//   HWND_TOPMOST    = -1
//   HWND_NOTOPMOST  = -2
const (
	HWND_TOPMOST   = ^uintptr(0)         // -1
	HWND_NOTOPMOST = ^uintptr(0) - 1     // -2
)

// GetSystemMetrics 索引（winuser.h）。
const (
	SM_CXSCREEN = 0
	SM_CYSCREEN = 1
)

// IFileOpenDialog vtable 偏移（按 IUnknown → IModalWindow → IFileDialog →
// IFileOpenDialog 继承顺序累加；COM 规范保证 vtable 与声明顺序一致）：
//
//   IUnknown:        0=QueryInterface, 1=AddRef, 2=Release
//   IModalWindow:    3=Show
//   IFileDialog:     4=SetFileTypes,   5=SetFileTypeIndex,  6=GetFileTypeIndex,
//                    7=Advise,         8=Unadvise,          9=SetOptions,
//                    10=GetOptions,    11=SetDefaultFolder, 12=SetFolder,
//                    13=GetFolder,     14=GetCurrentSelection,
//                    15=SetFileName,   16=GetFileName,      17=SetTitle,
//                    18=SetOkButtonLabel, 19=SetFileNameLabel,
//                    20=GetResult,     21=AddPlace,         22=SetDefaultExtension,
//                    23=Close,         24=SetClientGuid,    25=ClearClientData,
//                    26=SetFilter
//   IFileOpenDialog: 27=GetResults,    28=GetSelectedItems
//
// 5.6.1 修正：上一版误把 IModalWindow 当成继承自 IOleWindow，导致所有 IFileDialog
// 方法偏移整体多算 2，Show 实际上调到了 GetWindow、SetOptions 调到了 SetFolder，
// 对话框因此没有被真正创建出来。
//
// 我们用：Show(3) / SetOptions(9) / SetTitle(17) / GetResult(20)
const (
	vtblShow       = 3
	vtblSetOptions = 9
	vtblSetTitle   = 17
	vtblGetResult  = 20
)

// IOleWindow vtable 偏移（IUnknown 之后）。
const (
	vtblOleWindowGetWindow = 3
)

// selectDirectory opens a modern Windows folder picker (IFileOpenDialog with
// FOS_PICKFOLDERS). Compared to the legacy SHBrowseForFolderW, this dialog
// respects per-monitor DPI scaling, uses the system font, and visually
// matches Explorer's "Select Folder" dialog.
//
// Returns an empty string if the user cancels.
//
// 5.6.1：把整个 COM 流程丢到独立 OS 线程跑（goroutine + LockOSThread），
// 主 HTTP handler 通过 channel 等结果：
//   1) CoInit / Show / IOleWindow::GetWindow 严格同线程，COM STA 单元稳定；
//   2) 不阻塞 HTTP worker，长任务不会拖死 server；
//   3) 加 120s 兜底超时，Show 内部卡死不会永久占住 OS 线程。
func selectDirectory(title string) (string, error) {
	type result struct {
		path string
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		path, err := selectDirectoryImpl(title)
		resultCh <- result{path, err}
	}()

	select {
	case r := <-resultCh:
		return r.path, r.err
	case <-time.After(120 * time.Second):
		return "", fmt.Errorf("selectDirectory timeout after 120s")
	}
}

// selectDirectoryImpl runs on a dedicated OS thread.
func selectDirectoryImpl(title string) (string, error) {
	// COM apartment：第二个参数 2 = COINIT_APARTMENTTHREADED；
	// 返回 1 (S_FALSE) 表示该线程 COM 已初始化过，可忽略。
	hr, _, _ := procCoInit.Call(0, 2)
	if hr != S_OK && hr != 1 {
		return "", fmt.Errorf("CoInitializeEx failed: 0x%x", hr)
	}
	defer procCoUninit.Call()

	// 给自己进程「允许置前」权利。Win10 起后台进程调 SetForegroundWindow
	// 会被静默忽略；显式打开这个权限，配合下面 SetWindowPos(TOPMOST)
	// 才能在用户已经聚焦浏览器时抢到前台。
	pid, _, _ := procGetCurrentProcessId.Call()
	if pid != 0 {
		procAllowSetForegroundWindow.Call(pid)
	}

	// 创建 IFileOpenDialog（必须指定 CLSCTX_INPROC_SERVER，传 0 会返回
	// E_INVALIDARG，在部分系统上表现为静默失败）。
	var pUnk unsafe.Pointer
	hr, _, _ = procCoCreate.Call(
		uintptr(unsafe.Pointer(&CLSID_FileOpenDialog)),
		0,
		uintptr(CLSCTX_INPROC_SERVER),
		uintptr(unsafe.Pointer(&IID_IFileOpenDialog)),
		uintptr(unsafe.Pointer(&pUnk)),
	)
	if hr != S_OK || pUnk == nil {
		return "", fmt.Errorf("CoCreateInstance(FileOpenDialog) failed: 0x%x", hr)
	}
	defer releaseIUnknown(pUnk)

	// SetOptions(FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM)
	if hr := callVtbl(pUnk, vtblSetOptions, 0, uintptr(FOS_PICKFOLDERS|FOS_FORCEFILESYSTEM)); hr != S_OK {
		return "", fmt.Errorf("SetOptions failed: 0x%x", hr)
	}

	// SetTitle(title)
	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return "", fmt.Errorf("title encoding: %w", err)
	}
	if hr := callVtbl(pUnk, vtblSetTitle, 0, uintptr(unsafe.Pointer(titlePtr))); hr != S_OK {
		return "", fmt.Errorf("SetTitle failed: 0x%x", hr)
	}

	// Show(NULL owner)
	//
	// 5.6.1 关键：owner 显式传 NULL，不再用 GetForegroundWindow。
	// GetForegroundWindow 在浏览器抢焦点的瞬间拿到的是 chrome / edge / firefox
	// 的 HWND（跨进程），UIPI 限制下非浏览器进程拿不到这个 HWND 的 foreground
	// 状态，模态对话框会被卡在 Z 序下层；同时 Win10 的 Foreground Window
	// Restriction 会拒绝把这个跨进程 HWND 提升到前台。
	// NULL owner 让对话框作为独立 top-level 创建，Shell 自己的置顶策略直接
	// 生效；配合 Show 之后的 GetWindow + SetWindowPos(TOPMOST) 能稳定抢到焦点。
	hrShow := callVtbl(pUnk, vtblShow, 0, 0)
	if hrShow != S_OK {
		// 用户取消：0x800704C7 = ERROR_CANCELLED，0x80004004 = E_ABORT
		if hrShow == HRESULT_CANCELLED || hrShow == HRESULT_ABORT {
			return "", nil
		}
		return "", fmt.Errorf("Show failed: 0x%x", hrShow)
	}

	// Show 成功 → 对话框已显示。IFileOpenDialog 本身不暴露 GetWindow，需要
	// QueryInterface 到 IOleWindow 才能拿到 HWND，再强制置顶/抢焦点，避免被
	// 浏览器压在后面。
	var pOle unsafe.Pointer
	if hr := callVtbl(pUnk, 0, 0, uintptr(unsafe.Pointer(&IID_IOleWindow)), uintptr(unsafe.Pointer(&pOle))); hr != S_OK || pOle == nil {
		log.Printf("selectDirectory: QueryInterface(IOleWindow) failed hr=0x%x（对话框已显示但拿不到 HWND，后续取结果照常）", hr)
	} else {
		defer releaseIUnknown(pOle)
		var hwnd uintptr
		if hr := callVtbl(pOle, vtblOleWindowGetWindow, 0, uintptr(unsafe.Pointer(&hwnd))); hr != S_OK || hwnd == 0 {
			log.Printf("selectDirectory: IOleWindow::GetWindow failed hr=0x%x, hwnd=0x%x", hr, hwnd)
		} else {
			forceWindowToFront(hwnd)
		}
	}

	// GetResult
	var pItem unsafe.Pointer
	if hr := callVtbl(pUnk, vtblGetResult, 0, uintptr(unsafe.Pointer(&pItem))); hr != S_OK || pItem == nil {
		return "", fmt.Errorf("GetResult failed: 0x%x", hr)
	}
	defer releaseIUnknown(pItem)

	// IShellItem vtable：0=QI, 1=AddRef, 2=Release, 3=BindToHandler,
	//                   4=GetParent, 5=GetDisplayName, 6=GetAttributes
	var namePtr *uint16
	if hr := callVtbl(pItem, 5, 0, uintptr(SIGDN_FILESYSPATH), uintptr(unsafe.Pointer(&namePtr))); hr != S_OK || namePtr == nil {
		return "", fmt.Errorf("GetDisplayName failed: 0x%x", hr)
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(namePtr)))

	return windows.UTF16PtrToString(namePtr), nil
}

// forceWindowToFront 把对话框临时置顶抢一次焦点再降回 NOTOPMOST，避免一直
// 压住别的窗口。同时挪到主屏正中。
func forceWindowToFront(hwnd uintptr) {
	var rc struct{ Left, Top, Right, Bottom int32 }
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc))); ok == 0 {
		log.Printf("selectDirectory: GetWindowRect 失败 hwnd=0x%x", hwnd)
		return
	}
	w := rc.Right - rc.Left
	h := rc.Bottom - rc.Top
	if w <= 0 || h <= 0 {
		return
	}
	sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	if int32(sw) <= 0 || int32(sh) <= 0 {
		return
	}
	x := (int32(sw) - w) / 2
	y := (int32(sh) - h) / 2

	// 先临时 TOPMOST 抢一次前台；再降回 NOTOPMOST 释放「一直压顶」状态。
	procSetWindowPos.Call(
		hwnd, HWND_TOPMOST,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		SWP_SHOWWINDOW|SWP_NOACTIVATE,
	)
	procSetWindowPos.Call(
		hwnd, HWND_NOTOPMOST,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		SWP_SHOWWINDOW|SWP_NOACTIVATE,
	)
	// 再多调用一次 SetForegroundWindow（双重保险）。
	procSetForegroundWindow.Call(hwnd)
}

// callVtbl 调 p 的 vtable 偏移 idx 处的成员函数；retIdx 是「返回值位于栈中
// 的第几号槽」，因为 Go syscall 没法直接拿 syscall.SyscallN 的真实返回值，
// 所以用 retIdx 把 hr 找出来（vtable 调用的第一个返回值固定在 rax 里，但 Go
// 把 rax 放到 retIdx 索引位置）。
//
// 我们只关心 HRESULT，所以 retIdx=0 即可。
func callVtbl(p unsafe.Pointer, idx uintptr, retIdx uintptr, args ...uintptr) uintptr {
	vtbl := *(*unsafe.Pointer)(unsafe.Pointer(uintptr(p)))
	fn := *(*uintptr)(unsafe.Pointer(uintptr(vtbl) + idx*unsafe.Sizeof(uintptr(0))))
	all := append([]uintptr{uintptr(p)}, args...)
	r, _, _ := syscall.SyscallN(fn, all...)
	_ = retIdx
	return r
}

// releaseIUnknown 走 vtable 偏移 2（Release）来释放 COM 对象。
// IUnknown 的三方法布局在所有 COM 接口里都是固定 [QI, AddRef, Release]。
func releaseIUnknown(p unsafe.Pointer) {
	if p == nil {
		return
	}
	vtbl := *(*unsafe.Pointer)(unsafe.Pointer(uintptr(p)))
	fn := *(*uintptr)(unsafe.Pointer(uintptr(vtbl) + 2*unsafe.Sizeof(uintptr(0))))
	syscall.SyscallN(fn, uintptr(p))
}
