//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	processSynchronize = 0x00100000
)

type targetProcess struct {
	pid uint32
	exe string
}

func main() {
	os.Exit(run())
}

func run() int {
	installDir := flag.String("install-dir", "", "installed application directory")
	excludePath := flag.String("exclude", "", "executable path to exclude")
	timeoutSeconds := flag.Int("timeout", 10, "cleanup timeout in seconds")
	flag.Parse()

	root, err := normalizeDirectory(*installDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	exclude := normalizeFile(*excludePath)
	deadline := time.Now().Add(time.Duration(max(1, *timeoutSeconds)) * time.Second)

	for {
		targets, err := findTargetProcesses(root, exclude)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
		if len(targets) == 0 {
			return 0
		}
		for _, target := range targets {
			if err := terminateProcess(target.pid); err != nil {
				fmt.Fprintf(os.Stderr, "terminate %s#%d: %v\n", target.exe, target.pid, err)
			}
		}
		if time.Now().After(deadline) {
			for _, target := range targets {
				fmt.Fprintf(os.Stderr, "still running: %s#%d\n", target.exe, target.pid)
			}
			return 1
		}
		time.Sleep(400 * time.Millisecond)
	}
}

func normalizeDirectory(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("install-dir is required")
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	clean := strings.TrimRight(filepath.Clean(abs), `\/`)
	if clean == "" {
		return "", fmt.Errorf("install-dir is invalid")
	}
	return strings.ToLower(clean) + string(os.PathSeparator), nil
}

func normalizeFile(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return strings.ToLower(filepath.Clean(value))
	}
	return strings.ToLower(filepath.Clean(abs))
}

func findTargetProcesses(root string, exclude string) ([]targetProcess, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)

	currentPID := uint32(os.Getpid())
	entry := windows.ProcessEntry32{Size: uint32(unsafeSizeofProcessEntry32())}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		if err == windows.ERROR_NO_MORE_FILES {
			return nil, nil
		}
		return nil, err
	}

	var targets []targetProcess
	for {
		pid := entry.ProcessID
		if pid != 0 && pid != currentPID {
			if exe, ok := processImagePath(pid); ok {
				normalized := strings.ToLower(filepath.Clean(exe))
				if normalized != exclude && isUnderRoot(normalized, root) {
					targets = append(targets, targetProcess{pid: pid, exe: exe})
				}
			}
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				break
			}
			return nil, err
		}
	}
	return targets, nil
}

func isUnderRoot(normalizedPath string, normalizedRoot string) bool {
	if normalizedPath == "" || normalizedRoot == "" {
		return false
	}
	return strings.HasPrefix(normalizedPath+string(os.PathSeparator), normalizedRoot)
}

func processImagePath(pid uint32) (string, bool) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", false
	}
	defer windows.CloseHandle(handle)

	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil || size == 0 {
		return "", false
	}
	return windows.UTF16ToString(buf[:size]), true
}

func terminateProcess(pid uint32) error {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|processSynchronize, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	if err := windows.TerminateProcess(handle, 1); err != nil {
		return err
	}
	_, _ = windows.WaitForSingleObject(handle, 1000)
	return nil
}

func unsafeSizeofProcessEntry32() uintptr {
	var entry windows.ProcessEntry32
	return uintptr(unsafe.Sizeof(entry))
}
