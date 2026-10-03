// Package system handles privileged system operations (bootc, systemd).
package system

import (
	"os"
	"os/exec"
)

// UupdPresent reports whether uupd (Universal Blue's updater) is installed.
func UupdPresent() bool {
	return exec.Command("systemctl", "cat", "--", "uupd.service").Run() == nil
}

// Systemctl runs systemctl with args, streaming output.
func Systemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildBootcSwitchArgs(ref string, apply bool, softReboot string) []string {
	args := []string{"switch", "--transport=containers-storage"}
	if apply {
		args = append(args, "--apply")
	}
	if softReboot != "" {
		args = append(args, "--soft-reboot="+softReboot)
	}
	args = append(args, ref)
	return args
}

// BootcSwitch rebases the system onto ref. ref should carry a digest so that
// bootc sees a distinct target even when the tag is unchanged.
func BootcSwitch(ref string, apply bool, softReboot string) error {
	args := buildBootcSwitchArgs(ref, apply, softReboot)
	cmd := exec.Command("bootc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
