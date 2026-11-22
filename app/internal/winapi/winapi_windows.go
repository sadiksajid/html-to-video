//go:build windows

package winapi

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW      = kernel32.NewProc("CreateMutexW")
	procReleaseMutex      = kernel32.NewProc("ReleaseMutex")
	procCloseHandle       = kernel32.NewProc("CloseHandle")
	procGetDiskFreeSpaceW = kernel32.NewProc("GetDiskFreeSpaceExW")
)

const (
	ERROR_ALREADY_EXISTS = 183

	TBPF_NOPROGRESS    = 0x0
	TBPF_INDETERMINATE = 0x1
	TBPF_NORMAL        = 0x2
	TBPF_ERROR         = 0x4
	TBPF_PAUSED        = 0x8
)

type MutexLock struct {
	handle syscall.Handle
}

func AcquireSingleInstance(name string) (*MutexLock, bool, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, err
	}

	ret, _, callErr := procCreateMutexW.Call(
		0,
		1, // initial owner = true
		uintptr(unsafe.Pointer(namePtr)),
	)

	if ret == 0 {
		return nil, false, fmt.Errorf("CreateMutex failed: %v", callErr)
	}

	handle := syscall.Handle(ret)
	lastErr, ok := callErr.(syscall.Errno)
	if ok && lastErr == ERROR_ALREADY_EXISTS {
		return &MutexLock{handle: handle}, false, nil
	}

	return &MutexLock{handle: handle}, true, nil
}

func (m *MutexLock) Release() {
	if m != nil && m.handle != 0 {
		_, _, _ = procReleaseMutex.Call(uintptr(m.handle))
		_, _, _ = procCloseHandle.Call(uintptr(m.handle))
		m.handle = 0
	}
}

func CheckFreeDiskSpace(path string) (uint64, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	var freeBytesAvailable uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	r1, _, callErr := procGetDiskFreeSpaceW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	if r1 == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceEx failed: %v", callErr)
	}

	return freeBytesAvailable, nil
}

func OpenInExplorer(filePath string) error {
	cmd := exec.Command("explorer.exe", "/select,", filePath)
	return cmd.Start()
}

func OpenFile(filePath string) error {
	cmd := exec.Command("cmd.exe", "/c", "start", "", filePath)
	return cmd.Start()
}

func ShowToastNotification(title, message string) {
	// Send toast notification using PowerShell WinRT notification script
	psScript := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$template = @"
<toast>
    <visual>
        <binding template="ToastGeneric">
            <text>%s</text>
            <text>%s</text>
        </binding>
    </visual>
</toast>
"@
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Reel Studio").Show($toast)
`, title, message)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psScript)
	_ = cmd.Start()
}
