package protoipc

import "testing"

func TestBrowserSnapshotRoundTrip(t *testing.T) {
	listRequest, err := DecodeBrowserSnapshotProfileRequest(EncodeBrowserSnapshotProfileRequest(BrowserSnapshotProfileRequest{
		ProfileID: "profile-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserSnapshotProfileRequest failed: %v", err)
	}
	if listRequest.ProfileID != "profile-1" {
		t.Fatalf("snapshot profile request was not preserved: %#v", listRequest)
	}

	createRequest, err := DecodeBrowserSnapshotCreateRequest(EncodeBrowserSnapshotCreateRequest(BrowserSnapshotCreateRequest{
		ProfileID: "profile-1",
		Name:      "上线前",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserSnapshotCreateRequest failed: %v", err)
	}
	if createRequest.ProfileID != "profile-1" || createRequest.Name != "上线前" {
		t.Fatalf("snapshot create request was not preserved: %#v", createRequest)
	}

	actionRequest, err := DecodeBrowserSnapshotActionRequest(EncodeBrowserSnapshotActionRequest(BrowserSnapshotActionRequest{
		ProfileID:  "profile-1",
		SnapshotID: "snapshot-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserSnapshotActionRequest failed: %v", err)
	}
	if actionRequest.ProfileID != "profile-1" || actionRequest.SnapshotID != "snapshot-1" {
		t.Fatalf("snapshot action request was not preserved: %#v", actionRequest)
	}

	snapshot := BrowserSnapshotInfo{
		SnapshotID:  "snapshot-1",
		ProfileID:   "profile-1",
		Name:        "上线前",
		SizeMBMilli: 12567,
		CreatedAt:   "2026-05-24T00:00:00Z",
	}

	listResponse, err := DecodeBrowserSnapshotListResponse(EncodeBrowserSnapshotListResponse(BrowserSnapshotListResponse{
		Snapshots: []BrowserSnapshotInfo{snapshot},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserSnapshotListResponse failed: %v", err)
	}
	if len(listResponse.Snapshots) != 1 || listResponse.Snapshots[0].SizeMBMilli != 12567 {
		t.Fatalf("snapshot list response was not preserved: %#v", listResponse)
	}

	snapshotResponse, err := DecodeBrowserSnapshotResponse(EncodeBrowserSnapshotResponse(BrowserSnapshotResponse{
		Snapshot: snapshot,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserSnapshotResponse failed: %v", err)
	}
	if snapshotResponse.Snapshot.SnapshotID != "snapshot-1" || snapshotResponse.Snapshot.Name != "上线前" {
		t.Fatalf("snapshot response was not preserved: %#v", snapshotResponse)
	}
}
