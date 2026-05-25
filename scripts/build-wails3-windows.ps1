param(
    [string]$Output = "build/bin/trace-browser.exe"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-TrimmedText {
    param([AllowNull()][string]$Value)

    if ($null -eq $Value) {
        return ""
    }
    return $Value.Trim()
}

function Resolve-BuildConfigInfoValue {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ConfigText,
        [Parameter(Mandatory = $true)]
        [string]$Key,
        [string]$Fallback = ""
    )

    $pattern = "(?m)^\s{2}$([regex]::Escape($Key)):\s*[""']?([^""'\r\n]+)[""']?\s*$"
    $match = [regex]::Match($ConfigText, $pattern)
    if (-not $match.Success) {
        return $Fallback
    }
    $value = Get-TrimmedText ([string]$match.Groups[1].Value)
    if ($value -eq "") {
        return $Fallback
    }
    return $value
}

function Resolve-GoBinary {
    $goCommand = Get-Command go -ErrorAction SilentlyContinue
    if ($null -ne $goCommand) {
        return $goCommand.Source
    }

    $toolchainRoot = Join-Path (Get-Location) ".tmp/toolchains"
    if (Test-Path -LiteralPath $toolchainRoot -PathType Container) {
        $localGo = Get-ChildItem -LiteralPath $toolchainRoot -Directory -ErrorAction SilentlyContinue |
            Sort-Object Name -Descending |
            ForEach-Object { Join-Path $_.FullName "go/bin/go.exe" } |
            Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } |
            Select-Object -First 1
        if (-not [string]::IsNullOrWhiteSpace($localGo)) {
            return $localGo
        }
    }

    throw "go command not found; cannot build Windows executable"
}

$version = $env:TRACE_BROWSER_VERSION
if ([string]::IsNullOrWhiteSpace($version)) {
    $version = $env:VERSION
}

$configPath = Join-Path (Get-Location) "build/config.yml"
if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
    throw "build/config.yml not found; cannot resolve Windows executable resources"
}
$configText = Get-Content -LiteralPath $configPath -Raw

if ([string]::IsNullOrWhiteSpace($version)) {
    $version = Resolve-BuildConfigInfoValue -ConfigText $configText -Key "version" -Fallback "0.0.0"
}
$version = $version.Trim()

$ldflags = @("-H windowsgui")
if (-not [string]::IsNullOrWhiteSpace($version)) {
    $ldflags += "-X main.appBuildVersion=$version"
}

$goPath = Resolve-GoBinary

$goarch = $env:GOARCH
if ([string]::IsNullOrWhiteSpace($goarch)) {
    $goarch = (& $goPath env GOARCH).Trim()
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

$sysoPath = Join-Path (Get-Location) "rsrc_windows_$goarch.syso"
$iconPath = Join-Path (Get-Location) "build/windows/icon.ico"
$manifestPath = Join-Path (Get-Location) "build/windows/wails.exe.manifest"
$tempDir = Join-Path (Get-Location) ".tmp"
New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
$infoPath = Join-Path $tempDir "windows-info-$goarch.json"
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

$productName = Resolve-BuildConfigInfoValue -ConfigText $configText -Key "productName" -Fallback "Trace Browser"
$companyName = Resolve-BuildConfigInfoValue -ConfigText $configText -Key "companyName" -Fallback $productName
$productIdentifier = Resolve-BuildConfigInfoValue -ConfigText $configText -Key "productIdentifier" -Fallback "com.tracebrowser.app"
$copyright = Resolve-BuildConfigInfoValue -ConfigText $configText -Key "copyright" -Fallback "Copyright (c) 2026"
$comments = Resolve-BuildConfigInfoValue -ConfigText $configText -Key "comments" -Fallback "$productName desktop application"

$versionInfo = [ordered]@{
    fixed = [ordered]@{
        file_version = $version
        product_version = $version
    }
    info = [ordered]@{
        "0409" = [ordered]@{
            ProductVersion = $version
            CompanyName = $companyName
            FileDescription = $productName
            LegalCopyright = $copyright
            ProductName = $productName
            Comments = $comments
        }
    }
}
[System.IO.File]::WriteAllText($infoPath, (($versionInfo | ConvertTo-Json -Depth 5) + [Environment]::NewLine), $utf8NoBom)

$manifestVersion = "0.0.0.0"
$versionMatch = [regex]::Match($version, '^(\d+)\.(\d+)\.(\d+)')
if ($versionMatch.Success) {
    $manifestVersion = "$($versionMatch.Groups[1].Value).$($versionMatch.Groups[2].Value).$($versionMatch.Groups[3].Value).0"
}
$manifestTemplate = Get-Content -LiteralPath $manifestPath -Raw
$tempManifestPath = Join-Path $tempDir "windows-manifest-$goarch.xml"
$resolvedManifest = [regex]::Replace($manifestTemplate, '(<assemblyIdentity\b[^>]*\bname=")[^"]+(")', "`${1}$productIdentifier`${2}", 1)
$resolvedManifest = [regex]::Replace($resolvedManifest, '(<assemblyIdentity\b[^>]*\bversion=")[^"]+(")', "`${1}$manifestVersion`${2}", 1)
[System.IO.File]::WriteAllText($tempManifestPath, $resolvedManifest, $utf8NoBom)

try {
    & $wails3Path generate syso -arch $goarch -icon $iconPath -info $infoPath -manifest $tempManifestPath -out $sysoPath
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

    & $goPath @goArgs
    exit $LASTEXITCODE
}
finally {
    if ([string]::IsNullOrWhiteSpace($env:TRACE_KEEP_WINDOWS_RESOURCES)) {
        Remove-Item -LiteralPath $sysoPath -Force -ErrorAction SilentlyContinue
        Remove-Item -LiteralPath $infoPath -Force -ErrorAction SilentlyContinue
        Remove-Item -LiteralPath $tempManifestPath -Force -ErrorAction SilentlyContinue
    }
}
