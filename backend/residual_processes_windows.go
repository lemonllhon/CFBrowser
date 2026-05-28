//go:build windows
// +build windows

package backend

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func killResidualRuntimeProcesses(appRoot string) error {
	appRoot = strings.TrimSpace(appRoot)
	if appRoot == "" {
		return nil
	}
	if _, err := os.Stat(appRoot); err != nil {
		return nil
	}

	exePath, _ := os.Executable()
	psScript := residualRuntimeCleanupScript()

	tempFile, err := os.CreateTemp("", "ant-chrome-cleanup-*.ps1")
	if err != nil {
		return fmt.Errorf("创建清理脚本失败: %w", err)
	}
	scriptPath := tempFile.Name()
	if _, err := tempFile.WriteString(psScript); err != nil {
		tempFile.Close()
		_ = os.Remove(scriptPath)
		return fmt.Errorf("写入清理脚本失败: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(scriptPath)
		return fmt.Errorf("关闭清理脚本失败: %w", err)
	}
	defer os.Remove(scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	powershellPath := `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	if _, err := os.Stat(powershellPath); err != nil {
		if fallbackPath, lookErr := exec.LookPath("powershell.exe"); lookErr == nil {
			powershellPath = fallbackPath
		} else {
			return fmt.Errorf("未找到 powershell.exe")
		}
	}

	cmd := exec.CommandContext(ctx, powershellPath, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Clean(scriptPath), "-Root", appRoot, "-CurrentPid", fmt.Sprint(os.Getpid()), "-CurrentExePath", exePath)
	hideWindow(cmd)

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("清理残留进程超时")
	}
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("清理残留进程失败: %s", message)
	}
	return nil
}

func residualRuntimeCleanupScript() string {
	return `param([string]$Root, [int]$CurrentPid, [string]$CurrentExePath)
$ErrorActionPreference = 'SilentlyContinue'
if ([string]::IsNullOrWhiteSpace($Root) -or -not (Test-Path -LiteralPath $Root)) { exit 0 }
$root = [System.IO.Path]::GetFullPath($Root).TrimEnd('\') + '\'
$currentExe = ''
if (-not [string]::IsNullOrWhiteSpace($CurrentExePath)) {
  $currentExe = [System.IO.Path]::GetFullPath($CurrentExePath)
}
$proxyNames = @('xray.exe', 'xray', 'sing-box.exe', 'sing-box', 'clash.exe', 'clash', 'clash-meta.exe', 'clash-meta', 'mihomo.exe', 'mihomo')
function Test-TraceBrowserResidualProcess($process) {
  if (-not $process -or -not $process.ExecutablePath) { return $false }
  if ($process.ProcessId -eq $CurrentPid) { return $false }

  $exe = [string]$process.ExecutablePath
  $name = ([string]$process.Name).ToLowerInvariant()
  $cmdline = [string]$process.CommandLine

  $isBundledRuntime = (
    $exe.StartsWith(($root + 'bin\'), [System.StringComparison]::OrdinalIgnoreCase) -or
    $exe.StartsWith(($root + 'chrome\'), [System.StringComparison]::OrdinalIgnoreCase)
  )
  $isCurrentApp = $currentExe -ne '' -and $exe.Equals($currentExe, [System.StringComparison]::OrdinalIgnoreCase)
  $isRootCommandProxy = (
    $proxyNames -contains $name -and
    -not [string]::IsNullOrWhiteSpace($cmdline) -and
    $cmdline.IndexOf($root, [System.StringComparison]::OrdinalIgnoreCase) -ge 0
  )

  return $isBundledRuntime -or $isCurrentApp -or $isRootCommandProxy
}
function Get-TraceBrowserResidualProcesses {
  @(
    Get-CimInstance Win32_Process | Where-Object { Test-TraceBrowserResidualProcess $_ }
  )
}
$targets = @(Get-TraceBrowserResidualProcesses | Sort-Object ProcessId -Descending)
foreach ($p in $targets) {
  try { Stop-Process -Id $p.ProcessId -Force -ErrorAction Stop } catch {}
}
Start-Sleep -Milliseconds 400
$left = @(Get-TraceBrowserResidualProcesses)
if ($left.Count -gt 0) {
  $taskkill = Join-Path $env:SystemRoot 'System32\taskkill.exe'
  foreach ($p in $left) {
    try {
      if (Test-Path -LiteralPath $taskkill) {
        & $taskkill /F /T /PID $p.ProcessId | Out-Null
      }
    } catch {}
  }
  Start-Sleep -Milliseconds 400
}
$left = @(Get-TraceBrowserResidualProcesses)
if ($left.Count -gt 0) {
  $names = ($left | ForEach-Object { $_.Name + '#' + $_.ProcessId }) -join ', '
  Write-Host ('still running: ' + $names)
  exit 1
}
exit 0
`
}
