# Third-Party Notices & Licenses

Reel Studio bundles and interfaces with several open-source technologies. This document acknowledges their copyright holders and license terms.

---

## 1. Node.js (Portable Runtime)
- **Version**: v20.18.1 (LTS, win-x64)
- **Source**: https://nodejs.org/
- **License**: MIT License
- **Copyright**: Copyright Joyent, Inc. and other Node contributors. All rights reserved.

---

## 2. FFmpeg & FFprobe
- **Version**: 9.0.2 (Essentials Build by Gyan.dev)
- **Source**: https://www.gyan.dev/ffmpeg/builds/
- **License**: GNU General Public License v3.0+ (GPLv3+) / GNU Lesser General Public License v2.1+ (LGPLv2.1+)
- **Notices**:
  FFmpeg is an open-source multimedia framework developed by the FFmpeg team.
  Source code for FFmpeg can be downloaded from: https://ffmpeg.org/download.html
  Reel Studio invokes `ffmpeg.exe` and `ffprobe.exe` as separate, standalone child processes via standard system command execution and standard pipes (`stdin`/`stdout`). It does not link directly with FFmpeg static or shared libraries.

---

## 3. Microsoft Playwright
- **Version**: 1.65.1
- **Source**: https://github.com/microsoft/playwright
- **License**: Apache License 2.0
- **Copyright**: Copyright (c) Microsoft Corporation.

---

## 4. Wails v2
- **Version**: v2.17.0
- **Source**: https://wails.io / https://github.com/wailsapp/wails
- **License**: MIT License
- **Copyright**: Copyright (c) 2019-present Lea Anthony

---

## 5. Microsoft Edge & WebView2
- **Component**: Microsoft Edge WebView2 Runtime & Microsoft Edge Browser
- **License**: Microsoft Software License Terms
- **Notice**: Utilizes the system-installed Evergreen Microsoft Edge / WebView2 runtime present on Windows 10 (22H2+) and Windows 11.
