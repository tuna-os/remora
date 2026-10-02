package bootc

import (
	"testing"
)

func TestBuildSwitchArgs(t *testing.T) {
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
		got := buildSwitchArgs(c.ref, c.apply, c.softReboot)
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
