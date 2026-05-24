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

& go @goArgs
exit $LASTEXITCODE
