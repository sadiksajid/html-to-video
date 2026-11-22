package app

import (
	"log"
	"os"

	"reelstudio/app/internal/winapi"
)

// RunApp is a helper to launch the application.
func RunApp() {
	lock, acquired, err := winapi.AcquireSingleInstance("Local\\ReelStudioSingleInstanceMutex")
	if err == nil && !acquired {
		println("Reel Studio is already running.")
		os.Exit(0)
	}
	defer lock.Release()

	workDir, _ := os.Getwd()
	application, err := NewApp(workDir)
	if err != nil {
		log.Fatalf("Failed to initialize Reel Studio: %v", err)
	}

	println("Reel Studio backend initialized:", application)
}
