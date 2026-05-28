//go:build windows
// +build windows

package backend

import (
	"strings"
	"testing"
)

func TestResidualRuntimeCleanupScriptTargetsOtherAppInstances(t *testing.T) {
	script := residualRuntimeCleanupScript()
	required := []string{
		"param([string]$Root, [int]$CurrentPid, [string]$CurrentExePath)",
		"$process.ProcessId -eq $CurrentPid",
		"$isCurrentApp",
		"$exe.Equals($currentExe",
		"$root + 'bin\\'",
		"$root + 'chrome\\'",
		"taskkill.exe",
		"mihomo.exe",
	}
	for _, needle := range required {
		if !strings.Contains(script, needle) {
			t.Fatalf("expected cleanup script to contain %q", needle)
		}
	}
	if strings.Contains(script, "ExcludePath") {
		t.Fatal("cleanup script should exclude only the current PID, not every process using the same executable path")
	}
}
