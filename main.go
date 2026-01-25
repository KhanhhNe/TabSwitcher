package main

import (
	"embed"
	_ "embed"
	"log"
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

var (
	user32             = windows.NewLazySystemDLL("user32.dll")
	procRegisterHotKey = user32.NewProc("RegisterHotKey")
	procGetMessage     = user32.NewProc("GetMessage")

	MOD_ALT   uint    = 0x0001
	VK_TAB    uint    = 0x09
	WM_HOTKEY uint32  = 0x0312
	NULL      uintptr = 0

	HK_ID_ALT_TAB int = 1
)

type WIN_MSG struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      POINT
}

type POINT struct {
	x int32
	y int32
}

func RegisterHotKey(hWnd uintptr, id int, fsModifiers uint, vk uint) bool {
	ret, _, err := procRegisterHotKey.Call(hWnd, uintptr(id), uintptr(fsModifiers), uintptr(vk))
	if err != windows.ERROR_SUCCESS {
		log.Println("RegisterHotKey error:", err)
	}
	return ret != 0
}

func GetMessage(msg *WIN_MSG, hWnd uintptr, wMsgFilterMin uint32, wMsgFilterMax uint32) bool {
	ret, _, err := procGetMessage.Call(uintptr(unsafe.Pointer(msg)), hWnd, uintptr(wMsgFilterMin), uintptr(wMsgFilterMax))
	if err != windows.ERROR_SUCCESS {
		log.Println("GetMessage error:", err)
	}
	return ret != 0
}

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
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Window 1",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// Create a goroutine that emits an event containing the current time every second.
	// The frontend can listen to this event and update the UI accordingly.
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.Event.Emit("time", now)
			time.Sleep(time.Second)
		}
	}()

	hwnd := window.NativeWindow()
	log.Println("Window handle:", hwnd)
	if RegisterHotKey(uintptr(hwnd), HK_ID_ALT_TAB, MOD_ALT, VK_TAB) {
		log.Println("Successfully registered ALT+TAB hotkey")
	} else {
		log.Println("Failed to register ALT+TAB hotkey")
	}

	go func() {
		msg := WIN_MSG{}
		for {
			if GetMessage(&msg, NULL, WM_HOTKEY, WM_HOTKEY) {
				if msg.message == WM_HOTKEY {
					if int(msg.wParam) == HK_ID_ALT_TAB {
						log.Println("ALT+TAB hotkey pressed")
					}
				}
			}
		}
	}()

	// Run the application. This blocks until the application has been exited.
	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
		return
	}
}
