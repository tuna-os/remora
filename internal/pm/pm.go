// Package pm detects the package manager for a running system or container image.
package pm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DetectPM picks the package manager for the running system. The booted
// image is the base image, so inspecting the host is inspecting the base.
func DetectPM() (string, error) {
	return Detect(osReleaseID(), binaryExists)
}

// Detect is the core PM detection logic, parameterized for testing.
// It prefers an actual package-manager binary over os-release heuristics;
// os-release breaks ties for derivatives.
func Detect(osID string, exists func(string) bool) (string, error) {
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

// DetectInImage probes the base image itself for its package manager,
// rather than inferring it from the host. The two only agree when the base
// image is the booted image; an explicit `base:` in the manifest can be any
// distribution at all, so asking the host would happily generate a
// Containerfile that runs dnf inside an apt image.
func DetectInImage(image string) (string, error) {
	out, err := exec.Command("podman", "run", "--rm", "--entrypoint", "", image, "sh", "-c", imageProbe).Output()
	if err != nil {
		return "", fmt.Errorf("probing %s for its package manager: %w", image, err)
	}
	return parseImageProbe(out)
}

// imageProbe is the script run inside the base image. It reports the first
// package-manager binary on PATH and the os-release ID.
const imageProbe = `. /etc/os-release 2>/dev/null || true
for b in dnf5 dnf zypper pacman apt-get emerge apk; do
  command -v "$b" >/dev/null 2>&1 && { echo "bin=$b"; break; }
done
echo "id=${ID:-}"`

// parseImageProbe turns imageProbe's output into a package-manager name.
// Split out from DetectInImage so the classification is testable without
// a container runtime.
func parseImageProbe(out []byte) (string, error) {
	var bin, osID string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "bin="); ok {
			bin = after
		}
		if after, ok := strings.CutPrefix(line, "id="); ok {
			osID = strings.Trim(after, `"`)
		}
	}
	return Detect(osID, func(name string) bool { return name == bin })
}

// osReleaseID returns the ID field from /etc/os-release.
func osReleaseID() string {
	return osReleaseIDFromPath("/etc/os-release")
}

// osReleaseIDFromPath reads the ID field from the os-release file at path.
func osReleaseIDFromPath(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return parseOSReleaseID(data)
}

// parseOSReleaseID extracts the ID field from os-release file contents.
func parseOSReleaseID(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if after, ok := strings.CutPrefix(line, "ID="); ok {
			return strings.Trim(after, `"`)
		}
	}
	return ""
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
