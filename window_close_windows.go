//go:build windows

package main

import (
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	gwlpWndProc    = -4
	swHide         = 0
	swMinimize     = 6
	swRestore      = 9
	wmClose        = 0x0010
	wmSysCommand   = 0x0112
	scMinimize     = 0xF020
	sysCommandMask = 0xFFF0
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	procCallWindowProcW       = user32.NewProc("CallWindowProcW")
	procEnumWindows           = user32.NewProc("EnumWindows")
	procGetWindowTextW        = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcID = user32.NewProc("GetWindowThreadProcessId")
	procIsWindow              = user32.NewProc("IsWindow")
	procSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
	procSetWindowLongPtrW     = user32.NewProc("SetWindowLongPtrW")
	procShowWindow            = user32.NewProc("ShowWindow")

	closeToTrayInterceptor = &windowCloseInterceptor{}
)

type windowCloseInterceptor struct {
	mu              sync.Mutex
	app             *App
	hwnd            uintptr
	originalWndProc uintptr
	callback        uintptr
}

func installCloseToTrayHandler(app *App) {
	closeToTrayInterceptor.install(app)
}

func shouldHandleCloseToTrayInBeforeClose() bool {
	return false
}

func showMainWindowNative() bool {
	hwnd := closeToTrayInterceptor.windowHandle()
	if hwnd == 0 {
		return false
	}

	isWindow, _, _ := procIsWindow.Call(hwnd)
	if isWindow == 0 {
		return false
	}

	procShowWindow.Call(hwnd, swRestore)
	procSetForegroundWindow.Call(hwnd)
	return true
}

func (i *windowCloseInterceptor) install(app *App) {
	i.mu.Lock()
	if i.callback != 0 {
		i.mu.Unlock()
		return
	}
	i.app = app
	i.callback = syscall.NewCallback(i.windowProc)
	i.mu.Unlock()

	go i.installWhenWindowReady()
}

func (i *windowCloseInterceptor) windowHandle() uintptr {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.hwnd
}

func (i *windowCloseInterceptor) installWhenWindowReady() {
	for attempt := 0; attempt < 50; attempt++ {
		hwnd := findCodexSwitchWindow()
		if hwnd == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		i.mu.Lock()
		if i.hwnd != 0 {
			i.mu.Unlock()
			return
		}
		wndProcIndex := int32(gwlpWndProc)
		previous, _, err := procSetWindowLongPtrW.Call(hwnd, uintptr(wndProcIndex), i.callback)
		if previous == 0 {
			i.mu.Unlock()
			log.Printf("window close interceptor: SetWindowLongPtrW failed: %v", err)
			return
		}
		i.hwnd = hwnd
		i.originalWndProc = previous
		i.mu.Unlock()
		return
	}

	log.Println("window close interceptor: main window not found")
}

func (i *windowCloseInterceptor) windowProc(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	if msg == wmSysCommand && wParam&sysCommandMask == scMinimize {
		procShowWindow.Call(hwnd, swMinimize)
		return 0
	}
	if msg == wmClose && i.shouldHideOnClose() {
		procShowWindow.Call(hwnd, swHide)
		return 0
	}
	return i.callOriginal(hwnd, msg, wParam, lParam)
}

func (i *windowCloseInterceptor) shouldHideOnClose() bool {
	i.mu.Lock()
	app := i.app
	i.mu.Unlock()
	if app == nil || app.service == nil {
		return false
	}

	settings, err := app.service.GetSettings()
	if err != nil {
		log.Printf("window close interceptor: read settings failed: %v", err)
		return false
	}
	return settings.MinimizeToTrayOnClose
}

func (i *windowCloseInterceptor) callOriginal(hwnd uintptr, msg uint32, wParam uintptr, lParam uintptr) uintptr {
	i.mu.Lock()
	originalWndProc := i.originalWndProc
	i.mu.Unlock()
	if originalWndProc == 0 {
		return 0
	}

	result, _, _ := procCallWindowProcW.Call(originalWndProc, hwnd, uintptr(msg), wParam, lParam)
	return result
}

func findCodexSwitchWindow() uintptr {
	targetPID := uint32(os.Getpid())
	var result uintptr

	callback := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid != targetPID {
			return 1
		}
		if strings.TrimSpace(windowText(hwnd)) != "CodexSwitch" {
			return 1
		}

		result = hwnd
		return 0
	})

	procEnumWindows.Call(callback, 0)
	return result
}

func windowText(hwnd uintptr) string {
	buffer := make([]uint16, 256)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return syscall.UTF16ToString(buffer)
}
