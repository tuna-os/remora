// Package buildplan reconciles a manifest into remora's generated state.
//
// It owns the policy that connects a manifest to a pinned base image, the
// package manager of that image, an optional resolved package lockfile, and
// the runtime units that build the desired local image on its desired
// schedule. Keeping that policy here prevents the CLI transport from becoming
// the build state machine.
package buildplan

import (
	"fmt"
	"os"

	"github.com/tuna-os/remora/internal/factory"
	"github.com/tuna-os/remora/internal/host"
	"github.com/tuna-os/remora/internal/manifest"
	"github.com/tuna-os/remora/internal/resolve"
)

// ResolveBase returns the digest-pinned base that remora should build from.
func ResolveBase(dir string, m *manifest.Manifest) (string, error) {
	want := m.Base
	if pinned := manifest.LoadBase(dir); pinned != "" {
		name, _, _ := host.SplitDigest(pinned)
		if want == "" || name == want || pinned == want {
			return pinned, nil
		}
	}
	if want == "" {
		ref, digest, err := host.BootedImageDigest()
		if err != nil {
			return "", err
		}
		if digest != "" {
			ref += "@" + digest
		}
		if err := manifest.SaveBase(dir, ref); err != nil {
			return "", err
		}
		return ref, nil
	}
	pinned, err := host.PinBase(want)
	if err != nil {
		fmt.Fprintf(os.Stderr, "remora: could not pin %s to a digest (%v); building from the unpinned ref\n", want, err)
		return want, nil
	}
	if err := manifest.SaveBase(dir, pinned); err != nil {
		return "", err
	}
	return pinned, nil
}

// ResolvePM selects the package manager belonging to the selected base image.
func ResolvePM(m *manifest.Manifest, base string) (string, error) {
	if m.PackageManager != "" {
		return m.PackageManager, nil
	}
	if m.Base == "" {
		return host.DetectPM()
	}
	pm, err := host.DetectPMInImage(base)
	if err != nil {
		return "", fmt.Errorf("%w; set package_manager in remora.yaml", err)
	}
	return pm, nil
}

// Reconcile resolves the build inputs and rewrites the complete generated
// state: build context, quadlet, and timer. installUnits creates runtime units
// during initialization; later reconciliations update installed units. The
// default state directory always restores missing units on a managed host.
func Reconcile(dir string, m *manifest.Manifest, wantLock, installUnits bool, root, exe string, reload func() error) error {
	base, err := ResolveBase(dir, m)
	if err != nil {
		return err
	}
	pm, err := ResolvePM(m, base)
	if err != nil {
		return err
	}
	lock := ""
	if wantLock {
		lock = ResolveLock(dir, m, base, pm)
	} else if err := resolve.Clear(dir); err != nil {
		fmt.Fprintf(os.Stderr, "remora: could not remove a stale %s: %v\n", resolve.LockFile, err)
	}
	if err := factory.WriteContext(dir, m, base, pm, lock); err != nil {
		return err
	}
	how := "spec list"
	if lock != "" {
		how = "lockfile"
	}
	fmt.Printf("generated %s/Containerfile (base=%s, pm=%s, %d packages, %s)\n",
		dir, base, pm, len(m.Packages), how)
	if installUnits || dir == manifest.DefaultDir {
		return factory.InstallUnits(m, dir, root, exe, reload)
	}
	return factory.UpdateUnits(m, dir, root, exe, reload)
}

// ResolveLock creates a fresh lockfile or clears stale lock state on fallback.
func ResolveLock(dir string, m *manifest.Manifest, base, pm string) string {
	fallback := func(format string, args ...any) string {
		if format != "" {
			fmt.Fprintf(os.Stderr, "remora: "+format+"\n", args...)
		}
		if err := resolve.Clear(dir); err != nil {
			fmt.Fprintf(os.Stderr, "remora: could not remove a stale %s: %v\n", resolve.LockFile, err)
		}
		return ""
	}

	if len(m.Packages) == 0 || (m.Lockfile != nil && !*m.Lockfile) {
		return fallback("")
	}
	r, ok := resolve.For(pm)
	if !ok {
		return fallback("")
	}
	if !r.Available(base) {
		return fallback("%s is unavailable in %s; installing from the package list instead", r.Name(), base)
	}
	if err := r.Resolve(base, dir, m.Packages); err != nil {
		return fallback("%s failed (%v); installing from the package list instead", r.Name(), err)
	}
	return resolve.LockFile
}
