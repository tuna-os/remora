package factory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
)

func TestQuadletContent(t *testing.T) {
	m := &manifest.Manifest{}
	q := Quadlet(m, "/etc/remora")
	for _, w := range []string{
		"ImageTag=localhost/remora:latest",
		"Pull=newer",
		"SetWorkingDirectory=/etc/remora",
		"bootc switch --quiet --transport=containers-storage localhost/remora:latest",
		"bootc update --quiet",
		"podman image prune",
	} {
		if !strings.Contains(q, w) {
			t.Errorf("quadlet missing %q", w)
		}
	}
}

func TestQuadletCustomImage(t *testing.T) {
	m := &manifest.Manifest{Image: "localhost/myhost:latest"}
	q := Quadlet(m, "/etc/remora")
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
