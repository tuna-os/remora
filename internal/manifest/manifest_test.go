package manifest

import (
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
	m := &Manifest{PackageManager: "portage"}
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
