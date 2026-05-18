//go:build !windows
// +build !windows

package backend

func primaryWorkArea() workAreaRect {
	return workAreaRect{Left: 0, Top: 0, Width: 1280, Height: 800}
}

func appWindowCenterPoint() (int, int, bool) {
	return 0, 0, false
}

func workAreaForPoint(x int, y int) workAreaRect {
	return primaryWorkArea()
}

func setBrowserWindowsTopmostByPID(pid int, left int, top int, width int, height int) error {
	return nil
}
