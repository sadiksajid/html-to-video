package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"reelstudio/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func getAppRootDir() string {
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		// On macOS, if inside an App bundle (.../ReelStudio.app/Contents/MacOS), root is Contents/MacOS
		if runtime.GOOS == "darwin" && strings.Contains(exe, ".app/Contents/MacOS") {
			return exeDir
		}
		// If third_party or worker exists next to exe, use exeDir
		if _, err := os.Stat(filepath.Join(exeDir, "worker")); err == nil {
			return exeDir
		}
		if _, err := os.Stat(filepath.Join(exeDir, "third_party")); err == nil {
			return exeDir
		}
	}
	wd, err := os.Getwd()
	if err == nil && wd != "/" && wd != "" {
		return wd
	}
	if err == nil && exe != "" {
		return filepath.Dir(exe)
	}
	return "."
}

func main() {
	// 1. Single instance lock
	lock, acquired, err := app.AcquireSingleInstance("Local\\ReelStudioSingleInstanceMutex")
	if err == nil && !acquired {
		// Another instance is already running
		println("Reel Studio is already running.")
		os.Exit(0)
	}
	defer lock.Release()

	// 2. Initialize application with robust root directory resolution
	workDir := getAppRootDir()
	application, err := app.NewApp(workDir)
	if err != nil {
		log.Fatalf("Failed to initialize Reel Studio: %v", err)
	}

	// 3. Launch Wails application
	err = wails.Run(&options.App{
		Title:  "Reel Studio",
		Width:  1180,
		Height: 820,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 1}, // Modern dark slate
		OnStartup:        application.Startup,
		OnShutdown:       application.Shutdown,
		Bind: []interface{}{
			application,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
			Theme:                windows.Dark,
		},
	})

	if err != nil {
		log.Fatalf("Error running application: %v", err)
	}
}
