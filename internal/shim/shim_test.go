package shim

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptPerPM(t *testing.T) {
	cases := []struct{ pm, name, mustHave string }{
		{"dnf", "dnf", "install|in) op=install"},
		{"apt", "apt-get", `real="/usr/bin/apt-get"`},
		{"zypper", "zypper", "remove|rm|erase|purge"},
		{"pacman", "pacman", "-S|-Sy) op=install"},
	}
	for _, c := range cases {
		s, err := Script(c.pm, c.name)
		if err != nil {
			t.Fatalf("%s: %v", c.pm, err)
		}
		if !strings.Contains(s, c.mustHave) {
			t.Errorf("%s shim missing %q", c.pm, c.mustHave)
		}
		if !strings.Contains(s, "remora shim for "+c.name) {
			t.Errorf("%s shim missing ownership marker", c.name)
		}
	}
}

func TestScriptUnknownPM(t *testing.T) {
	if _, err := Script("portage", "emerge"); err == nil {
		t.Fatal("expected error for unknown pm")
	}
}

// TestShimBehaviour runs the generated dnf shim with a fake "real" binary
// and asserts pass-through vs redirect behaviour.
func TestShimBehaviour(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	dir := t.TempDir()
	script, err := Script("dnf", "dnf")
	if err != nil {
		t.Fatal(err)
	}
	// Point the shim at a fake real dnf that echoes its args.
	fake := filepath.Join(dir, "real-dnf")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho REAL:$@\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	script = strings.Replace(script, `real="/usr/bin/dnf"`, `real="`+fake+`"`, 1)
	shimPath := filepath.Join(dir, "dnf")
	if err := os.WriteFile(shimPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	// Read-only subcommand passes through.
	out, err := exec.Command(shimPath, "search", "htop").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "REAL:search htop") {
		t.Fatalf("search should pass through, got %q err=%v", out, err)
	}

	// install is intercepted, exits 1 (non-interactive), mentions remora.
	cmd := exec.Command(shimPath, "install", "htop")
	cmd.Stdin = nil
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatal("install should exit non-zero when not confirmed")
	}
	if !strings.Contains(string(out), "remora install htop") {
		t.Fatalf("install output should point at remora, got %q", out)
	}

	// upgrade points at remora build.
	out, _ = exec.Command(shimPath, "upgrade").CombinedOutput()
	if !strings.Contains(string(out), "remora build") {
		t.Fatalf("upgrade output should point at remora build, got %q", out)
	}
}

func TestInstallRemoveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	installed, err := Install(dir, "apt")
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 2 {
		t.Fatalf("expected apt + apt-get shims, got %v", installed)
	}
	removed, err := Remove(dir, "apt")
	if err != nil || len(removed) != 2 {
		t.Fatalf("remove failed: %v %v", removed, err)
	}
}

func TestInstallRefusesForeignFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zypper"), []byte("#!/bin/sh\n# the real thing\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(dir, "zypper"); err == nil {
		t.Fatal("must refuse to overwrite a non-shim file")
	}
}

func TestRemoveLeavesForeignFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pacman")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n# the real thing\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	removed, err := Remove(dir, "pacman")
	if err != nil || len(removed) != 0 {
		t.Fatalf("must not remove foreign files: %v %v", removed, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("foreign file was deleted")
	}
}
