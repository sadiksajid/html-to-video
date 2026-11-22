package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"reelstudio/app/internal/iframe"
	"reelstudio/app/internal/jobs"
	"reelstudio/app/internal/settings"
	"reelstudio/app/internal/winapi"
	"reelstudio/app/internal/worker"
)

type BatchItem struct {
	ReelPath   string `json:"reelPath"`
	AudioPath  string `json:"audioPath"`
	Resolution string `json:"resolution"`
	Fps        int    `json:"fps"`
	Quality    string `json:"quality"`
}

type BatchPairResult struct {
	Pairs   []BatchItem `json:"pairs"`
	Unpairs []string    `json:"unpairs"`
}

type BrowserStatus struct {
	EdgeAvailable     bool   `json:"edgeAvailable"`
	ChromiumAvailable bool   `json:"chromiumAvailable"`
	SelectedEngine    string `json:"selectedEngine"`
}

type App struct {
	ctx         context.Context
	rootDir     string
	settingsMgr *settings.Manager
	supervisor  *worker.Supervisor
	queueMgr    *jobs.QueueManager
	ffmpegPath  string
	ffprobePath string
}

func AcquireSingleInstance(name string) (*winapi.MutexLock, bool, error) {
	return winapi.AcquireSingleInstance(name)
}

func NewApp(rootDir string) (*App, error) {
	settingsMgr, err := settings.NewManager()
	if err != nil {
		return nil, fmt.Errorf("settings manager init failed: %w", err)
	}

	nodePath := filepath.Join(rootDir, "third_party", "node.exe")
	ffmpegPath := filepath.Join(rootDir, "third_party", "ffmpeg.exe")
	ffprobePath := filepath.Join(rootDir, "third_party", "ffprobe.exe")
	workerScript := filepath.Join(rootDir, "worker", "render-worker.js")

	supervisor := worker.NewSupervisor(nodePath, workerScript, rootDir)

	a := &App{
		rootDir:     rootDir,
		settingsMgr: settingsMgr,
		supervisor:  supervisor,
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
	}

	queueMgr, err := jobs.NewQueueManager(
		settingsMgr,
		supervisor,
		ffmpegPath,
		ffprobePath,
		func(j *jobs.Job) {
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "job:update", j)
			}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("queue manager init failed: %w", err)
	}
	a.queueMgr = queueMgr

	return a, nil
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Shutdown(ctx context.Context) {
	if a.queueMgr != nil {
		a.queueMgr.Close()
	}
	if a.supervisor != nil {
		a.supervisor.Stop()
	}
}

func (a *App) GetSettings() settings.Settings {
	return a.settingsMgr.Get()
}

func (a *App) UpdateSettings(s settings.Settings) error {
	return a.settingsMgr.Update(s)
}

func (a *App) GetJobs() []*jobs.Job {
	return a.queueMgr.GetJobs()
}

func (a *App) CheckReelIframes(reelPath string) ([]iframe.CheckResult, error) {
	return iframe.CheckReelFile(reelPath)
}

func (a *App) SelectMockupImage() (string, error) {
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Mockup Replacement Image",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Images (*.png;*.jpg;*.jpeg;*.webp)", Pattern: "*.png;*.jpg;*.jpeg;*.webp"},
		},
	})
}

func (a *App) AddJob(reelPath, audioPath, resolution string, fps int, quality string, outDir string, iframeReplacements map[string]string) (*jobs.Job, error) {
	return a.queueMgr.AddJob(reelPath, audioPath, resolution, fps, quality, outDir, iframeReplacements)
}

func (a *App) AddBatchJobs(items []BatchItem) ([]*jobs.Job, error) {
	s := a.settingsMgr.Get()
	var created []*jobs.Job
	for _, it := range items {
		res := it.Resolution
		if res == "" {
			res = s.DefaultResolution
		}
		fps := it.Fps
		if fps <= 0 {
			fps = s.DefaultFps
		}
		q := it.Quality
		if q == "" {
			q = s.DefaultQuality
		}

		j, err := a.queueMgr.AddJob(it.ReelPath, it.AudioPath, res, fps, q, s.DefaultOutputDir, nil)
		if err != nil {
			return created, err
		}
		created = append(created, j)
	}
	return created, nil
}

func (a *App) CancelJob(id string) error {
	return a.queueMgr.CancelJob(id)
}

