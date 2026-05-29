package backend

import (
	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/database"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ant-chrome/backend/internal/browser"
)

func TestChromeExtensionIDForDirectoryUsesManifestKey(t *testing.T) {
	dir := t.TempDir()
	key := []byte("trace-browser-test-public-key")
	manifest := `{"manifest_version":3,"name":"Shared","version":"1.0","key":"` + base64ForTest(key) + `"}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := chromeExtensionIDForDirectory(dir)
	if err != nil {
		t.Fatalf("chromeExtensionIDForDirectory failed: %v", err)
	}
	want := chromeExtensionIDFromBytes(key)
	if got != want {
		t.Fatalf("extension id mismatch: got=%s want=%s", got, want)
	}
	if len(got) != 32 || strings.Trim(got, "abcdefghijklmnop") != "" {
		t.Fatalf("extension id should be 32 chars in a-p alphabet: %s", got)
	}
}

func TestPrepareSharedExtensionDataBindingLinksProfileStorage(t *testing.T) {
	app := NewApp(t.TempDir())
	userDataDir := filepath.Join(app.appRootAbs(), "data", "profile-1")
	extensionDir := filepath.Join(app.extensionLibraryRoot(), "extension-1")
	if err := os.MkdirAll(filepath.Join(userDataDir, "Default", "Local Extension Settings"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(extensionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extensionDir, "manifest.json"), []byte(`{"manifest_version":3,"name":"Shared","version":"1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	chromeID, err := chromeExtensionIDForDirectory(extensionDir)
	if err != nil {
		t.Fatal(err)
	}
	existingDataDir := filepath.Join(userDataDir, "Default", "Local Extension Settings", chromeID)
	if err := os.MkdirAll(existingDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existingDataDir, "state.json"), []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	err = app.prepareSharedExtensionDataBinding(userDataDir, browser.ExtensionBinding{
		ProfileId:   "profile-1",
		ExtensionId: "extension-1",
		Mode:        "shared",
		Enabled:     true,
	}, &browser.Extension{
		ExtensionId: "extension-1",
		Name:        "Shared",
		InstallDir:  filepath.Join("data", "extensions", "library", "extension-1"),
	})
	if err != nil {
		t.Fatalf("prepareSharedExtensionDataBinding failed: %v", err)
	}

	sharedData := filepath.Join(app.extensionSharedDataRoot(), "extension-1", chromeID, "local-extension-settings", "state.json")
	if _, err := os.Stat(sharedData); err != nil {
		t.Fatalf("expected existing profile extension data to be migrated to shared dir: %v", err)
	}
	if !linkedToTarget(existingDataDir, filepath.Dir(sharedData)) {
		t.Fatalf("expected profile extension data dir to link to shared dir")
	}
}

func TestPrepareSharedExtensionDataBindingLinksLegacyLocalStorageFile(t *testing.T) {
	app := NewApp(t.TempDir())
	userDataDir := filepath.Join(app.appRootAbs(), "data", "profile-1")
	extensionDir := filepath.Join(app.extensionLibraryRoot(), "extension-1")
	if err := os.MkdirAll(extensionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extensionDir, "manifest.json"), []byte(`{"manifest_version":2,"name":"Shared","version":"1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	chromeID, err := chromeExtensionIDForDirectory(extensionDir)
	if err != nil {
		t.Fatal(err)
	}
	localStorageFile := filepath.Join(userDataDir, "Default", "Local Storage", "chrome-extension_"+chromeID+"_0.localstorage")
	if err := os.MkdirAll(filepath.Dir(localStorageFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localStorageFile, []byte("legacy-state"), 0644); err != nil {
		t.Fatal(err)
	}

	err = app.prepareSharedExtensionDataBinding(userDataDir, browser.ExtensionBinding{
		ProfileId:   "profile-1",
		ExtensionId: "extension-1",
		Mode:        "shared",
		Enabled:     true,
	}, &browser.Extension{
		ExtensionId: "extension-1",
		Name:        "Shared",
		InstallDir:  filepath.Join("data", "extensions", "library", "extension-1"),
	})
	if err != nil {
		t.Fatalf("prepareSharedExtensionDataBinding failed: %v", err)
	}

	sharedFile := filepath.Join(app.extensionSharedDataRoot(), "extension-1", chromeID, "local-storage", "chrome-extension_"+chromeID+"_0.localstorage")
	data, err := os.ReadFile(sharedFile)
	if err != nil {
		t.Fatalf("expected legacy local storage to be migrated to shared file: %v", err)
	}
	if string(data) != "legacy-state" {
		t.Fatalf("unexpected shared local storage content: %q", string(data))
	}
	if !linkedToTarget(localStorageFile, sharedFile) {
		t.Fatalf("expected profile legacy local storage file to link to shared file")
	}
}

func TestMaterializeSharedExtensionDataBindingRestoresProfileStorage(t *testing.T) {
	app := NewApp(t.TempDir())
	userDataDir := filepath.Join(app.appRootAbs(), "data", "profile-1")
	extensionDir := filepath.Join(app.extensionLibraryRoot(), "extension-1")
	if err := os.MkdirAll(extensionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extensionDir, "manifest.json"), []byte(`{"manifest_version":3,"name":"Shared","version":"1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	chromeID, err := chromeExtensionIDForDirectory(extensionDir)
	if err != nil {
		t.Fatal(err)
	}
	existingDataDir := filepath.Join(userDataDir, "Default", "Local Extension Settings", chromeID)
	if err := os.MkdirAll(existingDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existingDataDir, "state.json"), []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	binding := browser.ExtensionBinding{
		ProfileId:   "profile-1",
		ExtensionId: "extension-1",
		Mode:        "shared",
		Enabled:     true,
	}
	extension := &browser.Extension{
		ExtensionId: "extension-1",
		Name:        "Shared",
		InstallDir:  filepath.Join("data", "extensions", "library", "extension-1"),
	}
	if err := app.prepareSharedExtensionDataBinding(userDataDir, binding, extension); err != nil {
		t.Fatalf("prepareSharedExtensionDataBinding failed: %v", err)
	}
	sharedDir := filepath.Join(app.extensionSharedDataRoot(), "extension-1", chromeID, "local-extension-settings")
	if !linkedToTarget(existingDataDir, sharedDir) {
		t.Fatalf("expected shared storage link before materialize")
	}

	binding.Mode = "exclusive"
	if err := app.materializeSharedExtensionDataBinding(userDataDir, binding, extension); err != nil {
		t.Fatalf("materializeSharedExtensionDataBinding failed: %v", err)
	}
	if linkedToTarget(existingDataDir, sharedDir) {
		t.Fatalf("expected profile storage to be materialized back to a local directory")
	}
	if _, err := os.Stat(filepath.Join(existingDataDir, "state.json")); err != nil {
		t.Fatalf("expected materialized profile data to keep shared state: %v", err)
	}
}

func TestBrowserExtensionDeleteRemovesSharedDataDir(t *testing.T) {
	root := t.TempDir()
	app := NewApp(root)
	cfg := config.DefaultConfig()
	app.config = cfg
	dbPath := filepath.Join(root, "data", "test.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatal(err)
	}
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}
	app.browserMgr = browser.NewManager(cfg, root)
	app.browserMgr.ExtensionDAO = browser.NewSQLiteExtensionDAO(db.GetConn())
	dao, err := app.extensionDAO()
	if err != nil {
		t.Fatal(err)
	}

	extensionDir := filepath.Join(app.extensionLibraryRoot(), "extension-1")
	if err := os.MkdirAll(extensionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extensionDir, "manifest.json"), []byte(`{"manifest_version":3,"name":"Shared","version":"1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	sharedDir := app.extensionSharedDataDir("extension-1")
	if err := os.MkdirAll(filepath.Join(sharedDir, "state"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := dao.Upsert(browser.Extension{
		ExtensionId:     "extension-1",
		Name:            "Shared",
		Version:         "1.0",
		ManifestVersion: 3,
		SourceType:      "directory",
		InstallDir:      filepath.Join("data", "extensions", "library", "extension-1"),
	}); err != nil {
		t.Fatal(err)
	}

	if err := app.BrowserExtensionDelete("extension-1"); err != nil {
		t.Fatalf("BrowserExtensionDelete failed: %v", err)
	}
	if _, err := os.Stat(sharedDir); !os.IsNotExist(err) {
		t.Fatalf("expected shared data dir to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(extensionDir); !os.IsNotExist(err) {
		t.Fatalf("expected extension library dir to be removed, stat err=%v", err)
	}
}

func TestBrowserExtensionSyncProfileDataCopiesMasterDataToTargets(t *testing.T) {
	app, dao := newExtensionSyncTestApp(t)
	extension := seedExtensionSyncData(t, app, dao)
	sourceProfile := &browser.Profile{ProfileId: "source", ProfileName: "主实例", UserDataDir: "source"}
	targetProfile := &browser.Profile{ProfileId: "target", ProfileName: "副实例", UserDataDir: "target"}
	app.browserMgr.Profiles[sourceProfile.ProfileId] = sourceProfile
	app.browserMgr.Profiles[targetProfile.ProfileId] = targetProfile
	for _, profile := range []*browser.Profile{sourceProfile, targetProfile} {
		if err := dao.UpsertBinding(browser.ExtensionBinding{
			ProfileId:   profile.ProfileId,
			ExtensionId: extension.ExtensionId,
			Mode:        "shared",
			Enabled:     true,
		}); err != nil {
			t.Fatal(err)
		}
	}

	chromeID, err := chromeExtensionIDForDirectory(app.resolveAppPath(extension.InstallDir))
	if err != nil {
		t.Fatal(err)
	}
	sourceUserDataDir := app.browserMgr.ResolveUserDataDir(sourceProfile)
	targetUserDataDir := app.browserMgr.ResolveUserDataDir(targetProfile)
	sourceStateDir := filepath.Join(sourceUserDataDir, "Default", "Local Extension Settings", chromeID)
	targetStateDir := filepath.Join(targetUserDataDir, "Default", "Local Extension Settings", chromeID)
	targetSharedStateDir := filepath.Join(app.extensionSharedDataDir(extension.ExtensionId), chromeID, "local-extension-settings")
	if err := os.MkdirAll(sourceStateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceStateDir, "state.json"), []byte(`{"master":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetSharedStateDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetSharedStateDir, "state.json"), []byte(`{"old":true}`), 0644); err != nil {
		t.Fatal(err)
	}
	linkDirForTest(t, targetStateDir, targetSharedStateDir)
	sourceLocalStorage := filepath.Join(sourceUserDataDir, "Default", "Local Storage", "chrome-extension_"+chromeID+"_0.localstorage")
	targetLocalStorage := filepath.Join(targetUserDataDir, "Default", "Local Storage", "chrome-extension_"+chromeID+"_0.localstorage")
	if err := os.MkdirAll(filepath.Dir(sourceLocalStorage), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceLocalStorage, []byte("master-local-storage"), 0644); err != nil {
		t.Fatal(err)
	}
	sourceLocalStorageLevelDB := filepath.Join(sourceUserDataDir, "Default", "Local Storage", "leveldb")
	targetLocalStorageLevelDB := filepath.Join(targetUserDataDir, "Default", "Local Storage", "leveldb")
	if err := os.MkdirAll(sourceLocalStorageLevelDB, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceLocalStorageLevelDB, "000003.log"), []byte("master-leveldb"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetLocalStorageLevelDB, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetLocalStorageLevelDB, "000003.log"), []byte("old-leveldb"), 0644); err != nil {
		t.Fatal(err)
	}
	sourceStorageExt := filepath.Join(sourceUserDataDir, "Default", "Storage", "ext", chromeID, "def")
	targetStorageExt := filepath.Join(targetUserDataDir, "Default", "Storage", "ext", chromeID, "def")
	if err := os.MkdirAll(sourceStorageExt, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceStorageExt, "bucket.json"), []byte(`{"source":"master"}`), 0644); err != nil {
		t.Fatal(err)
	}
	targetSessionStorage := filepath.Join(targetUserDataDir, "Default", "Session Storage")
	if err := os.MkdirAll(targetSessionStorage, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetSessionStorage, "old.log"), []byte("keep-when-source-missing"), 0644); err != nil {
		t.Fatal(err)
	}
	staleTargetIndexedDB := filepath.Join(targetUserDataDir, "Default", "IndexedDB", "chrome-extension_"+chromeID+"_0.indexeddb.leveldb")
	if err := os.MkdirAll(staleTargetIndexedDB, 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := app.BrowserExtensionSyncProfileData(BrowserExtensionSyncDataInput{
		ExtensionId:      extension.ExtensionId,
		SourceProfileId:  sourceProfile.ProfileId,
		TargetProfileIds: []string{targetProfile.ProfileId},
	}); err != nil {
		t.Fatalf("BrowserExtensionSyncProfileData failed: %v", err)
	}

	state, err := os.ReadFile(filepath.Join(targetStateDir, "state.json"))
	if err != nil {
		t.Fatalf("expected target state to be copied: %v", err)
	}
	if string(state) != `{"master":true}` {
		t.Fatalf("unexpected target state: %s", state)
	}
	if !linkedToTarget(targetStateDir, targetSharedStateDir) {
		t.Fatalf("expected target shared data link to be preserved")
	}
	sharedState, err := os.ReadFile(filepath.Join(targetSharedStateDir, "state.json"))
	if err != nil {
		t.Fatalf("expected target shared backing data to be copied: %v", err)
	}
	if string(sharedState) != `{"master":true}` {
		t.Fatalf("unexpected target shared backing state: %s", sharedState)
	}
	localStorage, err := os.ReadFile(targetLocalStorage)
	if err != nil {
		t.Fatalf("expected target local storage to be copied: %v", err)
	}
	if string(localStorage) != "master-local-storage" {
		t.Fatalf("unexpected target local storage: %s", localStorage)
	}
	levelDBData, err := os.ReadFile(filepath.Join(targetLocalStorageLevelDB, "000003.log"))
	if err != nil {
		t.Fatalf("expected target Local Storage leveldb to be copied: %v", err)
	}
	if string(levelDBData) != "master-leveldb" {
		t.Fatalf("unexpected target Local Storage leveldb: %s", levelDBData)
	}
	storageExtData, err := os.ReadFile(filepath.Join(targetStorageExt, "bucket.json"))
	if err != nil {
		t.Fatalf("expected target Storage/ext data to be copied: %v", err)
	}
	if string(storageExtData) != `{"source":"master"}` {
		t.Fatalf("unexpected target Storage/ext data: %s", storageExtData)
	}
	sessionData, err := os.ReadFile(filepath.Join(targetSessionStorage, "old.log"))
	if err != nil {
		t.Fatalf("expected target Session Storage to remain when source is missing: %v", err)
	}
	if string(sessionData) != "keep-when-source-missing" {
		t.Fatalf("unexpected target Session Storage data: %s", sessionData)
	}
	if _, err := os.Stat(staleTargetIndexedDB); !os.IsNotExist(err) {
		t.Fatalf("expected stale target IndexedDB to be removed, stat err=%v", err)
	}
}

