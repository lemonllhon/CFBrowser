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

func TestBrowserProfileListRefreshesProfileConfigFromStore(t *testing.T) {
	ln := mustListenLoopback(t)
	defer ln.Close()

	app := newRuntimeStateTestApp(t.TempDir())
	app.browserMgr.Profiles["profile-1"] = &BrowserProfile{
		ProfileId:   "profile-1",
		ProfileName: "Old name",
		Running:     true,
		DebugPort:   listenerPort(t, ln),
		LastStartAt: "2026-05-26T00:00:00Z",
	}
	app.browserMgr.Profiles["deleted-profile"] = &BrowserProfile{
		ProfileId:   "deleted-profile",
		ProfileName: "Deleted",
	}
	app.browserMgr.ProfileDAO = &profileDAOListStub{profiles: []*BrowserProfile{
		{
			ProfileId:   "profile-1",
			ProfileName: "Updated name",
			Tags:        []string{"synced"},
		},
		{
			ProfileId:   "created-profile",
			ProfileName: "Created elsewhere",
		},
	}}

	profiles := app.BrowserProfileList()
	if len(profiles) != 2 {
		t.Fatalf("expected refreshed profile list, got %d: %#v", len(profiles), profiles)
	}

	byID := map[string]BrowserProfile{}
	for _, profile := range profiles {
		byID[profile.ProfileId] = profile
	}
	updated := byID["profile-1"]
	if updated.ProfileName != "Updated name" || len(updated.Tags) != 1 || updated.Tags[0] != "synced" {
		t.Fatalf("expected persisted config changes to be visible: %#v", updated)
	}
	if !updated.Running || updated.DebugPort != listenerPort(t, ln) || updated.LastStartAt == "" {
		t.Fatalf("expected local runtime fields to survive config refresh: %#v", updated)
	}
	if _, exists := byID["created-profile"]; !exists {
		t.Fatalf("expected created profile from store to be visible: %#v", profiles)
	}
	if _, exists := byID["deleted-profile"]; exists {
		t.Fatalf("expected deleted profile to disappear: %#v", profiles)
	}
}

func TestBrowserProfileCreateRefreshesBeforeWriting(t *testing.T) {
	app := newRuntimeStateTestApp(t.TempDir())
	app.browserMgr.Profiles["deleted-profile"] = &BrowserProfile{
		ProfileId:   "deleted-profile",
		ProfileName: "Deleted elsewhere",
	}
	dao := &profileDAOListStub{}
	app.browserMgr.ProfileDAO = dao

	created, err := app.BrowserProfileCreate(BrowserProfileInput{ProfileName: "Created here"})
	if err != nil {
		t.Fatalf("BrowserProfileCreate failed: %v", err)
	}
	if created == nil || created.ProfileId == "" {
		t.Fatalf("expected created profile with id, got %#v", created)
	}
	if len(dao.upsertedIDs) != 1 || dao.upsertedIDs[0] != created.ProfileId {
		t.Fatalf("expected create to avoid resurrecting stale cached profiles, upserted=%v created=%s", dao.upsertedIDs, created.ProfileId)
	}
}

func newRuntimeStateTestApp(appRoot string) *App {
	app := NewApp(appRoot)
	app.browserMgr = browser.NewManager(config.DefaultConfig(), appRoot)
	app.browserMgr.Profiles = map[string]*BrowserProfile{}
	app.browserMgr.BrowserProcesses = map[string]*exec.Cmd{}
	return app
}

type profileDAOListStub struct {
	profiles    []*BrowserProfile
	upsertedIDs []string
}

func (s *profileDAOListStub) List() ([]*BrowserProfile, error) {
	out := make([]*BrowserProfile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		if profile == nil {
			continue
		}
		snapshot := *profile
		out = append(out, &snapshot)
	}
	return out, nil
}

func (s *profileDAOListStub) GetById(profileId string) (*BrowserProfile, error) {
	for _, profile := range s.profiles {
		if profile != nil && profile.ProfileId == profileId {
			snapshot := *profile
			return &snapshot, nil
		}
	}
	return nil, os.ErrNotExist
}

func (s *profileDAOListStub) Upsert(profile *BrowserProfile) error {
	if profile != nil {
		s.upsertedIDs = append(s.upsertedIDs, profile.ProfileId)
	}
	return nil
}

func (s *profileDAOListStub) Delete(profileId string) error {
	return nil
}
