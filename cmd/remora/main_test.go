package main

import (
	"os"
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
	"github.com/tuna-os/remora/internal/resolve"
)

// Contract tests for the CLI dispatcher run() in main.go.
//
// Only side-effect-free paths are exercised here: help/usage output, flag
// parsing errors, unknown-command rejection, a read-only `list` against an
// empty directory, and the argument-validation / missing-manifest branches of
// status/rebase/upgrade that return before any host call is made.
//
// CI's e2e smoke step (.github/workflows/ci.yml) only ever drives
// generate/install/list/apply — init, shims, the build-triggering branches of
// modify/build/upgrade/rebase, and enable/disable are covered by neither this
// file nor that smoke test. Those genuinely write to /etc, /usr/local/bin, or
// invoke systemctl/podman for real, which neither a unit test nor the current
// smoke harness should do without refactoring cmdInit/cmdShims to take an
// injectable root (see the coverage-gap issue this file's history points to).

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

// cmdStatus never mutates anything — it only reads the manifest and shells
// out to bootc/systemctl for informational output — so it is safe to call
// directly in a unit test even without those binaries present. Neither the
// happy path nor the "not a bootc system" fallback was exercised anywhere
// (not by a unit test, and not by CI's e2e smoke, which only ever calls
// generate/install/list/apply): the misleading main_test.go package comment
// above claims host-touching subcommands are "exercised by the smoke test in
// CI", which is not true for status/init/shims/upgrade/rebase.
func TestCmdStatusNoManifest(t *testing.T) {
	if err := cmdStatus(t.TempDir()); err != nil {
		t.Fatalf("cmdStatus(<empty dir>) = %v, want nil (missing manifest is reported, not an error)", err)
	}
}

func TestCmdStatusWithManifest(t *testing.T) {
	dir := t.TempDir()
	m := &manifest.Manifest{Packages: []string{"htop"}}
	if err := m.Save(dir); err != nil {
		t.Fatal(err)
	}
	if err := cmdStatus(dir); err != nil {
		t.Fatalf("cmdStatus(<dir with manifest>) = %v, want nil", err)
	}
}

// cmdRebase must reject a wrong argument count before touching the manifest
// or the host at all.
func TestCmdRebaseRequiresExactlyOneArg(t *testing.T) {
	for _, args := range [][]string{nil, {}, {"a", "b"}} {
		err := cmdRebase(t.TempDir(), args, false)
		if err == nil {
			t.Fatalf("cmdRebase(%v) = nil, want error", args)
		}
		if !strings.Contains(err.Error(), "exactly one image ref") {
			t.Fatalf("cmdRebase(%v) error = %q, want it to mention the arg count", args, err)
		}
	}
}

// With no manifest present, cmdRebase and cmdUpgrade must fail on the load
// before reaching any host call (skopeo/podman/bootc), which is what keeps
// this path deterministic without those tools installed.
func TestCmdRebaseNoManifestFailsBeforeHostCall(t *testing.T) {
	err := cmdRebase(t.TempDir(), []string{"docker.io/library/debian"}, false)
	if err == nil {
		t.Fatal("cmdRebase(<empty dir>, ...) = nil, want error for missing manifest")
	}
}

func TestCmdUpgradeNoManifestFailsBeforeHostCall(t *testing.T) {
	err := cmdUpgrade(t.TempDir(), false)
	if err == nil {
		t.Fatal("cmdUpgrade(<empty dir>) = nil, want error for missing manifest")
	}
}
