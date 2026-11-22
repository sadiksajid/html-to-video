package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"reelstudio/app/internal/ffmpeg"
	"reelstudio/app/internal/settings"
	"reelstudio/app/internal/winapi"
	"reelstudio/app/internal/worker"
)

type Status string

const (
	StatusQueued      Status = "queued"
	StatusRunning     Status = "running"
	StatusDone        Status = "done"
	StatusError       Status = "error"
	StatusCancelled   Status = "cancelled"
	StatusInterrupted Status = "interrupted"
)

type JobOptions struct {
	Resolution string `json:"resolution"`
	Fps        int    `json:"fps"`
	Quality    string `json:"quality"`
}

type Job struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Status    Status      `json:"status"`
	Phase     string      `json:"phase"`
	Frame     int         `json:"frame"`
	Frames    int         `json:"frames"`
	Pct       int         `json:"pct"`
	ETA       int         `json:"eta"`
	Error     string      `json:"error,omitempty"`
	ReelPath  string      `json:"reelPath"`
	AudioPath string      `json:"audioPath"`
	OutPath   string      `json:"outPath"`
	Options   JobOptions  `json:"options"`
	Duration  float64     `json:"duration"`
	FileSize  int64       `json:"fileSize"`
	Created   int64       `json:"created"`
	Completed int64       `json:"completed"`
	Logs      []string    `json:"logs"`
}

type JobUpdateCallback func(job *Job)

type QueueManager struct {
	mu          sync.RWMutex
	storePath   string
	jobsDir     string
	jobs        map[string]*Job
	queue       []string // job IDs in FIFO order
	settingsMgr *settings.Manager
	supervisor  *worker.Supervisor
	ffprobePath string
	ffmpegPath  string
	onUpdate    JobUpdateCallback
	activeJobs  map[string]bool
	stopChan    chan struct{}
}

func randomID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func sanitizeFilename(name string) string {
	reg := regexp.MustCompile(`[^\w.-]+`)
	return reg.ReplaceAllString(name, "-")
}

func FormatOutputName(template string, reelBase string, res string, fps int) string {
	if template == "" {
		template = "{reel}-{res}p{fps}.mp4"
	}
	r := strings.ReplaceAll(template, "{reel}", reelBase)
	r = strings.ReplaceAll(r, "{res}", res)
	r = strings.ReplaceAll(r, "{fps}", fmt.Sprint(fps))
	if !strings.HasSuffix(strings.ToLower(r), ".mp4") {
		r += ".mp4"
	}
	return r
}

func NewQueueManager(
	settingsMgr *settings.Manager,
	supervisor *worker.Supervisor,
	ffmpegPath string,
	ffprobePath string,
	onUpdate JobUpdateCallback,
) (*QueueManager, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	appDir := filepath.Join(appData, "ReelStudio")
	_ = os.MkdirAll(appDir, 0755)

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		home, _ := os.UserHomeDir()
		localAppData = filepath.Join(home, "AppData", "Local")
	}
	jobsDir := filepath.Join(localAppData, "ReelStudio", "jobs")
	_ = os.MkdirAll(jobsDir, 0755)

	qm := &QueueManager{
		storePath:   filepath.Join(appDir, "jobs.json"),
		jobsDir:     jobsDir,
		jobs:        make(map[string]*Job),
		queue:       make([]string, 0),
		settingsMgr: settingsMgr,
		supervisor:  supervisor,
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		onUpdate:    onUpdate,
		activeJobs:  make(map[string]bool),
		stopChan:    make(chan struct{}),
	}

	qm.load()
	go qm.listenWorkerMessages()
	go qm.processLoop()

	return qm, nil
}

func (qm *QueueManager) load() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	data, err := os.ReadFile(qm.storePath)
	if err != nil {
		return
	}

	var list []*Job
	if err := json.Unmarshal(data, &list); err != nil {
		return
	}

	for _, j := range list {
		// Mark unfinished jobs as interrupted on restart
		if j.Status == StatusRunning || j.Status == StatusQueued {
			j.Status = StatusInterrupted
			j.Phase = "Interrupted on previous session"
		}
		qm.jobs[j.ID] = j
	}
}

