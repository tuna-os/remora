package buildplan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/factory"
	"github.com/tuna-os/remora/internal/manifest"
	"github.com/tuna-os/remora/internal/resolve"
)

// putOnPath writes an executable stub to a temp dir and prepends it to PATH.
func putOnPath(t *testing.T, name, script string) {
	t.Helper()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

// ResolveBase must reuse an existing pin rather than re-resolving, which is
// what makes two rebuilds a week apart produce the same image. Only
// `remora upgrade` (or a changed base: in the manifest) moves it.
func TestResolveBaseReusesPin(t *testing.T) {
	dir := t.TempDir()
	const pin = "quay.io/fedora/fedora-bootc@sha256:abc"
	if err := manifest.SaveBase(dir, pin); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		m    *manifest.Manifest
	}{
		// "" means follow the booted image; the pin records what that
		// resolved to, so it is reused rather than re-read from bootc.
		{"follow booted", &manifest.Manifest{}},
		// An explicit base naming the same image keeps the pin.
		{"same image", &manifest.Manifest{Base: "quay.io/fedora/fedora-bootc"}},
		// A base given with its digest already is the pin.
		{"same pinned ref", &manifest.Manifest{Base: pin}},
	}
	for _, c := range cases {
		got, err := ResolveBase(dir, c.m)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != pin {
			t.Errorf("%s: got %q, want the existing pin %q", c.name, got, pin)
		}
	}
}

// Pointing base: at a different image must not silently keep building the
// old one: the stale pin is dropped, the new ref is pinned from the registry,
// and the old pin is kept as base.previous for rollback.
func TestResolveBaseRepinsChangedBase(t *testing.T) {
	putOnPath(t, "skopeo", "#!/bin/sh\necho sha256:new\n")
	dir := t.TempDir()
	const stale = "quay.io/fedora/fedora-bootc@sha256:abc"
	if err := manifest.SaveBase(dir, stale); err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{Base: "docker.io/library/debian"}
	got, err := ResolveBase(dir, m)
	if err != nil {
		t.Fatal(err)
	}
	const want = "docker.io/library/debian@sha256:new"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if pin := manifest.LoadBase(dir); pin != want {
		t.Errorf("saved pin = %q, want %q", pin, want)
	}
	if prev := manifest.LoadBasePrevious(dir); prev != stale {
		t.Errorf("base.previous = %q, want the replaced pin %q", prev, stale)
	}
}

// An unreachable registry degrades to building from the unpinned ref, but
// that ref must never be recorded as the pin: a later run with the registry
// back must still pin it to a digest.
func TestResolveBaseUnpinnedWhenRegistryUnreachable(t *testing.T) {
	putOnPath(t, "skopeo", "#!/bin/sh\nexit 1\n")
	dir := t.TempDir()
	m := &manifest.Manifest{Base: "docker.io/library/debian"}
	got, err := ResolveBase(dir, m)
	if err != nil {
		t.Fatalf("registry failure must fall back, not fail: %v", err)
	}
	if got != m.Base {
		t.Errorf("got %q, want the unpinned ref %q", got, m.Base)
	}
	if pin := manifest.LoadBase(dir); pin != "" {
		t.Errorf("unpinned ref recorded as the pin: %q", pin)
	}
}

// A base given with its digest is already a pin; it is saved without asking
// the registry.
func TestResolveBaseExplicitDigestSkipsRegistry(t *testing.T) {
	t.Setenv("PATH", "/nonexistent-dir")
	dir := t.TempDir()
	const ref = "docker.io/library/debian@sha256:abc"
	got, err := ResolveBase(dir, &manifest.Manifest{Base: ref})
	if err != nil {
		t.Fatal(err)
	}
	if got != ref {
		t.Errorf("got %q, want %q", got, ref)
	}
	if pin := manifest.LoadBase(dir); pin != ref {
		t.Errorf("saved pin = %q, want %q", pin, ref)
	}
}

// With no base: and no pin, the booted image is followed and pinned to the
// digest bootc reports, so the next rebuild does not chase a moving tag.
func TestResolveBaseFollowsBootedImageDigest(t *testing.T) {
	cases := []struct {
		name, status, want string
	}{
		{
			"digest reported",
			`{"status":{"booted":{"image":{"image":{"image":"quay.io/fedora/fedora-bootc:42"},"imageDigest":"sha256:abc"}}}}`,
			"quay.io/fedora/fedora-bootc:42@sha256:abc",
		},
		{
			"no digest reported",
			`{"status":{"booted":{"image":{"image":{"image":"quay.io/fedora/fedora-bootc:42"}}}}}`,
			"quay.io/fedora/fedora-bootc:42",
		},
	}
	for _, c := range cases {
		putOnPath(t, "bootc", "#!/bin/sh\ncat <<'EOF'\n"+c.status+"\nEOF\n")
		dir := t.TempDir()
		got, err := ResolveBase(dir, &manifest.Manifest{})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if pin := manifest.LoadBase(dir); pin != c.want {
			t.Errorf("%s: saved pin = %q, want %q", c.name, pin, c.want)
		}
	}
}

// Following the booted image off a bootc system is an error, and nothing is
// pinned.
func TestResolveBaseNotABootcSystem(t *testing.T) {
	t.Setenv("PATH", "/nonexistent-dir")
	dir := t.TempDir()
	if _, err := ResolveBase(dir, &manifest.Manifest{}); err == nil {
		t.Fatal("expected an error without bootc")
	}
	if pin := manifest.LoadBase(dir); pin != "" {
		t.Errorf("pin written on failure: %q", pin)
	}
}

