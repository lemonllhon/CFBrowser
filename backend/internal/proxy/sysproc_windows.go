//go:build windows
// +build windows

package proxy

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func stopProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid
	if pid <= 0 {
		return nil
	}

	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	hideWindow(killCmd)
	err := killCmd.Run()
	if err == nil || waitProcessExit(pid, 2*time.Second) {
		return nil
	}
	return err
}

func waitProcessExit(pid int, timeout time.Duration) bool {
	if pid <= 0 {
		return true
	}

	deadline := time.Now().Add(timeout)
	for {
		alive, err := isProcessAlive(pid)
		if err == nil && !alive {
			return true
		}
		if time.Now().After(deadline) {
			alive, err = isProcessAlive(pid)
			return err == nil && !alive
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func isProcessAlive(pid int) (bool, error) {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	hideWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	text := strings.TrimSpace(string(output))
	if text == "" || strings.Contains(text, "No tasks are running") || strings.Contains(text, "没有运行的任务") {
		return false, nil
	}
	return strings.Contains(text, fmt.Sprintf(`"%d"`, pid)) || strings.Contains(text, fmt.Sprintf(",%d,", pid)), nil
}
