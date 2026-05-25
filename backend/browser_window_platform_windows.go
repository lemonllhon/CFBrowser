//go:build windows
// +build windows

package backend

import (
	"syscall"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procEnumDisplayMonitors      = user32.NewProc("EnumDisplayMonitors")
	procGetCurrentProcessId      = kernel32.NewProc("GetCurrentProcessId")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procMonitorFromPoint         = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSystemParametersInfoW    = user32.NewProc("SystemParametersInfoW")
)

const (
	hwndTopmost             = ^uintptr(0)
	swpShowWindow           = 0x0040
	swpNoZOrder             = 0x0004
	spiGetWorkArea          = 0x0030
	monitorDefaultToNearest = 0x00000002
	swRestore               = 9
)

type winRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type winPoint struct {
	X int32
	Y int32
}

type winMonitorInfo struct {
	CbSize    uint32
	RcMonitor winRect
	RcWork    winRect
	DwFlags   uint32
}

func appWindowCenterPoint() (int, int, bool) {
	pidRet, _, _ := procGetCurrentProcessId.Call()
	if pidRet == 0 {
		return 0, 0, false
	}
	currentPID := uint32(pidRet)
	var rect winRect
	found := false
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}
		var windowPID uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
		if windowPID != currentPID {
			return 1
		}
		ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
		if ret == 0 || rect.Right <= rect.Left || rect.Bottom <= rect.Top {
			return 1
		}
		found = true
		return 0
	})
	procEnumWindows.Call(cb, 0)
	if !found {
		return 0, 0, false
	}
	return int(rect.Left + (rect.Right-rect.Left)/2), int(rect.Top + (rect.Bottom-rect.Top)/2), true
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

func workAreaForPoint(x int, y int) workAreaRect {
	info := winMonitorInfo{CbSize: uint32(unsafe.Sizeof(winMonitorInfo{}))}
	point := winPoint{X: int32(x), Y: int32(y)}
	monitor, _, _ := procMonitorFromPoint.Call(
		uintptr(*(*uint64)(unsafe.Pointer(&point))),
		uintptr(monitorDefaultToNearest),
	)
	if monitor == 0 {
		return primaryWorkArea()
	}
	ret, _, _ := procGetMonitorInfoW.Call(
		monitor,
		uintptr(unsafe.Pointer(&info)),
	)
	if ret == 0 || info.RcWork.Right <= info.RcWork.Left || info.RcWork.Bottom <= info.RcWork.Top {
		return primaryWorkArea()
	}
	return workAreaRect{
		Left:   int(info.RcWork.Left),
		Top:    int(info.RcWork.Top),
		Width:  int(info.RcWork.Right - info.RcWork.Left),
		Height: int(info.RcWork.Bottom - info.RcWork.Top),
	}
}

func allWorkAreasUnion() workAreaRect {
	areas := make([]workAreaRect, 0, 2)
	cb := syscall.NewCallback(func(monitor uintptr, hdc uintptr, rect uintptr, lparam uintptr) uintptr {
		info := winMonitorInfo{CbSize: uint32(unsafe.Sizeof(winMonitorInfo{}))}
		ret, _, _ := procGetMonitorInfoW.Call(
			monitor,
			uintptr(unsafe.Pointer(&info)),
		)
		if ret == 0 || info.RcWork.Right <= info.RcWork.Left || info.RcWork.Bottom <= info.RcWork.Top {
			return 1
		}
		areas = append(areas, workAreaRect{
			Left:   int(info.RcWork.Left),
			Top:    int(info.RcWork.Top),
			Width:  int(info.RcWork.Right - info.RcWork.Left),
			Height: int(info.RcWork.Bottom - info.RcWork.Top),
		})
		return 1
	})
	procEnumDisplayMonitors.Call(0, 0, cb, 0)
	if len(areas) == 0 {
		return primaryWorkArea()
	}
	left := areas[0].Left
	top := areas[0].Top
	right := areas[0].Left + areas[0].Width
	bottom := areas[0].Top + areas[0].Height
	for _, area := range areas[1:] {
		left = minInt(left, area.Left)
		top = minInt(top, area.Top)
		right = maxInt(right, area.Left+area.Width)
		bottom = maxInt(bottom, area.Top+area.Height)
	}
	return workAreaRect{
		Left:   left,
		Top:    top,
		Width:  maxInt(1, right-left),
		Height: maxInt(1, bottom-top),
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

func setBrowserWindowsBoundsByPID(pid int, left int, top int, width int, height int) error {
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
			procShowWindow.Call(hwnd, uintptr(swRestore))
			procSetWindowPos.Call(
				hwnd,
				0,
				uintptr(left),
				uintptr(top),
				uintptr(width),
				uintptr(height),
				uintptr(swpShowWindow|swpNoZOrder),
			)
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return nil
}
