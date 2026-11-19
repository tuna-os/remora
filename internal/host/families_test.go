package host

import (
	"strings"
	"testing"

	"github.com/tuna-os/remora/internal/pkgmgr"
)

// probeBinaries returns the binaries imageProbe looks for, in order.
func probeBinaries(t *testing.T) []string {
	t.Helper()
	_, rest, ok := strings.Cut(imageProbe, "for b in ")
	if !ok {
		t.Fatal("imageProbe has no `for b in` loop")
	}
	list, _, ok := strings.Cut(rest, ";")
	if !ok {
		t.Fatal("imageProbe `for b in` loop is not terminated by ;")
	}
	return strings.Fields(list)
}

// Detection must be able to return every family remora can build for, from
// a binary the image probe actually looks for. Otherwise a family is only
// reachable through an explicit package_manager in the manifest.
func TestDetectionReachesEveryFamily(t *testing.T) {
	reached := map[string]bool{}
	for _, b := range probeBinaries(t) {
		got, err := parseImageProbe([]byte("bin=" + b + "\nid=\n"))
		if err != nil {
			t.Errorf("probe binary %q is not classified: %v", b, err)
			continue
		}
		reached[got] = true
	}
	for _, f := range pkgmgr.Families() {
		if !reached[f] {
			t.Errorf("family %q is not detectable from any binary in imageProbe", f)
		}
	}
}
