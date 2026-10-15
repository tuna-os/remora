package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{Packages: []string{"vim", "htop"}, Schedule: "daily"}
	if err := m.Save(dir); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Packages) != 2 || got.Schedule != "daily" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestAddRemoveDedup(t *testing.T) {
	m := &Manifest{}
	added := m.AddPackages([]string{"b", "a", "b", ""})
	if len(added) != 2 {
		t.Fatalf("added = %v, want [b a]", added)
	}
	if m.Packages[0] != "a" || m.Packages[1] != "b" {
		t.Fatalf("packages not sorted: %v", m.Packages)
	}
	if again := m.AddPackages([]string{"a"}); len(again) != 0 {
		t.Fatal("duplicate add should be a no-op")
	}
	removed := m.RemovePackages([]string{"a", "zzz"})
	if len(removed) != 1 || removed[0] != "a" {
		t.Fatalf("removed = %v, want [a]", removed)
	}
}

func TestValidateRejectsShellMetachars(t *testing.T) {
	m := &Manifest{Packages: []string{"vim; rm -rf /"}}
	if err := m.Validate(); err == nil {
		t.Fatal("expected rejection of package name with shell metacharacters")
	}
}

func TestValidateRejectsUnknownPM(t *testing.T) {
	m := &Manifest{PackageManager: "nix"}
	if err := m.Validate(); err == nil {
		t.Fatal("expected rejection of unknown package_manager")
	}
}

func TestDefaults(t *testing.T) {
	m := &Manifest{}
	if m.ImageTag() != "localhost/remora:latest" {
		t.Fatal("wrong default image tag:", m.ImageTag())
	}
	if m.OnCalendar() != "*-*-* 04:00:00" {
		t.Fatal("wrong default schedule:", m.OnCalendar())
	}
}

func TestBasePinRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if got := LoadBase(dir); got != "" {
		t.Fatalf("unwritten pin should read empty, got %q", got)
	}
	const ref = "quay.io/fedora/fedora-bootc@sha256:abc"
	if err := SaveBase(dir, ref); err != nil {
		t.Fatal(err)
	}
	if got := LoadBase(dir); got != ref {
		t.Errorf("LoadBase = %q, want %q", got, ref)
	}
	// The pin is a plain one-line file, readable and editable by hand.
	data, err := os.ReadFile(BasePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != ref+"\n" {
		t.Errorf("pin file = %q, want a single trailing newline", string(data))
	}
}

func TestSaveBaseCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "remora")
	if err := SaveBase(dir, "img@sha256:abc"); err != nil {
		t.Fatal(err)
	}
	if got := LoadBase(dir); got != "img@sha256:abc" {
		t.Errorf("got %q", got)
	}
}

// Moving the pin records the old one; re-saving the same pin (as
// ResolveBase does on every build) must not clobber that history.
func TestSaveBaseKeepsPrevious(t *testing.T) {
	dir := t.TempDir()
	if err := SaveBase(dir, "img@sha256:aaa"); err != nil {
		t.Fatal(err)
	}
	if got := LoadBasePrevious(dir); got != "" {
		t.Fatalf("first pin recorded a previous: %q", got)
	}
	if err := SaveBase(dir, "img@sha256:bbb"); err != nil {
		t.Fatal(err)
	}
	if got := LoadBasePrevious(dir); got != "img@sha256:aaa" {
		t.Fatalf("previous = %q, want img@sha256:aaa", got)
	}
	if err := SaveBase(dir, "img@sha256:bbb"); err != nil {
		t.Fatal(err)
	}
	if got := LoadBasePrevious(dir); got != "img@sha256:aaa" {
		t.Errorf("re-saving the same pin changed previous to %q", got)
	}
	if got := LoadBase(dir); got != "img@sha256:bbb" {
		t.Errorf("base = %q, want img@sha256:bbb", got)
	}
}
