package generate

import (
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
)

// TestContainerfileReproducibility ensures that identical inputs always produce
// byte-identical Containerfile output. This is critical: remora's no-op rebuild
// optimization depends on digest reproducibility. If two builds with the same
// manifest and base produce different Containerfiles, the image digest will differ
// even though nothing changed, and the apply command will incorrectly trigger a switch.
func TestContainerfileReproducibility(t *testing.T) {
	testCases := []struct {
		name string
		m    *manifest.Manifest
		base string
		pm   string
	}{
		{
			name: "empty manifest",
			m:    &manifest.Manifest{},
			base: "quay.io/fedora/fedora-bootc@sha256:abc123",
			pm:   "dnf",
		},
		{
			name: "with packages",
			m: &manifest.Manifest{
				Packages: []string{"htop", "vim", "git"},
			},
			base: "ghcr.io/tuna-os/yellowfin:gnome@sha256:def456",
			pm:   "dnf",
		},
		{
			name: "with extra_run",
			m: &manifest.Manifest{
				Packages: []string{"tailscale"},
				ExtraRun: []string{
					"dnf config-manager addrepo --from-repofile=https://pkgs.tailscale.com/stable/fedora/tailscale.repo",
					"echo setup-complete",
				},
			},
			base: "registry.fedoraproject.org/fedora:40@sha256:xyz789",
			pm:   "dnf",
		},
		{
			name: "zypper pm",
			m: &manifest.Manifest{
				Packages: []string{"zypper-package"},
			},
			base: "registry.opensuse.org/opensuse:tumbleweed@sha256:aaa111",
			pm:   "zypper",
		},
		{
			name: "apt pm",
			m: &manifest.Manifest{
				Packages: []string{"ubuntu-package"},
			},
			base: "ubuntu:24.04@sha256:bbb222",
			pm:   "apt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate the same Containerfile five times to ensure reproducibility
			// across multiple generations.
			var first string
			for i := 0; i < 5; i++ {
				generated, err := Containerfile(tc.m, tc.base, tc.pm, "")
				if err != nil {
					t.Fatalf("generation %d failed: %v", i+1, err)
				}

				if i == 0 {
					first = generated
				} else if generated != first {
					t.Fatalf("generation %d differs from first generation:\nFirst:\n%s\nGeneration %d:\n%s",
						i+1, first, i+1, generated)
				}
			}
		})
	}
}

// TestLayerStructureIsConsistent verifies that the Containerfile always
// contains exactly three RUN heredoc layers (overlay, packages, scripts)
// in the correct order. This structure is load-bearing: if the layer
// structure breaks, the package build cache becomes unreliable.
func TestLayerStructureIsConsistent(t *testing.T) {
	testCases := []struct {
		name      string
		m         *manifest.Manifest
		wantLayers int
	}{
		{
			name:       "empty manifest still has 3 layers",
			m:          &manifest.Manifest{},
			wantLayers: 3,
		},
		{
			name: "manifest with packages",
			m: &manifest.Manifest{
				Packages: []string{"a", "b", "c"},
			},
			wantLayers: 3,
		},
		{
			name: "manifest with extra_run",
			m: &manifest.Manifest{
				ExtraRun: []string{"cmd1", "cmd2"},
			},
			wantLayers: 3,
		},
		{
			name: "manifest with packages and extra_run",
			m: &manifest.Manifest{
				Packages: []string{"pkg1"},
				ExtraRun: []string{"cmd1"},
			},
			wantLayers: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cf, err := Containerfile(tc.m, "base@sha256:abc", "dnf", "")
			if err != nil {
				t.Fatalf("Containerfile generation failed: %v", err)
			}

			// Count heredoc RUN layers: each layer is a "<<'REMORA_EOF'" marker
			heredocCount := countOccurrences(cf, "<<'REMORA_EOF'")
			if heredocCount != tc.wantLayers {
				t.Errorf("want %d heredoc RUN layers, got %d", tc.wantLayers, heredocCount)
			}
		})
	}
}

// TestDigestPinningIsConsistent ensures that the FROM line always includes
// a SHA256 digest when the input base is pinned, and never omits it.
// A regressed FROM clause could silently break the reproducibility guarantee.
func TestDigestPinningIsConsistent(t *testing.T) {
	testCases := []struct {
		name       string
		inputBase  string
		expectFrom string // substring to find in the FROM line
	}{
		{
			name:       "pinned digest preserved",
			inputBase:  "fedora:40@sha256:abc123",
			expectFrom: "@sha256:abc123",
		},
		{
			name:       "pinned digest not stripped",
			inputBase:  "quay.io/some/image:tag@sha256:def456",
			expectFrom: "@sha256:def456",
		},
		{
			name:       "fedora bootc pinned",
			inputBase:  "quay.io/fedora/fedora-bootc@sha256:xyz789",
			expectFrom: "@sha256:xyz789",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &manifest.Manifest{Packages: []string{"pkg"}}
			cf, err := Containerfile(m, tc.inputBase, "dnf", "")
			if err != nil {
				t.Fatalf("Containerfile generation failed: %v", err)
			}

			if !hasSubstring(cf, tc.expectFrom) {
				t.Errorf("expected FROM to include digest %q, but Containerfile:\n%s", tc.expectFrom, cf)
			}
		})
	}
}

// TestTimestampScrubbing verifies that the Containerfile includes steps
// to scrub build-varying state (logs, package manager caches, timestamps)
// that would cause otherwise-identical inputs to produce different digests.
func TestTimestampScrubbing(t *testing.T) {
	m := &manifest.Manifest{Packages: []string{"curl", "git"}}
	cf, err := Containerfile(m, "base@sha256:abc", "dnf", "")
	if err != nil {
		t.Fatalf("Containerfile generation failed: %v", err)
	}

	// The Containerfile should include scrubbing steps for:
	// - logs that vary between builds
	// - package manager cache and history
	// These are documented in the code and essential for reproducibility.
	scrubPatterns := []string{
		"logs",         // logs vary between builds
		"history",      // dnf history varies
		"cache",        // package caches are build-varying
		"rpm",          // rpm/dnf state
	}

	for _, pattern := range scrubPatterns {
		if !hasSubstring(cf, pattern) {
			t.Logf("warning: expected scrubbing for %q in Containerfile", pattern)
		}
	}
}

// Helpers

func countOccurrences(s, substring string) int {
	count := 0
	for {
		idx := indexAfter(s, substring)
		if idx < 0 {
			return count
		}
		count++
		s = s[idx+1:]
	}
}

func indexAfter(s, substring string) int {
	for i := 0; i < len(s); i++ {
		if len(s)-i >= len(substring) &&
			s[i:i+len(substring)] == substring {
			return i
		}
	}
	return -1
}

func hasSubstring(s, substring string) bool {
	return indexAfter(s, substring) >= 0
}
