package generate

import (
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
)

func TestContainerfilePerPM(t *testing.T) {
	m := &manifest.Manifest{Packages: []string{"htop", "vim"}}
	cases := map[string][]string{
		"dnf":     {"dnf -y install", "/var/cache/libdnf5"},
		"zypper":  {"zypper --non-interactive install", "/var/cache/zypp"},
		"pacman":  {"pacman -Sy --noconfirm --needed", "/var/cache/pacman/pkg"},
		"apt":     {"apt-get install -y", "/var/cache/apt"},
		"portage": {"emerge --verbose", "/var/cache/distfiles"},
		"apk":     {"apk add --no-interactive"},
	}
	for pm, wants := range cases {
		out, err := Containerfile(m, "ghcr.io/tuna-os/yellowfin:gnome", pm, "")
		if err != nil {
			t.Fatalf("%s: %v", pm, err)
		}
		for _, w := range append(wants,
			"FROM ghcr.io/tuna-os/yellowfin:gnome",
			"htop",
			"vim",
			"bootc container lint",
			"cp -avf /ctx/system_files/. /",
			"/ctx/build_files/*.sh",
		) {
			if !strings.Contains(out, w) {
				t.Errorf("%s: generated Containerfile missing %q", pm, w)
			}
		}
	}
}

func TestContainerfileUnsupportedPM(t *testing.T) {
	if _, err := Containerfile(&manifest.Manifest{}, "x", "nix", ""); err == nil {
		t.Fatal("expected error for unsupported package manager")
	}
}

func TestContainerfileExtraRunOrdering(t *testing.T) {
	m := &manifest.Manifest{
		Packages: []string{"tailscale"},
		ExtraRun: []string{"dnf config-manager addrepo --from-repofile=https://pkgs.tailscale.com/stable/fedora/tailscale.repo"},
	}
	out, err := Containerfile(m, "quay.io/fedora/fedora-bootc:latest", "dnf", "")
	if err != nil {
		t.Fatal(err)
	}
	repoIdx := strings.Index(out, "addrepo")
	pkgIdx := strings.Index(out, "dnf -y install")
	if repoIdx < 0 || pkgIdx < 0 || repoIdx > pkgIdx {
		t.Fatal("extra_run must render before package installation")
	}
}

func TestContainerfileNoPackages(t *testing.T) {
	out, err := Containerfile(&manifest.Manifest{}, "base:latest", "apt", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "apt-get install") {
		t.Fatal("no install command expected when the manifest has no packages")
	}
	if !strings.Contains(out, "bootc container lint") {
		t.Fatal("lint step must always be present")
	}
}

