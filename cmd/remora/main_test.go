package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
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

// putOnPath writes an executable stub to a temp dir and prepends it to PATH,
// so cmdStatus's bootc/systemctl calls can be driven explicitly instead of
// depending on whatever is (or isn't) actually installed on the test host.
func putOnPath(t *testing.T, name, script string) {
	t.Helper()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

// cmdStatus's fallback when bootc itself fails (not a bootc host, or no bootc
// binary at all) must not turn into an error — status is meant to degrade
// gracefully, not refuse to run off-target.
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

// exePath is what units call back into (`remora enable` installs a systemd
// unit that re-execs this path); a relative or empty result would install a
// unit that stops working the moment the working directory changes.
func TestExePathReturnsAbsolutePath(t *testing.T) {
	got := exePath()
	if got == "" {
		t.Fatal("exePath returned empty string")
	}
	if !filepath.IsAbs(got) {
		t.Errorf("exePath = %q, want an absolute path", got)
	}
}

// rebase --previous must fail clearly, before any host call, when no
// previous pin has been recorded.
func TestCmdRebasePreviousWithoutHistory(t *testing.T) {
	err := cmdRebase(t.TempDir(), []string{"--previous"}, false)
	if err == nil || !strings.Contains(err.Error(), "no previous base pin") {
		t.Fatalf("cmdRebase(--previous) error = %v, want missing-history error", err)
	}
}
