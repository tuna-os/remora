// Package host inspects the running bootc system: booted image, package
// manager, and systemd interaction.
package host

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tuna-os/remora/internal/pm"
)

// BootedImage returns the image ref the system is currently booted from,
// via bootc status --json.
func BootedImage() (string, error) {
	out, err := exec.Command("bootc", "status", "--json").Output()
	if err != nil {
		return "", fmt.Errorf("bootc status (is this a bootc system?): %w", err)
	}
	return parseBootedImage(out)
}

func parseBootedImage(statusJSON []byte) (string, error) {
	var status struct {
		Status struct {
			Booted struct {
				Image struct {
					Image struct {
						Image string `json:"image"`
					} `json:"image"`
				} `json:"image"`
			} `json:"booted"`
		} `json:"status"`
	}
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		return "", fmt.Errorf("parsing bootc status: %w", err)
	}
	ref := status.Status.Booted.Image.Image.Image
	if ref == "" {
		return "", fmt.Errorf("bootc status reports no booted image")
	}
	return ref, nil
}

// DetectPM picks the package manager for the running system.
// Deprecated: use pm.DetectPM instead.
func DetectPM() (string, error) {
	return pm.DetectPM()
}

// DetectPMInImage probes the base image itself for its package manager.
// Deprecated: use pm.DetectInImage instead.
func DetectPMInImage(image string) (string, error) {
	return pm.DetectInImage(image)
}

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

// BootedImageDigest returns the image ref and digest the system is booted
// from. The digest is what makes a base pin reproducible: a tag moves, a
// digest does not.
func BootedImageDigest() (ref, digest string, err error) {
	out, err := exec.Command("bootc", "status", "--json").Output()
	if err != nil {
		return "", "", fmt.Errorf("bootc status (is this a bootc system?): %w", err)
	}
	return parseBootedImageDigest(out)
}

func parseBootedImageDigest(statusJSON []byte) (string, string, error) {
	var status struct {
		Status struct {
			Booted struct {
				Image struct {
					Image struct {
						Image     string `json:"image"`
						Transport string `json:"transport"`
					} `json:"image"`
					ImageDigest string `json:"imageDigest"`
				} `json:"image"`
			} `json:"booted"`
		} `json:"status"`
	}
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		return "", "", fmt.Errorf("parsing bootc status: %w", err)
	}
	img := status.Status.Booted.Image
	if img.Image.Image == "" {
		return "", "", fmt.Errorf("bootc status reports no booted image")
	}
	return img.Image.Image, img.ImageDigest, nil
}

// PinBase returns ref pinned to a digest. A ref that already carries a
// digest is returned unchanged; otherwise the digest is resolved from the
// registry with skopeo.
func PinBase(ref string) (string, error) {
	if _, _, ok := SplitDigest(ref); ok {
		return ref, nil
	}
	digest, err := LatestDigest(ref)
	if err != nil {
		return "", err
	}
	return ref + "@" + digest, nil
}

// SplitDigest splits "image@sha256:..." into its name and digest. ok is
// false when ref carries no digest.
func SplitDigest(ref string) (name, digest string, ok bool) {
	i := strings.LastIndex(ref, "@")
	if i < 0 {
		return ref, "", false
	}
	return ref[:i], ref[i+1:], true
}

// LatestDigest asks the registry for the current digest behind ref.
func LatestDigest(ref string) (string, error) {
	name, _, _ := SplitDigest(ref)
	out, err := exec.Command("skopeo", "inspect", "--format", "{{.Digest}}", "docker://"+name).Output()
	if err != nil {
		return "", fmt.Errorf("skopeo inspect %s: %w", name, err)
	}
	digest := strings.TrimSpace(string(out))
	if digest == "" {
		return "", fmt.Errorf("skopeo returned an empty digest for %s", name)
	}
	return digest, nil
}

// LocalDigest returns the digest podman records for a local image ref, or
// "" if the image is not present locally.
func LocalDigest(ref string) (string, error) {
	out, err := exec.Command("podman", "inspect", "--format", "{{.Digest}}", ref).Output()
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
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

// StagedOrBootedDigest returns the digest of the image the system will next
// boot: the staged deployment if there is one, otherwise the booted one.
// remora compares a freshly built image against this to decide whether a
// switch would change anything.
func StagedOrBootedDigest() (string, error) {
	out, err := exec.Command("bootc", "status", "--json").Output()
	if err != nil {
		return "", fmt.Errorf("bootc status: %w", err)
	}
	return parseStagedOrBootedDigest(out)
}

func parseStagedOrBootedDigest(statusJSON []byte) (string, error) {
	var status struct {
		Status struct {
			Staged *struct {
				Image struct {
					ImageDigest string `json:"imageDigest"`
				} `json:"image"`
			} `json:"staged"`
			Booted *struct {
				Image struct {
					ImageDigest string `json:"imageDigest"`
				} `json:"image"`
			} `json:"booted"`
		} `json:"status"`
	}
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		return "", fmt.Errorf("parsing bootc status: %w", err)
	}
	if s := status.Status.Staged; s != nil && s.Image.ImageDigest != "" {
		return s.Image.ImageDigest, nil
	}
	if b := status.Status.Booted; b != nil {
		return b.Image.ImageDigest, nil
	}
	return "", nil
}
