//go:build !windows

package winapi

type MutexLock struct{}

func AcquireSingleInstance(name string) (*MutexLock, bool, error) {
	return &MutexLock{}, true, nil
}

func (m *MutexLock) Release() {}

func CheckFreeDiskSpace(path string) (uint64, error) {
	return 100 * 1024 * 1024 * 1024, nil
}

func OpenInExplorer(filePath string) error {
	return nil
}

func OpenFile(filePath string) error {
	return nil
}

func ShowToastNotification(title, message string) {}
