package pm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		{"unknown derivative with apt", "bin=apt-get\nid=somederivative\n", "apt"},
		{"no binary, known id", "id=fedora\n", "dnf"},
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

func TestParseImageProbeUnknown(t *testing.T) {
	_, err := parseImageProbe([]byte("id=\n"))
	if err == nil {
		t.Fatal("expected an error when nothing identifies the package manager")
	}
	if !strings.Contains(err.Error(), "package_manager") {
		t.Errorf("error should point at the manifest override, got: %v", err)
	}
}
