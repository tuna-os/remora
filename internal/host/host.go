// Package host provides backward-compatible accessors to bootc system state.
// Deprecated: Use internal/bootc, internal/pm, internal/digest, internal/system packages directly.
package host

import (
	"tuna-os/remora/internal/bootc"
	"tuna-os/remora/internal/digest"
	"tuna-os/remora/internal/pm"
	"tuna-os/remora/internal/system"
)

// BootedImage returns the image ref the system is currently booted from.
// Deprecated: Use bootc.BootedImage instead.
func BootedImage() (string, error) {
	return bootc.BootedImage()
}

// DetectPM picks the package manager for the running system.
// Deprecated: Use pm.DetectPM instead.
func DetectPM() (string, error) {
	return pm.DetectPM()
}

// PinBase returns ref pinned to a digest.
// Deprecated: Use digest.PinBase instead.
func PinBase(ref string) (string, error) {
	return digest.PinBase(ref)
}

// SplitDigest splits "image@sha256:..." into its name and digest.
// Deprecated: Use digest.SplitDigest instead.
func SplitDigest(ref string) (name, digest string, ok bool) {
	return digest.SplitDigest(ref)
}

// LatestDigest asks the registry for the current digest behind ref.
// Deprecated: Use digest.LatestDigest instead.
func LatestDigest(ref string) (string, error) {
	return digest.LatestDigest(ref)
}

// LocalDigest returns the digest podman records for a local image ref.
// Deprecated: Use digest.LocalDigest instead.
func LocalDigest(ref string) (string, error) {
	return digest.LocalDigest(ref)
}

// UupdPresent reports whether uupd is installed.
// Deprecated: Use system.UupdPresent instead.
func UupdPresent() bool {
	return system.UupdPresent()
}

// Systemctl runs systemctl with args.
// Deprecated: Use system.Systemctl instead.
func Systemctl(args ...string) error {
	return system.Systemctl(args...)
}

// BootedImageDigest returns the image ref and digest the system is booted from.
// Deprecated: Use bootc.BootedImageDigest instead.
func BootedImageDigest() (ref, digest string, err error) {
	return bootc.BootedImageDigest()
}

// BootcSwitch rebases the system onto ref.
// Deprecated: Use system.BootcSwitch instead.
func BootcSwitch(ref string, apply bool, softReboot string) error {
	return system.BootcSwitch(ref, apply, softReboot)
}

// StagedOrBootedDigest returns the digest of the image the system will next boot.
// Deprecated: Use bootc.StagedOrBootedDigest instead.
func StagedOrBootedDigest() (string, error) {
	return bootc.StagedOrBootedDigest()
}

// DetectPMInImage probes the base image for its package manager.
// Deprecated: Use pm.DetectPMInImage instead.
func DetectPMInImage(image string) (string, error) {
	return pm.DetectPMInImage(image)
}
