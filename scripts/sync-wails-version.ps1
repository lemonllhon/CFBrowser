param(
    [string]$RepoRoot
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

function Assert-VersionValue {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Value,
        [string]$Source = "version"
    )

    $trimmed = Get-TrimmedText $Value
    if ($trimmed -eq "") {
        throw "$Source is empty"
    }
    if ($trimmed -notmatch '^\d+\.\d+\.\d+(?:-[0-9A-Za-z\.-]+)?(?:\+[0-9A-Za-z\.-]+)?$') {
        throw "$Source has invalid format: $trimmed"
    }
    return $trimmed
}

function Invoke-Git {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)

    $output = & git @Arguments 2>$null
    if ($LASTEXITCODE -ne 0) {
        return ""
    }
    return (($output | Out-String).Trim())
}

function Resolve-BuildTag {
    $exactTag = Get-TrimmedText (Invoke-Git -Arguments @("describe", "--tags", "--exact-match", "HEAD"))
    if ($exactTag -ne "") {
        return $exactTag
    }

    $latestTag = Get-TrimmedText (Invoke-Git -Arguments @("tag", "--sort=-v:refname"))
    if ($latestTag -eq "") {
        throw "No local git tags found. Create a version tag such as v0.0.24 before building."
    }

    return (($latestTag -split "(`r`n|`n|`r)") | Where-Object { (Get-TrimmedText $_) -ne "" } | Select-Object -First 1)
}

if ((Get-TrimmedText $RepoRoot) -eq "") {
    $RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
}
else {
    $RepoRoot = (Resolve-Path $RepoRoot).Path
}

$wailsConfigPath = Join-Path $RepoRoot "build/config.yml"
if (-not (Test-Path -LiteralPath $wailsConfigPath -PathType Leaf)) {
    throw "build/config.yml not found: $wailsConfigPath"
}

$tagName = Get-TrimmedText (Resolve-BuildTag)
if ($tagName -eq "") {
    throw "git tag is empty"
}

Write-Host "Syncing build/config.yml info.version from local git tag..."
Write-Host "  Tag: $tagName"

$releaseVersion = Assert-VersionValue -Value ($tagName -replace '^[vV]', '') -Source "local git tag"
$configText = Get-Content -LiteralPath $wailsConfigPath -Raw
$versionPattern = '(?m)^(\s{2}version:\s*["'']?)([^"''\r\n]+)(["'']?)\s*$'
$match = [regex]::Match($configText, $versionPattern)
if (-not $match.Success) {
    throw "Cannot find info.version in build/config.yml"
}
$currentVersion = Get-TrimmedText ([string]$match.Groups[2].Value)

if ($currentVersion -eq $releaseVersion) {
    Write-Host "OK build/config.yml info.version already matches: $releaseVersion"
    exit 0
}

$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
$updatedText = [regex]::Replace($configText, $versionPattern, "`${1}$releaseVersion`${3}", 1)
[System.IO.File]::WriteAllText($wailsConfigPath, $updatedText, $utf8NoBom)

Write-Host "OK build/config.yml info.version synced: $currentVersion -> $releaseVersion"
