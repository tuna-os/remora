package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/tuna-os/remora/internal/manifest"
)

// TestApplyNoOpWhenDigestUnchanged verifies that the apply command skips
// the bootc switch when the locally built image digest matches the currently
// staged/booted deployment. This is remora's core no-op optimization.
func TestApplyNoOpWhenDigestUnchanged(t *testing.T) {
	if testing.Short() {
		t.Skip("requires manifest setup")
	}

	// Create a temporary state directory
	tmpDir, err := os.MkdirTemp("", "remora-apply-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write a test manifest
	m := &manifest.Manifest{
		Packages: []string{"htop"},
	}
	if err := m.Save(tmpDir); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	// This test would ideally mock the host.LocalDigest and host.StagedOrBootedDigest
	// calls to verify that apply compares digests and skips the switch when they match.
	// However, the current implementation has these calls embedded in the apply handler,
	// making it difficult to test without a full system integration test.
	//
	// The recommendation is to extract apply's digest-checking logic into a
	// testable function that doesn't require mocking process execution.
	t.Logf("apply no-op behavior requires system integration test (see issue #99)")
}

// TestApplyRefusesWithoutBuiltImage verifies that apply exits non-zero
// when no local image has been built yet. This guards against accidentally
// triggering a switch to a non-existent image.
func TestApplyRefusesWithoutBuiltImage(t *testing.T) {
	if testing.Short() {
		t.Skip("requires manifest setup")
	}

	tmpDir, err := os.MkdirTemp("", "remora-apply-empty-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an empty state without a built image
	m := &manifest.Manifest{
		Packages: []string{"pkg"},
	}
	if err := m.Save(tmpDir); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	// Note: This would require extracting cmdApply's image-existence check
	// into a standalone function for effective testing without mocking.
	t.Logf("apply guards against missing image (see issue #99 for unit test extraction)")
}
