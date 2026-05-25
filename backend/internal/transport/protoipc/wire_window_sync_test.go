package protoipc

import "testing"

func TestWindowSyncStateRoundTrip(t *testing.T) {
	state := &WindowSyncState{
		SessionID:       "session-1",
		Active:          true,
		Paused:          true,
		MasterProfileID: "p1",
		ProfileIDs:      []string{"p1", "p2"},
		Windows: []WindowSyncCandidate{
			{
				ProfileID:   "p1",
				ProfileName: "Master",
				DebugPort:   9222,
				PID:         1001,
				Running:     true,
				DebugReady:  true,
				Role:        "master",
				Master:      true,
				CanSync:     true,
			},
			{
				ProfileID:    "p2",
				ProfileName:  "Worker",
				DebugPort:    9223,
				PID:          1002,
				Running:      true,
				DebugReady:   true,
				Role:         "controlled",
				CanSync:      true,
				CanAutoStart: true,
			},
		},
		MasterColor:  "#2563eb",
		SyncKeyboard: true,
		SyncMouse:    false,
		Layout: WindowSyncLayoutSettings{
			Mode:      "custom",
			Width:     1200,
			Height:    720,
			GapX:      12,
			GapY:      16,
			PerRow:    2,
			UpdatedAt: "2026-05-24T12:00:00+08:00",
		},
		StartedAt: "2026-05-24T11:59:00+08:00",
		UpdatedAt: "2026-05-24T12:00:00+08:00",
	}

	decoded, err := DecodeWindowSyncStateResponse(EncodeWindowSyncStateResponse(WindowSyncStateResponse{State: state}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncStateResponse failed: %v", err)
	}
	if decoded.State == nil {
		t.Fatalf("expected state to be present")
	}
	if decoded.State.SessionID != "session-1" || !decoded.State.Paused {
		t.Fatalf("state scalar fields were not preserved: %#v", decoded.State)
	}
	if len(decoded.State.Windows) != 2 || decoded.State.Windows[0].Role != "master" || !decoded.State.Windows[1].CanAutoStart {
		t.Fatalf("state windows were not preserved: %#v", decoded.State.Windows)
	}
	if decoded.State.Layout.Mode != "custom" || decoded.State.Layout.GapY != 16 {
		t.Fatalf("state layout was not preserved: %#v", decoded.State.Layout)
	}
	if !decoded.State.SyncKeyboard || decoded.State.SyncMouse {
		t.Fatalf("state settings were not preserved: %#v", decoded.State)
	}

	empty, err := DecodeWindowSyncStateResponse(EncodeWindowSyncStateResponse(WindowSyncStateResponse{}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncStateResponse empty failed: %v", err)
	}
	if empty.State != nil {
		t.Fatalf("expected nil state, got %#v", empty.State)
	}
}

func TestWindowSyncRequestsRoundTrip(t *testing.T) {
	start, err := DecodeWindowSyncStartRequest(EncodeWindowSyncStartRequest(WindowSyncStartRequest{
		ProfileIDs:      []string{"p1", "p2"},
		MasterProfileID: "p1",
	}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncStartRequest failed: %v", err)
	}
	if len(start.ProfileIDs) != 2 || start.MasterProfileID != "p1" {
		t.Fatalf("start request was not preserved: %#v", start)
	}

	layout, err := DecodeWindowSyncLayoutSettings(EncodeWindowSyncLayoutSettings(WindowSyncLayoutSettings{
		Mode:   "grid",
		Scope:  "toolbar-screen",
		Width:  1500,
		Height: 500,
		GapX:   10,
		GapY:   10,
		PerRow: 2,
	}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncLayoutSettings failed: %v", err)
	}
	if layout.Mode != "grid" || layout.Scope != "toolbar-screen" || layout.Width != 1500 || layout.PerRow != 2 {
		t.Fatalf("layout request was not preserved: %#v", layout)
	}

	settings, err := DecodeWindowSyncSettings(EncodeWindowSyncSettings(WindowSyncSettings{
		MasterColor:  "#22c55e",
		SyncKeyboard: false,
		SyncMouse:    true,
	}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncSettings failed: %v", err)
	}
	if settings.MasterColor != "#22c55e" || settings.SyncKeyboard || !settings.SyncMouse {
		t.Fatalf("settings request was not preserved: %#v", settings)
	}
}

func TestWindowSyncBatchAndActionRoundTrip(t *testing.T) {
	different, err := DecodeWindowSyncBatchInputDifferentRequest(EncodeWindowSyncBatchInputDifferentRequest(WindowSyncBatchInputDifferentRequest{
		Items: []WindowSyncBatchInputDifferentItem{
			{ProfileID: "p1", Text: "one"},
			{ProfileID: "p2", Text: "two"},
		},
	}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncBatchInputDifferentRequest failed: %v", err)
	}
	if len(different.Items) != 2 || different.Items[1].Text != "two" {
		t.Fatalf("different input request was not preserved: %#v", different)
	}

	batch, err := DecodeWindowSyncBatchInputResult(EncodeWindowSyncBatchInputResult(WindowSyncBatchInputResult{
		Total:   2,
		Success: 1,
		Failed:  1,
		Results: []WindowSyncBatchInputResultItem{
			{ProfileID: "p1", ProfileName: "Master", Master: true, Success: true},
			{ProfileID: "p2", ProfileName: "Worker", Error: "failed"},
		},
	}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncBatchInputResult failed: %v", err)
	}
	if batch.Total != 2 || batch.Success != 1 || len(batch.Results) != 2 || batch.Results[1].Error != "failed" {
		t.Fatalf("batch result was not preserved: %#v", batch)
	}

	action, err := DecodeWindowSyncActionResult(EncodeWindowSyncActionResult(WindowSyncActionResult{
		Total:   1,
		Success: 1,
		Results: []WindowSyncActionResultItem{{ProfileID: "p1", ProfileName: "Master", Master: true, Success: true}},
	}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncActionResult failed: %v", err)
	}
	if action.Total != 1 || len(action.Results) != 1 || !action.Results[0].Master {
		t.Fatalf("action result was not preserved: %#v", action)
	}

	resize, err := DecodeWindowSyncToolbarResizeRequest(EncodeWindowSyncToolbarResizeRequest(WindowSyncToolbarResizeRequest{
		Width:  780,
		Height: 430,
	}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncToolbarResizeRequest failed: %v", err)
	}
	if resize.Width != 780 || resize.Height != 430 {
		t.Fatalf("toolbar resize request was not preserved: %#v", resize)
	}

	response, err := DecodeWindowSyncToolbarResizeResponse(EncodeWindowSyncToolbarResizeResponse(WindowSyncToolbarResizeResponse{OK: true}))
	if err != nil {
		t.Fatalf("DecodeWindowSyncToolbarResizeResponse failed: %v", err)
	}
	if !response.OK {
		t.Fatalf("toolbar resize response was not preserved: %#v", response)
	}
}
