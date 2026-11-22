# Reel Studio

**Reel Studio** is a native Windows desktop application (Go + Wails v2 + Node.js render worker) that transforms ReelSync interactive reels (`.html`) and voice-over audio files into Instagram-ready MP4 videos (H.264 High profile, level 4.2, AAC 48 kHz stereo, 9:16 aspect ratio, faststart enabled).

---

## Key Features

- **Native Windows Desktop Shell**: Built with Go and Wails v2 utilizing the Microsoft Edge WebView2 runtime.
- **Frame-Accurate Virtual Clock**: Stepped frame-by-frame via an injected virtual time engine (`__vtAdvance`), pausing CSS animations, timers, and `requestAnimationFrame`. Zero dropped frames regardless of system load.
- **Bundled Portable Environment**: Ships with portable Node.js LTS, ffmpeg, ffprobe, and Playwright worker. No Node.js or ffmpeg installation required on the target machine.
- **Browser Automation**: Uses Microsoft Edge (`msedge`) present on Windows 10/11 or Playwright Chromium.
- **Batch Processing**: Drag and drop multiple reels and audio files with automatic base-name pairing.
- **Job Queue & Resilience**: Full queue management with real-time frame progress, ETA, and cancellation. Unfinished jobs are persisted in `%APPDATA%\ReelStudio\` and recovered as "Interrupted" with a 1-click retry.
- **Windows Integration**: Taskbar progress (`ITaskbarList3`), Windows Toast Notifications, and Single-Instance application lock.
- **Multilingual Support**: English, German, and Arabic with dynamic Right-to-Left (RTL) layout switching.

---

## Architecture Overview

```
Reel Studio (Go + Wails v2)
 ├── WebView2 Frontend (Vanilla TypeScript + Vite)
 ├── Go Core Engine (Job Queue, Settings, Win32 API, SQLite/JSON Store)
 └── Render Worker (Isolated Node.js Process)
      ├── Playwright + Edge (Virtual Clock Automation)
      └── Bundled FFmpeg (Direct Frame Pipe -> H.264 + AAC MP4)
```

For complete technical specifications, IPC protocols, and design decisions, see [DESIGN.md](file:///c:/Users/surface/OneDrive/Bureau/Projects/desctop/html%20to%20video/DESIGN.md).

---

## Prerequisites & Building

### Requirements
- **OS**: Windows 10 (22H2+) or Windows 11 (x64)
- **Go**: 1.22+
- **Node.js & npm**: Node 18+ (for building frontend and worker)
- **Wails v2 CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **NSIS**: Version 3.x+ (installed in `C:\Program Files (x86)\NSIS`)

### Step 1: Fetch Pinned Dependencies
Downloads portable Node.js LTS, ffmpeg essentials, and installs pinned worker packages (verifying SHA-256 hashes):
```powershell
powershell -ExecutionPolicy Bypass -File scripts\fetch-deps.ps1
```

### Step 2: Build Application & Installer
Runs Go unit tests, worker end-to-end tests, builds the Vite frontend, compiles the native Wails binary, generates the NSIS installer, and bundles the portable zip release:
```powershell
powershell -ExecutionPolicy Bypass -File scripts\build.ps1
```

The resulting binaries will be in `build/bin/`:
- `ReelStudio.exe` — Application executable
- `ReelStudio-amd64-installer.exe` — Setup installer
- `ReelStudio-portable-win-x64.zip` — Standalone portable package

---

## Running the Automated Tests

### 1. Go Unit Tests
Tests queue FIFO ordering, cancel handling, retry logic, persistence/recovery, IPC serialization, and ffmpeg argument construction:
```powershell
go test ./... -v
```

### 2. Worker E2E Test
Automates the render of `/testdata/sample-reel.html` with `/testdata/sample-audio.wav` at 720p/30fps/balanced. Validates H.264 High, AAC audio, exact duration (audio + 1.0s tail), and frame movement between $t = 0.0\text{s}$ and $t = 1.0\text{s}$:
```powershell
& "third_party\node.exe" test\worker.test.js
```

---

## Manual Verification Checklist

Follow this checklist on a clean Windows 10 / 11 test machine or VM:

- [ ] **1. Clean Installation from NSIS**:
  1. Run `ReelStudio-amd64-installer.exe` on a machine without Node.js or ffmpeg installed.
  2. Complete installation to `%PROGRAMFILES%\Reel Studio` (or `%LOCALAPPDATA%\Programs`).
  3. Verify Desktop and Start Menu shortcuts are created with the app icon.
  4. Launch Reel Studio from the shortcut and verify the window opens.

- [ ] **2. Render Standard 1080p30 High Video**:
  1. Drop `sample-reel.html` into the Reel HTML dropzone.
  2. Drop `sample-audio.wav` (or an `.mp3` file) into the Voice-over dropzone.
  3. Leave settings at default: Resolution `1080x1920`, Frame Rate `30 fps`, Quality `High`.
  4. Click **Render Video**.
  5. Watch the progress bar advance through `Opening reel`, `Loading the websites in the reel`, and `Rendering frames (x/y)` with live ETA.
  6. Verify taskbar shows green progress bar.
  7. When complete, verify toast notification appears.
  8. Click **Preview** on the completed card to watch the video in the in-app player.
  9. Click **Show in Explorer** to reveal the output MP4 in `Videos\ReelStudio`.

- [ ] **3. Mid-Render Cancellation**:
  1. Start rendering a video.
  2. While frames are rendering, click the red **Cancel** button on the job card.
  3. Verify the worker terminates ffmpeg and the browser immediately.
  4. Confirm partial output file is deleted from disk and job status changes to `Cancelled`.

- [ ] **4. Interrupted Job Recovery on App Restart**:
  1. Start rendering a video.
  2. Force close the application (e.g. `Alt + F4` or Task Manager).
  3. Re-launch Reel Studio.
  4. Verify the unfinished job appears in the queue marked with status **Interrupted**.
  5. Click the **Retry** button on the interrupted card and confirm rendering starts and finishes successfully.

- [ ] **5. Multilingual Layout & RTL Verification**:
  1. Open **Settings** (gear icon).
  2. Switch Language to **العربية (Arabic RTL)** and click **Save Changes**.
  3. Verify the interface flips direction to Right-to-Left (`dir="rtl"`) with all labels translated into Arabic.
  4. Switch back to English or German and confirm proper LTR rendering.

- [ ] **6. Single Instance Lock**:
  1. With Reel Studio running, attempt to launch a second instance.
  2. Verify the secondary process exits immediately without opening a duplicate window.

---

## Third-Party Components & Licenses

See [THIRD_PARTY.md](file:///c:/Users/surface/OneDrive/Bureau/Projects/desctop/html%20to%20video/THIRD_PARTY.md) for licenses and copyright notices for Node.js, FFmpeg, Playwright, and Wails.
