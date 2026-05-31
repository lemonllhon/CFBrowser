//go:build windows
// +build windows

package backend

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	procLoadImageW          = user32.NewProc("LoadImageW")
	procSendMessageW        = user32.NewProc("SendMessageW")
	procGetWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	procGetWindowText       = user32.NewProc("GetWindowTextW")
)

const (
	winImageIcon      = 1
	winLRLoadFromFile = 0x0010
	winLRDefaultSize  = 0x0040
	winWMSetIcon      = 0x0080
	winIconSmall      = 0
	winIconBig        = 1
)

func setBrowserWindowsIcon(pid int, titleMarker string, iconPath string) error {
	if (pid <= 0 && titleMarker == "") || iconPath == "" {
		return nil
	}
	pathPtr, err := syscall.UTF16PtrFromString(iconPath)
	if err != nil {
		return err
	}
	hicon, _, callErr := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(winImageIcon),
		0,
		0,
		uintptr(winLRLoadFromFile|winLRDefaultSize),
	)
	if hicon == 0 {
		if callErr != syscall.Errno(0) {
			return callErr
		}
		return fmt.Errorf("LoadImageW failed")
	}

	applied := false
	applyToMatchingWindows := func(match func(hwnd uintptr) bool) {
		cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
			visible, _, _ := procIsWindowVisible.Call(hwnd)
			if visible == 0 || !match(hwnd) {
				return 1
			}
			procSendMessageW.Call(hwnd, uintptr(winWMSetIcon), uintptr(winIconSmall), hicon)
			procSendMessageW.Call(hwnd, uintptr(winWMSetIcon), uintptr(winIconBig), hicon)
			applied = true
			return 1
		})
		procEnumWindows.Call(cb, 0)
	}

	if pid > 0 {
		applyToMatchingWindows(func(hwnd uintptr) bool {
			var windowPID uint32
			procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
			return int(windowPID) == pid
		})
	}
	if !applied && titleMarker != "" {
		applyToMatchingWindows(func(hwnd uintptr) bool {
			return strings.Contains(windowText(hwnd), titleMarker)
		})
	}
	if !applied {
		return fmt.Errorf("未找到浏览器窗口")
	}
	return nil
}

func windowText(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLength.Call(hwnd)
	if length == 0 {
		return ""
	}
	buf := make([]uint16, int(length)+1)
	ret, _, _ := procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
