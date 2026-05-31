package backend

import (
	"strings"
	"testing"

	"ant-chrome/backend/internal/browser"
)

func TestBrowserInstanceIdentityUsesStableProfileOrder(t *testing.T) {
	app := NewApp(t.TempDir())
	app.browserMgr = &browser.Manager{
		Profiles: map[string]*browser.Profile{
			"profile-b": {ProfileId: "profile-b", ProfileName: "Beta", CreatedAt: "2026-01-02T00:00:00Z", DebugPort: 9223, Pid: 12},
			"profile-a": {ProfileId: "profile-a", ProfileName: "Alpha", CreatedAt: "2026-01-01T00:00:00Z", DebugPort: 9222, Pid: 11},
		},
	}

	identity := app.browserInstanceIdentityLocked(app.browserMgr.Profiles["profile-b"])
	if identity.Index != 2 {
		t.Fatalf("expected Beta to be second in profile order, got %d", identity.Index)
	}
	if !strings.HasPrefix(identity.Marker, "[Trace #02] Beta") {
		t.Fatalf("unexpected marker: %q", identity.Marker)
	}

	protoProfile := app.browserProfileToProto(*app.browserMgr.Profiles["profile-b"])
	if protoProfile.InstanceMarkerIndex != 2 {
		t.Fatalf("expected proto profile marker index 2, got %d", protoProfile.InstanceMarkerIndex)
	}
	if protoProfile.InstanceMarker != identity.Marker {
		t.Fatalf("expected proto profile marker %q, got %q", identity.Marker, protoProfile.InstanceMarker)
	}
}

func TestRenderBrowserInstanceIconWritesICOHeader(t *testing.T) {
	data, err := renderBrowserInstanceIcon(12)
	if err != nil {
		t.Fatalf("render icon: %v", err)
	}
	if len(data) < 6 {
		t.Fatalf("icon data too short: %d", len(data))
	}
	if data[0] != 0 || data[1] != 0 || data[2] != 1 || data[3] != 0 {
		t.Fatalf("unexpected ico header: %v", data[:4])
	}
	if data[4] != 7 || data[5] != 0 {
		t.Fatalf("expected seven icon frames, header bytes=%v", data[4:6])
	}
}

func TestDecodeBrowserInstanceBaseIconFrames(t *testing.T) {
	frames, err := decodeBrowserInstanceBaseIconFrames(browserInstanceBaseIconICO)
	if err != nil {
		t.Fatalf("decode base icon: %v", err)
	}
	for _, size := range []int{16, 24, 32, 48, 64, 128, 256} {
		frame := frames[size]
		if frame == nil {
			t.Fatalf("missing %dpx base icon frame", size)
		}
		if frame.Bounds().Dx() != size || frame.Bounds().Dy() != size {
			t.Fatalf("unexpected %dpx frame bounds: %v", size, frame.Bounds())
		}
	}
}