func (qm *QueueManager) saveLocked() {
	list := make([]*Job, 0, len(qm.jobs))
	for _, j := range qm.jobs {
		list = append(list, j)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err == nil {
		_ = os.WriteFile(qm.storePath, data, 0644)
	}
}

func (qm *QueueManager) notify(j *Job) {
	if qm.onUpdate != nil {
		qm.onUpdate(j)
	}
}

func (qm *QueueManager) AddJob(
	reelPath, audioPath, resolution string,
	fps int, quality string,
	outDir string,
) (*Job, error) {
	// 1. Validate inputs exist
	if _, err := os.Stat(reelPath); err != nil {
		return nil, fmt.Errorf("reel HTML file does not exist: %w", err)
	}
	if _, err := os.Stat(audioPath); err != nil {
		return nil, fmt.Errorf("audio file does not exist: %w", err)
	}

	// 2. Validate HTML is a ReelSync reel
	htmlBytes, err := os.ReadFile(reelPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read reel HTML: %w", err)
	}
	htmlContent := string(htmlBytes)
	if !strings.Contains(htmlContent, "silentBtn") || !strings.Contains(htmlContent, "silent") {
		return nil, errors.New("the selected file is not a valid ReelSync reel (silent clock/button missing)")
	}

	// 3. Probe audio duration
	duration, err := ffmpeg.ProbeAudioDuration(qm.ffprobePath, audioPath)
	if err != nil {
		return nil, fmt.Errorf("audio unreadable by ffprobe: %w", err)
	}

	// 4. Resolve options
	s := qm.settingsMgr.Get()
	if resolution == "" {
		resolution = s.DefaultResolution
	}
	if fps <= 0 {
		fps = s.DefaultFps
	}
	if quality == "" {
		quality = s.DefaultQuality
	}
	if outDir == "" {
		outDir = s.DefaultOutputDir
	}
	_ = os.MkdirAll(outDir, 0755)

	reelBase := filepath.Base(reelPath)
	ext := filepath.Ext(reelBase)
	rawName := strings.TrimSuffix(reelBase, ext)
	cleanName := sanitizeFilename(rawName)
	if cleanName == "" {
		cleanName = "reel"
	}

	fileName := FormatOutputName(s.FilenameTemplate, cleanName, resolution, fps)
	outPath := filepath.Join(outDir, fileName)

	// Ensure unique output filename if file already exists
	if _, err := os.Stat(outPath); err == nil {
		baseWithoutExt := strings.TrimSuffix(fileName, ".mp4")
		outPath = filepath.Join(outDir, fmt.Sprintf("%s_%d.mp4", baseWithoutExt, time.Now().Unix()))
	}

	id := "job-" + randomID()
	job := &Job{
		ID:        id,
		Name:      filepath.Base(outPath),
		Status:    StatusQueued,
		Phase:     "Waiting in queue",
		Pct:       0,
		ReelPath:  reelPath,
		AudioPath: audioPath,
		OutPath:   outPath,
		Options: JobOptions{
			Resolution: resolution,
			Fps:        fps,
			Quality:    quality,
		},
		Duration: duration,
		Created:  time.Now().UnixMilli(),
		Logs:     make([]string, 0),
	}

	qm.mu.Lock()
	qm.jobs[id] = job
	qm.queue = append(qm.queue, id)
	qm.saveLocked()
	qm.mu.Unlock()

	qm.notify(job)
	return job, nil
}

func (qm *QueueManager) GetJobs() []*Job {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	list := make([]*Job, 0, len(qm.jobs))
	for _, j := range qm.jobs {
		// Return copy
		copyJob := *j
		list = append(list, &copyJob)
	}
	return list
}

func (qm *QueueManager) GetJob(id string) (*Job, bool) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	j, ok := qm.jobs[id]
	if !ok {
		return nil, false
	}
	copyJob := *j
	return &copyJob, true
}

