# Reel Studio — Technical Design Document

## 1. System Overview

Reel Studio is a native Windows desktop application that converts ReelSync interactive reels (`.html`) and voice-over audio files into Instagram-ready MP4 videos (H.264 High profile, level 4.2, AAC 48 kHz stereo, 9:16 aspect ratio, faststart enabled).

The application combines:
1. **Desktop Shell (Go + Wails v2)**: Provides windowing, native Windows API integration (Taskbar progress, single-instance mutex, Explorer reveal, toast notifications), file dialogs, job queue state machine, persistence, and process management.
2. **Frontend UI (Vanilla TypeScript + Vite)**: High-performance, lightweight UI running inside WebView2, supporting drag-and-drop, real-time queue updates, in-app video playback, batch processing, and multi-language localization (English, German, and Arabic with RTL layout).
3. **Render Worker (Node.js + Playwright)**: Isolated worker executed by a bundled portable `node.exe`. It loads the HTML reel in Microsoft Edge (or Playwright Chromium), drives a frame-accurate virtual clock, captures CDP screenshots, and streams them into ffmpeg.
4. **Media Engine (ffmpeg & ffprobe)**: Bundled standalone binaries that probe audio durations and encode video frames into standards-compliant MP4.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Reel Studio Desktop (Go)                           │
│  ┌───────────────────────────────┐     ┌─────────────────────────────────┐  │
│  │     WebView2 Frontend UI      │     │         Go Core Engine          │  │
│  │ (Vite + TS, en/de/ar RTL)     │◄───►│ • Job Queue & Persistence       │  │
│  │ • Reel & Audio Dropzones      │IPC  │ • Win32 API (Taskbar, Mutex)    │  │
│  │ • Batch Pairing & Queue Panel │     │ • System Health & Disk Validator│  │
│  │ • Video Player Preview        │     │ • Worker Process Supervisor     │  │
│  └───────────────────────────────┘     └───────────────┬─────────────────┘  │
└────────────────────────────────────────────────────────┼────────────────────┘
                                                         │ stdin / stdout IPC
                                                         │ (newline-delimited JSON)
                                                         ▼
                                         ┌─────────────────────────────────┐
                                         │   Render Worker (node.exe)      │
                                         │  (worker/render-worker.js)      │
                                         │ • Edge / Chromium via CDP       │
                                         │ • Virtual Clock (__vtAdvance)   │
                                         │ • Page.captureScreenshot        │
                                         └───────────────┬─────────────────┘
                                                         │ raw frame pipe
                                                         │ (stdin streaming)
                                                         ▼
                                         ┌─────────────────────────────────┐
                                         │      ffmpeg.exe (bundled)       │
                                         │ • libx264 High 4.2, yuv420p     │
                                         │ • AAC 48kHz, audio pad tail     │
                                         │ • output.mp4                    │
                                         └─────────────────────────────────┘
