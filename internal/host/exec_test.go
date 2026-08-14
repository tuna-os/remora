package host

// Tests for the exec-backed host helpers that had zero coverage:
// BootedImage (via a fake bootc in PATH), UupdPresent (fake systemctl),
// and Systemctl. osReleaseID is covered through /etc/os-release reads
// only where the environment permits; the pure detectPM logic is already
// tested in host_test.go.

import (
	"os"
	"path/filepath"
	"testing"
)

// putOnPath writes an executable stub to a temp dir and prepends it to PATH.
func putOnPath(t *testing.T, name, script string) string {
	t.Helper()
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	old := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", old) })
	os.Setenv("PATH", binDir+":"+old)
	return binDir
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
	old := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", old) })
	os.Setenv("PATH", "/nonexistent-dir")

	_, err := BootedImage()
	if err == nil {
		t.Fatal("BootedImage without bootc: expected error")
	}
	if !contains(err.Error(), "bootc status") {
		t.Errorf("error = %v, want bootc status context", err)
	}
}

func TestUupdPresent_WhenInstalled(t *testing.T) {
	putOnPath(t, "systemctl", "#!/bin/sh\nexit 0\n")
	if !UupdPresent() {
		t.Error("UupdPresent with systemctl success: want true")
	}
}

func TestUupdPresent_WhenMissing(t *testing.T) {
	putOnPath(t, "systemctl", "#!/bin/sh\nexit 1\n")
	if UupdPresent() {
		t.Error("UupdPresent with systemctl failure: want false")
	}
}

func TestSystemctl_Success(t *testing.T) {
	binDir := putOnPath(t, "systemctl", "#!/bin/sh\nexit 0\n")
	logPath := filepath.Join(binDir, "systemctl.log")
	os.WriteFile(logPath, []byte("#!/bin/sh\necho \"$@\" > \""+logPath+"\"\nexit 0\n"), 0o755)

	if err := Systemctl("daemon-reload"); err != nil {
		t.Fatalf("Systemctl: %v", err)
	}
}

func TestSystemctl_Error(t *testing.T) {
	putOnPath(t, "systemctl", "#!/bin/sh\nexit 1\n")
	if err := Systemctl("start", "remora-build.timer"); err == nil {
		t.Fatal("Systemctl with failing systemctl: expected error")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
