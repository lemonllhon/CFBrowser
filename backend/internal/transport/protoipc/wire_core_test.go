package protoipc

import "testing"

func TestBrowserCoreRoundTrip(t *testing.T) {
	core := BrowserCore{
		CoreID:    "core-1",
		CoreName:  "Chrome 142",
		CorePath:  "chrome/142",
		IsDefault: true,
	}

	decodedCore, err := DecodeBrowserCore(EncodeBrowserCore(core))
	if err != nil {
		t.Fatalf("DecodeBrowserCore failed: %v", err)
	}
	if decodedCore.CoreID != core.CoreID || decodedCore.CorePath != core.CorePath || !decodedCore.IsDefault {
		t.Fatalf("core was not preserved: %#v", decodedCore)
	}

	listResponse, err := DecodeBrowserCoreListResponse(EncodeBrowserCoreListResponse(BrowserCoreListResponse{
		Cores: []BrowserCore{core},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCoreListResponse failed: %v", err)
	}
	if len(listResponse.Cores) != 1 || listResponse.Cores[0].CoreName != core.CoreName {
		t.Fatalf("core list was not preserved: %#v", listResponse)
	}

	saveRequest, err := DecodeBrowserCoreSaveRequest(EncodeBrowserCoreSaveRequest(BrowserCoreSaveRequest{Core: core}))
	if err != nil {
		t.Fatalf("DecodeBrowserCoreSaveRequest failed: %v", err)
	}
	if saveRequest.Core.CoreID != core.CoreID {
		t.Fatalf("save request was not preserved: %#v", saveRequest)
	}
}

func TestBrowserCoreOperationRoundTrip(t *testing.T) {
	idRequest, err := DecodeBrowserCoreIDRequest(EncodeBrowserCoreIDRequest(BrowserCoreIDRequest{CoreID: "core-1"}))
	if err != nil {
		t.Fatalf("DecodeBrowserCoreIDRequest failed: %v", err)
	}
	if idRequest.CoreID != "core-1" {
		t.Fatalf("id request was not preserved: %#v", idRequest)
	}

	pathRequest, err := DecodeBrowserCorePathRequest(EncodeBrowserCorePathRequest(BrowserCorePathRequest{CorePath: "chrome/142"}))
	if err != nil {
		t.Fatalf("DecodeBrowserCorePathRequest failed: %v", err)
	}
	if pathRequest.CorePath != "chrome/142" {
		t.Fatalf("path request was not preserved: %#v", pathRequest)
	}

	validateResponse, err := DecodeBrowserCoreValidateResponse(EncodeBrowserCoreValidateResponse(BrowserCoreValidateResponse{
		Valid:   true,
		Message: "路径有效",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCoreValidateResponse failed: %v", err)
	}
	if !validateResponse.Valid || validateResponse.Message == "" {
		t.Fatalf("validate response was not preserved: %#v", validateResponse)
	}

	renameRequest, err := DecodeBrowserCoreRenamePathRequest(EncodeBrowserCoreRenamePathRequest(BrowserCoreRenamePathRequest{
		CorePath:      "chrome/142",
		NewFolderName: "chromium-142",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCoreRenamePathRequest failed: %v", err)
	}
	if renameRequest.CorePath != "chrome/142" || renameRequest.NewFolderName != "chromium-142" {
		t.Fatalf("rename path request was not preserved: %#v", renameRequest)
	}

	extended := BrowserCoreExtendedInfo{
		CoreID:        "core-1",
		ChromeVersion: "142.0.0.0",
		InstanceCount: 3,
	}
	extendedResponse, err := DecodeBrowserCoreExtendedInfoResponse(EncodeBrowserCoreExtendedInfoResponse(BrowserCoreExtendedInfoResponse{
		Items: []BrowserCoreExtendedInfo{extended},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCoreExtendedInfoResponse failed: %v", err)
	}
	if len(extendedResponse.Items) != 1 || extendedResponse.Items[0].InstanceCount != 3 {
		t.Fatalf("extended response was not preserved: %#v", extendedResponse)
	}

	downloadRequest, err := DecodeBrowserCoreDownloadRequest(EncodeBrowserCoreDownloadRequest(BrowserCoreDownloadRequest{
		CoreName:    "Chrome 142",
		URL:         "https://example.test/chrome.zip",
		ProxyConfig: "__system__",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCoreDownloadRequest failed: %v", err)
	}
	if downloadRequest.CoreName != "Chrome 142" || downloadRequest.ProxyConfig != "__system__" {
		t.Fatalf("download request was not preserved: %#v", downloadRequest)
	}
}

func TestBrowserCoreDownloadProgressRoundTrip(t *testing.T) {
	progress, err := DecodeBrowserCoreDownloadProgress(EncodeBrowserCoreDownloadProgress(BrowserCoreDownloadProgress{
		Phase:    "downloading",
		Progress: 42,
		Message:  "正在下载",
		CorePath: "chrome/142",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCoreDownloadProgress failed: %v", err)
	}
	if progress.Phase != "downloading" || progress.Progress != 42 || progress.Message == "" || progress.CorePath != "chrome/142" {
		t.Fatalf("download progress was not preserved: %#v", progress)
	}
}
