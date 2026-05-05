//go:build windows
// +build windows

package backend

import (
	"syscall"
	"unsafe"
)

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	procEnumWindows                = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible            = user32.NewProc("IsWindowVisible")
	procSetWindowPos               = user32.NewProc("SetWindowPos")
	procSystemParametersInfoW      = user32.NewProc("SystemParametersInfoW")
)

const (
	hwndTopmost       = ^uintptr(0)
	swpShowWindow     = 0x0040
	spiGetWorkArea    = 0x0030
)

type winRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func primaryWorkArea() workAreaRect {
	rect := winRect{}
	ret, _, _ := procSystemParametersInfoW.Call(
		uintptr(spiGetWorkArea),
		0,
		uintptr(unsafe.Pointer(&rect)),
		0,
	)
	if ret == 0 || rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		return workAreaRect{Left: 0, Top: 0, Width: 1280, Height: 800}
	}
	return workAreaRect{
		Left:   int(rect.Left),
		Top:    int(rect.Top),
		Width:  int(rect.Right - rect.Left),
		Height: int(rect.Bottom - rect.Top),
	}
}

func setBrowserWindowsTopmostByPID(pid int, left int, top int, width int, height int) error {
	if pid <= 0 {
		return nil
	}
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}
		var windowPID uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
		if int(windowPID) == pid {
			procSetWindowPos.Call(
				hwnd,
				hwndTopmost,
				uintptr(left),
				uintptr(top),
				uintptr(width),
				uintptr(height),
				uintptr(swpShowWindow),
			)
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return nil
}
