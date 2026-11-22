package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type Settings struct {
	DefaultOutputDir       string `json:"defaultOutputDir"`
	DefaultResolution      string `json:"defaultResolution"`
	DefaultFps             int    `json:"defaultFps"`
	DefaultQuality         string `json:"defaultQuality"`
	MaxParallelRenders     int    `json:"maxParallelRenders"`
	KeepIntermediateFiles bool   `json:"keepIntermediateFiles"`
	BrowserChoice          string `json:"browserChoice"` // "msedge" or "chromium"
	Language               string `json:"language"`      // "en", "de", "ar"
	FilenameTemplate       string `json:"filenameTemplate"`
	OpenWithEnabled        bool   `json:"openWithEnabled"`
}

type Manager struct {
	mu       sync.RWMutex
	filePath string
	current  Settings
}

func MaxAllowedConcurrency() int {
	max := runtime.NumCPU() / 4
	if max < 1 {
		return 1
	}
	return max
}

func DefaultSettings() Settings {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		userProfile, _ = os.UserHomeDir()
	}
	defaultVideos := filepath.Join(userProfile, "Videos", "ReelStudio")

	return Settings{
		DefaultOutputDir:       defaultVideos,
		DefaultResolution:      "1080",
		DefaultFps:             30,
		DefaultQuality:         "high",
		MaxParallelRenders:     1,
		KeepIntermediateFiles: false,
		BrowserChoice:          "msedge",
		Language:               "en",
		FilenameTemplate:       "{reel}-{res}p{fps}.mp4",
		OpenWithEnabled:        false,
	}
}

func NewManager() (*Manager, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, "AppData", "Roaming")
	}

	dir := filepath.Join(appData, "ReelStudio")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	filePath := filepath.Join(dir, "settings.json")
	m := &Manager{
		filePath: filePath,
		current:  DefaultSettings(),
	}

	if err := m.load(); err != nil {
		// If file doesn't exist, save default settings
		_ = m.save()
	}

	return m, nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Validate & Clamp settings
	if s.DefaultOutputDir == "" {
		s.DefaultOutputDir = DefaultSettings().DefaultOutputDir
	}
	if s.DefaultResolution == "" {
		s.DefaultResolution = "1080"
	}
	if s.DefaultFps <= 0 {
		s.DefaultFps = 30
	}
	if s.DefaultQuality == "" {
		s.DefaultQuality = "high"
	}
	maxAllowed := MaxAllowedConcurrency()
	if s.MaxParallelRenders < 1 {
		s.MaxParallelRenders = 1
	} else if s.MaxParallelRenders > maxAllowed {
		s.MaxParallelRenders = maxAllowed
	}
	if s.BrowserChoice == "" {
		s.BrowserChoice = "msedge"
	}
	if s.Language == "" {
		s.Language = "en"
	}
	if s.FilenameTemplate == "" {
		s.FilenameTemplate = "{reel}-{res}p{fps}.mp4"
	}

	m.current = s
	return nil
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.current, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0644)
}

func (m *Manager) Get() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manager) Update(s Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	maxAllowed := MaxAllowedConcurrency()
	if s.MaxParallelRenders < 1 {
		s.MaxParallelRenders = 1
	} else if s.MaxParallelRenders > maxAllowed {
		s.MaxParallelRenders = maxAllowed
	}

	m.current = s
	return m.save()
}
