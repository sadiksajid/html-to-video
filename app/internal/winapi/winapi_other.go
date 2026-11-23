//go:build !windows

package winapi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

type MutexLock struct {
	file *os.File
}

func AcquireSingleInstance(name string) (*MutexLock, bool, error) {
	lockPath := filepath.Join(os.TempDir(), name+".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, false, err
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		return nil, false, nil // already locked
	}
	return &MutexLock{file: f}, true, nil
}

func (m *MutexLock) Release() {
	if m != nil && m.file != nil {
		_ = syscall.Flock(int(m.file.Fd()), syscall.LOCK_UN)
		_ = m.file.Close()
	}
}

func CheckFreeDiskSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 100 * 1024 * 1024 * 1024, nil
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

func OpenInExplorer(filePath string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-R", filePath).Start()
	}
	dir := filepath.Dir(filePath)
	return exec.Command("xdg-open", dir).Start()
}

func OpenFile(filePath string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", filePath).Start()
	}
	return exec.Command("xdg-open", filePath).Start()
}

func ShowToastNotification(title, message string) {
	if runtime.GOOS == "darwin" {
		script := fmt.Sprintf(`display notification %q with title %q`, message, title)
		_ = exec.Command("osascript", "-e", script).Start()
	} else {
		_ = exec.Command("notify-send", title, message).Start()
	}
}
