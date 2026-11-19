// Package pkgmgr owns the set of package-manager families remora supports.
//
// Each family's behaviour lives with the package that needs it: the install
// recipe in internal/generate, the shimmed commands in internal/shim,
// detection in internal/host. This package only names the families, so that
// adding one is a change every one of those sites is tested against.
package pkgmgr

// families is the canonical, ordered list. Order is user-visible: it is the
// order error messages and documentation list the families in.
var families = []string{"dnf", "zypper", "pacman", "apt", "portage", "apk"}

// Families returns the supported package-manager families, in canonical
// order. The caller owns the returned slice.
func Families() []string {
	return append([]string(nil), families...)
}
