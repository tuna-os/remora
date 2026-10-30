package bootc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBootedImage(t *testing.T) {
	j := []byte(`{"status":{"booted":{"image":{"image":{"image":"ghcr.io/tuna-os/yellowfin:gnome"}}}}}`)
	got, err := parseBootedImage(j)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ghcr.io/tuna-os/yellowfin:gnome" {
		t.Fatal("wrong image:", got)
	}
}

func TestParseBootedImageEmpty(t *testing.T) {
	if _, err := parseBootedImage([]byte(`{"status":{}}`)); err == nil {
		t.Fatal("expected error for missing booted image")
	}
}

func TestBootedImage(t *testing.T) {
	putOnPath(t, "bootc", `#!/bin/sh
cat <<'EOF'
{"status":{"booted":{"image":{"image":{"image":"ghcr.io/tuna-os/yellowfin:gnome"}}}}}
EOF
`)
	got, err := BootedImage()
	if err != nil {
		t.Fatalf("BootedImage: %v", err)
	}
	if got != "ghcr.io/tuna-os/yellowfin:gnome" {
		t.Errorf("BootedImage = %q", got)
	}
}

func TestBootedImage_NotABootcSystem(t *testing.T) {
	// bootc absent from PATH → exec fails → wrapped error.
	t.Setenv("PATH", "/nonexistent-dir")

	_, err := BootedImage()
	if err == nil {
		t.Fatal("BootedImage without bootc: expected error")
	}
	if !strings.Contains(err.Error(), "bootc status") {
		t.Errorf("error = %v, want bootc status context", err)
	}
}

func TestParseBootedImageDigest(t *testing.T) {
	const status = `{"status":{"booted":{"image":{"image":{"image":"quay.io/fedora/fedora-bootc:42","transport":"registry"},"imageDigest":"sha256:abc"}}}}`
	ref, digest, err := parseBootedImageDigest([]byte(status))
	if err != nil {
		t.Fatal(err)
	}
	if ref != "quay.io/fedora/fedora-bootc:42" {
		t.Errorf("ref = %q", ref)
	}
	if digest != "sha256:abc" {
		t.Errorf("digest = %q", digest)
	}
}

func TestParseBootedImageDigestNoImage(t *testing.T) {
	if _, _, err := parseBootedImageDigest([]byte(`{"status":{}}`)); err == nil {
		t.Fatal("expected an error when no booted image is reported")
	}
}

// A staged deployment is what the system will boot next, so it wins over
// the booted one when deciding whether a switch would change anything.
func TestParseStagedOrBootedDigest(t *testing.T) {
	cases := []struct{ name, json, want string }{
		{
			"staged wins",
			`{"status":{"staged":{"image":{"imageDigest":"sha256:staged"}},"booted":{"image":{"imageDigest":"sha256:booted"}}}}`,
			"sha256:staged",
		},
		{
			"falls back to booted",
			`{"status":{"booted":{"image":{"imageDigest":"sha256:booted"}}}}`,
			"sha256:booted",
		},
		{
			"neither",
			`{"status":{}}`,
			"",
		},
	}
	for _, c := range cases {
		got, err := parseStagedOrBootedDigest([]byte(c.json))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestParseStagedOrBootedDigestBadJSON(t *testing.T) {
	if _, err := parseStagedOrBootedDigest([]byte("not json")); err == nil {
		t.Fatal("expected a parse error")
	}
}

func TestBuildBootcSwitchArgs(t *testing.T) {
	cases := []struct {
		name       string
		ref        string
		apply      bool
		softReboot string
		want       []string
	}{
		{
			name:       "basic switch",
			ref:        "quay.io/tuna-os/base@sha256:123",
			apply:      false,
			softReboot: "",
			want:       []string{"switch", "--transport=containers-storage", "quay.io/tuna-os/base@sha256:123"},
		},
		{
			name:       "switch with apply",
			ref:        "quay.io/tuna-os/base@sha256:123",
			apply:      true,
			softReboot: "",
			want:       []string{"switch", "--transport=containers-storage", "--apply", "quay.io/tuna-os/base@sha256:123"},
		},
		{
			name:       "switch with apply and soft reboot",
			ref:        "quay.io/tuna-os/base@sha256:123",
			apply:      true,
			softReboot: "auto",
			want:       []string{"switch", "--transport=containers-storage", "--apply", "--soft-reboot=auto", "quay.io/tuna-os/base@sha256:123"},
		},
	}

	for _, c := range cases {
		got := buildBootcSwitchArgs(c.ref, c.apply, c.softReboot)
		if len(got) != len(c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s[%d]: got %q, want %q", c.name, i, got[i], c.want[i])
			}
		}
	}
}

func putOnPath(t *testing.T, name, script string) string {
	t.Helper()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	return binDir
}
