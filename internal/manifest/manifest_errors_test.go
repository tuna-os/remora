package manifest

// Tests for the Load/Save error paths and defaults that were uncovered:
// missing manifest, malformed YAML, Save with an invalid manifest, and
// the ImageTag/OnCalendar defaults.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Fatal("Load of missing manifest: expected error")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "remora.yaml"), []byte("packages: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load of malformed YAML: expected error")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("error = %v, want parsing context", err)
	}
}

func TestLoad_InvalidManifestRejected(t *testing.T) {
	dir := t.TempDir()
	// Shell metachars in a package name must be rejected at load time.
	if err := os.WriteFile(filepath.Join(dir, "remora.yaml"),
		[]byte("packages:\n  - 'vim; rm -rf /'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load of invalid manifest: expected validation error")
	}
}

func TestSave_InvalidManifest(t *testing.T) {
	m := &Manifest{Packages: []string{"vim; rm -rf /"}}
	if err := m.Save(t.TempDir()); err == nil {
		t.Fatal("Save of invalid manifest: expected error")
	}
}

func TestImageTag_Default(t *testing.T) {
	m := &Manifest{}
	if got := m.ImageTag(); got != "localhost/remora:latest" {
		t.Errorf("ImageTag() default = %q", got)
	}
}

func TestImageTag_Custom(t *testing.T) {
	m := &Manifest{Image: "localhost/myhost:dev"}
	if got := m.ImageTag(); got != "localhost/myhost:dev" {
		t.Errorf("ImageTag() = %q", got)
	}
}

func TestOnCalendar_DefaultAndCustom(t *testing.T) {
	if got := (&Manifest{}).OnCalendar(); got != "*-*-* 04:00:00" {
		t.Errorf("OnCalendar() default = %q", got)
	}
	if got := (&Manifest{Schedule: "daily"}).OnCalendar(); got != "daily" {
		t.Errorf("OnCalendar() custom = %q", got)
	}
}
