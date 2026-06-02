package protoipc

import "testing"

func TestBrowserProfileBackupWireRoundTrip(t *testing.T) {
	exportRequest, err := DecodeBrowserProfileBackupExportRequest(EncodeBrowserProfileBackupExportRequest(BrowserProfileBackupExportRequest{
		Scope:                          "selected",
		ProfileIDs:                     []string{"profile-1", "profile-2"},
		IncludeCookies:                 true,
		IncludePlainCookiesWhenRunning: true,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileBackupExportRequest failed: %v", err)
	}
	if exportRequest.Scope != "selected" || len(exportRequest.ProfileIDs) != 2 || !exportRequest.IncludeCookies || !exportRequest.IncludePlainCookiesWhenRunning {
		t.Fatalf("export request mismatch: %#v", exportRequest)
	}

	importRequest, err := DecodeBrowserProfileBackupImportRequest(EncodeBrowserProfileBackupImportRequest(BrowserProfileBackupImportRequest{
		ZipPath:        "C:/tmp/instances.zip",
		RestoreCookies: true,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileBackupImportRequest failed: %v", err)
	}
	if importRequest.ZipPath != "C:/tmp/instances.zip" || !importRequest.RestoreCookies {
		t.Fatalf("import request mismatch: %#v", importRequest)
	}

	result := BrowserProfileBackupActionResult{
		Message:            "done",
		ZipPath:            "C:/tmp/instances.zip",
		CreatedAt:          "2026-06-02T00:00:00Z",
		Exported:           2,
		Imported:           1,
		Failed:             1,
		ProfileCount:       2,
		CookieProfileCount: 1,
		Summary: BrowserProfileBackupSummary{
			ZipPath:              "C:/tmp/instances.zip",
			Format:               "trace-browser-instance-backup",
			Version:              1,
			AppName:              "Trace Browser",
			AppVersion:           "dev",
			CreatedAt:            "2026-06-02T00:00:00Z",
			SourceOS:             "windows",
			ProfileCount:         2,
			CookieProfileCount:   1,
			IncludesCookies:      true,
			IncludesPlainCookies: true,
			CookieNotice:         "notice",
			Warnings:             []string{"warning"},
		},
		Warnings: []BrowserProfileBackupWarning{
			{ProfileID: "profile-1", ProfileName: "Profile 1", Message: "warning"},
		},
	}
	decoded, err := DecodeBrowserProfileBackupActionResult(EncodeBrowserProfileBackupActionResult(result))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileBackupActionResult failed: %v", err)
	}
	if decoded.ProfileCount != 2 || decoded.CookieProfileCount != 1 || !decoded.Summary.IncludesCookies || len(decoded.Warnings) != 1 {
		t.Fatalf("action result mismatch: %#v", decoded)
	}
}
