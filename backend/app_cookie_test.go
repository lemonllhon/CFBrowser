package backend

import (
	"os"
	"path/filepath"
	"testing"

	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
)

func TestBrowserClearCookiesClearsStoppedProfileUserData(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	app := NewApp(root)
	app.config = cfg
	app.browserMgr = browser.NewManager(cfg, root)
	app.browserMgr.Profiles = map[string]*BrowserProfile{
		"profile-1": {
			ProfileId:   "profile-1",
			ProfileName: "Profile 1",
			UserDataDir: "profile-1",
			Running:     false,
		},
	}

	userDataDir := filepath.Join(root, "data", "profile-1")
	if err := os.MkdirAll(filepath.Join(userDataDir, "Default"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDataDir, "Default", "Cookies"), []byte("cookie-db"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDataDir, "Preferences"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := app.BrowserClearCookies("profile-1"); err != nil {
		t.Fatalf("BrowserClearCookies returned error: %v", err)
	}

	if _, err := os.Stat(userDataDir); err != nil {
		t.Fatalf("expected user data dir to remain: %v", err)
	}
	entries, err := os.ReadDir(userDataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected user data dir contents to be removed, got %d entries", len(entries))
	}
}

func TestBrowserClearCookiesRejectsRunningProfileWithoutDebugReady(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	app := NewApp(root)
	app.config = cfg
	app.browserMgr = browser.NewManager(cfg, root)
	app.browserMgr.Profiles = map[string]*BrowserProfile{
		"profile-1": {
			ProfileId:  "profile-1",
			Running:    true,
			DebugReady: false,
		},
	}

	if err := app.BrowserClearCookies("profile-1"); err == nil {
		t.Fatal("expected running profile without debug readiness to return an error")
	}
}
