package backend

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	updatergithub "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestMatchOfficialInstallerAssetPrefersSetupExeOnWindows(t *testing.T) {
	assets := []updatergithub.ReleaseAsset{
		{Name: "TraceBrowser-Portable-0.2.7-win-x64.zip"},
		{Name: "TraceBrowser-SelfUpdate-0.2.7-windows-amd64.zip"},
		{Name: "TraceBrowser-Setup-0.2.7.exe"},
	}

	got := matchOfficialInstallerAsset(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets)
	if got != 2 {
		t.Fatalf("expected setup exe at index 2, got %d", got)
	}
}

func TestMatchOfficialInstallerAssetRejectsPortableAndSelfUpdateOnWindows(t *testing.T) {
	assets := []updatergithub.ReleaseAsset{
		{Name: "TraceBrowser-Portable-0.2.7-win-x64.zip"},
		{Name: "TraceBrowser-SelfUpdate-0.2.7-windows-amd64.zip"},
		{Name: "SHA256SUMS"},
	}

	got := matchOfficialInstallerAsset(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets)
	if got != -1 {
		t.Fatalf("expected no installable asset, got %d", got)
	}
}

func TestMatchOfficialInstallerAssetPrefersArchSpecificSetupExe(t *testing.T) {
	assets := []updatergithub.ReleaseAsset{
		{Name: "TraceBrowser-Setup-0.2.7.exe"},
		{Name: "TraceBrowser-Setup-0.2.7-win-x64.exe"},
	}

	got := matchOfficialInstallerAsset(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets)
	if got != 1 {
		t.Fatalf("expected arch-specific setup exe at index 1, got %d", got)
	}
}

func TestLooksLikeOfficialInstallerAsset(t *testing.T) {
	valid := []string{
		"TraceBrowser-Setup-0.2.7.exe",
		"Trace Browser Installer 0.2.7.exe",
	}
	for _, name := range valid {
		if !looksLikeOfficialInstallerAsset(name) {
			t.Fatalf("expected %q to be accepted", name)
		}
	}

	invalid := []string{
		"TraceBrowser-Portable-0.2.7-win-x64.zip",
		"TraceBrowser-SelfUpdate-0.2.7-windows-amd64.zip",
		"TraceBrowser-SelfUpdate-0.2.7.exe",
		"OtherApp-Setup-0.2.7.exe",
	}
	for _, name := range invalid {
		if looksLikeOfficialInstallerAsset(name) {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}