func TestBrowserExtensionSyncProfileDataRejectsRunningTarget(t *testing.T) {
	app, dao := newExtensionSyncTestApp(t)
	extension := seedExtensionSyncData(t, app, dao)
	sourceProfile := &browser.Profile{ProfileId: "source", ProfileName: "主实例", UserDataDir: "source"}
	targetProfile := &browser.Profile{ProfileId: "target", ProfileName: "副实例", UserDataDir: "target", Running: true}
	app.browserMgr.Profiles[sourceProfile.ProfileId] = sourceProfile
	app.browserMgr.Profiles[targetProfile.ProfileId] = targetProfile
	for _, profile := range []*browser.Profile{sourceProfile, targetProfile} {
		if err := dao.UpsertBinding(browser.ExtensionBinding{
			ProfileId:   profile.ProfileId,
			ExtensionId: extension.ExtensionId,
			Mode:        "shared",
			Enabled:     true,
		}); err != nil {
			t.Fatal(err)
		}
	}

	_, err := app.BrowserExtensionSyncProfileData(BrowserExtensionSyncDataInput{
		ExtensionId:      extension.ExtensionId,
		SourceProfileId:  sourceProfile.ProfileId,
		TargetProfileIds: []string{targetProfile.ProfileId},
	})
	if err == nil || !strings.Contains(err.Error(), "正在运行") {
		t.Fatalf("expected running target to be rejected, got %v", err)
	}
}

