<#
.SYNOPSIS
    Downloads pinned dependencies for Reel Studio: portable Node.js, ffmpeg, ffprobe, and worker npm packages.
#>

[CmdletBinding()]
param(
    [switch]$Force = $false
)

$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir
$ThirdPartyDir = Join-Path $RootDir "third_party"
$WorkerDir = Join-Path $RootDir "worker"

if (-not (Test-Path $ThirdPartyDir)) {
    New-Item -ItemType Directory -Path $ThirdPartyDir -Force | Out-Null
}

function Download-File-Fast {
    param(
        [string]$Url,
        [string]$OutFile
    )
    if (Get-Command "curl.exe" -ErrorAction SilentlyContinue) {
        & curl.exe -L --fail --retry 3 --silent --show-error -o $OutFile $Url
    } else {
        $wc = New-Object System.Net.WebClient
        $wc.DownloadFile($Url, $OutFile)
    }
}

function Verify-Sha256 {
    param(
        [string]$FilePath,
        [string]$ExpectedHash
    )
    $actual = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLowerInvariant()
    $expected = $ExpectedHash.ToLowerInvariant().Trim()
    if ($actual -ne $expected) {
        throw "SHA-256 mismatch for '$FilePath'! Expected: $expected, Actual: $actual"
    }
    Write-Host "  [OK] SHA-256 verified: $actual" -ForegroundColor Green
}

# 1. Node.js LTS portable (v20.18.1 win-x64)
$NodeExePath = Join-Path $ThirdPartyDir "node.exe"
$NodeUrl = "https://nodejs.org/dist/v20.18.1/win-x64/node.exe"
$NodeSha256 = "06c1dec1b428927d6ff01c8f5882f119ec13b61ac77483760aa7fba215c72cf5"

Write-Host "Checking portable Node.js..." -ForegroundColor Cyan
if ($Force -or -not (Test-Path $NodeExePath)) {
    Write-Host "Downloading Node.js from $NodeUrl..." -ForegroundColor Yellow
    Download-File-Fast -Url $NodeUrl -OutFile $NodeExePath
    Verify-Sha256 -FilePath $NodeExePath -ExpectedHash $NodeSha256
} else {
    try {
        Verify-Sha256 -FilePath $NodeExePath -ExpectedHash $NodeSha256
        Write-Host "  Existing node.exe is valid." -ForegroundColor Green
    } catch {
        Write-Host "  Existing node.exe hash mismatch, re-downloading..." -ForegroundColor Yellow
        Download-File-Fast -Url $NodeUrl -OutFile $NodeExePath
        Verify-Sha256 -FilePath $NodeExePath -ExpectedHash $NodeSha256
    }
}

# 2. ffmpeg & ffprobe essentials (9.0.2)
$FFmpegExe = Join-Path $ThirdPartyDir "ffmpeg.exe"
$FFprobeExe = Join-Path $ThirdPartyDir "ffprobe.exe"
$FFmpegZipUrl = "https://www.gyan.dev/ffmpeg/builds/packages/ffmpeg-9.0.2-essentials_build.zip"
$FFmpegSha256 = "60f467265b1e312373dbcd92200c2618a74850f98d3d078e94296bb3fa2047ba"
$TempZip = Join-Path $ThirdPartyDir "ffmpeg_temp.zip"

Write-Host "Checking ffmpeg and ffprobe..." -ForegroundColor Cyan
if ($Force -or -not (Test-Path $FFmpegExe) -or -not (Test-Path $FFprobeExe)) {
    Write-Host "Downloading ffmpeg from $FFmpegZipUrl..." -ForegroundColor Yellow
    Download-File-Fast -Url $FFmpegZipUrl -OutFile $TempZip
    Verify-Sha256 -FilePath $TempZip -ExpectedHash $FFmpegSha256

    $TempExtract = Join-Path $ThirdPartyDir "ffmpeg_extract"
    if (Test-Path $TempExtract) { Remove-Item -Recurse -Force $TempExtract }
    Write-Host "Extracting ffmpeg binaries..." -ForegroundColor Yellow
    Expand-Archive -Path $TempZip -DestinationPath $TempExtract -Force

    $foundFFmpeg = Get-ChildItem -Path $TempExtract -Filter "ffmpeg.exe" -Recurse | Select-Object -First 1
    $foundFFprobe = Get-ChildItem -Path $TempExtract -Filter "ffprobe.exe" -Recurse | Select-Object -First 1

    if (-not $foundFFmpeg -or -not $foundFFprobe) {
        throw "Could not locate ffmpeg.exe or ffprobe.exe inside downloaded archive!"
    }

    Move-Item -Path $foundFFmpeg.FullName -Destination $FFmpegExe -Force
    Move-Item -Path $foundFFprobe.FullName -Destination $FFprobeExe -Force

    Remove-Item -Recurse -Force $TempExtract
    Remove-Item -Force $TempZip
    Write-Host "  [OK] ffmpeg and ffprobe installed." -ForegroundColor Green
} else {
    Write-Host "  Existing ffmpeg.exe and ffprobe.exe found." -ForegroundColor Green
}

# 3. Worker Node dependencies
Write-Host "Installing worker dependencies..." -ForegroundColor Cyan
$env:PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD = "1"
Push-Location $WorkerDir
try {
    if (Test-Path "package-lock.json") {
        npm ci --no-audit
    } else {
        npm install --no-audit
    }
} finally {
    Pop-Location
}

# 4. Verify executables
Write-Host "Verifying tool versions..." -ForegroundColor Cyan
$nodeVer = & $NodeExePath -v
$ffmpegVer = & $FFmpegExe -version | Select-Object -First 1
$ffprobeVer = & $FFprobeExe -version | Select-Object -First 1

Write-Host "  Node.js:  $nodeVer" -ForegroundColor Green
Write-Host "  ffmpeg:   $ffmpegVer" -ForegroundColor Green
Write-Host "  ffprobe:  $ffprobeVer" -ForegroundColor Green
Write-Host "Dependency fetch complete!" -ForegroundColor Green