// The package transaction must sit in its own RUN layer, separate from the
// overlay and the build scripts, so that editing a build script does not
// invalidate the cached package install.
func TestContainerfileLayerSplit(t *testing.T) {
	m := &manifest.Manifest{
		Packages: []string{"htop"},
		ExtraRun: []string{"echo repo-setup"},
	}
	cf, err := Containerfile(m, "base@sha256:abc", "dnf", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(cf, "<<'REMORA_EOF'"); got != 3 {
		t.Fatalf("want 3 heredoc RUN layers (overlay, packages, scripts), got %d:\n%s", got, cf)
	}
	overlay := strings.Index(cf, "cp -avf /ctx/system_files/. /")
	pkgs := strings.Index(cf, "dnf -y install")
	scripts := strings.Index(cf, "for script in /ctx/build_files/*.sh")
	if !(overlay < pkgs && pkgs < scripts) {
		t.Fatalf("layer order must be overlay < packages < scripts, got %d/%d/%d", overlay, pkgs, scripts)
	}
	// extra_run sets up repositories, so it must precede the install.
	if er := strings.Index(cf, "echo repo-setup"); er > pkgs {
		t.Error("extra_run must run before the package transaction")
	}
}

// Identical inputs must render an identical Containerfile — the first half
// of "identical inputs produce an identical image digest".
func TestContainerfileDeterministic(t *testing.T) {
	m := &manifest.Manifest{Packages: []string{"htop", "vim"}}
	for _, pm := range SupportedPMs() {
		a, err := Containerfile(m, "base@sha256:abc", pm, "")
		if err != nil {
			t.Fatal(err)
		}
		b, err := Containerfile(m, "base@sha256:abc", pm, "")
		if err != nil {
			t.Fatal(err)
		}
		if a != b {
			t.Errorf("%s: Containerfile not deterministic", pm)
		}
	}
}

// Logs vary between otherwise-identical builds, so every package manager
// must scrub them; the rpmdb (the installed-package record) must survive.
func TestContainerfileScrubsVaryingState(t *testing.T) {
	m := &manifest.Manifest{Packages: []string{"htop"}}
	for _, pm := range SupportedPMs() {
		cf, err := Containerfile(m, "base@sha256:abc", pm, "")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(cf, "/var/log/*") {
			t.Errorf("%s: does not scrub /var/log", pm)
		}
		if strings.Contains(cf, "/usr/lib/sysimage/rpm") {
			t.Errorf("%s: must not scrub the rpmdb", pm)
		}
	}
}

// With no packages the transaction layer is omitted entirely.
func TestContainerfileNoPackagesOmitsLayer(t *testing.T) {
	cf, err := Containerfile(&manifest.Manifest{}, "base@sha256:abc", "dnf", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cf, "dnf -y install") {
		t.Error("empty manifest must not emit an install command")
	}
	if got := strings.Count(cf, "<<'REMORA_EOF'"); got != 2 {
		t.Errorf("want 2 layers without packages, got %d", got)
	}
}

// With a lockfile the package layer installs from it and never from the spec
// list — installing both would double-resolve and defeat the cache key.
func TestContainerfileLockfileInstall(t *testing.T) {
	m := &manifest.Manifest{Packages: []string{"htop", "vim"}}
	cf, err := Containerfile(m, "base@sha256:abc", "dnf", "remora.lock.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cf, "COPY remora.lock.yaml /run/remora/remora.lock.yaml") {
		t.Error("lockfile must be COPYied into the build so podman keys the layer on it")
	}
	if !strings.Contains(cf, "dnf5 manifest install --assumeyes --manifest /run/remora/remora.lock.yaml") {
		t.Error("missing lockfile install command")
	}
	if strings.Contains(cf, "dnf -y install") {
		t.Error("must not also install from the spec list when a lockfile is used")
	}
	// The COPY must precede the RUN that consumes it.
	if strings.Index(cf, "COPY remora.lock.yaml") > strings.Index(cf, "manifest install") {
		t.Error("COPY must come before the install that reads it")
	}
	// The lockfile is build input, not image content.
	if !strings.Contains(cf, "rm -rf /run/remora ") {
		t.Error("lockfile must be scrubbed from the image")
	}
}

// A package manager with no resolver must ignore the lockfile argument
// entirely rather than emitting a COPY for a file that will not exist.
func TestContainerfileLockfileIgnoredWithoutResolver(t *testing.T) {
	m := &manifest.Manifest{Packages: []string{"htop"}}
	for _, pmName := range []string{"apt", "zypper", "pacman", "portage", "apk"} {
		cf, err := Containerfile(m, "base@sha256:abc", pmName, "remora.lock.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(cf, "remora.lock.yaml") {
			t.Errorf("%s has no resolver but referenced a lockfile", pmName)
		}
		if !strings.Contains(cf, "htop") {
			t.Errorf("%s must fall back to installing from the spec list", pmName)
		}
	}
}

// An empty package list emits no transaction layer at all, lockfile or not.
func TestContainerfileLockfileNoPackages(t *testing.T) {
	cf, err := Containerfile(&manifest.Manifest{}, "base@sha256:abc", "dnf", "remora.lock.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cf, "remora.lock.yaml") {
		t.Error("no packages means no lockfile COPY")
	}
}

// Determinism has to hold on the lockfile path too — it is the whole point.
func TestContainerfileLockfileDeterministic(t *testing.T) {
	m := &manifest.Manifest{Packages: []string{"htop", "vim"}}
	a, _ := Containerfile(m, "base@sha256:abc", "dnf", "remora.lock.yaml")
	b, _ := Containerfile(m, "base@sha256:abc", "dnf", "remora.lock.yaml")
	if a != b {
		t.Error("lockfile rendering is not deterministic")
	}
}