func TestExtensionDirForBindingSyncsLocalDirectorySourceToLibrary(t *testing.T) {
	app := NewApp(t.TempDir())
	sourceDir := filepath.Join(app.appRootAbs(), "source-extension")
	libraryDir := filepath.Join(app.extensionLibraryRoot(), "extension-1")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "manifest.json"), []byte(`{"manifest_version":3,"name":"Local","version":"2.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "content.js"), []byte(`console.log("new")`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libraryDir, "manifest.json"), []byte(`{"manifest_version":3,"name":"Local","version":"1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := app.extensionDirForBinding(browser.ExtensionBinding{
		ProfileId:   "profile-1",
		ExtensionId: "extension-1",
		Mode:        "shared",
		Enabled:     true,
	}, &browser.Extension{
		ExtensionId: "extension-1",
		Name:        "Local",
		SourceType:  "directory",
		SourceURL:   sourceDir,
		InstallDir:  filepath.Join("data", "extensions", "library", "extension-1"),
	})
	if err != nil {
		t.Fatalf("extensionDirForBinding returned error: %v", err)
	}
	if !strings.EqualFold(filepath.Clean(got), filepath.Clean(libraryDir)) {
		t.Fatalf("expected stable library dir, got=%s want=%s", got, libraryDir)
	}
	if _, err := os.Stat(filepath.Join(libraryDir, "content.js")); err != nil {
		t.Fatalf("expected local source update to be synced into library dir: %v", err)
	}
}

