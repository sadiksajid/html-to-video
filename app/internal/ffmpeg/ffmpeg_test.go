package ffmpeg

import (
	"strconv"
	"strings"
	"testing"
)

func TestBuildFFmpegArgs(t *testing.T) {
	fps := 30
	crf := 18
	preset := "veryfast"
	width := 1080
	height := 1920
	totalDuration := 15.500
	audioPath := "C:\\audio.mp3"
	outPath := "C:\\out.mp4"

	args := BuildFFmpegArgs(fps, crf, preset, width, height, totalDuration, audioPath, outPath)

	argStr := strings.Join(args, " ")

	expectedTokens := []string{
		"-y",
		"-f image2pipe",
		"-framerate 30",
		"-i pipe:0",
		"-i C:\\audio.mp3",
		"-map 0:v:0",
		"-map 1:a:0",
		"scale=1080:1920:flags=lanczos,format=yuv420p",
		"-c:v libx264",
		"-preset veryfast",
		"-crf 18",
		"-profile:v high",
		"-level 4.2",
		"-r 30",
		"-g 60",
		"-bf 2",
		"-c:a aac",
		"-b:a 192k",
		"-ar 48000",
		"-ac 2",
		"apad=whole_dur=15.500",
		"-t 15.500",
		"-movflags +faststart",
		"C:\\out.mp4",
	}

	for _, token := range expectedTokens {
		if !strings.Contains(argStr, token) {
			t.Errorf("FFmpeg args missing expected token '%s'. Full args: %s", token, argStr)
		}
	}

	// Verify -g is exactly 2 * fps
	foundG := false
	for i, a := range args {
		if a == "-g" && i+1 < len(args) {
			foundG = true
			if args[i+1] != strconv.Itoa(fps*2) {
				t.Errorf("Expected GOP = 2 * fps (%d), got %s", fps*2, args[i+1])
			}
		}
	}
	if !foundG {
		t.Error("-g flag not found in ffmpeg args")
	}
}

func TestResolutionsAndQualityPresets(t *testing.T) {
	if res, ok := Resolutions["1080"]; !ok || res.Width != 1080 || res.Height != 1920 {
		t.Errorf("Resolution 1080 incorrect: %+v", res)
	}

	if q, ok := QualityPresets["high"]; !ok || q.CRF != 18 || q.Preset != "veryfast" || q.ImageFormat != "png" {
		t.Errorf("Quality preset 'high' incorrect: %+v", q)
	}

	if q, ok := QualityPresets["balanced"]; !ok || q.CRF != 21 || q.Preset != "veryfast" || q.ImageFormat != "jpeg" {
		t.Errorf("Quality preset 'balanced' incorrect: %+v", q)
	}
}
