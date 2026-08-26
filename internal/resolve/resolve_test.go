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
