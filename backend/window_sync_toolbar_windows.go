//go:build windows
// +build windows

package backend

import (
	"fmt"
	goruntime "runtime"
	"sync"
	"syscall"
	"unsafe"
)

const (
	windowSyncToolbarClass = "TraceBrowserWindowSyncToolbar"
	windowSyncToolbarWidth = 420
	windowSyncToolbarHeight = 46

	windowSyncToolbarShowAllID = 1001
	windowSyncToolbarPauseID   = 1002
	windowSyncToolbarStopID    = 1003

	wsOverlapped = 0x00000000
	wsVisible    = 0x10000000
	wsChild      = 0x40000000
	wsBorder     = 0x00800000
	wsCaption    = 0x00C00000
	wsSysMenu    = 0x00080000

	wsExToolWindow = 0x00000080
	wsExTopmost    = 0x00000008

	wmDestroy    = 0x0002
	wmCommand    = 0x0111
	wmLButtonDown = 0x0201
	wmAppUpdate = 0x8001
	wmAppHide   = 0x8002

	swShow = 5

	htCaption = 2
)

var (
	windowSyncToolbarUser32              = syscall.NewLazyDLL("user32.dll")
	windowSyncToolbarKernel32            = syscall.NewLazyDLL("kernel32.dll")
	procToolbarRegisterClassW            = windowSyncToolbarUser32.NewProc("RegisterClassW")
	procToolbarCreateWindowExW           = windowSyncToolbarUser32.NewProc("CreateWindowExW")
	procToolbarDefWindowProcW            = windowSyncToolbarUser32.NewProc("DefWindowProcW")
	procToolbarDestroyWindow             = windowSyncToolbarUser32.NewProc("DestroyWindow")
	procToolbarShowWindow                = windowSyncToolbarUser32.NewProc("ShowWindow")
	procToolbarSetWindowTextW            = windowSyncToolbarUser32.NewProc("SetWindowTextW")
	procToolbarSetWindowPos              = windowSyncToolbarUser32.NewProc("SetWindowPos")
	procToolbarReleaseCapture            = windowSyncToolbarUser32.NewProc("ReleaseCapture")
	procToolbarSendMessageW              = windowSyncToolbarUser32.NewProc("SendMessageW")
	procToolbarPostMessageW              = windowSyncToolbarUser32.NewProc("PostMessageW")
	procToolbarGetMessageW               = windowSyncToolbarUser32.NewProc("GetMessageW")
	procToolbarTranslateMessage          = windowSyncToolbarUser32.NewProc("TranslateMessage")
	procToolbarDispatchMessageW          = windowSyncToolbarUser32.NewProc("DispatchMessageW")
	procToolbarPostQuitMessage           = windowSyncToolbarUser32.NewProc("PostQuitMessage")
	procToolbarGetModuleHandleW          = windowSyncToolbarKernel32.NewProc("GetModuleHandleW")
	windowSyncToolbarClassRegisterOnce   sync.Once
	windowSyncToolbarClassRegisterErr    error
	windowSyncToolbarInstances           sync.Map
)

type windowSyncToolbarController struct {
	mu         sync.Mutex
	hwnd       uintptr
	statusHwnd uintptr
	pauseHwnd  uintptr
	app        *App
	paused     bool
	started    bool
	lastState  *WindowSyncState
}

type windowSyncToolbarWndClass struct {
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
}

type windowSyncToolbarMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct {
		X int32
		Y int32
	}
}

func (c *windowSyncToolbarController) Show(app *App, state *WindowSyncState) error {
	if state == nil {
		return nil
	}
	c.mu.Lock()
	c.app = app
	c.paused = state.Paused
	c.lastState = cloneWindowSyncState(state)
	if c.hwnd != 0 {
		hwnd := c.hwnd
		c.mu.Unlock()
		procToolbarPostMessageW.Call(hwnd, uintptr(wmAppUpdate), 0, 0)
		procToolbarShowWindow.Call(hwnd, uintptr(swShow))
		return nil
	}
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	ready := make(chan error, 1)
	go c.run(app, cloneWindowSyncState(state), ready)
	c.mu.Unlock()
	return <-ready
}

