package ffmpeg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type QualityPreset struct {
	CRF         int    `json:"crf"`
	Preset      string `json:"preset"`
	ImageFormat string `json:"imageFormat"`
}

var QualityPresets = map[string]QualityPreset{
	"best":     {CRF: 16, Preset: "slow", ImageFormat: "png"},
	"high":     {CRF: 18, Preset: "veryfast", ImageFormat: "png"},
	"balanced": {CRF: 21, Preset: "veryfast", ImageFormat: "jpeg"},
	"small":    {CRF: 25, Preset: "veryfast", ImageFormat: "jpeg"},
}

type Resolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

var Resolutions = map[string]Resolution{
	"720":  {Width: 720, Height: 1280},
	"1080": {Width: 1080, Height: 1920},
	"1440": {Width: 1440, Height: 2560},
}

func findWorkingFFprobe(path string) string {
	if path != "" {
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			return path
		}
	}
	if p, err := exec.LookPath("ffprobe"); err == nil {
		return p
	}
	if runtime.GOOS == "darwin" {
		for _, cand := range []string{
			"/opt/homebrew/bin/ffprobe",
			"/usr/local/bin/ffprobe",
			"/opt/local/bin/ffprobe",
			"/usr/bin/ffprobe",
		} {
			if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
				return cand
			}
		}
	}
	return path
}

func ProbeAudioDuration(ffprobePath, audioPath string) (float64, error) {
	resolvedPath := findWorkingFFprobe(ffprobePath)
	if fi, err := os.Stat(resolvedPath); err != nil || fi.IsDir() {
		hint := "Please ensure FFmpeg & FFprobe are installed and available in PATH."
		if runtime.GOOS == "darwin" {
			hint = "Install FFmpeg on your Mac by running: brew install ffmpeg"
		}
		return 0, fmt.Errorf("ffprobe not found at '%s' (%s)", ffprobePath, hint)
	}

	cmd := exec.Command(resolvedPath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		audioPath,
	)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe failed: %v (stderr: %s)", err, errOut.String())
	}

	str := strings.TrimSpace(out.String())
	val, err := strconv.ParseFloat(str, 64)
	if err != nil || val <= 0 {
		return 0, fmt.Errorf("invalid audio duration '%s': %w", str, err)
	}

	return val, nil
}

func BuildFFmpegArgs(fps int, crf int, preset string, width int, height int, totalDuration float64, audioPath string, outPath string) []string {
	return []string{
		"-y",
		"-f", "image2pipe",
		"-framerate", strconv.Itoa(fps),
		"-i", "pipe:0",
		"-i", audioPath,
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-vf", fmt.Sprintf("scale=%d:%d:flags=lanczos,format=yuv420p", width, height),
		"-c:v", "libx264",
		"-preset", preset,
		"-crf", strconv.Itoa(crf),
		"-profile:v", "high",
		"-level", "4.2",
		"-r", strconv.Itoa(fps),
		"-g", strconv.Itoa(fps * 2),
		"-bf", "2",
		"-c:a", "aac",
		"-b:a", "192k",
		"-ar", "48000",
		"-ac", "2",
		"-af", fmt.Sprintf("apad=whole_dur=%.3f", totalDuration),
		"-t", fmt.Sprintf("%.3f", totalDuration),
		"-movflags", "+faststart",
		outPath,
	}
}