```

---

## 2. Frame Transport: Decision & Justification

### Options Evaluated
1. **Option A: Windows Named Pipe (`\\.\pipe\reelstudio-frames-{id}`) connecting Worker to Go/ffmpeg**
2. **Option B: Worker directly spawns bundled `ffmpeg.exe` and pipes screenshot buffers to `ffmpeg.stdin` with stream backpressure**

### Decision: Option B (Worker-Spawned ffmpeg with Path Provided by Go)
**Justification:**
1. **Elimination of Cross-Process Deadlocks & Buffering Bottlenecks**:
   At 1080x1920 or 1440x2560 resolutions, uncompressed PNG/JPEG frame buffers are between 1 MB and 11 MB per frame. Windows named pipes require asynchronous Overlapped I/O with rigid OS buffer quotas. If ffmpeg or Go encounters a micro-stall while reading from a named pipe, the worker's write blocks or errors out.
2. **Direct Stream Backpressure**:
   Node.js streams provide built-in backpressure handling:
   ```javascript
   if (!ff.stdin.write(buf)) {
     await new Promise(r => ff.stdin.once('drain', r));
   }
   ```
   This ensures screenshots are only captured as fast as ffmpeg can encode them, maintaining stable memory consumption (~150MB RSS) throughout the entire render.
3. **Exact 1:1 Compatibility with Verified Prototype**:
   The prototype `render.js` relies on this exact architecture. Porting this mechanism directly guarantees identical behavior and eliminates multi-hop IPC latency.
4. **Clean Lifecycle & Process Isolation**:
   Go passes the validated absolute path of `ffmpeg.exe` to the worker in the `render` command. Go places the worker process into a Windows Job Object (`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`). When the worker is canceled or killed, Windows kernel immediately terminates both Node.js and ffmpeg, preventing orphan processes.

---

## 3. IPC Schema (Go ⟷ Worker)

Communication occurs over standard input and standard output using newline-delimited JSON (`ndjson`). Neither stderr nor stdout ever carries binary frame data.

### Go to Worker Commands

#### `render`
Instructs the worker to start rendering a reel:
```json
{
  "cmd": "render",
  "id": "job-a1b2c3d4",
  "html": "C:\\Users\\surface\\AppData\\Local\\ReelStudio\\jobs\\job-a1b2c3d4\\reel.html",
  "audio": "C:\\Users\\surface\\AppData\\Local\\ReelStudio\\jobs\\job-a1b2c3d4\\audio.mp3",
  "out": "C:\\Users\\surface\\Videos\\ReelStudio\\demo-1080p30.mp4",
  "audioDuration": 14.52,
  "width": 1080,
  "height": 1920,
  "dpr": 1.0,
  "fps": 30,
  "quality": "high",
  "crf": 18,
  "preset": "veryfast",
  "imageFormat": "png",
  "tail": 1.0,
  "ffmpegPath": "C:\\Program Files\\Reel Studio\\third_party\\ffmpeg.exe",
  "browserChannel": "msedge",
  "browserExecutable": ""
}
```

#### `cancel`
Signals the worker to abort an active render, terminate ffmpeg and the browser, and clean up temporary files:
```json
{
  "cmd": "cancel",
  "id": "job-a1b2c3d4"
}
```

#### `exit`
Requests graceful shutdown of the worker:
```json
{
  "cmd": "exit"
}
```

### Worker to Go Messages

#### `progress`
Sent periodically as frames render and encode:
```json
{
  "type": "progress",
  "id": "job-a1b2c3d4",
  "phase": "Rendering frames",
  "frame": 120,
  "frames": 466,
  "pct": 27,
  "eta": 18
}
```

#### `done`
Emitted when the final MP4 is completely encoded and verified on disk:
```json
{
  "type": "done",
  "id": "job-a1b2c3d4",
  "result": {
    "width": 1080,
    "height": 1920,
    "fps": 30,
    "duration": 15.52,
    "frames": 466,
    "size": 14258900
  }
}
```

#### `error`
Emitted if any stage fails (e.g., timeout loading iframes, invalid HTML, ffmpeg error):
```json
{
  "type": "error",
  "id": "job-a1b2c3d4",
  "error": "The reel did not start (loading took more than 90 s)."
}
```

#### `log`
Diagnostics stream captured by Go into per-job and rotating daily log files:
```json
{
  "type": "log",
  "id": "job-a1b2c3d4",
  "level": "info",
  "message": "Edge CDP session connected."
}
```

---

## 4. ReelSync Virtual Time Clock Details

ReelSync reels are self-contained single HTML files that coordinate CSS animations, timers, and live website iframes (`iframe.pif[data-src]`).

To guarantee zero dropped frames on any PC hardware:
1. `VIRTUAL_TIME` script is injected into the browser via Playwright `page.addInitScript()` before any page script executes.
2. It overrides `performance.now`, `Date`, `setTimeout`, `setInterval`, `requestAnimationFrame`, and captures `document.getAnimations()`.
3. Loading step: virtual time is ticked in 50 ms increments until `started === true` is observed in page context (with 90s timeout).
4. Time zero reset: `silent.currentTime = 0`.
5. Frame loop: Virtual time is advanced by `1000 / fps` ms per frame, `document.getAnimations()` are synchronized, and the frame is captured via Chrome DevTools Protocol (`Page.captureScreenshot`).

---

## 5. Job Queue State Machine & Concurrency

Jobs transition through the following states:

```
[Queued] ──► [Preparing] ──► [Rendering] ──► [Done]
   │              │               │
   ▼              ▼               ▼
[Cancelled]   [Failed]        [Failed]
   ▲
   │ (On app restart)
[Interrupted] ──► [Queued] (Retry)
```

- **Persistence**: Job metadata and history are stored in `%APPDATA%\ReelStudio\queue.json` (or SQLite) with atomic file writes.
- **Interruption Recovery**: If the application crashes or exits while a job is in `Preparing` or `Rendering`, the job is marked `Interrupted` on the next launch with an available "Retry" action.
- **Worker Concurrency**: Defaults to 1 job at a time. Configurable up to `max(1, NumCPU / 4)` in Settings.
- **Temp Workspace Isolation**: Uploaded reel HTML and audio are copied into `%LOCALAPPDATA%\ReelStudio\jobs\{id}\`. Original user files are never touched.

---

## 6. Windows OS Integration

1. **Taskbar Progress (`ITaskbarList3`)**:
   Reflects the overall or active job progress in the Windows taskbar icon (Normal green progress, Paused yellow, or Error red).
2. **Single Instance Guard**:
   Win32 Named Mutex `Global\ReelStudio_SingleInstance_Mutex`. When a secondary instance launches, it detects the existing mutex, posts the file arguments to the running instance via named pipe or window message, and exits.
3. **Shell Associations**:
   Optional "Open with Reel Studio" registration for `.html` files in Windows Registry (`HKCU\Software\Classes`).
4. **Diagnostics**:
   Rotating log stored in `%APPDATA%\ReelStudio\logs\app-YYYY-MM-DD.log`.

---

## 7. Step Plan

- **Step 1**: Technical Design Document (`DESIGN.md`).
- **Step 2**: Dependency downloader script (`scripts/fetch-deps.ps1`) for portable Node, ffmpeg, ffprobe, and Playwright modules.
- **Step 3**: Implementation of `/worker` (`render-worker.js`, `package.json`, IPC handler, Edge launcher).
- **Step 4**: Test dataset (`/testdata`) and end-to-end worker verification test script.
- **Step 5**: Go backend architecture:
  - Settings manager (`internal/settings`)
  - Audio probe & media validator (`internal/ffmpeg`)
  - Process supervisor & IPC parser (`internal/worker`)
  - Queue manager & state machine (`internal/jobs`)
  - Win32 integrations (`internal/winapi`)
  - Wails application bindings (`app.go`, `main.go`)
- **Step 6**: Go unit test suite (queue ordering, retry/cancel, IPC parsing, ffmpeg args).
- **Step 7**: Frontend UI in Vite + TypeScript (en/de/ar RTL, dropzones, queue, player, settings).
- **Step 8**: Packaging, NSIS installer script, build script (`scripts/build.ps1`), icons.
- **Step 9**: Verification of full build, test execution, and documentation (`README.md`, `THIRD_PARTY.md`).