func (c *windowSyncToolbarController) run(app *App, state *WindowSyncState, ready chan<- error) {
	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()
	if err := registerWindowSyncToolbarClass(); err != nil {
		ready <- err
		return
	}
	area := primaryWorkArea()
	x := area.Left + (area.Width-windowSyncToolbarWidth)/2
	y := area.Top + 10
	className, _ := syscall.UTF16PtrFromString(windowSyncToolbarClass)
	title, _ := syscall.UTF16PtrFromString("窗口同步")
	hwnd, _, _ := procToolbarCreateWindowExW.Call(
		uintptr(wsExToolWindow|wsExTopmost),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsOverlapped|wsCaption|wsSysMenu|wsVisible),
		uintptr(x),
		uintptr(y),
		uintptr(windowSyncToolbarWidth),
		uintptr(windowSyncToolbarHeight),
		0,
		0,
		0,
		0,
	)
	if hwnd == 0 {
		ready <- fmt.Errorf("创建窗口同步工具栏失败")
		return
	}
	c.mu.Lock()
	c.hwnd = hwnd
	c.paused = state.Paused
	c.app = app
	c.lastState = cloneWindowSyncState(state)
	c.mu.Unlock()
	windowSyncToolbarInstances.Store(hwnd, c)
	c.createChildren()
	c.applyLatestState()
	procToolbarSetWindowPos.Call(c.hwnd, hwndTopmost, uintptr(x), uintptr(y), uintptr(windowSyncToolbarWidth), uintptr(windowSyncToolbarHeight), uintptr(swpShowWindow))
	ready <- nil

	var msg windowSyncToolbarMsg
	for {
		ret, _, _ := procToolbarGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procToolbarTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procToolbarDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (c *windowSyncToolbarController) Update(state *WindowSyncState) error {
	c.mu.Lock()
	if state != nil {
		c.paused = state.Paused
		c.lastState = cloneWindowSyncState(state)
	}
	if c.hwnd == 0 || state == nil {
		c.mu.Unlock()
		return nil
	}
	hwnd := c.hwnd
	c.mu.Unlock()
	procToolbarPostMessageW.Call(hwnd, uintptr(wmAppUpdate), 0, 0)
	return nil
}

func (c *windowSyncToolbarController) Hide() error {
	c.mu.Lock()
	if c.hwnd == 0 {
		c.started = false
		c.mu.Unlock()
		return nil
	}
	hwnd := c.hwnd
	c.mu.Unlock()
	procToolbarPostMessageW.Call(hwnd, uintptr(wmAppHide), 0, 0)
	return nil
}

func (c *windowSyncToolbarController) createChildren() {
	statusClass, _ := syscall.UTF16PtrFromString("STATIC")
	buttonClass, _ := syscall.UTF16PtrFromString("BUTTON")
	status, _ := syscall.UTF16PtrFromString("窗口同步中")
	showText, _ := syscall.UTF16PtrFromString("展示")
	pauseText, _ := syscall.UTF16PtrFromString("暂停")
	stopText, _ := syscall.UTF16PtrFromString("停止")
	c.statusHwnd, _, _ = procToolbarCreateWindowExW.Call(0, uintptr(unsafe.Pointer(statusClass)), uintptr(unsafe.Pointer(status)), uintptr(wsChild|wsVisible), 12, 10, 150, 24, c.hwnd, 0, 0, 0)
	procToolbarCreateWindowExW.Call(0, uintptr(unsafe.Pointer(buttonClass)), uintptr(unsafe.Pointer(showText)), uintptr(wsChild|wsVisible|wsBorder), 190, 8, 62, 26, c.hwnd, uintptr(windowSyncToolbarShowAllID), 0, 0)
	c.pauseHwnd, _, _ = procToolbarCreateWindowExW.Call(0, uintptr(unsafe.Pointer(buttonClass)), uintptr(unsafe.Pointer(pauseText)), uintptr(wsChild|wsVisible|wsBorder), 260, 8, 62, 26, c.hwnd, uintptr(windowSyncToolbarPauseID), 0, 0)
	procToolbarCreateWindowExW.Call(0, uintptr(unsafe.Pointer(buttonClass)), uintptr(unsafe.Pointer(stopText)), uintptr(wsChild|wsVisible|wsBorder), 330, 8, 62, 26, c.hwnd, uintptr(windowSyncToolbarStopID), 0, 0)
}

func (c *windowSyncToolbarController) applyLatestState() {
	c.mu.Lock()
	state := cloneWindowSyncState(c.lastState)
	c.mu.Unlock()
	if state == nil {
		return
	}
	status := "窗口同步中"
	if state.Paused {
		status = "窗口同步已暂停"
	}
	statusPtr, _ := syscall.UTF16PtrFromString(status)
	pauseText := "暂停"
	if state.Paused {
		pauseText = "恢复"
	}
	pausePtr, _ := syscall.UTF16PtrFromString(pauseText)
	if c.statusHwnd != 0 {
		procToolbarSetWindowTextW.Call(c.statusHwnd, uintptr(unsafe.Pointer(statusPtr)))
	}
	if c.pauseHwnd != 0 {
		procToolbarSetWindowTextW.Call(c.pauseHwnd, uintptr(unsafe.Pointer(pausePtr)))
	}
}

func registerWindowSyncToolbarClass() error {
	windowSyncToolbarClassRegisterOnce.Do(func() {
		className, _ := syscall.UTF16PtrFromString(windowSyncToolbarClass)
		instance, _, _ := procToolbarGetModuleHandleW.Call(0)
		wndClass := windowSyncToolbarWndClass{
			LpfnWndProc:   syscall.NewCallback(windowSyncToolbarWndProc),
			HInstance:     instance,
			HbrBackground: 6,
			LpszClassName: className,
		}
		ret, _, _ := procToolbarRegisterClassW.Call(uintptr(unsafe.Pointer(&wndClass)))
		if ret == 0 {
			windowSyncToolbarClassRegisterErr = fmt.Errorf("注册窗口同步工具栏窗口类失败")
		}
	})
	return windowSyncToolbarClassRegisterErr
}

func windowSyncToolbarWndProc(hwnd uintptr, msg uint32, wparam uintptr, lparam uintptr) uintptr {
	switch msg {
	case wmCommand:
		id := int(wparam & 0xffff)
		if value, ok := windowSyncToolbarInstances.Load(hwnd); ok {
			controller := value.(*windowSyncToolbarController)
			controller.handleCommand(id)
		}
		return 0
	case wmLButtonDown:
		procToolbarReleaseCapture.Call()
		procToolbarSendMessageW.Call(hwnd, 0x00A1, uintptr(htCaption), 0)
		return 0
	case wmAppUpdate:
		if value, ok := windowSyncToolbarInstances.Load(hwnd); ok {
			controller := value.(*windowSyncToolbarController)
			controller.applyLatestState()
		}
		return 0
	case wmAppHide:
		procToolbarDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		if value, ok := windowSyncToolbarInstances.Load(hwnd); ok {
			controller := value.(*windowSyncToolbarController)
			controller.resetAfterDestroy()
		}
		windowSyncToolbarInstances.Delete(hwnd)
		procToolbarPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procToolbarDefWindowProcW.Call(hwnd, uintptr(msg), wparam, lparam)
	return ret
}

func (c *windowSyncToolbarController) resetAfterDestroy() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hwnd = 0
	c.statusHwnd = 0
	c.pauseHwnd = 0
	c.started = false
	c.app = nil
	c.lastState = nil
}

func (c *windowSyncToolbarController) handleCommand(id int) {
	c.mu.Lock()
	app := c.app
	paused := c.paused
	c.mu.Unlock()
	if app == nil {
		return
	}
	go func() {
		switch id {
		case windowSyncToolbarShowAllID:
			_, _ = app.WindowSyncShowAll()
		case windowSyncToolbarPauseID:
			if paused {
				_, _ = app.WindowSyncResume()
			} else {
				_, _ = app.WindowSyncPause()
			}
		case windowSyncToolbarStopID:
			_, _ = app.WindowSyncStop()
		}
	}()
}
