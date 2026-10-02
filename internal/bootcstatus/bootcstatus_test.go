package bootcstatus

import (
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
