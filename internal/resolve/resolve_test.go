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

// writeAtomic must produce a readable file with the exact bytes given,
// whether or not something already sits at path.
func TestWriteAtomicCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.yaml")
	if err := writeAtomic(path, []byte("pinned: true")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "pinned: true" {
		t.Errorf("content = %q, want %q", got, "pinned: true")
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temp file survived the rename: %v", err)
	}
}

func TestWriteAtomicOverwritesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.yaml")
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
		t.Errorf("content = %q, want %q", got, "fresh")
	}
}

// A resolver with no podman on PATH must report itself unavailable rather
// than let callers hit a "command not found" failure mid-build.
func TestAvailableFalseWithoutPodman(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	r, _ := For("dnf")
	if r.Available("base@sha256:abc") {
		t.Error("Available should be false when podman is not on PATH")
	}
}
