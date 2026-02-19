package win32

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"slices"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procSetWindowsHookExW        = user32.NewProc("SetWindowsHookExW")
	procLowLevelKeyboard         = user32.NewProc("LowLevelKeyboardProc")
	procCallNextHookEx           = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx      = user32.NewProc("UnhookWindowsHookEx")
	procGetMessage               = user32.NewProc("GetMessageW")
	procTranslateMessage         = user32.NewProc("TranslateMessage")
	procDispatchMessage          = user32.NewProc("DispatchMessageW")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procEnumDesktopWindows       = user32.NewProc("EnumDesktopWindows")
	procGetWindowInfo            = user32.NewProc("GetWindowInfo")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetShellWindow           = user32.NewProc("GetShellWindow")
	procGetAncestor              = user32.NewProc("GetAncestor")
	procGetLastActivePopup       = user32.NewProc("GetLastActivePopup")
	procGetClassNameW            = user32.NewProc("GetClassNameW")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procGetWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")
	procGetClassLongPtrW         = user32.NewProc("GetClassLongPtrW")
	procSendMessageW             = user32.NewProc("SendMessageW")
	procSendMessageCallbackW     = user32.NewProc("SendMessageCallbackW")
	procLoadIconW                = user32.NewProc("LoadIconW")
	procGetIconInfo              = user32.NewProc("GetIconInfo")
	procGetIconInfoExW           = user32.NewProc("GetIconInfoExW")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")

	shell32            = windows.NewLazySystemDLL("shell32.dll")
	procExtractIconExW = shell32.NewProc("ExtractIconExW")

	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")

	dwmapi                    = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")

	gdi32            = windows.NewLazySystemDLL("gdi32.dll")
	procDeleteObject = gdi32.NewProc("DeleteObject")
	procGetObjectW   = gdi32.NewProc("GetObjectW")
	procGetDC        = user32.NewProc("GetDC")
	procReleaseDC    = user32.NewProc("ReleaseDC")
	procGetDIBits    = gdi32.NewProc("GetDIBits")

	gdiplusDLL                   = windows.NewLazySystemDLL("gdiplus.dll")
	procGdipGetImageEncodersSize = gdiplusDLL.NewProc("GdipGetImageEncodersSize")
	procGdipGetImageEncoders     = gdiplusDLL.NewProc("GdipGetImageEncoders")
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
	WM_GETICON     = 0x007F

	// Icon types for WM_GETICON
	ICON_SMALL  = 0
	ICON_BIG    = 1
	ICON_SMALL2 = 2

	// Standard icon IDs
	IDI_APPLICATION = 32512
	IDI_WINLOGO     = 32517

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
	GWL_EXSTYLE    = -20
	GWLP_HINSTANCE = -6

	// GetClassLong indices
	GCLP_HICON   = -14
	GCLP_HICONSM = -34

	// DWM window attributes
	DWMWA_CLOAKED = 14

	// DWM cloaked flags
	DWM_CLOAKED_APP       = 0x00000001
	DWM_CLOAKED_SHELL     = 0x00000002
	DWM_CLOAKED_INHERITED = 0x00000004

	// MaxLastActivePopupIterations limits iterations when finding last active popup
	MaxLastActivePopupIterations = 50

	// MAX_PATH is the maximum path length in Windows
	MAX_PATH = 260

	// Bitmap compression types
	BI_RGB = 0

	// DIB color table identifiers
	DIB_RGB_COLORS = 0

	// Process access rights
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

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
	HICON     HANDLE
	HBITMAP   HANDLE
	HGDIOBJ   HANDLE
	HDC       HANDLE
	WORD      uint16
	BOOL      int32
	LONG      int32
)

type HOOKPROC func(int, WPARAM, LPARAM) LRESULT
type WNDENUMPROC func(windows.HWND, LPARAM) uintptr
type SENDASYNCPROC func(windows.HWND, uint32, uintptr, LRESULT) uintptr

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

// BITMAP contains information about a bitmap
type BITMAP struct {
	BmType       LONG
	BmWidth      LONG
	BmHeight     LONG
	BmWidthBytes LONG
	BmPlanes     WORD
	BmBitsPixel  WORD
	BmBits       uintptr
}

// BITMAPINFOHEADER contains information about the dimensions and color format of a DIB
type BITMAPINFOHEADER struct {
	BiSize          DWORD
	BiWidth         LONG
	BiHeight        LONG
	BiPlanes        WORD
	BiBitCount      WORD
	BiCompression   DWORD
	BiSizeImage     DWORD
	BiXPelsPerMeter LONG
	BiYPelsPerMeter LONG
	BiClrUsed       DWORD
	BiClrImportant  DWORD
}

