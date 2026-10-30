package host

// Tests for the exec-backed host helpers:
// UupdPresent and Systemctl. osReleaseID is covered through /etc/os-release
// reads only where the environment permits; the pure detectPM logic is
// already tested in host_test.go.

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
	if err := os.WriteFile(logPath, []byte("#!/bin/sh\necho \"$@\" > \""+logPath+"\"\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

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
