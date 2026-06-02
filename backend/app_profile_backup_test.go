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
	if err := os.WriteFile(filepath.Join(root, "data", "profile-1", "Local State"), []byte(`{"os_crypt":{"encrypted_key":"key"}}`), 0644); err != nil {
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
		path.Join("payload", "cookies", "profile-1", "Local State"):                     true,
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

func TestProfileBackupFindsCookiesInNonDefaultChromeProfile(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	app := NewApp(root)
	app.config = cfg
	app.browserMgr = browser.NewManager(cfg, root)

	profile := BrowserProfile{
		ProfileId:   "profile-2",
		ProfileName: "Profile 2",
		UserDataDir: "profile-2",
	}
	userDataDir := filepath.Join(root, "data", "profile-2", "Profile 1", "Network")
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDataDir, "Cookies"), []byte("cookie-db"), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(root, "instances.zip")
	result, err := app.writeProfileBackupZip(zipPath, []BrowserProfile{profile}, ProfileBackupExportRequest{IncludeCookies: true})
	if err != nil {
		t.Fatalf("writeProfileBackupZip failed: %v", err)
	}
	if result.CookieProfileCount != 1 {
		t.Fatalf("expected cookie profile to be backed up: %#v", result)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	found := false
	for _, file := range reader.File {
		if path.Clean(file.Name) == path.Join("payload", "cookies", "profile-2", "Profile 1", "Network", "Cookies") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("backup zip missing non-default profile cookie database")
	}
}

func TestRestoreProfileBackupCookiesCopiesLocalStateWithCookieDatabase(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	app := NewApp(root)
	app.config = cfg
	app.browserMgr = browser.NewManager(cfg, root)

	zipPath := filepath.Join(root, "instances.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(out)
	cookieRoot := profileBackupCookieArchiveRoot("source-profile")
	if err := profileBackupZipWriteString(writer, path.Join(cookieRoot, "Local State"), `{"os_crypt":{"encrypted_key":"key"}}`); err != nil {
		t.Fatal(err)
	}
	if err := profileBackupZipWriteString(writer, path.Join(cookieRoot, "Default", "Network", "Cookies"), "cookie-db"); err != nil {
		t.Fatal(err)
	}
	meta := profileBackupCookieMeta{
		ProfileID:   "source-profile",
		ProfileName: "Source",
		UserDataDir: "source-profile",
		Files: []profileBackupCookieFile{
			{
				RelativePath: "Local State",
				ArchivePath:  path.Join(cookieRoot, "Local State"),
				Kind:         "local_state",
			},
			{
				RelativePath: "Default/Network/Cookies",
				ArchivePath:  path.Join(cookieRoot, "Default", "Network", "Cookies"),
				Kind:         "cookie_db",
			},
		},
	}
	if err := profileBackupZipWriteJSON(writer, path.Join(cookieRoot, "cookie-meta.json"), meta); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	target := &BrowserProfile{ProfileId: "target-profile", UserDataDir: "target-profile"}
	restored, warnings := app.restoreProfileBackupCookies(&reader.Reader, profileBackupProfile{
		ProfileID:   "source-profile",
		ProfileName: "Source",
	}, target)
	if !restored {
		t.Fatalf("expected cookies to restore, warnings=%v", warnings)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected restore warnings: %v", warnings)
	}
	for _, rel := range []string{"Local State", "Default/Network/Cookies"} {
		if _, err := os.Stat(filepath.Join(root, "data", "target-profile", filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected restored file %s: %v", rel, err)
		}
	}
}

func TestSelectProfilesForBackupEmptySelectedDoesNotExportAll(t *testing.T) {
	root := t.TempDir()
	cfg := config.DefaultConfig()
	app := NewApp(root)
	app.config = cfg
	app.browserMgr = browser.NewManager(cfg, root)
	app.browserMgr.Profiles = map[string]*BrowserProfile{
		"profile-1": {ProfileId: "profile-1", ProfileName: "Profile 1"},
	}

	got := app.selectProfilesForBackup(ProfileBackupExportRequest{Scope: "selected"})
	if len(got) != 0 {
		t.Fatalf("expected no profiles for empty selected scope, got %#v", got)
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
