// Package host inspects the running bootc system: booted image, package
// manager, and systemd interaction.
package host

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

// DetectPM picks the package manager for the running system. The booted
// image is the base image, so inspecting the host is inspecting the base.
func DetectPM() (string, error) {
	return detectPM(osReleaseID(), binaryExists)
}

// detectPM prefers an actual package-manager binary over os-release
// heuristics; os-release breaks ties for derivatives.
func detectPM(osID string, exists func(string) bool) (string, error) {
	switch {
	case exists("dnf5") || exists("dnf"):
		return "dnf", nil
	case exists("zypper"):
		return "zypper", nil
	case exists("pacman"):
		return "pacman", nil
	case exists("apt-get"):
		return "apt", nil
	case exists("emerge"):
		return "portage", nil
	case exists("apk"):
		return "apk", nil
	}
	switch osID {
	case "fedora", "rhel", "centos", "almalinux", "rocky":
		return "dnf", nil
	case "opensuse", "opensuse-tumbleweed", "opensuse-leap", "sles":
		return "zypper", nil
	case "arch", "cachyos", "manjaro":
		return "pacman", nil
	case "debian", "ubuntu":
		return "apt", nil
	case "gentoo":
		return "portage", nil
	case "alpine", "postmarketos":
		return "apk", nil
	}
	return "", fmt.Errorf("could not detect package manager (os-release ID=%q); set package_manager in remora.yaml", osID)
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func osReleaseID() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	return parseOSReleaseID(data)
}

func parseOSReleaseID(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if after, ok := strings.CutPrefix(line, "ID="); ok {
			return strings.Trim(after, `"`)
		}
	}
	return ""
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

// DetectPMInImage probes the base image itself for its package manager,
// rather than inferring it from the host. The two only agree when the base
// image is the booted image; an explicit `base:` in the manifest can be any
// distribution at all.
func DetectPMInImage(image string) (string, error) {
	const probe = `. /etc/os-release 2>/dev/null || true
for b in dnf5 dnf zypper pacman apt-get emerge apk; do
  command -v "$b" >/dev/null 2>&1 && { echo "bin=$b"; break; }
done
echo "id=${ID:-}"`
	out, err := exec.Command("podman", "run", "--rm", "--entrypoint", "", image, "sh", "-c", probe).Output()
	if err != nil {
		return "", fmt.Errorf("probing %s for its package manager: %w", image, err)
	}
	var bin, osID string
	for _, line := range strings.Split(string(out), "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "bin="); ok {
			bin = after
		}
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "id="); ok {
			osID = strings.Trim(after, `"`)
		}
	}
	return detectPM(osID, func(name string) bool { return name == bin })
}

// BootcSwitch rebases the system onto ref. ref should carry a digest so that
// bootc sees a distinct target even when the tag is unchanged.
func BootcSwitch(ref string, apply bool, softReboot string) error {
	args := []string{"switch", "--transport=containers-storage"}
	if apply {
		args = append(args, "--apply")
	}
	if softReboot != "" {
		args = append(args, "--soft-reboot="+softReboot)
	}
	args = append(args, ref)
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
