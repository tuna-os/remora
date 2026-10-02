// Package bootc handles bootc system queries and switching.
package bootc

import (
	"os"
	"os/exec"

	"github.com/tuna-os/remora/internal/bootcstatus"
)

// BootedImage returns the image ref the system is currently booted from.
func BootedImage() (string, error) {
	return bootcstatus.BootedImage()
}

// BootedImageDigest returns the image ref and digest the system is booted from.
func BootedImageDigest() (ref, digest string, err error) {
	return bootcstatus.BootedImageDigest()
}

// StagedOrBootedDigest returns the digest of the image the system will next boot.
func StagedOrBootedDigest() (string, error) {
	return bootcstatus.StagedOrBootedDigest()
}

func buildSwitchArgs(ref string, apply bool, softReboot string) []string {
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

// Switch rebases the system onto ref. ref should carry a digest so that
// bootc sees a distinct target even when the tag is unchanged.
func Switch(ref string, apply bool, softReboot string) error {
	args := buildSwitchArgs(ref, apply, softReboot)
	cmd := exec.Command("bootc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
