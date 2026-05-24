package protoipc

import "testing"

func TestBrowserSettingsRoundTrip(t *testing.T) {
	settings := BrowserSettings{
		UserDataRoot:           "data/profiles",
		DefaultFingerprintArgs: []string{"--fingerprint-brand=Chrome", "--fingerprint-platform=windows"},
		DefaultLaunchArgs:      []string{"--disable-features=Translate", "--start-maximized"},
		DefaultProxy:           "socks5://127.0.0.1:1080",
		StartReadyTimeoutMs:    15000,
		StartStableWindowMs:    2400,
	}

	decodedRequest, err := DecodeBrowserSettingsSaveRequest(EncodeBrowserSettingsSaveRequest(BrowserSettingsSaveRequest{
		Settings: settings,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserSettingsSaveRequest failed: %v", err)
	}
	if decodedRequest.Settings.UserDataRoot != "data/profiles" {
		t.Fatalf("user data root was not preserved: %#v", decodedRequest.Settings)
	}
	if len(decodedRequest.Settings.DefaultFingerprintArgs) != 2 || decodedRequest.Settings.DefaultFingerprintArgs[1] != "--fingerprint-platform=windows" {
		t.Fatalf("fingerprint args were not preserved: %#v", decodedRequest.Settings.DefaultFingerprintArgs)
	}
	if len(decodedRequest.Settings.DefaultLaunchArgs) != 2 || decodedRequest.Settings.DefaultLaunchArgs[0] != "--disable-features=Translate" {
		t.Fatalf("launch args were not preserved: %#v", decodedRequest.Settings.DefaultLaunchArgs)
	}
	if decodedRequest.Settings.StartReadyTimeoutMs != 15000 || decodedRequest.Settings.StartStableWindowMs != 2400 {
		t.Fatalf("start timing was not preserved: %#v", decodedRequest.Settings)
	}

	decodedResponse, err := DecodeBrowserSettingsResponse(EncodeBrowserSettingsResponse(BrowserSettingsResponse{
		Settings: settings,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserSettingsResponse failed: %v", err)
	}
	if decodedResponse.Settings.DefaultProxy != "socks5://127.0.0.1:1080" {
		t.Fatalf("settings response was not preserved: %#v", decodedResponse.Settings)
	}
}
