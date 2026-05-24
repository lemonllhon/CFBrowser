package protoipc

import "testing"

func TestBrowserCookieRoundTrip(t *testing.T) {
	profileRequest, err := DecodeBrowserCookieProfileRequest(EncodeBrowserCookieProfileRequest(BrowserCookieProfileRequest{
		ProfileID: "profile-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCookieProfileRequest failed: %v", err)
	}
	if profileRequest.ProfileID != "profile-1" {
		t.Fatalf("cookie profile request was not preserved: %#v", profileRequest)
	}

	importRequest, err := DecodeBrowserCookieImportRequest(EncodeBrowserCookieImportRequest(BrowserCookieImportRequest{
		ProfileID: "profile-1",
		Content:   "# Netscape HTTP Cookie File",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCookieImportRequest failed: %v", err)
	}
	if importRequest.ProfileID != "profile-1" || importRequest.Content == "" {
		t.Fatalf("cookie import request was not preserved: %#v", importRequest)
	}

	cookie := BrowserCookieInfo{
		Name:     "session",
		Value:    "abc123",
		Domain:   ".example.com",
		Path:     "/",
		Expires:  -1,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	}

	listResponse, err := DecodeBrowserCookieListResponse(EncodeBrowserCookieListResponse(BrowserCookieListResponse{
		Cookies: []BrowserCookieInfo{cookie},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCookieListResponse failed: %v", err)
	}
	if len(listResponse.Cookies) != 1 || listResponse.Cookies[0].Expires != -1 || !listResponse.Cookies[0].HTTPOnly {
		t.Fatalf("cookie list response was not preserved: %#v", listResponse)
	}

	exportResponse, err := DecodeBrowserCookieExportResponse(EncodeBrowserCookieExportResponse(BrowserCookieExportResponse{
		Content: "cookie text",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCookieExportResponse failed: %v", err)
	}
	if exportResponse.Content != "cookie text" {
		t.Fatalf("cookie export response was not preserved: %#v", exportResponse)
	}

	importResult, err := DecodeBrowserCookieImportResult(EncodeBrowserCookieImportResult(BrowserCookieImportResult{
		Imported: 3,
		Skipped:  1,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserCookieImportResult failed: %v", err)
	}
	if importResult.Imported != 3 || importResult.Skipped != 1 {
		t.Fatalf("cookie import result was not preserved: %#v", importResult)
	}
}
