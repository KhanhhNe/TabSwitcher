package main

import (
	"embed"
	_ "embed"
	"fmt"
	"log"
	"sync"
	"tabswitcher/win32"
	"time"
	"unsafe"

	"github.com/shahfarhadreza/go-gdiplus"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

type UserWindow struct {
	touched      bool
	IsForeground bool
	LastActive   int
	Hwnd         windows.HWND
	Caption      string
	IconBase64   string
	IconSource   string
	ExePath      string
}

var userWindows sync.Map

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[[]UserWindow]("userWindowsChanged")
	application.RegisterEvent[string]("systemKeyPressed")
	application.RegisterEvent[windows.HWND]("activateWindow")
}

var (
	gdipInput  = gdiplus.GdiplusStartupInput{GdiplusVersion: 1}
	gdipOutput = gdiplus.GdiplusStartupOutput{}
	pngClsId   = &windows.GUID{}
)

func GetAltTabWindows() []UserWindow {
	foreground := win32.GetForegroundWindow()

	userWindows.Range(func(key, val any) bool {
		window := val.(UserWindow)
		window.touched = false
		userWindows.Store(key, window)
		return true
	})

	for res := range win32.ListDesktopWindows() {
		if res.Error != nil {
			log.Printf("Error enumerating windows: %v", res.Error)
			continue
		}

		hWnd := res.Window
		if win32.IsAltTabWindow(hWnd) {
			caption := make([]uint16, 256)
			_, err := win32.GetWindowTextW(hWnd, &caption[0], int32(len(caption)))
			if err != nil {
				continue
			}
			capStr := windows.UTF16ToString(caption)

			// Get the executable path for this window
			var processId win32.DWORD
			win32.GetWindowThreadProcessId(hWnd, &processId)
			exePath := ""
			hProcess, err := windows.OpenProcess(win32.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(processId))
			if err == nil {
				defer windows.CloseHandle(hProcess)
				var exePathBuf [win32.MAX_PATH]uint16
				exePathSize := win32.DWORD(win32.MAX_PATH)
				err = win32.QueryFullProcessImageNameW(hProcess, 0, &exePathBuf[0], &exePathSize)
				if err == nil {
					exePath = windows.UTF16ToString(exePathBuf[:])
				}
			}

			iconInfo := win32.GetWindowIcon(hWnd, exePath)
			iconB64, err := win32.HICONToBase64Png(iconInfo.Icon, pngClsId)
			if err != nil {
				continue
			}

			isForeground := foreground == hWnd

			win, ok := userWindows.Load(hWnd)
			if ok {
				window := win.(UserWindow)
				window.touched = true
				window.Caption = capStr
				window.IconBase64 = "data:image/png;base64," + iconB64
				window.IconSource = iconInfo.Source
				window.IsForeground = isForeground
				window.ExePath = exePath
				userWindows.Store(hWnd, window)
			} else {
				userWindows.Store(hWnd, UserWindow{
					touched:      true,
					Hwnd:         hWnd,
					Caption:      capStr,
					IconBase64:   "data:image/png;base64," + iconB64,
					IconSource:   iconInfo.Source,
					IsForeground: isForeground,
					ExePath:      exePath,
				})
			}
		}
	}

	var userWindowsSlice []UserWindow
	userWindows.Range(func(key, val any) bool {
		window := val.(UserWindow)
		if window.touched {
			userWindowsSlice = append(userWindowsSlice, window)
		} else {
			userWindows.Delete(key)
		}
		return true
	})

	return userWindowsSlice
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "TabSwitcher",
		Description: "A demo of using raw HTML & CSS",
		Services: []application.Service{
			application.NewService(&GreetService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		// BackgroundColour: application.NewRGBA(239, 244, 249, 50),
		BackgroundType:  application.BackgroundTypeTranslucent,
		URL:             "/",
		AlwaysOnTop:     true,
		Frameless:       true,
		Width:           400,
		Height:          200,
		InitialPosition: application.WindowCentered,
		Windows: application.WindowsWindow{
			BackdropType:    application.Acrylic,
			HiddenOnTaskbar: true,
		},
	})
	log.Println("Application set up finished.")

	window.Show()

	app.Event.On("activateWindow", func(event *application.CustomEvent) {
		hwnd := event.Data.(windows.HWND)
		success := win32.SetForegroundWindow(hwnd)
		if !success {
			log.Printf("Failed to set window %v to foreground\n", hwnd)
			return
		}

		win, ok := userWindows.Load(hwnd)
		if ok {
			window := win.(UserWindow)
			window.LastActive = int(time.Now().UnixMilli())
			userWindows.Store(hwnd, window)

			log.Printf("Activated window: %s\n", window.Caption)
			app.Event.Emit("userWindowsChanged", GetAltTabWindows())
		}
	})

	ret := gdiplus.GdiplusStartup(&gdipInput, &gdipOutput)
	fmt.Println(ret.String())
	defer gdiplus.GdiplusShutdown()

	clsId, err := win32.GetEncoderClsid("image/png")
	pngClsId = clsId

	// Create a goroutine that emits an event containing the current time every second.
	// The frontend can listen to this event and update the UI accordingly.
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.Event.Emit("time", now)
			time.Sleep(time.Second)
		}
	}()

	hook, err := win32.SetWindowsHookExW(
		win32.WH_KEYBOARD_LL,
		(win32.HOOKPROC)(func(nCode int, wParam win32.WPARAM, lParam win32.LPARAM) win32.LRESULT {
			// SYSKEYDOWN is for Alt+Key combinations & F10
			if nCode == 0 && wParam == win32.WM_SYSKEYDOWN {
				fmt.Print("key pressed:")
				kbdstruct := (*win32.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
				code := byte(kbdstruct.VkCode)
				if code == windows.VK_TAB {
					app.Event.Emit("systemKeyPressed", "tab")
					fmt.Printf("(tab)")
				}
				if code == windows.VK_OEM_3 {
					app.Event.Emit("systemKeyPressed", "tilde")
					fmt.Printf("(`~)")
				}
				fmt.Printf("%q\n", code)
			}
			return win32.CallNextHookEx(win32.HHOOK(0), nCode, wParam, lParam)
		}),
		0,
		0,
	)
	if err != nil {
		log.Fatal("Failed to set keyboard hook:", err)
	}
	log.Println("Keyboard hook installed")

	go func() {
		msg := &win32.MSG{}
		for {
			if _, err := win32.GetMessage(msg, 0, 0, 0); err != nil {
				break
			}

			win32.TranslateMessage(msg)
			win32.DispatchMessage(msg)
		}

		win32.UnhookWindowsHookEx(hook)
		hook = 0
	}()

	go func() {
		for {
			app.Event.Emit("userWindowsChanged", GetAltTabWindows())
			<-time.After(time.Second)
		}
	}()

	// Run the application. This blocks until the application has been exited.
	log.Println("Running the application...")
	err = app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
		return
	}
}
