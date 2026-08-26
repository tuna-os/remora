package factory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
)

func TestQuadletContent(t *testing.T) {
	m := &manifest.Manifest{}
	q := Quadlet(m, "/etc/remora", "/usr/bin/remora")
	for _, w := range []string{
		"ImageTag=localhost/remora:latest",
		"Pull=newer",
		"SetWorkingDirectory=/etc/remora",
		"PodmanArgs=--timestamp 0",
		"ExecStartPre=-/usr/bin/remora --dir /etc/remora upgrade --no-build",
		"ExecStartPost=/usr/bin/remora --dir /etc/remora apply",
		"podman image prune",
	} {
		if !strings.Contains(q, w) {
			t.Errorf("quadlet missing %q", w)
		}
	}
}

func TestQuadletCustomImage(t *testing.T) {
	m := &manifest.Manifest{Image: "localhost/myhost:latest"}
	q := Quadlet(m, "/etc/remora", "/usr/bin/remora")
	if !strings.Contains(q, "ImageTag=localhost/myhost:latest") {
		t.Fatal("custom image tag not respected")
	}
}

func TestTimerSchedule(t *testing.T) {
	m := &manifest.Manifest{Schedule: "weekly"}
	tm := Timer(m)
	if !strings.Contains(tm, "OnCalendar=weekly") {
		t.Fatal("custom schedule not respected")
	}
	if !strings.Contains(tm, "Persistent=true") {
		t.Fatal("timer must be persistent")
	}
}

func TestUupdDropin(t *testing.T) {
	d := UupdDropin()
	if !strings.Contains(d, "Wants="+ServiceName) || !strings.Contains(d, "After="+ServiceName) {
		t.Fatal("uupd drop-in must order the rebuild before uupd:", d)
	}
}

func TestWriteContext(t *testing.T) {
	dir := t.TempDir()
	m := &manifest.Manifest{Packages: []string{"htop"}}
	if err := WriteContext(dir, m, "base:latest", "dnf"); err != nil {
		t.Fatal(err)
	}
	cf, err := os.ReadFile(filepath.Join(dir, "Containerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cf), "FROM base:latest") {
		t.Fatal("Containerfile missing base")
	}
	for _, sub := range []string{"build_files", "system_files"} {
		if fi, err := os.Stat(filepath.Join(dir, sub)); err != nil || !fi.IsDir() {
			t.Fatalf("missing %s dir", sub)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestInstallUupdHook(t *testing.T) {
	root := t.TempDir()
	if err := InstallUupdHook(root); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, UupdDropinPath)
	if got := readFile(t, path); got != UupdDropin() {
		t.Errorf("drop-in content mismatch:\n got %q\nwant %q", got, UupdDropin())
	}
	// Parent dirs must be created (uupd.service.d is a new tree under root).
	if fi, err := os.Stat(filepath.Dir(path)); err != nil || !fi.IsDir() {
		t.Fatalf("drop-in parent dir not created: %v", err)
	}
	// Nothing may be written outside root.
	if _, err := os.Stat(filepath.Join("/etc/systemd/system/uupd.service.d", "10-remora.conf")); !os.IsNotExist(err) {
		t.Errorf("install wrote outside root (host /etc touched): %v", err)
	}
}

func TestInstallUnits(t *testing.T) {
	root := t.TempDir()
	dir := "/srv/remora"
	exe := "/usr/bin/remora"
	m := &manifest.Manifest{}
	// /etc/systemd/system exists on real systemd hosts but not under a temp
	// root; InstallUnits only creates the quadlet dir (its own tree).
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(TimerPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	var reloaded int
	err := InstallUnits(m, dir, root, exe, func() error {
		reloaded++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if reloaded != 1 {
		t.Errorf("reload called %d times, want 1", reloaded)
	}
	quadletPath := filepath.Join(root, QuadletPath)
	timerPath := filepath.Join(root, TimerPath)
	if got := readFile(t, quadletPath); got != Quadlet(m, dir, exe) {
		t.Errorf("quadlet content mismatch:\n got %q\nwant %q", got, Quadlet(m, dir, exe))
	}
	if got := readFile(t, timerPath); got != Timer(m) {
		t.Errorf("timer content mismatch:\n got %q\nwant %q", got, Timer(m))
	}
	for _, p := range []string{quadletPath, timerPath} {
		if fi, err := os.Stat(filepath.Dir(p)); err != nil || !fi.IsDir() {
			t.Fatalf("parent dir %s not created: %v", filepath.Dir(p), err)
		}
	}
}

func TestInstallUnitsReloadErrorPropagates(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(TimerPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	want := errors.New("daemon-reload failed")
	err := InstallUnits(&manifest.Manifest{}, "/srv/remora", root, "/usr/bin/remora", func() error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("reload error not propagated: got %v, want %v", err, want)
	}
	// Files must still have been written before the reload attempt.
	if _, err := os.Stat(filepath.Join(root, QuadletPath)); err != nil {
		t.Errorf("quadlet missing when reload fails: %v", err)
	}
}

func TestInstallUnitsFailureStopsBeforeReload(t *testing.T) {
	// Make the quadlet dir uncreatable so the first MkdirAll fails.
	root := t.TempDir()
	blocker := filepath.Join(root, "etc")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	reloaded := false
	err := InstallUnits(&manifest.Manifest{}, "/srv/remora", root, "/usr/bin/remora", func() error {
		reloaded = true
		return nil
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if reloaded {
		t.Error("reload must not run when a write step fails")
	}
}
