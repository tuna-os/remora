package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
	"github.com/tuna-os/remora/internal/resolve"
)

// Contract tests for the CLI dispatcher run() in main.go.
//
// Side-effect-free paths are exercised directly: help/usage output, flag
// parsing errors, unknown-command rejection, and a read-only `list` against
// an empty directory. Subcommands that shell out to bootc/systemctl (status)
// are exercised against fake binaries on PATH, the same technique
// internal/host/exec_test.go uses. cmdInit, cmdModify's build path, cmdBuild,
// cmdShims, cmdUpgrade, and cmdRebase still are not: init/build/upgrade/
// rebase invoke `systemctl start` for a real build, and shims hardcodes
// /usr/local/bin — none of that is a unit test's business to touch, and none
// of it is currently driven by CI's smoke script either (which only calls
// generate, install --no-build, list, and apply's no-local-image failure
// path). See the coverage-gap issue for what's still open.

func TestRunEmptyArgsPrintsUsage(t *testing.T) {
	for _, args := range [][]string{nil, {}} {
		if err := run(args); err != nil {
			t.Fatalf("run(%v) = %v, want nil (usage printed)", args, err)
		}
	}
}

func TestRunHelpVariantsSucceed(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"help"}} {
		if err := run(args); err != nil {
			t.Fatalf("run(%v) = %v, want nil", args, err)
		}
	}
}

func TestRunDirFlagRequiresValue(t *testing.T) {
	err := run([]string{"--dir"})
	if err == nil {
		t.Fatal("run([--dir]) = nil, want error")
	}
	if !strings.Contains(err.Error(), "--dir needs a value") {
		t.Fatalf("run([--dir]) error = %q, want it to mention the missing value", err)
	}
}

func TestRunUnknownCommandRejected(t *testing.T) {
	for _, args := range [][]string{{"bogus"}, {"--no-build", "bogus"}, {"--dir", "/tmp/x", "frobnicate"}} {
		err := run(args)
		if err == nil {
			t.Fatalf("run(%v) = nil, want unknown-command error", args)
		}
		if !strings.Contains(err.Error(), "unknown command") {
			t.Fatalf("run(%v) error = %q, want 'unknown command'", args, err)
		}
	}
}

func TestRunFlagsWithoutCommandPrintUsage(t *testing.T) {
	for _, args := range [][]string{{"--no-build"}, {"--remove"}, {"--dir", "/tmp/remora-test-does-not-exist"}} {
		if err := run(args); err != nil {
			t.Fatalf("run(%v) = %v, want nil (usage printed)", args, err)
		}
	}
}

func TestRunListInEmptyDirFailsSafely(t *testing.T) {
	err := run([]string{"--dir", t.TempDir(), "list"})
	if err == nil {
		t.Fatal("run([--dir <empty> list]) = nil, want error for missing manifest")
	}
}

func TestRunEnableDisabledNotInvokedOnHelp(t *testing.T) {
	// Sanity: --help must never reach the host-facing dispatch.
	if err := run([]string{"--help"}); err != nil {
		t.Fatalf("run([--help]) = %v, want nil", err)
	}
}

// resolveBase must reuse an existing pin rather than re-resolving, which is
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
		got, err := resolveBase(dir, c.m)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != pin {
			t.Errorf("%s: got %q, want the existing pin %q", c.name, got, pin)
		}
	}
}

// Pointing base: at a different image must not silently keep building the
// old one. Without network access here we can only assert that the stale pin
// is rejected — resolveBase then tries to resolve the new ref and fails.
func TestResolveBaseRejectsStalePin(t *testing.T) {
	dir := t.TempDir()
	if err := manifest.SaveBase(dir, "quay.io/fedora/fedora-bootc@sha256:abc"); err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{Base: "docker.io/library/debian"}
	got, err := resolveBase(dir, m)
	if err == nil && got == "quay.io/fedora/fedora-bootc@sha256:abc" {
		t.Fatal("stale pin reused after base: changed to a different image")
	}
}

