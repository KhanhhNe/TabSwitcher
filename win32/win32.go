package win32

import (
	"slices"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procLowLevelKeyboard    = user32.NewProc("LowLevelKeyboardProc")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessage          = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessage     = user32.NewProc("DispatchMessageW")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procEnumDesktopWindows  = user32.NewProc("EnumDesktopWindows")
	procGetWindowInfo       = user32.NewProc("GetWindowInfo")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procIsIconic            = user32.NewProc("IsIconic")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procGetShellWindow      = user32.NewProc("GetShellWindow")
	procGetAncestor         = user32.NewProc("GetAncestor")
	procGetLastActivePopup  = user32.NewProc("GetLastActivePopup")
	procGetClassNameW       = user32.NewProc("GetClassNameW")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procGetWindowLongPtrW   = user32.NewProc("GetWindowLongPtrW")

	dwmapi                     = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmGetWindowAttribute  = dwmapi.NewProc("DwmGetWindowAttribute")
)

const (
	WH_KEYBOARD_LL = 13
	WH_KEYBOARD    = 2

	WM_KEYDOWN     = 256
	WM_SYSKEYDOWN  = 260
	WM_KEYUP       = 257
	WM_SYSKEYUP    = 261
	WM_KEYFIRST    = 256
	WM_KEYLAST     = 264
	WM_LBUTTONDOWN = 513
	WM_RBUTTONDOWN = 516

	WS_OVERLAPPED       = 0x00000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_THICKFRAME       = 0x00040000
	WS_MINIMIZEBOX      = 0x00020000
	WS_MAXIMIZEBOX      = 0x00010000
	WS_OVERLAPPEDWINDOW = WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU | WS_THICKFRAME | WS_MINIMIZEBOX | WS_MAXIMIZEBOX
	WS_VISIBLE          = 0x10000000

	// Extended window styles
	WS_EX_TOOLWINDOW = 0x00000080

	PM_NOREMOVE = 0x000
	PM_REMOVE   = 0x001
	PM_NOYIELD  = 0x002

	// GetAncestor flags
	GA_PARENT    = 1
	GA_ROOT      = 2
	GA_ROOTOWNER = 3

	// GetWindowLong indices
	GWL_EXSTYLE = -20

	// DWM window attributes
	DWMWA_CLOAKED = 14

	// DWM cloaked flags
	DWM_CLOAKED_APP       = 0x00000001
	DWM_CLOAKED_SHELL     = 0x00000002
	DWM_CLOAKED_INHERITED = 0x00000004

	// MaxLastActivePopupIterations limits iterations when finding last active popup
	MaxLastActivePopupIterations = 50

	NULL = 0
)

type (
	DWORD     uint32
	WPARAM    uintptr
	LPARAM    uintptr
	LRESULT   uintptr
	HANDLE    uintptr
	HINSTANCE HANDLE
	HHOOK     HANDLE
	HDESK     HANDLE
	WORD      uint16
)

type HOOKPROC func(int, WPARAM, LPARAM) LRESULT
type WNDENUMPROC func(windows.HWND, LPARAM) uintptr

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type KBDLLHOOKSTRUCT struct {
	VkCode      DWORD
	ScanCode    DWORD
	Flags       DWORD
	Time        DWORD
	DwExtraInfo uintptr
}

