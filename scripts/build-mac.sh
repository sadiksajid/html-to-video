#!/usr/bin/env bash
# ==============================================================================
#  Reel Studio - macOS Build & Packaging Script
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=========================================="
echo "    Reel Studio - macOS Build Pipeline    "
echo "=========================================="

# 1. Check prerequisites
echo "[1/5] Checking build prerequisites..."
command -v go >/dev/null 2>&1 || { echo >&2 "Go is required. Install via: brew install go"; exit 1; }
command -v node >/dev/null 2>&1 || { echo >&2 "Node.js is required. Install via: brew install node"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo >&2 "npm is required."; exit 1; }

# Check for Wails CLI
if ! command -v wails >/dev/null 2>&1; then
    if [ -f "$HOME/go/bin/wails" ]; then
        export PATH="$HOME/go/bin:$PATH"
    else
        echo "Wails CLI not found. Installing..."
        go install github.com/wailsapp/wails/v2/cmd/wails@latest
        export PATH="$HOME/go/bin:$PATH"
    fi
fi

# Check for ffmpeg (optional if bundled, but recommended via Homebrew)
if ! command -v ffmpeg >/dev/null 2>&1; then
    echo "Notice: ffmpeg not found in PATH. Install via 'brew install ffmpeg' or place binaries in third_party/"
fi

# 2. Build Frontend
echo "[2/5] Building Frontend (Vite + TypeScript)..."
cd "${ROOT_DIR}/frontend"
npm install
npm run build

# 3. Setup Worker Dependencies
echo "[3/5] Setting up Worker node_modules..."
cd "${ROOT_DIR}/worker"
npm install --omit=dev

# 4. Compile macOS App using Wails
echo "[4/5] Compiling macOS App via Wails..."
cd "${ROOT_DIR}"

# Determine architecture: universal (Intel + Apple Silicon) or native
ARCH="darwin/universal"
if [ "${1:-}" = "--arm64" ]; then
    ARCH="darwin/arm64"
elif [ "${1:-}" = "--amd64" ]; then
    ARCH="darwin/amd64"
fi

echo "Target architecture: ${ARCH}"
wails build -platform "${ARCH}" -clean

APP_PATH="${ROOT_DIR}/build/bin/ReelStudio.app"

# 5. Bundle Worker & third_party into App Bundle
echo "[5/5] Bundling worker & assets into ${APP_PATH}..."
CONTENTS_DIR="${APP_PATH}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

mkdir -p "${MACOS_DIR}/worker"
mkdir -p "${MACOS_DIR}/third_party"

# Copy worker script and node_modules
cp "${ROOT_DIR}/worker/render-worker.js" "${MACOS_DIR}/worker/"
cp -R "${ROOT_DIR}/worker/node_modules" "${MACOS_DIR}/worker/"
cp "${ROOT_DIR}/worker/package.json" "${MACOS_DIR}/worker/"

# Copy third_party binaries if present
if [ -d "${ROOT_DIR}/third_party" ]; then
    cp -R "${ROOT_DIR}/third_party/"* "${MACOS_DIR}/third_party/" 2>/dev/null || true
fi

# Create a portable zip package for distribution
ZIP_NAME="ReelStudio-mac-${ARCH//\//-}.zip"
cd "${ROOT_DIR}/build/bin"
zip -r -y -q "${ZIP_NAME}" "ReelStudio.app"

echo "=========================================="
echo "      BUILD COMPLETED SUCCESSFULLY!       "
echo "=========================================="
echo "Application: ${APP_PATH}"
echo "Zip Archive: ${ROOT_DIR}/build/bin/${ZIP_NAME}"
