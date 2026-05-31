//go:build !windows

package backend

import "os/exec"

func launchOfficialUpdateInstaller(installerPath string) error {
	return exec.Command(installerPath).Start()
}
