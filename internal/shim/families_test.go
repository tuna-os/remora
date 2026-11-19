package shim

import (
	"slices"
	"testing"

	"github.com/tuna-os/remora/internal/pkgmgr"
)

// Every family remora can build for must have shims, and Remove must know
// every shim name, or `remora shims --remove` leaves files behind.
func TestShimsCoverEveryFamily(t *testing.T) {
	all := allNames()
	for _, f := range pkgmgr.Families() {
		cmds := names(f)
		if len(cmds) == 0 {
			t.Errorf("family %q has no shimmed commands", f)
		}
		for _, name := range cmds {
			if _, err := Script(f, name); err != nil {
				t.Errorf("Script(%q, %q): %v", f, name, err)
			}
			if !slices.Contains(all, name) {
				t.Errorf("allNames() omits %q (family %q), so Remove would leave it behind", name, f)
			}
		}
	}
}
