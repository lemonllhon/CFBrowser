package backend

import "testing"

type noopWindowSyncToolbarAdapter struct{}

func (noopWindowSyncToolbarAdapter) Show(_ *App, _ *WindowSyncState) error { return nil }
func (noopWindowSyncToolbarAdapter) Update(_ *WindowSyncState) error       { return nil }
func (noopWindowSyncToolbarAdapter) Hide() error                           { return nil }
func (noopWindowSyncToolbarAdapter) SetSize(_ int, _ int) error            { return nil }
func (noopWindowSyncToolbarAdapter) CenterPoint() (int, int, bool)         { return 0, 0, false }

func TestHandleWindowSyncControlledStoppedRemovesWindow(t *testing.T) {
	app := NewApp(t.TempDir())
	app.SetWindowSyncToolbarAdapter(noopWindowSyncToolbarAdapter{})
	app.windowSyncState = testWindowSyncState([]string{"p1", "p2", "p3"}, "p1")

	app.handleWindowSyncProfileStopped("p2", "stopped")

	state := app.windowSyncState
	if state == nil || !state.Active {
		t.Fatalf("expected window sync to remain active, got %#v", state)
	}
	if len(state.Windows) != 2 || len(state.ProfileIds) != 2 {
		t.Fatalf("expected one controlled window to be removed, got %#v", state)
	}
	if windowSyncStateHasProfile(state, "p2") {
		t.Fatalf("expected removed profile to disappear from state: %#v", state)
	}
	if !windowSyncStateHasProfile(state, "p1") || !windowSyncStateHasProfile(state, "p3") {
		t.Fatalf("expected remaining profiles to stay in sync state: %#v", state)
	}
}

func TestHandleWindowSyncControlledStoppedStopsWhenOnlyMasterRemains(t *testing.T) {
	app := NewApp(t.TempDir())
	app.SetWindowSyncToolbarAdapter(noopWindowSyncToolbarAdapter{})
	app.windowSyncState = testWindowSyncState([]string{"p1", "p2"}, "p1")

	app.handleWindowSyncProfileStopped("p2", "stopped")

	if app.windowSyncState != nil {
		t.Fatalf("expected window sync to stop when fewer than two windows remain, got %#v", app.windowSyncState)
	}
}

func TestHandleWindowSyncMasterStoppedStopsSession(t *testing.T) {
	app := NewApp(t.TempDir())
	app.SetWindowSyncToolbarAdapter(noopWindowSyncToolbarAdapter{})
	app.windowSyncState = testWindowSyncState([]string{"p1", "p2", "p3"}, "p1")

	app.handleWindowSyncProfileStopped("p1", "stopped")

	if app.windowSyncState != nil {
		t.Fatalf("expected master stop to clear active window sync state, got %#v", app.windowSyncState)
	}
}

func testWindowSyncState(profileIds []string, masterProfileId string) *WindowSyncState {
	windows := make([]WindowSyncCandidate, 0, len(profileIds))
	for index, profileId := range profileIds {
		isMaster := profileId == masterProfileId
		role := "controlled"
		if isMaster {
			role = "master"
		}
		windows = append(windows, WindowSyncCandidate{
			ProfileId:   profileId,
			ProfileName: profileId,
			DebugPort:   9000 + index,
			Pid:         1000 + index,
			Running:     true,
			DebugReady:  true,
			Role:        role,
			Master:      isMaster,
			CanSync:     true,
		})
	}
	return &WindowSyncState{
		SessionId:       "test-session",
		Active:          true,
		MasterProfileId: masterProfileId,
		ProfileIds:      append([]string{}, profileIds...),
		Windows:         windows,
		MasterColor:     defaultWindowSyncMasterColor,
		SyncKeyboard:    true,
		SyncMouse:       true,
		Layout:          defaultWindowSyncLayoutSettings(),
	}
}
