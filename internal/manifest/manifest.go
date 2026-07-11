// Package manifest defines remora's user-facing configuration: what to layer
// on top of the booted bootc image. It is deliberately small — packages, an
// overlay, scripts, and a shell escape hatch cover everything the generated
// Containerfile needs without becoming a build system.
package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultDir is where remora keeps its manifest and build context.
const DefaultDir = "/etc/remora"

// Manifest is remora.yaml.
type Manifest struct {
	// Base is the image to layer on. Empty means "follow the booted image"
	// (resolved from bootc status at generation time).
	Base string `yaml:"base,omitempty"`
	// PackageManager overrides detection: dnf, zypper, pacman, apt.
	// Empty means detect from the base image / host os-release.
	PackageManager string `yaml:"package_manager,omitempty"`
	// Packages are installed with the base's package manager.
	Packages []string `yaml:"packages"`
	// ExtraRun lines are executed verbatim (shell) before package
	// installation — the escape hatch for extra repos, keys, or exotic
	// builders like BuildStream. Each entry is one shell command.
	ExtraRun []string `yaml:"extra_run,omitempty"`
	// Image is the local tag the factory builds. Defaults to
	// localhost/remora:latest.
	Image string `yaml:"image,omitempty"`
	// Schedule is the systemd OnCalendar expression for automatic rebuilds.
	// Defaults to daily at 04:00.
	Schedule string `yaml:"schedule,omitempty"`
}

// Path returns the manifest path inside dir.
func Path(dir string) string { return filepath.Join(dir, "remora.yaml") }

// Load reads and validates the manifest in dir.
func Load(dir string) (*Manifest, error) {
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", Path(dir), err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Save writes the manifest to dir, creating it if needed.
func (m *Manifest) Save(dir string) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	header := "# remora manifest — packages and customizations layered on the booted image.\n" +
		"# Rebuilds happen locally (podman) and apply via bootc switch; see remora(1).\n"
	return os.WriteFile(Path(dir), append([]byte(header), data...), 0o644)
}

// ImageTag returns the local image tag the factory builds.
func (m *Manifest) ImageTag() string {
	if m.Image != "" {
		return m.Image
	}
	return "localhost/remora:latest"
}

// OnCalendar returns the rebuild schedule.
func (m *Manifest) OnCalendar() string {
	if m.Schedule != "" {
		return m.Schedule
	}
	return "*-*-* 04:00:00"
}

// AddPackages appends packages, deduplicating; returns the ones actually added.
func (m *Manifest) AddPackages(pkgs []string) []string {
	var added []string
	for _, p := range pkgs {
		if p == "" || slices.Contains(m.Packages, p) {
			continue
		}
		m.Packages = append(m.Packages, p)
		added = append(added, p)
	}
	slices.Sort(m.Packages)
	return added
}

// RemovePackages removes packages; returns the ones actually removed.
func (m *Manifest) RemovePackages(pkgs []string) []string {
	var removed []string
	for _, p := range pkgs {
		if i := slices.Index(m.Packages, p); i >= 0 {
			m.Packages = slices.Delete(m.Packages, i, i+1)
			removed = append(removed, p)
		}
	}
	return removed
}

var validPMs = []string{"", "dnf", "zypper", "pacman", "apt"}

// Validate rejects manifests the generator can't act on. Package names are
// checked against shell metacharacters because they are interpolated into
// the generated Containerfile RUN line.
func (m *Manifest) Validate() error {
	if !slices.Contains(validPMs, m.PackageManager) {
		return fmt.Errorf("package_manager must be one of dnf, zypper, pacman, apt (or empty for auto), got %q", m.PackageManager)
	}
	for _, p := range m.Packages {
		if strings.ContainsAny(p, " \t\n'\"`$&|;<>(){}\\") {
			return fmt.Errorf("package name %q contains shell metacharacters", p)
		}
	}
	if m.Image != "" && strings.ContainsAny(m.Image, " \t\n'\"") {
		return fmt.Errorf("image tag %q contains whitespace or quotes", m.Image)
	}
	return nil
}
