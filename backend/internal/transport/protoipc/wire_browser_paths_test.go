package protoipc

import "testing"

func TestBrowserPathRequestsRoundTrip(t *testing.T) {
	userDataDirRequest, err := DecodeBrowserUserDataDirOpenRequest(EncodeBrowserUserDataDirOpenRequest(BrowserUserDataDirOpenRequest{
		UserDataDir: "profile-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserUserDataDirOpenRequest failed: %v", err)
	}
	if userDataDirRequest.UserDataDir != "profile-1" {
		t.Fatalf("user data dir request was not preserved: %#v", userDataDirRequest)
	}

	profileRequest, err := DecodeBrowserProfileUserDataDirOpenRequest(EncodeBrowserProfileUserDataDirOpenRequest(BrowserProfileUserDataDirOpenRequest{
		ProfileID: "profile-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileUserDataDirOpenRequest failed: %v", err)
	}
	if profileRequest.ProfileID != "profile-1" {
		t.Fatalf("profile user data dir request was not preserved: %#v", profileRequest)
	}
}
