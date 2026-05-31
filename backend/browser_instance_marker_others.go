//go:build !windows
// +build !windows

package backend

func setBrowserWindowsIcon(pid int, titleMarker string, iconPath string) error {
	return nil
}
