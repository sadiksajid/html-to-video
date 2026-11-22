package worker

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

type RenderCommand struct {
	Cmd               string  `json:"cmd"` // "render"
	ID                string  `json:"id"`
	HTML              string  `json:"html"`
	Audio             string  `json:"audio"`
	Out               string  `json:"out"`
	AudioDuration     float64 `json:"audioDuration"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	Dpr               float64 `json:"dpr"`
	Fps               int     `json:"fps"`
	Quality           string  `json:"quality"`
	CRF               int     `json:"crf"`
	Preset            string  `json:"preset"`
	ImageFormat       string  `json:"imageFormat"`
	Tail              float64 `json:"tail"`
	FFmpegPath        string  `json:"ffmpegPath"`
	BrowserChannel    string  `json:"browserChannel"`
	BrowserExecutable string  `json:"browserExecutable"`
}

type CancelCommand struct {
	Cmd string `json:"cmd"` // "cancel"
	ID  string `json:"id"`
}

type ExitCommand struct {
	Cmd string `json:"cmd"` // "exit"
}

type RenderResult struct {
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Fps      int     `json:"fps"`
	Duration float64 `json:"duration"`
	Frames   int     `json:"frames"`
	Size     int64   `json:"size"`
}

type WorkerMessage struct {
	Type    string        `json:"type"` // "progress", "done", "error", "log"
	ID      string        `json:"id"`
	Phase   string        `json:"phase,omitempty"`
	Frame   int           `json:"frame,omitempty"`
	Frames  int           `json:"frames,omitempty"`
	Pct     int           `json:"pct,omitempty"`
	ETA     int           `json:"eta,omitempty"`
	Error   string        `json:"error,omitempty"`
	Result  *RenderResult `json:"result,omitempty"`
	Level   string        `json:"level,omitempty"`
	Message string        `json:"message,omitempty"`
}

type RingBuffer struct {
	mu    sync.Mutex
	lines []string
	size  int
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		lines: make([]string, 0, size),
		size:  size,
	}
}

func (r *RingBuffer) Add(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.lines) >= r.size {
		r.lines = r.lines[1:]
	}
	r.lines = append(r.lines, line)
}

func (r *RingBuffer) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	res := make([]string, len(r.lines))
	copy(res, r.lines)
	return res
}

type Supervisor struct {
	mu         sync.Mutex
	nodePath   string
	scriptPath string
	workDir    string
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	recentLogs *RingBuffer
	msgChan    chan WorkerMessage
	doneChan   chan struct{}
	running    bool
}

func NewSupervisor(nodePath, scriptPath, workDir string) *Supervisor {
	return &Supervisor{
		nodePath:   nodePath,
		scriptPath: scriptPath,
		workDir:    workDir,
		recentLogs: NewRingBuffer(50),
		msgChan:    make(chan WorkerMessage, 100),
		doneChan:   make(chan struct{}),
	}
}

func (s *Supervisor) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	cmd := exec.Command(s.nodePath, s.scriptPath)
	cmd.Dir = s.workDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe error: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("stdout pipe error: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("stderr pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return fmt.Errorf("failed starting worker: %w", err)
	}

	s.cmd = cmd
	s.stdin = stdin
	s.running = true
	s.doneChan = make(chan struct{})

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		// Increase buffer size if necessary
		buf := make([]byte, 1024*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			var msg WorkerMessage
			if err := json.Unmarshal([]byte(line), &msg); err == nil {
				if msg.Type == "log" {
					s.recentLogs.Add(fmt.Sprintf("[%s] %s", msg.Level, msg.Message))
				}
				select {
				case s.msgChan <- msg:
				default:
				}
			} else {
				s.recentLogs.Add("stdout: " + line)
			}
		}

		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		close(s.doneChan)
	}()

	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			s.recentLogs.Add("stderr: " + line)
		}
	}()

	return nil
}

func (s *Supervisor) Messages() <-chan WorkerMessage {
	return s.msgChan
}

func (s *Supervisor) RecentLogs() []string {
	return s.recentLogs.Lines()
}

func (s *Supervisor) SendRender(cmd RenderCommand) error {
	cmd.Cmd = "render"
	return s.writeJSON(cmd)
}

func (s *Supervisor) SendCancel(jobID string) error {
	return s.writeJSON(CancelCommand{Cmd: "cancel", ID: jobID})
}

func (s *Supervisor) writeJSON(v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.stdin == nil {
		return errors.New("worker process is not running")
	}

	bytes, err := json.Marshal(v)
	if err != nil {
		return err
	}

	bytes = append(bytes, '\n')
	_, err = s.stdin.Write(bytes)
	return err
}

func (s *Supervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.cmd == nil {
		return
	}

	_ = s.writeJSON(ExitCommand{Cmd: "exit"})
	s.running = false

	done := make(chan struct{})
	go func() {
		_ = s.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
	}
}

func (s *Supervisor) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