// An explicit base: can be any distribution, so the package manager must not
// be inferred from the host. An explicit package_manager always wins.
func TestResolvePMExplicitWins(t *testing.T) {
	m := &manifest.Manifest{Base: "docker.io/library/debian", PackageManager: "apt"}
	got, err := resolvePM(m, "docker.io/library/debian@sha256:abc")
	if err != nil {
		t.Fatal(err)
	}
	if got != "apt" {
		t.Errorf("got %q, want apt", got)
	}
}

// resolveLock must never leave a lockfile behind on any path that returns "".
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
		got := resolveLock(dir, c.m, "localhost/definitely-not-a-real-image:missing", c.pm)
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
	if got := resolveLock(dir, m, "base@sha256:abc", "dnf"); got != "" {
		t.Errorf("lockfile: false must force the spec-list path, got %q", got)
	}
}

func boolPtr(b bool) *bool { return &b }

// putOnPath writes an executable stub to a temp dir and prepends it to PATH,
// the same fake-binary technique internal/host/exec_test.go uses to test
// exec-backed helpers without touching the real host.
func putOnPath(t *testing.T, name, script string) string {
	t.Helper()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	old := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", old) })
	os.Setenv("PATH", binDir+":"+old)
	return binDir
}

// cmdStatus was 0% covered: it is a read-only report (booted image, manifest
// summary, timer status), but nothing exercised it — not a unit test, and
// not CI's smoke script, which only drives generate/install/list/apply.

func TestCmdStatusNoManifestReturnsNilWithoutQueryingTheTimer(t *testing.T) {
	putOnPath(t, "bootc", "#!/bin/sh\nexit 1\n")
	// No systemctl stub: cmdStatus must return before reaching it, since
	// manifest.Load fails first on an empty directory.
	if err := cmdStatus(t.TempDir()); err != nil {
		t.Fatalf("cmdStatus(empty dir) = %v, want nil", err)
	}
}

func TestCmdStatusWithManifestQueriesTheTimer(t *testing.T) {
	putOnPath(t, "bootc", `#!/bin/sh
cat <<'EOF'
{"status":{"booted":{"image":{"image":{"image":"ghcr.io/tuna-os/yellowfin:gnome"}}}}}
EOF
`)
	marker := filepath.Join(t.TempDir(), "systemctl-args")
	putOnPath(t, "systemctl", `#!/bin/sh
echo "$@" > `+marker+`
exit 0
`)

	dir := t.TempDir()
	m := &manifest.Manifest{Packages: []string{"htop"}}
	if err := m.Save(dir); err != nil {
		t.Fatal(err)
	}

	if err := cmdStatus(dir); err != nil {
		t.Fatalf("cmdStatus(dir with manifest) = %v, want nil", err)
	}

	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("cmdStatus never invoked systemctl: %v", err)
	}
	if !strings.Contains(string(got), "status") || !strings.Contains(string(got), "remora-build.timer") {
		t.Errorf("systemctl invoked with %q, want it to query remora-build.timer's status", got)
	}
}

// cmdStatus's fallback message when bootc itself fails (not a bootc host, or
// no bootc binary at all) must not turn into an error — status is meant to
// degrade gracefully, not refuse to run off-target.
func TestCmdStatusToleratesMissingBootc(t *testing.T) {
	putOnPath(t, "bootc", "#!/bin/sh\nexit 1\n")
	putOnPath(t, "systemctl", "#!/bin/sh\nexit 0\n")

	dir := t.TempDir()
	m := &manifest.Manifest{}
	if err := m.Save(dir); err != nil {
		t.Fatal(err)
	}

	if err := cmdStatus(dir); err != nil {
		t.Fatalf("cmdStatus with a failing bootc = %v, want nil (degrades gracefully)", err)
	}
}
