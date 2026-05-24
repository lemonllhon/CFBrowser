package protoipc

import "testing"

func TestBrowserExtensionRoundTrip(t *testing.T) {
	extension := BrowserExtension{
		ExtensionID:     "extension-1",
		Name:            "Trace Helper",
		Version:         "1.2.3",
		ManifestVersion: 3,
		Description:     "helper",
		SourceType:      "zip",
		SourceURL:       "https://example.test/helper.zip",
		InstallDir:      "data/extensions/library/extension-1",
		PackagePath:     "data/extensions/packages/extension-1.zip",
		ManifestJSON:    `{"manifest_version":3}`,
		BoundCount:      2,
		AutoBindEnabled: true,
		AutoBindMode:    "exclusive",
		CreatedAt:       "2026-05-24T00:00:00Z",
		UpdatedAt:       "2026-05-24T00:01:00Z",
	}

	decodedExtension, err := DecodeBrowserExtension(EncodeBrowserExtension(extension))
	if err != nil {
		t.Fatalf("DecodeBrowserExtension failed: %v", err)
	}
	if decodedExtension.ExtensionID != extension.ExtensionID || decodedExtension.ManifestVersion != 3 || !decodedExtension.AutoBindEnabled {
		t.Fatalf("extension was not preserved: %#v", decodedExtension)
	}

	listResponse, err := DecodeBrowserExtensionListResponse(EncodeBrowserExtensionListResponse(BrowserExtensionListResponse{
		Extensions: []BrowserExtension{extension},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionListResponse failed: %v", err)
	}
	if len(listResponse.Extensions) != 1 || listResponse.Extensions[0].Name != extension.Name {
		t.Fatalf("extension list response was not preserved: %#v", listResponse)
	}

	singleResponse, err := DecodeBrowserExtensionResponse(EncodeBrowserExtensionResponse(BrowserExtensionResponse{
		Extension: extension,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionResponse failed: %v", err)
	}
	if singleResponse.Extension.ExtensionID != extension.ExtensionID {
		t.Fatalf("extension response was not preserved: %#v", singleResponse)
	}
}

func TestBrowserExtensionMutationRoundTrip(t *testing.T) {
	idRequest, err := DecodeBrowserExtensionIDRequest(EncodeBrowserExtensionIDRequest(BrowserExtensionIDRequest{
		ExtensionID: "extension-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionIDRequest failed: %v", err)
	}
	if idRequest.ExtensionID != "extension-1" {
		t.Fatalf("extension id request was not preserved: %#v", idRequest)
	}

	profileRequest, err := DecodeBrowserExtensionProfileRequest(EncodeBrowserExtensionProfileRequest(BrowserExtensionProfileRequest{
		ProfileID: "profile-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionProfileRequest failed: %v", err)
	}
	if profileRequest.ProfileID != "profile-1" {
		t.Fatalf("extension profile request was not preserved: %#v", profileRequest)
	}

	importRequest, err := DecodeBrowserExtensionImportRequest(EncodeBrowserExtensionImportRequest(BrowserExtensionImportRequest{
		Path:     `D:\extensions\helper.zip`,
		Mode:     "overwrite",
		Existing: "extension-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionImportRequest failed: %v", err)
	}
	if importRequest.Path == "" || importRequest.Mode != "overwrite" || importRequest.Existing != "extension-1" {
		t.Fatalf("extension import request was not preserved: %#v", importRequest)
	}

	assignRequest, err := DecodeBrowserExtensionAssignRequest(EncodeBrowserExtensionAssignRequest(BrowserExtensionAssignRequest{
		ExtensionID: "extension-1",
		ProfileIDs:  []string{"profile-1", "profile-2"},
		Mode:        "exclusive",
		Enabled:     true,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionAssignRequest failed: %v", err)
	}
	if assignRequest.ExtensionID != "extension-1" || len(assignRequest.ProfileIDs) != 2 || !assignRequest.Enabled {
		t.Fatalf("extension assign request was not preserved: %#v", assignRequest)
	}

	autoBindRequest, err := DecodeBrowserExtensionAutoBindRequest(EncodeBrowserExtensionAutoBindRequest(BrowserExtensionAutoBindRequest{
		ExtensionID: "extension-1",
		Enabled:     true,
		Mode:        "shared",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionAutoBindRequest failed: %v", err)
	}
	if autoBindRequest.ExtensionID != "extension-1" || !autoBindRequest.Enabled || autoBindRequest.Mode != "shared" {
		t.Fatalf("extension auto bind request was not preserved: %#v", autoBindRequest)
	}

	unassignRequest, err := DecodeBrowserExtensionUnassignRequest(EncodeBrowserExtensionUnassignRequest(BrowserExtensionUnassignRequest{
		ExtensionID: "extension-1",
		ProfileIDs:  []string{"profile-1"},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionUnassignRequest failed: %v", err)
	}
	if unassignRequest.ExtensionID != "extension-1" || len(unassignRequest.ProfileIDs) != 1 {
		t.Fatalf("extension unassign request was not preserved: %#v", unassignRequest)
	}
}

func TestBrowserExtensionResultRoundTrip(t *testing.T) {
	extension := BrowserExtension{ExtensionID: "extension-1", Name: "Trace Helper"}
	importResult, err := DecodeBrowserExtensionImportResult(EncodeBrowserExtensionImportResult(BrowserExtensionImportResult{
		Duplicate: true,
		Message:   "duplicate",
		Existing:  &extension,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionImportResult failed: %v", err)
	}
	if !importResult.Duplicate || importResult.Existing == nil || importResult.Existing.ExtensionID != "extension-1" {
		t.Fatalf("extension import result was not preserved: %#v", importResult)
	}

	chooseResponse, err := DecodeBrowserExtensionChoosePathResponse(EncodeBrowserExtensionChoosePathResponse(BrowserExtensionChoosePathResponse{
		Cancelled: false,
		Path:      `D:\extensions\helper.zip`,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionChoosePathResponse failed: %v", err)
	}
	if chooseResponse.Cancelled || chooseResponse.Path == "" {
		t.Fatalf("extension choose path response was not preserved: %#v", chooseResponse)
	}

	binding := BrowserExtensionBinding{
		ID:               42,
		ProfileID:        "profile-1",
		ProfileName:      "默认",
		ExtensionID:      "extension-1",
		ExtensionName:    "Trace Helper",
		ExtensionVersion: "1.2.3",
		Mode:             "exclusive",
		Enabled:          true,
		ExclusiveDir:     "data/extensions/exclusive/profile-1/extension-1",
		CreatedAt:        "2026-05-24T00:00:00Z",
		UpdatedAt:        "2026-05-24T00:01:00Z",
	}
	bindings, err := DecodeBrowserExtensionBindingListResponse(EncodeBrowserExtensionBindingListResponse(BrowserExtensionBindingListResponse{
		Bindings: []BrowserExtensionBinding{binding},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserExtensionBindingListResponse failed: %v", err)
	}
	if len(bindings.Bindings) != 1 || bindings.Bindings[0].ID != 42 || !bindings.Bindings[0].Enabled {
		t.Fatalf("extension bindings were not preserved: %#v", bindings)
	}
}
