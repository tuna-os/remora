package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitDigest(t *testing.T) {
	cases := []struct {
		ref, name, digest string
		ok                bool
	}{
		{"quay.io/fedora/fedora-bootc@sha256:abc", "quay.io/fedora/fedora-bootc", "sha256:abc", true},
		{"quay.io/fedora/fedora-bootc:42", "quay.io/fedora/fedora-bootc:42", "", false},
		{"localhost/remora:latest", "localhost/remora:latest", "", false},
		// A registry port contains a colon but never an @, so the last @
		// is unambiguous.
		{"registry:5000/img@sha256:def", "registry:5000/img", "sha256:def", true},
	}
	for _, c := range cases {
		name, digest, ok := SplitDigest(c.ref)
		if name != c.name || digest != c.digest || ok != c.ok {
			t.Errorf("SplitDigest(%q) = %q,%q,%v; want %q,%q,%v",
				c.ref, name, digest, ok, c.name, c.digest, c.ok)
		}
	}
}

// A ref that already carries a digest must be returned untouched — PinBase
// must not reach for the network in that case.
func TestPinBaseAlreadyPinned(t *testing.T) {
	ref := "quay.io/fedora/fedora-bootc@sha256:abc"
	got, err := PinBase(ref)
	if err != nil {
		t.Fatal(err)
	}
	if got != ref {
		t.Errorf("PinBase(%q) = %q, want it unchanged", ref, got)
	}
}

func TestOSReleaseIDFromPath(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "os-release")
	if err := os.WriteFile(file, []byte("NAME=Fedora\nID=fedora\nVERSION=42\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := osReleaseIDFromPath(file); got != "fedora" {
		t.Errorf("osReleaseIDFromPath(%q) = %q, want %q", file, got, "fedora")
	}

	nonExistent := filepath.Join(tmpDir, "does-not-exist")
	if got := osReleaseIDFromPath(nonExistent); got != "" {
		t.Errorf("osReleaseIDFromPath(%q) = %q, want empty string for missing file", nonExistent, got)
	}
}
