package backend

import (
	"strings"
	"testing"
)

func TestResolveFingerprintArgsForLaunchAutoHardwareKeepsLocaleAndLocalizesFonts(t *testing.T) {
	got := resolveFingerprintArgsForLaunch([]string{
		"--fingerprint-auto-hardware=true",
		"--lang=ja-JP",
		"--timezone=Asia/Tokyo",
	})

	if containsLaunchArg(got, "--fingerprint-auto-hardware=true") {
		t.Fatalf("auto hardware marker should not be passed to browser: %v", got)
	}
	if !containsLaunchArg(got, "--lang=ja-JP") || !containsLaunchArg(got, "--timezone=Asia/Tokyo") {
		t.Fatalf("locale args should be preserved: %v", got)
	}

	fonts := launchArgValue(got, "--fingerprint-fonts")
	if fonts == "" {
		t.Fatalf("expected localized fonts arg: %v", got)
	}
	if !containsAnyFold(fonts, []string{"Yu Gothic", "Meiryo", "Hiragino", "Noto Sans CJK JP", "Noto Sans JP"}) {
		t.Fatalf("expected Japanese font candidates, got %q", fonts)
	}
}

func TestResolveFingerprintArgsForLaunchAutoHardwareDoesNotOverrideExplicitHardware(t *testing.T) {
	got := resolveFingerprintArgsForLaunch([]string{
		"--fingerprint-auto-hardware=true",
		"--lang=zh-CN",
		"--fingerprint-platform=mac",
		"--fingerprint-webgl-vendor=Apple",
		"--fingerprint-device-memory=16",
	})

	if value := launchArgValue(got, "--fingerprint-platform"); value != "mac" {
		t.Fatalf("explicit platform should be preserved, got %q in %v", value, got)
	}
	if value := launchArgValue(got, "--fingerprint-webgl-vendor"); value != "Apple" {
		t.Fatalf("explicit WebGL vendor should be preserved, got %q in %v", value, got)
	}
	if value := launchArgValue(got, "--fingerprint-device-memory"); value != "16" {
		t.Fatalf("explicit device memory should be preserved, got %q in %v", value, got)
	}
}

func containsLaunchArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func launchArgValue(args []string, key string) string {
	prefix := key + "="
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
}

func containsAnyFold(value string, needles []string) bool {
	normalized := strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(normalized, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
