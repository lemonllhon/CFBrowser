//go:build !windows
// +build !windows

package proxy

import "os/exec"

func hideWindow(cmd *exec.Cmd) {
	// do nothing on non-windows platforms
}

func stopProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