func (qm *QueueManager) CancelJob(id string) error {
	qm.mu.Lock()
	j, ok := qm.jobs[id]
	if !ok {
		qm.mu.Unlock()
		return errors.New("job not found")
	}

	if j.Status == StatusRunning {
		j.Status = StatusCancelled
		j.Phase = "Cancelled"
		qm.saveLocked()
		qm.mu.Unlock()
		_ = qm.supervisor.SendCancel(id)
		qm.notify(j)
		return nil
	}

	if j.Status == StatusQueued {
		j.Status = StatusCancelled
		j.Phase = "Cancelled"
		// Remove from queue
		newQ := make([]string, 0, len(qm.queue))
		for _, qid := range qm.queue {
			if qid != id {
				newQ = append(newQ, qid)
			}
		}
		qm.queue = newQ
		qm.saveLocked()
		qm.mu.Unlock()
		qm.notify(j)
		return nil
	}

	qm.mu.Unlock()
	return nil
}

func (qm *QueueManager) RetryJob(id string) error {
	qm.mu.Lock()
	j, ok := qm.jobs[id]
	if !ok {
		qm.mu.Unlock()
		return errors.New("job not found")
	}

	j.Status = StatusQueued
	j.Phase = "Waiting in queue"
	j.Pct = 0
	j.Frame = 0
	j.Error = ""
	qm.queue = append(qm.queue, id)
	qm.saveLocked()
	qm.mu.Unlock()

	qm.notify(j)
	return nil
}

func (qm *QueueManager) DeleteJob(id string) error {
	qm.mu.Lock()
	j, ok := qm.jobs[id]
	if !ok {
		qm.mu.Unlock()
		return errors.New("job not found")
	}

	if j.Status == StatusRunning {
		qm.mu.Unlock()
		return errors.New("cannot delete an active rendering job; cancel it first")
	}

	delete(qm.jobs, id)
	// Remove from queue if present
	newQ := make([]string, 0, len(qm.queue))
	for _, qid := range qm.queue {
		if qid != id {
			newQ = append(newQ, qid)
		}
	}
	qm.queue = newQ

	qm.saveLocked()
	qm.mu.Unlock()

	// Clean job workspace
	jobDir := filepath.Join(qm.jobsDir, id)
	_ = os.RemoveAll(jobDir)

	return nil
}

func (qm *QueueManager) listenWorkerMessages() {
	for msg := range qm.supervisor.Messages() {
		qm.mu.Lock()
		j, ok := qm.jobs[msg.ID]
		if !ok {
			qm.mu.Unlock()
			continue
		}

		switch msg.Type {
		case "progress":
			j.Phase = msg.Phase
			j.Frame = msg.Frame
			j.Frames = msg.Frames
			j.Pct = msg.Pct
			j.ETA = msg.ETA
			qm.mu.Unlock()
			qm.notify(j)

		case "done":
			j.Status = StatusDone
			j.Phase = "Done"
			j.Pct = 100
			j.Completed = time.Now().UnixMilli()
			if msg.Result != nil {
				j.Duration = msg.Result.Duration
				j.FileSize = msg.Result.Size
			}
			delete(qm.activeJobs, j.ID)
			qm.saveLocked()
			qm.mu.Unlock()
			qm.notify(j)

			// Clean temp directory if setting disabled
			if !qm.settingsMgr.Get().KeepIntermediateFiles {
				_ = os.RemoveAll(filepath.Join(qm.jobsDir, j.ID))
			}

			// Show native desktop notification
			winapi.ShowToastNotification("Reel Studio", fmt.Sprintf("Rendering complete: %s", j.Name))

		case "error":
			j.Status = StatusError
			j.Phase = "Failed"
			j.Error = msg.Error
			j.Completed = time.Now().UnixMilli()
			delete(qm.activeJobs, j.ID)
			qm.saveLocked()
			qm.mu.Unlock()
			qm.notify(j)

		case "log":
			if len(j.Logs) < 200 {
				j.Logs = append(j.Logs, fmt.Sprintf("[%s] %s", msg.Level, msg.Message))
			}
			qm.mu.Unlock()
		}
	}
}

func (qm *QueueManager) processLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-qm.stopChan:
			return
		case <-ticker.C:
			qm.checkAndRunNext()
		}
	}
}

