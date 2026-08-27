package host

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

func TestDetectPMByBinary(t *testing.T) {
	has := func(names ...string) func(string) bool {
		return func(n string) bool {
			for _, x := range names {
				if n == x {
					return true
				}
			}
			return false
		}
	}
	cases := []struct {
		bins []string
		want string
	}{
		{[]string{"dnf5"}, "dnf"},
		{[]string{"dnf"}, "dnf"},
		{[]string{"zypper"}, "zypper"},
		{[]string{"pacman"}, "pacman"},
		{[]string{"apt-get"}, "apt"},
		{[]string{"emerge"}, "portage"},
		{[]string{"apk"}, "apk"},
	}
	for _, c := range cases {
		got, err := detectPM("", has(c.bins...))
		if err != nil || got != c.want {
			t.Fatalf("bins=%v: got %q err=%v, want %q", c.bins, got, err, c.want)
		}
	}
}

func TestDetectPMByOSRelease(t *testing.T) {
	none := func(string) bool { return false }
	for id, want := range map[string]string{
		"fedora": "dnf", "centos": "dnf", "opensuse-tumbleweed": "zypper",
		"arch": "pacman", "debian": "apt", "ubuntu": "apt",
		"gentoo": "portage", "alpine": "apk",
	} {
		got, err := detectPM(id, none)
		if err != nil || got != want {
			t.Fatalf("ID=%s: got %q err=%v, want %q", id, got, err, want)
		}
	}
	if _, err := detectPM("nixos", none); err == nil {
		t.Fatal("expected detection failure for nixos (no supported pm)")
	}
}

func TestParseOSReleaseID(t *testing.T) {
	cases := []struct {
		content string
		want    string
	}{
		{"NAME=Fedora\nID=fedora\nVERSION=42\n", "fedora"},
		{"NAME=\"Ubuntu\"\nID=\"ubuntu\"\n", "ubuntu"},
		{"NAME=Unknown\n", ""},
	}
	for _, c := range cases {
		got := parseOSReleaseID([]byte(c.content))
		if got != c.want {
			t.Errorf("content=%q: got %q, want %q", c.content, got, c.want)
		}
	}
}

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

// The whole point of probing the image is that the base need not match the
// host: a dnf host layering onto an apt base must render apt-get, not dnf.
// These cases are the cross-family combinations issue #18 calls out.
func TestParseImageProbeCrossFamily(t *testing.T) {
	cases := []struct {
		name  string
		probe string
		want  string
	}{
		{"debian base", "bin=apt-get\nid=debian\n", "apt"},
		{"ubuntu base", "bin=apt-get\nid=ubuntu\n", "apt"},
		{"fedora base", "bin=dnf5\nid=fedora\n", "dnf"},
		{"arch base", "bin=pacman\nid=arch\n", "pacman"},
		{"opensuse base", "bin=zypper\nid=opensuse-tumbleweed\n", "zypper"},
		{"gentoo base", "bin=emerge\nid=gentoo\n", "portage"},
		{"alpine base", "bin=apk\nid=alpine\n", "apk"},
		// A binary present but an unrecognized ID still classifies: the
		// binary is the stronger signal, and derivatives are common.
		{"unknown derivative with apt", "bin=apt-get\nid=somederivative\n", "apt"},
		// No binary found, but os-release still identifies the family.
		{"no binary, known id", "id=fedora\n", "dnf"},
		// Trailing whitespace and CRLF must not defeat the prefix match.
		{"crlf output", "bin=pacman\r\nid=arch\r\n", "pacman"},
	}
	for _, c := range cases {
		got, err := parseImageProbe([]byte(c.probe))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// Neither a binary nor a recognizable ID must be a clear error pointing at
// the manifest escape hatch, not a silent wrong guess.
func TestParseImageProbeUnknown(t *testing.T) {
	_, err := parseImageProbe([]byte("id=\n"))
	if err == nil {
		t.Fatal("expected an error when nothing identifies the package manager")
	}
	if !strings.Contains(err.Error(), "package_manager") {
		t.Errorf("error should point at the manifest override, got: %v", err)
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

