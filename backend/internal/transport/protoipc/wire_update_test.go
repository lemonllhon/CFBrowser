package protoipc

import "testing"

func TestAppUpdateInfoRoundTrip(t *testing.T) {
	original := AppUpdateInfo{
		CurrentVersion:         "1.0.0",
		LatestVersion:          "1.1.0",
		ReleaseName:            "v1.1.0",
		ReleaseURL:             "https://example.test/releases/1.1.0",
		PublishedAt:            "2026-05-24T00:00:00Z",
		Body:                   "release notes",
		HasUpdate:              true,
		Asset:                  &AppUpdateAsset{Name: "setup.exe", Size: 123, DownloadURL: "https://example.test/setup.exe", Checksum: "abc123"},
		InstallerAsset:         &AppUpdateAsset{Name: "installer.exe", Size: 456, DownloadURL: "https://example.test/installer.exe", Checksum: "def456"},
		PortableAsset:          &AppUpdateAsset{Name: "portable.zip", Size: 789, DownloadURL: "https://example.test/portable.zip"},
		DistributionKind:       "installer",
		RecommendedPackageKind: "installer",
		CanSelfUpdatePortable:  true,
		Message:                "检测到新版本",
	}
	decoded, err := DecodeAppUpdateInfo(EncodeAppUpdateInfo(original))
	if err != nil {
		t.Fatalf("DecodeAppUpdateInfo failed: %v", err)
	}
	if decoded.LatestVersion != original.LatestVersion || !decoded.HasUpdate {
		t.Fatalf("update info was not preserved: %#v", decoded)
	}
	if decoded.InstallerAsset == nil || decoded.InstallerAsset.Size != 456 || decoded.InstallerAsset.Checksum != "def456" {
		t.Fatalf("installer asset was not preserved: %#v", decoded.InstallerAsset)
	}
	if decoded.PortableAsset == nil || decoded.PortableAsset.DownloadURL != original.PortableAsset.DownloadURL {
		t.Fatalf("portable asset was not preserved: %#v", decoded.PortableAsset)
	}
}

func TestAppUpdateRequestAndResultRoundTrip(t *testing.T) {
	request, err := DecodeAppUpdateDownloadRequest(EncodeAppUpdateDownloadRequest(AppUpdateDownloadRequest{
		Info:             AppUpdateInfo{LatestVersion: "1.1.0"},
		InstallOnRestart: true,
	}))
	if err != nil {
		t.Fatalf("DecodeAppUpdateDownloadRequest failed: %v", err)
	}
	if request.Info.LatestVersion != "1.1.0" || !request.InstallOnRestart {
		t.Fatalf("download request was not preserved: %#v", request)
	}

	result, err := DecodeAppUpdateDownloadResult(EncodeAppUpdateDownloadResult(AppUpdateDownloadResult{
		Message:          "done",
		Version:          "1.1.0",
		InstallerPath:    "C:/tmp/setup.exe",
		PackagePath:      "C:/tmp/setup.exe",
		ExtractedPath:    "C:/tmp/app",
		InstallOnRestart: true,
		RestartScheduled: true,
		PackageKind:      "installer",
	}))
	if err != nil {
		t.Fatalf("DecodeAppUpdateDownloadResult failed: %v", err)
	}
	if result.PackageKind != "installer" || !result.RestartScheduled {
		t.Fatalf("download result was not preserved: %#v", result)
	}
}

func TestAppUpdateEventsRoundTrip(t *testing.T) {
	progress, err := DecodeAppUpdateDownloadProgress(EncodeAppUpdateDownloadProgress(AppUpdateDownloadProgress{
		Phase:    "downloading",
		Progress: 42,
		Message:  "正在下载",
	}))
	if err != nil {
		t.Fatalf("DecodeAppUpdateDownloadProgress failed: %v", err)
	}
	if progress.Progress != 42 || progress.Message != "正在下载" {
		t.Fatalf("progress was not preserved: %#v", progress)
	}

	pending, err := DecodeAppUpdatePendingUpdate(EncodeAppUpdatePendingUpdate(AppUpdatePendingUpdate{
		Version:            "1.1.0",
		InstallerPath:      "C:/tmp/setup.exe",
		ReleaseURL:         "https://example.test",
		InstallOnNextStart: true,
		CreatedAt:          "2026-05-24T00:00:00Z",
	}))
	if err != nil {
		t.Fatalf("DecodeAppUpdatePendingUpdate failed: %v", err)
	}
	if pending.InstallerPath == "" || !pending.InstallOnNextStart {
		t.Fatalf("pending update was not preserved: %#v", pending)
	}
}
