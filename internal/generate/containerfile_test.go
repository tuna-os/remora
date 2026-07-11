package generate

import (
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
)

func TestContainerfilePerPM(t *testing.T) {
	m := &manifest.Manifest{Packages: []string{"htop", "vim"}}
	cases := map[string][]string{
		"dnf":    {"dnf -y install", "/var/cache/libdnf5"},
		"zypper": {"zypper --non-interactive install", "/var/cache/zypp"},
		"pacman": {"pacman -Sy --noconfirm --needed", "/var/cache/pacman/pkg"},
		"apt":    {"apt-get install -y", "/var/cache/apt"},
	}
	for pm, wants := range cases {
		out, err := Containerfile(m, "ghcr.io/tuna-os/yellowfin:gnome", pm)
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
	if _, err := Containerfile(&manifest.Manifest{}, "x", "portage"); err == nil {
		t.Fatal("expected error for unsupported package manager")
	}
}

func TestContainerfileExtraRunOrdering(t *testing.T) {
	m := &manifest.Manifest{
		Packages: []string{"tailscale"},
		ExtraRun: []string{"dnf config-manager addrepo --from-repofile=https://pkgs.tailscale.com/stable/fedora/tailscale.repo"},
	}
	out, err := Containerfile(m, "quay.io/fedora/fedora-bootc:latest", "dnf")
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
	out, err := Containerfile(&manifest.Manifest{}, "base:latest", "apt")
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
