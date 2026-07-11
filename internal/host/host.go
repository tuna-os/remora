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
