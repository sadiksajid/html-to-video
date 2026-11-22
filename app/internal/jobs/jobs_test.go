package jobs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestQueueOrderingAndCancel(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "jobs.json")

	qm := &QueueManager{
		storePath: storePath,
		jobsDir:   filepath.Join(tmpDir, "work"),
		jobs:      make(map[string]*Job),
		queue:     make([]string, 0),
	}

	j1 := &Job{ID: "j1", Name: "job1.mp4", Status: StatusQueued}
	j2 := &Job{ID: "j2", Name: "job2.mp4", Status: StatusQueued}
	j3 := &Job{ID: "j3", Name: "job3.mp4", Status: StatusQueued}

	qm.jobs["j1"] = j1
	qm.jobs["j2"] = j2
	qm.jobs["j3"] = j3
	qm.queue = []string{"j1", "j2", "j3"}

	// Verify FIFO order
	if qm.queue[0] != "j1" || qm.queue[1] != "j2" || qm.queue[2] != "j3" {
		t.Fatalf("Queue ordering unexpected: %v", qm.queue)
	}

	// Cancel j2
	if err := qm.CancelJob("j2"); err != nil {
		t.Fatalf("Failed to cancel j2: %v", err)
	}

	if j2.Status != StatusCancelled {
		t.Errorf("Expected j2 status cancelled, got %s", j2.Status)
	}

	// Queue should now contain only j1 and j3
	if len(qm.queue) != 2 || qm.queue[0] != "j1" || qm.queue[1] != "j3" {
		t.Errorf("Queue after cancel unexpected: %v", qm.queue)
	}
}

func TestQueueRetry(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "jobs.json")

	qm := &QueueManager{
		storePath: storePath,
		jobsDir:   filepath.Join(tmpDir, "work"),
		jobs:      make(map[string]*Job),
		queue:     make([]string, 0),
	}

	j1 := &Job{ID: "j1", Name: "job1.mp4", Status: StatusError, Error: "timeout"}
	qm.jobs["j1"] = j1

	if err := qm.RetryJob("j1"); err != nil {
		t.Fatalf("RetryJob failed: %v", err)
	}

	if j1.Status != StatusQueued {
		t.Errorf("Expected status queued after retry, got %s", j1.Status)
	}
	if j1.Error != "" {
		t.Errorf("Expected empty error after retry, got %s", j1.Error)
	}
	if len(qm.queue) != 1 || qm.queue[0] != "j1" {
		t.Errorf("Queue should contain j1 after retry: %v", qm.queue)
	}
}

func TestQueuePersistenceAndInterruptedRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "jobs.json")

	// Pre-populate jobs file with running and queued jobs
	initialJobs := []*Job{
		{ID: "job-done", Name: "done.mp4", Status: StatusDone},
		{ID: "job-running", Name: "running.mp4", Status: StatusRunning},
		{ID: "job-queued", Name: "queued.mp4", Status: StatusQueued},
	}

	bytes, err := json.Marshal(initialJobs)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if err := os.WriteFile(storePath, bytes, 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	qm := &QueueManager{
		storePath: storePath,
		jobsDir:   filepath.Join(tmpDir, "work"),
		jobs:      make(map[string]*Job),
		queue:     make([]string, 0),
	}

	qm.load()

	if qm.jobs["job-done"].Status != StatusDone {
		t.Errorf("job-done should remain done, got %s", qm.jobs["job-done"].Status)
	}

	if qm.jobs["job-running"].Status != StatusInterrupted {
		t.Errorf("job-running should be marked interrupted on recovery, got %s", qm.jobs["job-running"].Status)
	}

	if qm.jobs["job-queued"].Status != StatusInterrupted {
		t.Errorf("job-queued should be marked interrupted on recovery, got %s", qm.jobs["job-queued"].Status)
	}
}

func TestFormatOutputName(t *testing.T) {
	template := "{reel}-{res}p{fps}.mp4"
	res := FormatOutputName(template, "my-reel", "1080", 30)
	if res != "my-reel-1080p30.mp4" {
		t.Errorf("Expected 'my-reel-1080p30.mp4', got '%s'", res)
	}

	// Without .mp4 extension in template
	res2 := FormatOutputName("{reel}_{res}", "demo", "720", 60)
	if res2 != "demo_720.mp4" {
		t.Errorf("Expected 'demo_720.mp4', got '%s'", res2)
	}
}
