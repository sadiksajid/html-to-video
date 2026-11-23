package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

type MockupImageResult struct {
	Path    string `json:"path"`
	DataURI string `json:"dataUri"`
}

type App struct {
	ctx         context.Context
	rootDir     string
	settingsMgr *settings.Manager
	supervisor  *worker.Supervisor
	queueMgr    *jobs.QueueManager
	nodePath    string
	ffmpegPath  string
	ffprobePath string
}

func AcquireSingleInstance(name string) (*winapi.MutexLock, bool, error) {
	return winapi.AcquireSingleInstance(name)
}

func init() {
	if runtime.GOOS == "darwin" {
		currentPath := os.Getenv("PATH")
		extraPaths := []string{
			"/opt/homebrew/bin",
			"/opt/homebrew/sbin",
			"/usr/local/bin",
			"/usr/bin",
			"/bin",
			"/usr/sbin",
			"/sbin",
		}
		for _, p := range extraPaths {
			if !strings.Contains(currentPath, p) {
				currentPath = p + ":" + currentPath
			}
		}
		_ = os.Setenv("PATH", currentPath)
	}
}

func resolveExecutable(rootDir, name string) string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	target := name + ext

	candidates := []string{
		filepath.Join(rootDir, "third_party", target),
		filepath.Join(rootDir, target),
	}

	if runtime.GOOS == "darwin" {
		parent := filepath.Dir(rootDir)
		candidates = append(candidates,
			filepath.Join(parent, "Resources", "third_party", target),
			filepath.Join(parent, "Resources", target),
			filepath.Join("/opt/homebrew/bin", target),
			filepath.Join("/usr/local/bin", target),
			filepath.Join("/opt/local/bin", target),
			filepath.Join("/usr/bin", target),
		)
	}

	for _, cand := range candidates {
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand
		}
	}

	// Fallback to exec.LookPath
	if p, err := exec.LookPath(target); err == nil {
		return p
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	return filepath.Join(rootDir, "third_party", target)
}

func NewApp(rootDir string) (*App, error) {
	settingsMgr, err := settings.NewManager()
	if err != nil {
		return nil, fmt.Errorf("settings manager init failed: %w", err)
	}

	nodePath := resolveExecutable(rootDir, "node")
	ffmpegPath := resolveExecutable(rootDir, "ffmpeg")
	ffprobePath := resolveExecutable(rootDir, "ffprobe")
	workerScript := filepath.Join(rootDir, "worker", "render-worker.js")

	supervisor := worker.NewSupervisor(nodePath, workerScript, rootDir)

	a := &App{
		rootDir:     rootDir,
		settingsMgr: settingsMgr,
		supervisor:  supervisor,
		nodePath:    nodePath,
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

func (a *App) SelectMockupImage() (*MockupImageResult, error) {
	filePath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Mockup Replacement Image",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Images (*.png;*.jpg;*.jpeg;*.webp)", Pattern: "*.png;*.jpg;*.jpeg;*.webp"},
		},
	})
	if err != nil || filePath == "" {
		return nil, err
	}
	dataURI := jobs.ImageToDataURI(filePath)
	return &MockupImageResult{
		Path:    filePath,
		DataURI: dataURI,
	}, nil
}

func (a *App) GetImageDataURI(filePath string) (string, error) {
	return jobs.ImageToDataURI(filePath), nil
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
		`/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge`,
		`/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`,
	}
	edgeFound := false
	for _, p := range edgePaths {
		if _, err := os.Stat(p); err == nil {
			edgeFound = true
			break
		}
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, _ := os.UserHomeDir()
		localAppData = filepath.Join(home, "Library", "Application Support")
	}
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
		a.nodePath, a.ffmpegPath, a.ffprobePath))

	sb.WriteString("\nRecent Worker Logs (Last 50 lines):\n")
	for _, line := range a.supervisor.RecentLogs() {
		sb.WriteString(line + "\n")
	}

	return sb.String()
}
