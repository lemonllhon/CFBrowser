//go:build windows

package backend

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32ShellExecute = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")
)

func launchOfficialUpdateInstaller(installerPath string) error {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(installerPath)
	if err != nil {
		return err
	}
	dir, err := windows.UTF16PtrFromString(filepath.Dir(installerPath))
	if err != nil {
		return err
	}
	ret, _, callErr := shell32ShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0,
		uintptr(unsafe.Pointer(dir)),
		uintptr(windows.SW_SHOWNORMAL),
	)
	if ret > 32 {
		return nil
	}
	if callErr != nil && callErr != windows.ERROR_SUCCESS {
		return callErr
	}
	return fmt.Errorf("ShellExecuteW returned %d", ret)
}
