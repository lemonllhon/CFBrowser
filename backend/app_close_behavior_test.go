package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
	"context"
	"testing"
)

func TestPlatformSupportsTrayCloseFlowForOS(t *testing.T) {
	if !platformSupportsTrayCloseFlowForOS("windows") {
		t.Fatal("expected Windows close to open the frontend confirmation flow")
	}
	if platformSupportsTrayCloseFlowForOS("linux") {
		t.Fatal("expected Linux to skip tray close flow")
	}
	if platformSupportsTrayCloseFlowForOS("darwin") {
		t.Fatal("expected macOS to skip tray close flow")
	}
}

func TestShouldBlockClose_InterceptsSupportedWindowClose(t *testing.T) {
	app := NewApp("")
	events := 0
	app.setProtoEventSink(func(eventName string, payload []byte) {
		if eventName == "app:request-close" {
			events++
		}
	})

	if !shouldBlockClose(app, context.Background(), true) {
		t.Fatal("expected supported window close to be intercepted")
	}
	if events != 1 {
		t.Fatalf("expected app:request-close event to be emitted once, got %d", events)
	}
}

func TestShouldBlockClose_SkipsUnsupportedWindowClose(t *testing.T) {
	app := NewApp("")
	if shouldBlockClose(app, context.Background(), false) {
		t.Fatal("expected unsupported window close to proceed without frontend interception")
	}
}

func TestShouldBlockClose_SkipsAfterQuitModeSelected(t *testing.T) {
	app := NewApp("")
	app.setQuitMode(quitModeFull)

	if shouldBlockClose(app, context.Background(), true) {
		t.Fatal("expected selected quit mode to bypass close interception")
	}
}

func TestQuitAppOnlyKeepsTrackedBrowsers(t *testing.T) {
	app := NewApp("")
	app.browserMgr = browser.NewManager(config.DefaultConfig(), "")
	app.browserMgr.Profiles = map[string]*BrowserProfile{
		"profile-1": {
			ProfileId: "profile-1",
			Running:   true,
		},
	}
	app.browserMgr.BrowserProcesses["profile-1"] = nil

	app.QuitAppOnly()

	if !app.forceQuit {
		t.Fatal("expected QuitAppOnly to set forceQuit")
	}
	if app.quitMode != quitModeAppOnly {
		t.Fatalf("expected quitModeAppOnly, got %v", app.quitMode)
	}
	if app.shouldStopRuntimeServicesOnShutdown() {
		t.Fatal("expected app-only quit to skip runtime service shutdown")
	}
	if _, ok := app.browserMgr.BrowserProcesses["profile-1"]; !ok {
		t.Fatal("expected tracked browser to remain untouched before process shutdown")
	}
	if !app.browserMgr.Profiles["profile-1"].Running {
		t.Fatal("expected app-only quit to keep running profile state intact")
	}
}

func TestForceQuitStopsTrackedBrowsers(t *testing.T) {
	app := NewApp("")
	app.browserMgr = browser.NewManager(config.DefaultConfig(), "")
	app.browserMgr.Profiles = map[string]*BrowserProfile{
		"profile-1": {
			ProfileId: "profile-1",
			Running:   true,
		},
	}
	app.browserMgr.BrowserProcesses["profile-1"] = nil

	app.ForceQuit()

	if !app.forceQuit {
		t.Fatal("expected ForceQuit to set forceQuit")
	}
	if app.quitMode != quitModeFull {
		t.Fatalf("expected quitModeFull, got %v", app.quitMode)
	}
	if !app.shouldStopRuntimeServicesOnShutdown() {
		t.Fatal("expected full quit to stop runtime services")
	}
	if _, ok := app.browserMgr.BrowserProcesses["profile-1"]; ok {
		t.Fatal("expected ForceQuit to clear tracked browser processes")
	}
	if app.browserMgr.Profiles["profile-1"].Running {
		t.Fatal("expected ForceQuit to mark the profile as stopped")
	}
}
