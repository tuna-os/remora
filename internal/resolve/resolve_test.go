package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestForKnownPackageManagers(t *testing.T) {
	if _, ok := For("dnf"); !ok {
		t.Error("dnf should have a resolver")
	}
	// The rest install from the spec list until they grow a resolver;
	// claiming one that does not exist would emit a COPY for a missing file.
	for _, pm := range []string{"apt", "zypper", "pacman", "portage", "apk", "nonsense"} {
		if _, ok := For(pm); ok {
			t.Errorf("%s unexpectedly reports a resolver", pm)
		}
	}
}

func TestPath(t *testing.T) {
	if got, want := Path("/etc/remora"), filepath.Join("/etc/remora", LockFile); got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}

// Clear is what stops a lockfile from outliving the resolver that produced
// it. If the resolver stops being available and a stale lockfile survives,
// the build would pin an old package set indefinitely.
func TestClearRemovesStaleLockfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(Path(dir), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Clear(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(Path(dir)); !os.IsNotExist(err) {
		t.Fatalf("lockfile survived Clear: %v", err)
	}
}

// Clearing when there is nothing to clear is the common case — it runs on
// every generate for package managers without a resolver.
func TestClearMissingIsNotAnError(t *testing.T) {
	if err := Clear(t.TempDir()); err != nil {
		t.Fatalf("Clear on a dir with no lockfile: %v", err)
	}
}

// Resolving nothing is a caller bug, not a silent no-op that would leave a
// previous lockfile in place.
func TestResolveRejectsEmptyPackageList(t *testing.T) {
	r, _ := For("dnf")
	if err := r.Resolve("base@sha256:abc", t.TempDir(), nil); err == nil {
		t.Fatal("expected an error resolving an empty package list")
	}
}

func TestResolverName(t *testing.T) {
	r, _ := For("dnf")
	if r.Name() == "" {
		t.Error("resolver must have a user-facing name")
	}
}

// writeAtomic is what Resolve uses to publish the lockfile; an interrupted
// resolution must never leave a half-written file where a build would COPY
// it in.
func TestWriteAtomicCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	if err := writeAtomic(path, []byte("content")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "content" {
		t.Errorf("wrote %q, want %q", got, "content")
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temp file survived writeAtomic: %v", err)
	}
}

func TestWriteAtomicOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	if err := os.WriteFile(path, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(path, []byte("fresh")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "fresh" {
		t.Errorf("got %q, want %q", got, "fresh")
	}
}

// Available must not attempt to run podman at all when podman is not on
// PATH — the common case on a system without the tool installed.
func TestDnfResolverAvailable_NoPodman(t *testing.T) {
	old := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", old) })
	os.Setenv("PATH", t.TempDir())

	if (dnfResolver{}).Available("base@sha256:abc") {
		t.Error("Available with no podman on PATH: want false")
	}
}

// When podman is present but the base image lacks dnf5's manifest plugin,
// Available must report false rather than erroring — callers fall back to
// the plain install path.
func TestDnfResolverAvailable_PodmanProbeFails(t *testing.T) {
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "podman"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", old) })
	os.Setenv("PATH", binDir)

	if (dnfResolver{}).Available("base@sha256:abc") {
		t.Error("Available with a failing probe: want false")
	}
}
