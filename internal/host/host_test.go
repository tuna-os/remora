package host

import "testing"

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