func TestPrepareExtensionExclusiveDirCopiesLocalDirectorySource(t *testing.T) {
	app := NewApp(t.TempDir())
	sourceDir := filepath.Join(app.appRootAbs(), "source-extension")
	libraryDir := filepath.Join(app.extensionLibraryRoot(), "extension-1")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(libraryDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "manifest.json"), []byte(`{"manifest_version":3,"name":"Local","version":"2.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "content.js"), []byte(`console.log("new")`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(libraryDir, "manifest.json"), []byte(`{"manifest_version":3,"name":"Local","version":"1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	exclusiveDir, err := app.prepareExtensionExclusiveDir(&browser.Extension{
		ExtensionId: "extension-1",
		Name:        "Local",
		SourceType:  "directory",
		SourceURL:   sourceDir,
		InstallDir:  filepath.Join("data", "extensions", "library", "extension-1"),
	}, "profile-1")
	if err != nil {
		t.Fatalf("prepareExtensionExclusiveDir returned error: %v", err)
	}
	copied := app.resolveAppPath(filepath.Join(exclusiveDir, "content.js"))
	if _, err := os.Stat(copied); err != nil {
		t.Fatalf("expected exclusive dir to include source file: %v", err)
	}
}

func base64ForTest(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	if len(data) == 0 {
		return ""
	}
	var builder strings.Builder
	for i := 0; i < len(data); i += 3 {
		remaining := len(data) - i
		b0 := data[i]
		b1 := byte(0)
		b2 := byte(0)
		if remaining > 1 {
			b1 = data[i+1]
		}
		if remaining > 2 {
			b2 = data[i+2]
		}
		builder.WriteByte(alphabet[b0>>2])
		builder.WriteByte(alphabet[((b0&0x03)<<4)|(b1>>4)])
		if remaining > 1 {
			builder.WriteByte(alphabet[((b1&0x0f)<<2)|(b2>>6)])
		} else {
			builder.WriteByte('=')
		}
		if remaining > 2 {
			builder.WriteByte(alphabet[b2&0x3f])
		} else {
			builder.WriteByte('=')
		}
	}
	return builder.String()
}