// An explicit base: can be any distribution, so the package manager must not
// be inferred from the host. An explicit package_manager always wins.
func TestResolvePMExplicitWins(t *testing.T) {
	m := &manifest.Manifest{Base: "docker.io/library/debian", PackageManager: "apt"}
	got, err := ResolvePM(m, "docker.io/library/debian@sha256:abc")
	if err != nil {
		t.Fatal(err)
	}
	if got != "apt" {
		t.Errorf("got %q, want apt", got)
	}
}

// An explicit base: is probed for its own package manager instead of
// inheriting the host's.
func TestResolvePMProbesExplicitBase(t *testing.T) {
	putOnPath(t, "podman", "#!/bin/sh\necho bin=apt-get\necho id=debian\n")
	m := &manifest.Manifest{Base: "docker.io/library/debian"}
	got, err := ResolvePM(m, "docker.io/library/debian@sha256:abc")
	if err != nil {
		t.Fatal(err)
	}
	if got != "apt" {
		t.Errorf("got %q, want apt", got)
	}
}

// When the base cannot be probed, the error names the manifest override.
func TestResolvePMProbeFailureNamesOverride(t *testing.T) {
	putOnPath(t, "podman", "#!/bin/sh\nexit 1\n")
	m := &manifest.Manifest{Base: "docker.io/library/debian"}
	_, err := ResolvePM(m, "docker.io/library/debian@sha256:abc")
	if err == nil {
		t.Fatal("expected an error when the probe fails")
	}
	if !strings.Contains(err.Error(), "set package_manager") {
		t.Errorf("error = %v, want it to name the package_manager override", err)
	}
}

// ResolveLock must never leave a lockfile behind on any path that returns "".
// A surviving lockfile would keep pinning an old package set forever once the
// resolver stopped being available — the Containerfile would stop COPYing it,
// but a later run that regained the resolver would silently reuse it.
func TestResolveLockClearsStaleLockfileOnEveryFallback(t *testing.T) {
	cases := []struct {
		name string
		m    *manifest.Manifest
		pm   string
	}{
		{"no packages", &manifest.Manifest{}, "dnf"},
		{"explicitly disabled", &manifest.Manifest{Packages: []string{"htop"}, Lockfile: boolPtr(false)}, "dnf"},
		{"package manager has no resolver", &manifest.Manifest{Packages: []string{"htop"}}, "apt"},
		// dnf with no reachable podman/base: Available() fails, so this
		// falls back too rather than failing the build.
		{"resolver unavailable", &manifest.Manifest{Packages: []string{"htop"}}, "dnf"},
	}
	for _, c := range cases {
		dir := t.TempDir()
		stale := resolve.Path(dir)
		if err := os.WriteFile(stale, []byte("stale lockfile"), 0o644); err != nil {
			t.Fatal(err)
		}
		got := ResolveLock(dir, c.m, "localhost/definitely-not-a-real-image:missing", c.pm)
		if got != "" {
			t.Errorf("%s: expected a fallback to the spec list, got lock %q", c.name, got)
		}
		if _, err := os.Stat(stale); !os.IsNotExist(err) {
			t.Errorf("%s: stale lockfile survived the fallback", c.name)
		}
	}
}

// An explicit `lockfile: false` is the documented escape hatch, so it must
// win even where the resolver would otherwise be used.
func TestResolveLockRespectsOptOut(t *testing.T) {
	dir := t.TempDir()
	m := &manifest.Manifest{Packages: []string{"htop"}, Lockfile: boolPtr(false)}
	if got := ResolveLock(dir, m, "base@sha256:abc", "dnf"); got != "" {
		t.Errorf("lockfile: false must force the spec-list path, got %q", got)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestReconcileUpdatesRuntimeUnitsFromManifest(t *testing.T) {
	dir := t.TempDir()
	root := t.TempDir()
	if err := manifest.SaveBase(dir, "quay.io/fedora/fedora-bootc@sha256:abc"); err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		PackageManager: "dnf",
		Image:          "localhost/remora:first",
		Schedule:       "daily",
	}
	reloads := 0
	installUnits := true
	reconcile := func() {
		t.Helper()
		if err := Reconcile(dir, m, false, installUnits, root, "/usr/bin/remora", func() error {
			reloads++
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		installUnits = false
	}
	readUnit := func(path string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}

	reconcile()
	if reloads != 1 {
		t.Fatalf("initial reconciliation reloaded systemd %d times, want 1", reloads)
	}
	reconcile()
	if reloads != 1 {
		t.Fatalf("unchanged reconciliation reloaded systemd; got %d reloads, want 1", reloads)
	}

	m.Image = "localhost/remora:second"
	m.Schedule = "weekly"
	reconcile()
	if reloads != 2 {
		t.Fatalf("changed reconciliation produced %d reloads, want 2", reloads)
	}
	if got, want := readUnit(factory.QuadletPath), factory.Quadlet(m, dir, "/usr/bin/remora"); got != want {
		t.Errorf("quadlet not reconciled from changed image:\n got %q\nwant %q", got, want)
	}
	if got, want := readUnit(factory.TimerPath), factory.Timer(m); got != want {
		t.Errorf("timer not reconciled from changed schedule:\n got %q\nwant %q", got, want)
	}
}
