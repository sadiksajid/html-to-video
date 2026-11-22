<#
.SYNOPSIS
    Builds Reel Studio: executes tests, compiles frontend, runs Wails build with NSIS, and creates a portable zip package.
#>

[CmdletBinding()]
param(
    [switch]$SkipTests = $false
)

$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir
$ThirdPartyDir = Join-Path $RootDir "third_party"
$WorkerDir = Join-Path $RootDir "worker"
$FrontendDir = Join-Path $RootDir "frontend"
$BuildBinDir = Join-Path $RootDir "build\bin"

# Add NSIS to PATH if present
$nsisDir = "C:\Program Files (x86)\NSIS"
if ((Test-Path (Join-Path $nsisDir "makensis.exe")) -and ($env:PATH -notlike "*$nsisDir*")) {
    $env:PATH = "$nsisDir;$env:PATH"
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "        Reel Studio - Build Pipeline       " -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# 1. Ensure Dependencies
$fetchScript = Join-Path $ScriptDir "fetch-deps.ps1"
if (-not (Test-Path (Join-Path $ThirdPartyDir "node.exe")) -or `
    -not (Test-Path (Join-Path $ThirdPartyDir "ffmpeg.exe")) -or `
    -not (Test-Path (Join-Path $WorkerDir "node_modules"))) {
    Write-Host "[1/6] Dependencies missing. Fetching..." -ForegroundColor Yellow
    & powershell -ExecutionPolicy Bypass -File $fetchScript
} else {
    Write-Host "[1/6] Dependencies verified." -ForegroundColor Green
}

# 2. Run Tests
if (-not $SkipTests) {
    Write-Host "[2/6] Running Go Unit Tests..." -ForegroundColor Yellow
    Push-Location $RootDir
    try {
        go test ./... -v
    } finally {
        Pop-Location
    }

    Write-Host "      Running Worker E2E Test..." -ForegroundColor Yellow
    $NodeExe = Join-Path $ThirdPartyDir "node.exe"
    $WorkerTest = Join-Path $RootDir "test\worker.test.js"
    & $NodeExe $WorkerTest
    Write-Host "      All tests passed successfully!" -ForegroundColor Green
} else {
    Write-Host "[2/6] Skipping tests (-SkipTests specified)." -ForegroundColor DarkGray
}

# 3. Build Frontend
Write-Host "[3/6] Building Frontend (Vite + TypeScript)..." -ForegroundColor Yellow
Push-Location $FrontendDir
try {
    npm run build
} finally {
    Pop-Location
}

# 4. Run Wails Build with NSIS
Write-Host "[4/6] Compiling Native App & NSIS Installer via Wails..." -ForegroundColor Yellow
$wailsExe = Join-Path $env:USERPROFILE "go\bin\wails.exe"
if (-not (Test-Path $wailsExe)) {
    $wailsExe = (Get-Command "wails.exe" -ErrorAction SilentlyContinue).Source
}
if (-not $wailsExe) {
    throw "Wails CLI executable could not be found!"
}

Push-Location $RootDir
try {
    & $wailsExe build -platform windows/amd64 -nsis -clean
} finally {
    Pop-Location
}

# 5. Bundle Third-Party and Worker into build/bin for Portable Release
Write-Host "[5/6] Bundling portable release files..." -ForegroundColor Yellow
$portableThirdParty = Join-Path $BuildBinDir "third_party"
$portableWorker = Join-Path $BuildBinDir "worker"

if (-not (Test-Path $portableThirdParty)) {
    New-Item -ItemType Directory -Path $portableThirdParty -Force | Out-Null
}
Copy-Item -Path (Join-Path $ThirdPartyDir "*") -Destination $portableThirdParty -Recurse -Force

if (-not (Test-Path $portableWorker)) {
    New-Item -ItemType Directory -Path $portableWorker -Force | Out-Null
}
Copy-Item -Path (Join-Path $WorkerDir "*") -Destination $portableWorker -Recurse -Force

# Create Portable Zip
$zipPath = Join-Path $BuildBinDir "ReelStudio-portable-win-x64.zip"
if (Test-Path $zipPath) { Remove-Item -Force $zipPath }

Write-Host "      Compressing portable package: $zipPath..." -ForegroundColor Yellow
Compress-Archive -Path (Join-Path $BuildBinDir "ReelStudio.exe"), $portableThirdParty, $portableWorker -DestinationPath $zipPath -Force
Write-Host "      Portable package created." -ForegroundColor Green

# 6. Optional Code Signing (signtool)
Write-Host "[6/6] Checking Code Signing..." -ForegroundColor Yellow
if ($env:CERT_FILE -and (Test-Path $env:CERT_FILE)) {
    Write-Host "      Signing executables with signtool..." -ForegroundColor Cyan
    $certPassword = if ($env:CERT_PASSWORD) { "/p $env:CERT_PASSWORD" } else { "" }
    & signtool.exe sign /f $env:CERT_FILE $certPassword /fd sha256 /tr http://timestamp.digicert.com /td sha256 (Join-Path $BuildBinDir "ReelStudio.exe")
} else {
    Write-Host "      Code signing skipped (no CERT_FILE environment variable set)." -ForegroundColor DarkGray
}

Write-Host "==========================================" -ForegroundColor Green
Write-Host "         BUILD COMPLETED SUCCESSFULLY!    " -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
Write-Host "Outputs:"
Get-ChildItem -Path $BuildBinDir | Select-Object Name, Length, LastWriteTime | Format-Table -AutoSize
