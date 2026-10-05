//go:build windows

package desktop

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ole32  = windows.NewLazySystemDLL("ole32.dll")
	user32 = windows.NewLazySystemDLL("user32.dll")

	procCoCreateInstance         = ole32.NewProc("CoCreateInstance")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
)

// ShowError shows a modal error message above other windows.
func ShowError(title, text string) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(text)
	_, _ = windows.MessageBox(0, m, t, windows.MB_OK|windows.MB_ICONERROR|windows.MB_TOPMOST|windows.MB_SETFOREGROUND)
}

// COM identifiers and constants for the folder picker (shobjidl.h).
var (
	clsidFileOpenDialog = windows.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE,
		Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileOpenDialog = windows.GUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768,
		Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
)

const (
	fosPickFolders     = 0x20
	fosForceFileSystem = 0x40
	fosPathMustExist   = 0x800
	sigdnFileSysPath   = 0x80058000
	hrCancelled        = 0x800704C7 // HRESULT_FROM_WIN32(ERROR_CANCELLED)
	sFalse             = syscall.Errno(1)

	// vtable slots: IUnknown, then IModalWindow / IFileDialog / IShellItem.
	slotRelease        = 2
	slotShow           = 3
	slotSetOptions     = 9
	slotGetOptions     = 10
	slotSetTitle       = 17
	slotGetResult      = 20
	slotGetDisplayName = 5 // IShellItem
)

// comObject is a COM interface pointer: its first word points to the vtable.
type comObject struct{ vtbl *[32]uintptr }

func (o *comObject) call(slot int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[slot], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

// pickMu allows one folder dialog at a time.
var pickMu sync.Mutex

// PickFolder shows the Windows folder picker and returns the chosen folder,
// or "" when the user cancels.
func PickFolder(title string) (string, error) {
	pickMu.Lock()
	defer pickMu.Unlock()

	type result struct {
		path string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		// The COM apartment and the dialog's windows belong to one OS thread.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		path, err := pickFolder(title)
		done <- result{path, err}
	}()
	r := <-done
	return r.path, r.err
}

func pickFolder(title string) (string, error) {
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err != nil && err != sFalse {
		return "", fmt.Errorf("CoInitializeEx: %w", err)
	}
	defer windows.CoUninitialize()

	var dialog *comObject
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)), 0, windows.CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(&iidIFileOpenDialog)), uintptr(unsafe.Pointer(&dialog)))
	if hr != 0 {
		return "", fmt.Errorf("creating folder dialog: HRESULT 0x%08X", uint32(hr))
	}
	defer dialog.call(slotRelease)

	var options uint32
	dialog.call(slotGetOptions, uintptr(unsafe.Pointer(&options)))
	dialog.call(slotSetOptions, uintptr(options|fosPickFolders|fosForceFileSystem|fosPathMustExist))
	if t, err := windows.UTF16PtrFromString(title); err == nil {
		dialog.call(slotSetTitle, uintptr(unsafe.Pointer(t)))
	}

	owner := newOwnerWindow()
	defer procDestroyWindow.Call(owner)

	hr = dialog.call(slotShow, owner)
	if uint32(hr) == hrCancelled {
		return "", nil
	}
	if hr != 0 {
		return "", fmt.Errorf("showing folder dialog: HRESULT 0x%08X", uint32(hr))
	}

	var item *comObject
	if hr := dialog.call(slotGetResult, uintptr(unsafe.Pointer(&item))); hr != 0 {
		return "", fmt.Errorf("reading folder dialog result: HRESULT 0x%08X", uint32(hr))
	}
	defer item.call(slotRelease)

	var name *uint16
	if hr := item.call(slotGetDisplayName, sigdnFileSysPath, uintptr(unsafe.Pointer(&name))); hr != 0 {
		return "", fmt.Errorf("reading folder path: HRESULT 0x%08X", uint32(hr))
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(name))
	return windows.UTF16PtrToString(name), nil
}

// newOwnerWindow creates an invisible topmost window in the middle of the
// screen to own the dialog. The browser has the focus when the user clicks
// "Browse", and without a topmost owner the dialog can open behind it.
func newOwnerWindow() uintptr {
	const (
		wsExTopmost    = 0x8
		wsExToolWindow = 0x80
		wsPopup        = 0x80000000
	)
	class, _ := windows.UTF16PtrFromString("STATIC")
	cx, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
	cy, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	hwnd, _, _ := procCreateWindowExW.Call(wsExTopmost|wsExToolWindow, uintptr(unsafe.Pointer(class)), 0,
		wsPopup, cx/2, cy/2, 0, 0, 0, 0, 0, 0)

	// Windows only lets the foreground thread hand over the focus; borrow its
	// input queue for a moment so the dialog receives the keyboard.
	if fg, _, _ := procGetForegroundWindow.Call(); fg != 0 {
		fgThread, _, _ := procGetWindowThreadProcessId.Call(fg, 0)
		ourThread := uintptr(windows.GetCurrentThreadId())
		if fgThread != ourThread {
			procAttachThreadInput.Call(ourThread, fgThread, 1)
			procSetForegroundWindow.Call(hwnd)
			procAttachThreadInput.Call(ourThread, fgThread, 0)
		}
	}
	return hwnd
}
