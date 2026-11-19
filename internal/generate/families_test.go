package generate

import (
	"slices"
	"testing"

	"github.com/tuna-os/remora/internal/pkgmgr"
)

// The determinism and scrub tests iterate SupportedPMs, so a recipe in pms
// that SupportedPMs does not list would escape them. Both must match the
// canonical family list exactly.
func TestRecipesCoverEveryFamily(t *testing.T) {
	for _, f := range pkgmgr.Families() {
		if _, ok := pms[f]; !ok {
			t.Errorf("no Containerfile recipe for family %q", f)
		}
	}
	for name := range pms {
		if !slices.Contains(pkgmgr.Families(), name) {
			t.Errorf("recipe %q is not a family in internal/pkgmgr", name)
		}
	}
	if !slices.Equal(SupportedPMs(), pkgmgr.Families()) {
		t.Errorf("SupportedPMs() = %v, want %v", SupportedPMs(), pkgmgr.Families())
	}
}
