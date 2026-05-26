package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
	"os"
	"os/exec"
	"testing"
)

func TestBrowserProfileListReconcilesSharedRuntimeState(t *testing.T) {
	ln := mustListenLoopback(t)
	defer ln.Close()

	appRoot := t.TempDir()
	app1 := newRuntimeStateTestApp(appRoot)
	app1.browserMgr.Profiles["profile-1"] = &BrowserProfile{ProfileId: "profile-1", ProfileName: "Profile 1"}
	app1.markProfileRunningLocked("profile-1", app1.browserMgr.Profiles["profile-1"], nil, 0, listenerPort(t, ln), false, "pending")

	app2 := newRuntimeStateTestApp(appRoot)
	app2.browserMgr.Profiles["profile-1"] = &BrowserProfile{ProfileId: "profile-1", ProfileName: "Profile 1"}

	profiles := app2.BrowserProfileList()
	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	got := profiles[0]
	if !got.Running || !got.DebugReady || got.DebugPort != listenerPort(t, ln) {
		t.Fatalf("shared runtime state was not reconciled: %#v", got)
	}
	if got.RuntimeWarning != "" {
		t.Fatalf("expected live debug port to clear runtime warning, got %q", got.RuntimeWarning)
	}
}

func TestBrowserProfileListClearsStaleSharedRuntimeState(t *testing.T) {
	app := newRuntimeStateTestApp(t.TempDir())
	app.browserMgr.Profiles["profile-1"] = &BrowserProfile{
		ProfileId: "profile-1",
		Running:   true,
		DebugPort: 9,
	}
	app.persistBrowserProfileRuntimeState(app.browserMgr.Profiles["profile-1"])

	profiles := app.BrowserProfileList()
	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	if profiles[0].Running || profiles[0].DebugPort != 0 {
		t.Fatalf("expected stale shared runtime state to be cleared: %#v", profiles[0])
	}
	if _, err := os.Stat(app.browserProfileRuntimeStatePath("profile-1")); !os.IsNotExist(err) {
		t.Fatalf("expected stale runtime state file to be removed, stat err=%v", err)
	}
}

func newRuntimeStateTestApp(appRoot string) *App {
	app := NewApp(appRoot)
	app.browserMgr = browser.NewManager(config.DefaultConfig(), appRoot)
	app.browserMgr.Profiles = map[string]*BrowserProfile{}
	app.browserMgr.BrowserProcesses = map[string]*exec.Cmd{}
	return app
}
