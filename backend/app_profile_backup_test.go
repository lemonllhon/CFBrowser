package backend

import (
	"archive/zip"
	"os"
	"path"
	"path/filepath"
	"testing"

	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
)

func TestProfileBackupZipIncludesProfilesAndCookieBundle(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	app := NewApp(root)
	app.config = cfg
	app.browserMgr = browser.NewManager(cfg, root)
	app.browserMgr.Profiles = map[string]*BrowserProfile{}

	profile := BrowserProfile{
		ProfileId:   "profile-1",
		ProfileName: "Profile 1",
		UserDataDir: "profile-1",
		Tags:        []string{"backup"},
		Keywords:    []string{"kw"},
	}
	userDataDir := filepath.Join(root, "data", "profile-1", "Default", "Network")
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDataDir, "Cookies"), []byte("cookie-db"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDataDir, "Cookies-wal"), []byte("cookie-wal"), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(root, "instances.zip")
	result, err := app.writeProfileBackupZip(zipPath, []BrowserProfile{profile}, ProfileBackupExportRequest{IncludeCookies: true})
	if err != nil {
		t.Fatalf("writeProfileBackupZip failed: %v", err)
	}
	if result.ProfileCount != 1 || result.CookieProfileCount != 1 {
		t.Fatalf("backup result mismatch: %#v", result)
	}

	summary, err := readProfileBackupSummary(zipPath)
	if err != nil {
		t.Fatalf("readProfileBackupSummary failed: %v", err)
	}
	if summary.ProfileCount != 1 || summary.CookieProfileCount != 1 || !summary.IncludesCookies {
		t.Fatalf("summary mismatch: %#v", summary)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	expected := map[string]bool{
		"manifest.json":         true,
		"payload/profiles.json": true,
		path.Join("payload", "cookies", "profile-1", "Default", "Network", "Cookies"):     true,
		path.Join("payload", "cookies", "profile-1", "Default", "Network", "Cookies-wal"): true,
		path.Join("payload", "cookies", "profile-1", "cookie-meta.json"):                  true,
	}
	for _, file := range reader.File {
		delete(expected, path.Clean(file.Name))
	}
	if len(expected) != 0 {
		t.Fatalf("backup zip missing expected entries: %#v", expected)
	}
}

func TestUniqueRestoredProfileName(t *testing.T) {
	used := map[string]struct{}{"demo": {}, "demo (恢复)": {}}
	if got := uniqueRestoredProfileName("Demo", used); got != "Demo (恢复 2)" {
		t.Fatalf("unexpected restored name: %s", got)
	}
	if got := uniqueRestoredProfileName("", used); got != "恢复实例" {
		t.Fatalf("unexpected fallback restored name: %s", got)
	}
}
