param(
    [ValidateSet("3")]
    [string]$WailsVersion = $(if ($env:WAILS_VERSION) { $env:WAILS_VERSION } else { "3" }),
    [string]$WailsBin = $env:WAILS_BIN
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $repoRoot

function Invoke-NativeCommand {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$Arguments = @()
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        $argText = if ($Arguments.Count -gt 0) { " $($Arguments -join ' ')" } else { "" }
        throw "$FilePath$argText failed with exit code $LASTEXITCODE"
    }
}

function Assert-RequiredSourceFiles {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Action,
        [Parameter(Mandatory = $true)]
        [string[]]$Paths
    )

    $missing = @()
    foreach ($relativePath in $Paths) {
        $fullPath = Join-Path $repoRoot $relativePath
        if (-not (Test-Path -LiteralPath $fullPath -PathType Leaf)) {
            $missing += $relativePath
        }
    }

    if ($missing.Count -gt 0) {
        throw "$Action requires a complete source tree. Missing files: $($missing -join ', ')"
    }
}

try {
    $previousWailsBin = $env:WAILS_BIN
    if (-not [string]::IsNullOrWhiteSpace($WailsBin)) {
        $env:WAILS_BIN = $WailsBin
    }

    Write-Host "========================================"
    Write-Host "  Trace Browser - Build Script"
    Write-Host "========================================"
    Write-Host ""
    Write-Host "Current workdir: $repoRoot"
    Write-Host "Wails target: v$WailsVersion"
    Write-Host ""

    $proxyHost = if ($env:TRACE_BUILD_PROXY_HOST) { $env:TRACE_BUILD_PROXY_HOST } else { "127.0.0.1" }
    $proxyPort = if ($env:TRACE_BUILD_PROXY_PORT) { $env:TRACE_BUILD_PROXY_PORT } else { "7890" }
    $useProxy = $env:TRACE_BUILD_USE_PROXY -in @("1", "true", "TRUE", "yes", "YES", "on", "ON")
    $scriptConfiguredNpmProxy = $false

    if ($useProxy) {
        Write-Host "[0/7] Configuring proxy..."
        $proxyValue = "http://${proxyHost}:${proxyPort}"
        $env:HTTP_PROXY = $proxyValue
        $env:HTTPS_PROXY = $proxyValue
        $env:http_proxy = $proxyValue
        $env:https_proxy = $proxyValue
        $env:GOPROXY = "https://goproxy.cn,direct"

        & npm config set proxy $proxyValue | Out-Null
        & npm config set https-proxy $proxyValue | Out-Null
        $scriptConfiguredNpmProxy = $true

        Write-Host "OK proxy configured: ${proxyHost}:${proxyPort}"
        Write-Host ""
    }

    $requiredSourceFiles = @(
        "go.mod",
        "go.sum",
        "main.go",
        "Taskfile.yml",
        "build/config.yml"
    )
    Assert-RequiredSourceFiles -Action "Building from source" -Paths $requiredSourceFiles

    Write-Host "[1/8] Syncing build version..."
    Invoke-NativeCommand -FilePath "powershell" -Arguments @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        "scripts\sync-wails-version.ps1",
        "-RepoRoot",
        $repoRoot
    )

    Write-Host ""
    Write-Host "[2/8] Installing frontend dependencies..."
    Push-Location (Join-Path $repoRoot "frontend")
    try {
        if (Test-Path -LiteralPath "package-lock.json" -PathType Leaf) {
            Invoke-NativeCommand -FilePath "npm" -Arguments @("ci", "--prefer-offline", "--no-audit", "--no-fund")
        }
        else {
            Invoke-NativeCommand -FilePath "npm" -Arguments @("install")
        }
        Invoke-NativeCommand -FilePath "npm" -Arguments @("run", "ensure:native")
    }
    finally {
        Pop-Location
    }

    Write-Host ""
    Write-Host "[3/8] Installing Go dependencies..."
    Invoke-NativeCommand -FilePath "go" -Arguments @("mod", "download")

    Write-Host ""
    Write-Host "[4/8] Ensuring frontend\dist exists..."
    $frontendDist = Join-Path $repoRoot "frontend/dist"
    $tempDistCreated = $false
    if (-not (Test-Path -LiteralPath $frontendDist)) {
        New-Item -ItemType Directory -Path $frontendDist -Force | Out-Null
        Set-Content -LiteralPath (Join-Path $frontendDist "index.html") -Value "" -Encoding ascii
        $tempDistCreated = $true
        Write-Host "OK temporary dist directory created"
    } else {
        Write-Host "OK dist directory already exists"
    }

    Write-Host ""
    Write-Host "[5/8] Generating Wails bindings..."
    Invoke-NativeCommand -FilePath "cmd" -Arguments @("/c", "call bat\generate-bindings.bat --no-pause --wails3")

    $binaryPath = Join-Path $repoRoot "build/bin/trace-browser.exe"

    Write-Host ""
    Write-Host "[6/8] Building frontend..."
    if ($tempDistCreated -and (Test-Path -LiteralPath $frontendDist)) {
        Remove-Item -LiteralPath $frontendDist -Recurse -Force -ErrorAction SilentlyContinue
    }
    Push-Location (Join-Path $repoRoot "frontend")
    try {
        Invoke-NativeCommand -FilePath "npm" -Arguments @("run", "build")
    }
    finally {
        Pop-Location
    }

    Write-Host ""
    Write-Host "[7/8] Building app..."
    Invoke-NativeCommand -FilePath "powershell" -Arguments @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        "scripts\build-wails3-windows.ps1",
        "-Output",
        $binaryPath
    )

    if ($tempDistCreated -and (Test-Path -LiteralPath $frontendDist)) {
        Remove-Item -LiteralPath $frontendDist -Recurse -Force -ErrorAction SilentlyContinue
    }

    Write-Host ""
    Write-Host "[8/8] Copying runtime dependencies..."
    $binDir = Join-Path $repoRoot "bin"
    $targetDir = Join-Path $repoRoot "build/bin/bin"
    if (Test-Path -LiteralPath $binDir -PathType Container) {
        Copy-Item -LiteralPath $binDir -Destination $targetDir -Recurse -Force
        Write-Host "OK copied bin directory to build\bin\bin\"
    } else {
        Write-Host "[WARN] bin directory not found, skipping copy"
    }

    Write-Host ""
    Write-Host "========================================"
    Write-Host "  OK build completed"
    Write-Host "========================================"
    Write-Host ""
    Write-Host "Executable: $($binaryPath.Substring($repoRoot.Length + 1))"
    exit 0
}
catch {
    Write-Host ""
    Write-Host "[ERROR] $($_.Exception.Message)"
    exit 1
}
finally {
    $env:WAILS_BIN = $previousWailsBin
    if ($scriptConfiguredNpmProxy) {
        & npm config delete proxy 2>$null | Out-Null
        & npm config delete https-proxy 2>$null | Out-Null
    }
}
