package worker

import (
	"encoding/json"
	"testing"
)

func TestIPCCommandsSerialization(t *testing.T) {
	renderCmd := RenderCommand{
		Cmd:            "render",
		ID:             "job-123",
		HTML:           "C:\\reel.html",
		Audio:          "C:\\audio.mp3",
		Out:            "C:\\out.mp4",
		AudioDuration:  10.5,
		Width:          1080,
		Height:         1920,
		Dpr:            1.0,
		Fps:            30,
		Quality:        "high",
		CRF:            18,
		Preset:         "veryfast",
		ImageFormat:    "png",
		Tail:           1.0,
		FFmpegPath:     "C:\\ffmpeg.exe",
		BrowserChannel: "msedge",
	}

	data, err := json.Marshal(renderCmd)
	if err != nil {
		t.Fatalf("Failed to marshal RenderCommand: %v", err)
	}

	var parsed RenderCommand
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal RenderCommand: %v", err)
	}

	if parsed.Cmd != "render" || parsed.ID != "job-123" || parsed.Width != 1080 || parsed.Height != 1920 {
		t.Errorf("Parsed command mismatch: %+v", parsed)
	}

	cancelCmd := CancelCommand{Cmd: "cancel", ID: "job-123"}
	cData, err := json.Marshal(cancelCmd)
	if err != nil {
		t.Fatalf("Failed to marshal CancelCommand: %v", err)
	}

	var parsedCancel CancelCommand
	if err := json.Unmarshal(cData, &parsedCancel); err != nil {
		t.Fatalf("Failed to unmarshal CancelCommand: %v", err)
	}
	if parsedCancel.Cmd != "cancel" || parsedCancel.ID != "job-123" {
		t.Errorf("Parsed cancel command mismatch: %+v", parsedCancel)
	}
}

func TestIPCWorkerMessageParsing(t *testing.T) {
	// Progress message
	progressJSON := `{"type":"progress","id":"job-1","phase":"Rendering frames","frame":45,"frames":300,"pct":15,"eta":25}`
	var pMsg WorkerMessage
	if err := json.Unmarshal([]byte(progressJSON), &pMsg); err != nil {
		t.Fatalf("Failed to parse progress message: %v", err)
	}
	if pMsg.Type != "progress" || pMsg.ID != "job-1" || pMsg.Frame != 45 || pMsg.Frames != 300 || pMsg.Pct != 15 || pMsg.ETA != 25 {
		t.Errorf("Progress message contents unexpected: %+v", pMsg)
	}

	// Done message
	doneJSON := `{"type":"done","id":"job-1","result":{"width":1080,"height":1920,"fps":30,"duration":10.5,"frames":315,"size":5242880}}`
	var dMsg WorkerMessage
	if err := json.Unmarshal([]byte(doneJSON), &dMsg); err != nil {
		t.Fatalf("Failed to parse done message: %v", err)
	}
	if dMsg.Type != "done" || dMsg.Result == nil || dMsg.Result.Width != 1080 || dMsg.Result.Size != 5242880 {
		t.Errorf("Done message contents unexpected: %+v", dMsg)
	}

	// Error message
	errorJSON := `{"type":"error","id":"job-1","error":"loading timeout exceeded"}`
	var eMsg WorkerMessage
	if err := json.Unmarshal([]byte(errorJSON), &eMsg); err != nil {
		t.Fatalf("Failed to parse error message: %v", err)
	}
	if eMsg.Type != "error" || eMsg.Error != "loading timeout exceeded" {
		t.Errorf("Error message contents unexpected: %+v", eMsg)
	}
}

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Add("line 1")
	rb.Add("line 2")
	rb.Add("line 3")
	rb.Add("line 4")

	lines := rb.Lines()
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line 2" || lines[1] != "line 3" || lines[2] != "line 4" {
		t.Errorf("RingBuffer eviction order incorrect: %v", lines)
	}
}