func (a *App) RetryJob(id string) error {
	return a.queueMgr.RetryJob(id)
}

func (a *App) DeleteJob(id string) error {
	return a.queueMgr.DeleteJob(id)
}

func (a *App) SelectReelFile() (string, error) {
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select ReelSync Reel HTML",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "ReelSync HTML (*.html;*.htm)", Pattern: "*.html;*.htm"},
		},
	})
}

func (a *App) SelectAudioFile() (string, error) {
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Voice-over Audio",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Audio Files (*.mp3;*.wav;*.m4a;*.aac;*.ogg)", Pattern: "*.mp3;*.wav;*.m4a;*.aac;*.ogg"},
		},
	})
}

func (a *App) SelectOutputDir() (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Output Video Folder",
	})
}

func (a *App) SelectBatchFiles() (BatchPairResult, error) {
	files, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Reel HTML & Audio Files",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Reels & Audios", Pattern: "*.html;*.htm;*.mp3;*.wav;*.m4a;*.aac;*.ogg"},
		},
	})
	if err != nil || len(files) == 0 {
		return BatchPairResult{}, err
	}

	htmlMap := make(map[string]string)
	audioMap := make(map[string]string)

	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		base := strings.ToLower(strings.TrimSuffix(filepath.Base(f), ext))
		if ext == ".html" || ext == ".htm" {
			htmlMap[base] = f
		} else {
			audioMap[base] = f
		}
	}

	var pairs []BatchItem
	var unpairs []string

	s := a.settingsMgr.Get()

	// Pair matching by base name
	for base, hPath := range htmlMap {
		if aPath, ok := audioMap[base]; ok {
			pairs = append(pairs, BatchItem{
				ReelPath:   hPath,
				AudioPath:  aPath,
				Resolution: s.DefaultResolution,
				Fps:        s.DefaultFps,
				Quality:    s.DefaultQuality,
			})
			delete(audioMap, base)
		} else {
			unpairs = append(unpairs, hPath)
		}
	}

	for _, aPath := range audioMap {
		unpairs = append(unpairs, aPath)
	}

	return BatchPairResult{Pairs: pairs, Unpairs: unpairs}, nil
}

func (a *App) OpenInExplorer(filePath string) error {
	return winapi.OpenInExplorer(filePath)
}

func (a *App) OpenFile(filePath string) error {
	return winapi.OpenFile(filePath)
}

func (a *App) CheckBrowserStatus() BrowserStatus {
	edgePaths := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	}
	edgeFound := false
	for _, p := range edgePaths {
		if _, err := os.Stat(p); err == nil {
			edgeFound = true
			break
		}
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	chromDir := filepath.Join(localAppData, "ReelStudio", "browsers")
	chromFound := false
	if _, err := os.Stat(chromDir); err == nil {
		chromFound = true
	}

	return BrowserStatus{
		EdgeAvailable:     edgeFound,
		ChromiumAvailable: chromFound,
		SelectedEngine:    a.settingsMgr.Get().BrowserChoice,
	}
}

func (a *App) GetDiagnostics() string {
	s := a.settingsMgr.Get()
	var sb strings.Builder

	sb.WriteString("=== Reel Studio Diagnostics ===\n")
	sb.WriteString(fmt.Sprintf("OS: %s %s (%s)\n", runtime.GOOS, runtime.GOARCH, runtime.Version()))
	sb.WriteString(fmt.Sprintf("CPUs: %d (Max Render Workers: %d)\n", runtime.NumCPU(), settings.MaxAllowedConcurrency()))
	sb.WriteString(fmt.Sprintf("Settings: Res=%s, FPS=%d, Quality=%s, MaxParallel=%d\n",
		s.DefaultResolution, s.DefaultFps, s.DefaultQuality, s.MaxParallelRenders))
	sb.WriteString(fmt.Sprintf("Binary Paths:\n  Node: %s\n  FFmpeg: %s\n  FFprobe: %s\n",
		filepath.Join(a.rootDir, "third_party", "node.exe"), a.ffmpegPath, a.ffprobePath))

	sb.WriteString("\nRecent Worker Logs (Last 50 lines):\n")
	for _, line := range a.supervisor.RecentLogs() {
		sb.WriteString(line + "\n")
	}

	return sb.String()
}