type WINDOWINFO struct {
	CbSize          DWORD
	RcWindow        RECT
	RcClient        RECT
	DwStyle         DWORD
	DwExStyle       DWORD
	DwWindowStatus  DWORD
	CxWindowBorders DWORD
	CyWindowBorders DWORD
	AtomWindowType  WORD
	WCreatorVersion WORD
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd162805.aspx
type POINT struct {
	X, Y int32
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/ms644958.aspx
type MSG struct {
	Hwnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

func SetWindowsHookExW(idHook int, lpfn HOOKPROC, hMod HINSTANCE, dwThreadId DWORD) (HHOOK, error) {
	ret, _, err := procSetWindowsHookExW.Call(
		uintptr(idHook),
		syscall.NewCallback(lpfn),
		uintptr(hMod),
		uintptr(dwThreadId),
	)
	if ret == 0 {
		return 0, err
	}
	return HHOOK(ret), nil
}

func CallNextHookEx(hhk HHOOK, nCode int, wParam WPARAM, lParam LPARAM) LRESULT {
	ret, _, _ := procCallNextHookEx.Call(
		uintptr(hhk),
		uintptr(nCode),
		uintptr(wParam),
		uintptr(lParam),
	)
	return LRESULT(ret)
}

func UnhookWindowsHookEx(hhk HHOOK) error {
	ret, _, err := procUnhookWindowsHookEx.Call(
		uintptr(hhk),
	)
	if ret != 0 {
		return err
	}
	return nil
}

func GetMessage(msg *MSG, hwnd windows.HWND, msgFilterMin uint32, msgFilterMax uint32) (int, error) {
	ret, _, err := procGetMessage.Call(
		uintptr(unsafe.Pointer(msg)),
		uintptr(hwnd),
		uintptr(msgFilterMin),
		uintptr(msgFilterMax))
	if int(ret) < -1 {
		return int(ret), err
	}
	return int(ret), nil
}

func TranslateMessage(msg *MSG) bool {
	ret, _, _ := procTranslateMessage.Call(
		uintptr(unsafe.Pointer(msg)))
	return ret != 0
}

func DispatchMessage(msg *MSG) uintptr {
	ret, _, _ := procDispatchMessage.Call(
		uintptr(unsafe.Pointer(msg)))
	return ret
}

func LowLevelKeyboardProc(nCode int, wParam WPARAM, lParam LPARAM) LRESULT {
	ret, _, _ := procLowLevelKeyboard.Call(
		uintptr(nCode),
		uintptr(wParam),
		uintptr(lParam),
	)
	return LRESULT(ret)
}

func EnumWindows(enumFunc WNDENUMPROC, lParam LPARAM) error {
	ret, _, err := procEnumWindows.Call(
		syscall.NewCallback(enumFunc),
		uintptr(lParam),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func EnumDesktopWindows(hDesktop HDESK, enumFunc WNDENUMPROC, lParam LPARAM) error {
	ret, _, err := procEnumDesktopWindows.Call(
		uintptr(hDesktop),
		syscall.NewCallback(enumFunc),
		uintptr(lParam),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func IswindowVisible(hwnd windows.HWND) bool {
	ret, _, _ := procIsWindowVisible.Call(
		uintptr(hwnd),
	)
	return ret != 0
}

func IsIconic(hwnd windows.HWND) bool {
	ret, _, _ := procIsIconic.Call(
		uintptr(hwnd),
	)
	return ret != 0
}

func GetWindowInfo(hwnd windows.HWND, pwi *WINDOWINFO) error {
	ret, _, err := procGetWindowInfo.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(pwi)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func GetWindowRect(hwnd windows.HWND, rect *RECT) error {
	ret, _, err := procGetWindowRect.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(rect)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func GetWindowTextW(hwnd windows.HWND, str *uint16, maxCount int32) (int32, error) {
	ret, _, err := procGetWindowTextW.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(str)),
		uintptr(maxCount),
	)
	if ret == 0 {
		return 0, err
	}
	return int32(ret), nil
}

func GetShellWindow() windows.HWND {
	ret, _, _ := procGetShellWindow.Call()
	return windows.HWND(ret)
}

func GetAncestor(hwnd windows.HWND, gaFlags uint32) windows.HWND {
	ret, _, _ := procGetAncestor.Call(
		uintptr(hwnd),
		uintptr(gaFlags),
	)
	return windows.HWND(ret)
}

func GetLastActivePopup(hwnd windows.HWND) windows.HWND {
	ret, _, _ := procGetLastActivePopup.Call(
		uintptr(hwnd),
	)
	return windows.HWND(ret)
}

func GetClassNameW(hwnd windows.HWND, str *uint16, maxCount int32) (int32, error) {
	ret, _, err := procGetClassNameW.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(str)),
		uintptr(maxCount),
	)
	if ret == 0 {
		return 0, err
	}
	return int32(ret), nil
}

func GetWindowLongPtrW(hwnd windows.HWND, nIndex int32) uintptr {
	ret, _, _ := procGetWindowLongPtrW.Call(
		uintptr(hwnd),
		uintptr(nIndex),
	)
	return ret
}

func DwmGetWindowAttribute(hwnd windows.HWND, dwAttribute uint32, pvAttribute unsafe.Pointer, cbAttribute uint32) error {
	ret, _, _ := procDwmGetWindowAttribute.Call(
		uintptr(hwnd),
		uintptr(dwAttribute),
		uintptr(pvAttribute),
		uintptr(cbAttribute),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

// WindowsClassNamesToSkip defines window classes that should not be activated
var WindowsClassNamesToSkip = []string{
	"Shell_TrayWnd",
	"DV2ControlHost",
	"MsgrIMEWindowClass",
	"SysShadow",
	"Button",
}

// GetLastVisibleActivePopUpOfWindow finds the last visible active popup of a window
func GetLastVisibleActivePopUpOfWindow(hwnd windows.HWND) windows.HWND {
	level := MaxLastActivePopupIterations
	currentWindow := hwnd

	for level > 0 {
		level--
		lastPopUp := GetLastActivePopup(currentWindow)

		if windows.IsWindowVisible(lastPopUp) {
			return lastPopUp
		}

		if lastPopUp == currentWindow {
			return 0
		}

		currentWindow = lastPopUp
	}

	return 0
}

// EligibleForActivation determines if a window is eligible for activation
// Based on: http://stackoverflow.com/questions/210504/enumerate-windows-like-alt-tab-does
func EligibleForActivation(hwnd windows.HWND, shellWindow windows.HWND) bool {
	if hwnd == shellWindow {
		return false
	}

	root := GetAncestor(hwnd, GA_ROOTOWNER)

	if GetLastVisibleActivePopUpOfWindow(root) != hwnd {
		return false
	}

	className := make([]uint16, 256)
	length, err := GetClassNameW(hwnd, &className[0], int32(len(className)))
	if err != nil || length == 0 {
		return false
	}

	classNameStr := windows.UTF16ToString(className)

	// Check if class name is in the skip list
	if slices.Contains(WindowsClassNamesToSkip, classNameStr) {
		return false
	}

	// Check for WMP9MediaBarFlyout (Windows Media Player's "now playing" taskbar-toolbar)
	if len(classNameStr) >= 18 && classNameStr[:18] == "WMP9MediaBarFlyout" {
		return false
	}

	return true
}

// IsAltTabWindow determines if a window should appear in Alt+Tab
// This is a more modern approach that includes DWM cloaking detection
func IsAltTabWindow(hwnd windows.HWND) bool {
	// The window must be visible
	if !windows.IsWindowVisible(hwnd) {
		return false
	}

	// The window must be a root owner
	if GetAncestor(hwnd, GA_ROOTOWNER) != hwnd {
		return false
	}

	// The window must not be cloaked by the shell
	var cloaked uint32
	err := DwmGetWindowAttribute(hwnd, DWMWA_CLOAKED, unsafe.Pointer(&cloaked), uint32(unsafe.Sizeof(cloaked)))
	if err == nil && cloaked == DWM_CLOAKED_SHELL {
		return false
	}

	// The window must not have the extended style WS_EX_TOOLWINDOW
	exStyle := GetWindowLongPtrW(hwnd, GWL_EXSTYLE)
	if (exStyle & WS_EX_TOOLWINDOW) != 0 {
		return false
	}

	return true
}
