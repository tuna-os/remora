package pkgmgr

import (
	"slices"
	"testing"
)

func TestFamiliesUniqueAndNonEmpty(t *testing.T) {
	fs := Families()
	if len(fs) == 0 {
		t.Fatal("no families")
	}
	for i, f := range fs {
		if f == "" {
			t.Errorf("family %d is empty", i)
		}
		if slices.Index(fs, f) != i {
			t.Errorf("family %q listed twice", f)
		}
	}
}

// Callers must not be able to change the canonical list through the slice
// they were handed.
func TestFamiliesReturnsCopy(t *testing.T) {
	Families()[0] = "mutated"
	if Families()[0] == "mutated" {
		t.Fatal("Families exposes its backing array")
	}
}
