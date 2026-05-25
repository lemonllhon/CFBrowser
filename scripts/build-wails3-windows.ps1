param(
    [string]$Output = "build/bin/trace-browser.exe"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$version = $env:TRACE_BROWSER_VERSION
if ([string]::IsNullOrWhiteSpace($version)) {
    $version = $env:VERSION
}

$ldflags = @("-H windowsgui")
if (-not [string]::IsNullOrWhiteSpace($version)) {
    $ldflags += "-X main.appBuildVersion=$($version.Trim())"
}

$goarch = $env:GOARCH
if ([string]::IsNullOrWhiteSpace($goarch)) {
    $goarch = (& go env GOARCH).Trim()
}
if ([string]::IsNullOrWhiteSpace($goarch)) {
    $goarch = "amd64"
}

$wails3Command = Get-Command wails3 -ErrorAction SilentlyContinue
$wails3Path = if ($null -ne $wails3Command) { $wails3Command.Source } else { "" }
if ([string]::IsNullOrWhiteSpace($wails3Path)) {
    $localWails3 = Join-Path (Get-Location) ".tmp/go/bin/wails3.exe"
    if (Test-Path -LiteralPath $localWails3 -PathType Leaf) {
        $wails3Path = $localWails3
    }
}
if ([string]::IsNullOrWhiteSpace($wails3Path)) {
    throw "wails3 command not found; cannot generate Windows executable resources"
}

$sysoPath = Join-Path (Get-Location) "trace-browser_windows_$goarch.syso"
$iconPath = Join-Path (Get-Location) "build/windows/icon.ico"
$infoPath = Join-Path (Get-Location) "build/windows/info.json"
$manifestPath = Join-Path (Get-Location) "build/windows/wails.exe.manifest"
& $wails3Path generate syso -arch $goarch -icon $iconPath -info $infoPath -manifest $manifestPath -out $sysoPath
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

$outputDir = Split-Path -Parent $Output
if (-not [string]::IsNullOrWhiteSpace($outputDir)) {
    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
}

$goArgs = @(
    "build",
    "-trimpath",
    "-buildvcs=false",
    "-ldflags",
    ($ldflags -join " "),
    "-o",
    $Output,
    "."
)

try {
    & go @goArgs
    exit $LASTEXITCODE
}
finally {
    Remove-Item -LiteralPath $sysoPath -Force -ErrorAction SilentlyContinue
}
