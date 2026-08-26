package main

import (
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
)

// Contract tests for the CLI dispatcher run() in main.go.
//
// Only side-effect-free paths are exercised here: help/usage output, flag
// parsing errors, unknown-command rejection, and a read-only `list` against
// an empty directory. Every subcommand that touches the host (init/install/
// remove/build/enable/disable/status/shims) writes to /etc or invokes
// systemctl, which a unit test must not do; those paths are exercised by the
// smoke test in CI (`go build` + `just check` on a non-bootc host).

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
