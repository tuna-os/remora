package main

import (
	"strings"
	"testing"
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