func newExtensionSyncTestApp(t *testing.T) (*App, browser.ExtensionDAO) {
	t.Helper()
	root := t.TempDir()
	app := NewApp(root)
	cfg := config.DefaultConfig()
	app.config = cfg
	dbPath := filepath.Join(root, "data", "test.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatal(err)
	}
	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}
	app.browserMgr = browser.NewManager(cfg, root)
	app.browserMgr.ExtensionDAO = browser.NewSQLiteExtensionDAO(db.GetConn())
	dao, err := app.extensionDAO()
	if err != nil {
		t.Fatal(err)
	}
	return app, dao
}

func seedExtensionSyncData(t *testing.T, app *App, dao browser.ExtensionDAO) browser.Extension {
	t.Helper()
	extensionDir := filepath.Join(app.extensionLibraryRoot(), "extension-1")
	if err := os.MkdirAll(extensionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extensionDir, "manifest.json"), []byte(`{"manifest_version":3,"name":"Shared","version":"1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	extension := browser.Extension{
		ExtensionId:     "extension-1",
		Name:            "Shared",
		Version:         "1.0",
		ManifestVersion: 3,
		SourceType:      "directory",
		InstallDir:      filepath.Join("data", "extensions", "library", "extension-1"),
	}
	if err := dao.Upsert(extension); err != nil {
		t.Fatal(err)
	}
	return extension
}

func linkDirForTest(t *testing.T, linkPath string, targetPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetPath, linkPath); err == nil {
		return
	} else if runtime.GOOS != "windows" {
		t.Fatalf("create symlink failed: %v", err)
	}
	if err := createWindowsJunction(linkPath, targetPath); err != nil {
		t.Fatalf("create junction failed: %v", err)
	}
}