func (qm *QueueManager) checkAndRunNext() {
	qm.mu.Lock()
	maxParallel := qm.settingsMgr.Get().MaxParallelRenders
	if len(qm.activeJobs) >= maxParallel || len(qm.queue) == 0 {
		qm.mu.Unlock()
		return
	}

	// Pop next queued job
	jobID := qm.queue[0]
	qm.queue = qm.queue[1:]

	job, ok := qm.jobs[jobID]
	if !ok || job.Status != StatusQueued {
		qm.mu.Unlock()
		return
	}

	job.Status = StatusRunning
	job.Phase = "Preparing workspace"
	qm.activeJobs[jobID] = true
	qm.saveLocked()
	qm.mu.Unlock()

	qm.notify(job)

	// Execute job in goroutine
	go qm.executeJob(job)
}

func (qm *QueueManager) executeJob(job *Job) {
	// 1. Disk space check (>= 2GB free)
	freeBytes, err := winapi.CheckFreeDiskSpace(filepath.Dir(job.OutPath))
	if err == nil && freeBytes < 2*1024*1024*1024 {
		qm.failJob(job, fmt.Sprintf("Insufficient disk space on target drive (requires at least 2 GB free, found %.2f GB)", float64(freeBytes)/(1024*1024*1024)))
		return
	}

	// 2. Prepare temp working folder
	jobDir := filepath.Join(qm.jobsDir, job.ID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		qm.failJob(job, fmt.Sprintf("Failed creating temp job workspace: %v", err))
		return
	}

	tempReelPath := filepath.Join(jobDir, "reel.html")
	audioExt := filepath.Ext(job.AudioPath)
	if audioExt == "" {
		audioExt = ".mp3"
	}
	tempAudioPath := filepath.Join(jobDir, "audio"+audioExt)

	// Copy files to temp workspace to protect originals
	if err := copyFile(job.ReelPath, tempReelPath); err != nil {
		qm.failJob(job, fmt.Sprintf("Failed copying reel to workspace: %v", err))
		return
	}
	if err := copyFile(job.AudioPath, tempAudioPath); err != nil {
		qm.failJob(job, fmt.Sprintf("Failed copying audio to workspace: %v", err))
		return
	}

	// 3. Ensure supervisor is running
	if err := qm.supervisor.Start(); err != nil {
		qm.failJob(job, fmt.Sprintf("Failed starting render worker process: %v", err))
		return
	}

	// 4. Resolve presets
	res, ok := ffmpeg.Resolutions[job.Options.Resolution]
	if !ok {
		res = ffmpeg.Resolutions["1080"]
	}
	qPreset, ok := ffmpeg.QualityPresets[job.Options.Quality]
	if !ok {
		qPreset = ffmpeg.QualityPresets["high"]
	}

	dpr := 1.0
	if job.Options.Resolution == "1440" {
		dpr = 1.3333
	}

	s := qm.settingsMgr.Get()

	// 5. Send render command to worker
	cmd := worker.RenderCommand{
		Cmd:            "render",
		ID:             job.ID,
		HTML:           tempReelPath,
		Audio:          tempAudioPath,
		Out:            job.OutPath,
		AudioDuration:  job.Duration,
		Width:          res.Width,
		Height:         res.Height,
		Dpr:            dpr,
		Fps:            job.Options.Fps,
		Quality:        job.Options.Quality,
		CRF:            qPreset.CRF,
		Preset:         qPreset.Preset,
		ImageFormat:    qPreset.ImageFormat,
		Tail:           1.0,
		FFmpegPath:     qm.ffmpegPath,
		BrowserChannel: s.BrowserChoice,
	}

	if err := qm.supervisor.SendRender(cmd); err != nil {
		qm.failJob(job, fmt.Sprintf("Failed sending render command to worker: %v", err))
		return
	}
}

func (qm *QueueManager) failJob(job *Job, errMsg string) {
	qm.mu.Lock()
	job.Status = StatusError
	job.Phase = "Failed"
	job.Error = errMsg
	job.Completed = time.Now().UnixMilli()
	delete(qm.activeJobs, job.ID)
	qm.saveLocked()
	qm.mu.Unlock()

	qm.notify(job)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func (qm *QueueManager) Close() {
	close(qm.stopChan)
}
