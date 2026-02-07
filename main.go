package main

import (
	"changeme/win32"
	"embed"
	_ "embed"
	"fmt"
	"log"
	"sync"
	"time"
	"unsafe"

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
	hwnd     windows.HWND
	caption  string
	iconPath string
}

var userWindows sync.Map

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string]("time")
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
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Window 1",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})
	log.Println("Application set up finished.")

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
					fmt.Printf("(tab)")
				}
				if code == windows.VK_OEM_3 {
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
		err := win32.EnumDesktopWindows(
			win32.HDESK(0),
			(win32.WNDENUMPROC)(func(hWnd windows.HWND, lParam win32.LPARAM) uintptr {
				if win32.IsAltTabWindow(hWnd) {
					caption := make([]uint16, 256)
					_, err := win32.GetWindowTextW(hWnd, &caption[0], int32(len(caption)))
					if err != nil {
						return uintptr(1)
					}
					capStr := windows.UTF16ToString(caption)

					userWindows.Store(hWnd, UserWindow{hwnd: hWnd, caption: capStr})
				}
				return uintptr(1)
			}),
			win32.LPARAM(0),
		)
		if err != nil {
			log.Fatalf("Failed to enumerate windows: %v", err)
		}

		userWindows.Range(func(key, value any) bool {
			hwnd := key.(windows.HWND)
			userWindow := value.(UserWindow)

			fmt.Println("Window Handle:", hwnd, "UserWindow Struct:", userWindow)
			return true
		})
	}()

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

	// Run the application. This blocks until the application has been exited.
	log.Println("Running the application...")
	err = app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
		return
	}
}