// ICONINFO contains information about an icon or a cursor
type ICONINFO struct {
	FIcon    BOOL
	XHotspot DWORD
	YHotspot DWORD
	HbmMask  HBITMAP
	HbmColor HBITMAP
}

// ICONINFOEXW contains extended information about an icon or a cursor
type ICONINFOEXW struct {
	CbSize    DWORD
	FIcon     BOOL
	XHotspot  DWORD
	YHotspot  DWORD
	HbmMask   HBITMAP
	HbmColor  HBITMAP
	WResID    WORD
	SzModName [MAX_PATH]uint16
	SzResName [MAX_PATH]uint16
}

// ImageCodecInfo contains information about an image encoder/decoder
type ImageCodecInfo struct {
	Clsid             windows.GUID
	FormatID          windows.GUID
	CodecName         *uint16
	DllName           *uint16
	FormatDescription *uint16
	FilenameExtension *uint16
	MimeType          *uint16
	Flags             DWORD
	Version           DWORD
	SigCount          DWORD
	SigSize           DWORD
	SigPattern        *byte
	SigMask           *byte
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

type EnumWindowsResult struct {
	Window windows.HWND
	Error  error
}

func enumDesktopWindowsCallback(hwnd windows.HWND, lParam LPARAM) uintptr {
	ch := (*chan EnumWindowsResult)(unsafe.Pointer(lParam))
	*ch <- EnumWindowsResult{Window: hwnd}
	return 1
}

func ListDesktopWindows() chan EnumWindowsResult {
	ch := make(chan EnumWindowsResult)
	lParam := LPARAM(unsafe.Pointer(&ch))

	go func() {
		err := EnumDesktopWindows(0, (WNDENUMPROC)(enumDesktopWindowsCallback), lParam)
		if err != nil {
			ch <- EnumWindowsResult{Error: err}
		}
		close(ch)
	}()

	return ch
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

func GetClassLongPtrW(hwnd windows.HWND, nIndex int32) (uintptr, error) {
	ret, _, err := procGetClassLongPtrW.Call(
		uintptr(hwnd),
		uintptr(nIndex),
	)
	if ret == 0 {
		return 0, err
	}
	return ret, nil
}

func SendMessage(hwnd windows.HWND, msg uint32, wParam WPARAM, lParam LPARAM) LRESULT {
	ret, _, _ := procSendMessageW.Call(
		uintptr(hwnd),
		uintptr(msg),
		uintptr(wParam),
		uintptr(lParam),
	)
	return LRESULT(ret)
}

func SendMessageCallbackW(hwnd windows.HWND, msg uint32, wParam WPARAM, lParam LPARAM, lpResultCallBack SENDASYNCPROC, dwData uintptr) error {
	ret, _, err := procSendMessageCallbackW.Call(
		uintptr(hwnd),
		uintptr(msg),
		uintptr(wParam),
		uintptr(lParam),
		syscall.NewCallback(lpResultCallBack),
		dwData,
	)
	if ret == 0 {
		return err
	}
	return nil
}

func LoadIconW(hInstance HINSTANCE, lpIconName uintptr) HICON {
	ret, _, _ := procLoadIconW.Call(
		uintptr(hInstance),
		lpIconName,
	)
	return HICON(ret)
}

func GetIconInfo(hIcon HICON, piconinfo *ICONINFO) error {
	ret, _, err := procGetIconInfo.Call(
		uintptr(hIcon),
		uintptr(unsafe.Pointer(piconinfo)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func GetIconInfoExW(hIcon HICON, piconinfo *ICONINFOEXW) error {
	// Set the size before calling
	piconinfo.CbSize = DWORD(unsafe.Sizeof(*piconinfo))

	ret, _, err := procGetIconInfoExW.Call(
		uintptr(hIcon),
		uintptr(unsafe.Pointer(piconinfo)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

// MAKEINTRESOURCEW converts an integer resource ID to a resource pointer
func MAKEINTRESOURCEW(id uintptr) uintptr {
	return id & 0xFFFF
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

func DeleteObject(hObject HGDIOBJ) bool {
	ret, _, _ := procDeleteObject.Call(uintptr(hObject))
	return ret != 0
}

func GetObjectW(hObject HGDIOBJ, cbBuffer int32, lpvObject unsafe.Pointer) int32 {
	ret, _, _ := procGetObjectW.Call(
		uintptr(hObject),
		uintptr(cbBuffer),
		uintptr(lpvObject),
	)
	return int32(ret)
}

func GetDC(hwnd windows.HWND) HDC {
	ret, _, _ := procGetDC.Call(uintptr(hwnd))
	return HDC(ret)
}

func ReleaseDC(hwnd windows.HWND, hdc HDC) int32 {
	ret, _, _ := procReleaseDC.Call(
		uintptr(hwnd),
		uintptr(hdc),
	)
	return int32(ret)
}

func GetDIBits(hdc HDC, hbmp HBITMAP, uStartScan uint32, cScanLines uint32, lpvBits unsafe.Pointer, lpbi *BITMAPINFOHEADER, uUsage uint32) int32 {
	ret, _, _ := procGetDIBits.Call(
		uintptr(hdc),
		uintptr(hbmp),
		uintptr(uStartScan),
		uintptr(cScanLines),
		uintptr(lpvBits),
		uintptr(unsafe.Pointer(lpbi)),
		uintptr(uUsage),
	)
	return int32(ret)
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
	err := DwmGetWindowAttribute(
		hwnd,
		DWMWA_CLOAKED,
		unsafe.Pointer(&cloaked),
		uint32(unsafe.Sizeof(cloaked)),
	)
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

type IconInfo struct {
	Icon   HICON
	Source string
}

func GetWindowIcon(hwnd windows.HWND, exePath string) IconInfo {
	// Try WM_GETICON first
	icon := SendMessage(
		hwnd,
		WM_GETICON,
		ICON_BIG,
		0,
	)
	if icon != 0 {
		return IconInfo{
			Icon:   HICON(icon),
			Source: "WM_GETICON",
		}
	}

	icon = SendMessage(
		hwnd,
		WM_GETICON,
		ICON_SMALL,
		0,
	)
	if icon != 0 {
		return IconInfo{
			Icon:   HICON(icon),
			Source: "WM_GETICON_S",
		}
	}

	icon = SendMessage(
		hwnd,
		WM_GETICON,
		ICON_SMALL2,
		0,
	)
	if icon != 0 {
		return IconInfo{
			Icon:   HICON(icon),
			Source: "WM_GETICON_S2",
		}
	}

	// Try getting icon from window class
	ret, _ := GetClassLongPtrW(hwnd, GCLP_HICON)
	if ret != 0 {
		return IconInfo{
			Icon:   HICON(ret),
			Source: "GCLP_HICON",
		}
	}

	ret, _ = GetClassLongPtrW(hwnd, GCLP_HICONSM)
	if ret != 0 {
		return IconInfo{
			Icon:   HICON(ret),
			Source: "GCLP_HICONSM",
		}
	}

	// Try to extract icon from the executable path if provided
	if exePath != "" {
		exePathUTF16, err := windows.UTF16PtrFromString(exePath)
		if err == nil {
			var largeIcon HICON
			numIcons := ExtractIconExW(exePathUTF16, 0, &largeIcon, nil, 1)
			if numIcons > 0 && largeIcon != 0 {
				return IconInfo{
					Icon:   largeIcon,
					Source: "ExtractIconEx",
				}
			}
		}
	}

	// Fall back to default system icon
	return IconInfo{
		Icon:   LoadIconW(0, MAKEINTRESOURCEW(IDI_APPLICATION)),
		Source: "IDI_APPLICATION",
	}
}

func GetForegroundWindow() windows.HWND {
	ret, _, _ := procGetForegroundWindow.Call()
	return windows.HWND(ret)
}

func SetForegroundWindow(hwnd windows.HWND) bool {
	ret, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	return ret != 0
}

func GetWindowThreadProcessId(hwnd windows.HWND, lpdwProcessId *DWORD) DWORD {
	ret, _, _ := procGetWindowThreadProcessId.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(lpdwProcessId)),
	)
	return DWORD(ret)
}

func QueryFullProcessImageNameW(hProcess windows.Handle, dwFlags DWORD, lpExeName *uint16, lpdwSize *DWORD) error {
	ret, _, err := procQueryFullProcessImageNameW.Call(
		uintptr(hProcess),
		uintptr(dwFlags),
		uintptr(unsafe.Pointer(lpExeName)),
		uintptr(unsafe.Pointer(lpdwSize)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func ExtractIconExW(lpszFile *uint16, nIconIndex int32, phiconLarge *HICON, phiconSmall *HICON, nIcons uint32) uint32 {
	ret, _, _ := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(lpszFile)),
		uintptr(nIconIndex),
		uintptr(unsafe.Pointer(phiconLarge)),
		uintptr(unsafe.Pointer(phiconSmall)),
		uintptr(nIcons),
	)
	return uint32(ret)
}

// GetEncoderClsid finds the CLSID of an image encoder by MIME type
// mimeType examples: "image/png", "image/jpeg", "image/bmp", "image/gif"
// Returns the CLSID and an error if the encoder is not found
func GetEncoderClsid(mimeType string) (*windows.GUID, error) {
	var num, size uint32

	// Get the number of encoders and size of array
	ret, _, _ := procGdipGetImageEncodersSize.Call(
		uintptr(unsafe.Pointer(&num)),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret != 0 { // GDI+ returns 0 for Ok
		return nil, syscall.Errno(ret)
	}

	if size == 0 {
		return nil, syscall.EINVAL
	}

	// Allocate memory for the encoder array
	buffer := make([]byte, size)
	pImageCodecInfo := (*ImageCodecInfo)(unsafe.Pointer(&buffer[0]))

	// Get all encoder information
	ret, _, _ = procGdipGetImageEncoders.Call(
		uintptr(num),
		uintptr(size),
		uintptr(unsafe.Pointer(pImageCodecInfo)),
	)
	if ret != 0 {
		return nil, syscall.Errno(ret)
	}

	// Iterate through encoders to find matching MIME type
	encoders := unsafe.Slice(pImageCodecInfo, num)
	for i := uint32(0); i < num; i++ {
		// Compare MIME types
		if windows.UTF16PtrToString(encoders[i].MimeType) == mimeType {
			return &encoders[i].Clsid, nil
		}
	}

	return nil, syscall.ENOENT
}

func HICONToBase64Png(icon HICON, pngClsId *windows.GUID) (string, error) {
	// Get icon information
	var iconInfo ICONINFO
	err := GetIconInfo(icon, &iconInfo)
	if err != nil {
		return "", fmt.Errorf("GetIconInfo failed: %w", err)
	}

	// Delete mask bitmap as we don't need it
	DeleteObject(HGDIOBJ(iconInfo.HbmMask))
	defer DeleteObject(HGDIOBJ(iconInfo.HbmColor))

	// Get bitmap object information
	var bitmap BITMAP
	result := GetObjectW(
		HGDIOBJ(iconInfo.HbmColor),
		int32(unsafe.Sizeof(bitmap)),
		unsafe.Pointer(&bitmap),
	)
	if result == 0 {
		return "", fmt.Errorf("GetObjectW failed")
	}

	width := uint32(bitmap.BmWidth)
	height := uint32(bitmap.BmHeight)
	bufSize := int(width) * int(height) * 4
	buf := make([]byte, bufSize)

	// Get device context
	dc := GetDC(0)
	if dc == 0 {
		return "", fmt.Errorf("GetDC failed")
	}
	defer ReleaseDC(0, dc)

	// Setup bitmap info header
	bitmapInfo := BITMAPINFOHEADER{
		BiSize:        DWORD(unsafe.Sizeof(BITMAPINFOHEADER{})),
		BiWidth:       bitmap.BmWidth,
		BiHeight:      -bitmap.BmHeight, // Negative for top-down DIB
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: BI_RGB,
	}

	// Get DIB bits
	result = GetDIBits(
		dc,
		iconInfo.HbmColor,
		0,
		height,
		unsafe.Pointer(&buf[0]),
		&bitmapInfo,
		DIB_RGB_COLORS,
	)
	if result == 0 {
		return "", fmt.Errorf("GetDIBits failed")
	}

	// Swap B and R channels (BGRA to RGBA)
	for i := 0; i < len(buf); i += 4 {
		buf[i], buf[i+2] = buf[i+2], buf[i]
	}

	// Create RGBA image
	img := image.NewNRGBA(image.Rect(0, 0, int(width), int(height)))
	copy(img.Pix, buf)

	// Encode to PNG
	output := &bytes.Buffer{}
	err = png.Encode(output, img)
	if err != nil {
		return "", fmt.Errorf("PNG encode failed: %w", err)
	}

	// Return base64 encoded PNG
	return base64.StdEncoding.EncodeToString(output.Bytes()), nil
}
