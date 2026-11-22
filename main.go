package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"reelstudio/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 1. Single instance lock
	lock, acquired, err := app.AcquireSingleInstance("Local\\ReelStudioSingleInstanceMutex")
	if err == nil && !acquired {
		// Another instance is already running
		println("Reel Studio is already running.")
		os.Exit(0)
	}
	defer lock.Release()

	// 2. Initialize application
	workDir, _ := os.Getwd()
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
