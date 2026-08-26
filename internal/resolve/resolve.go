// Package resolve turns a manifest's package list into a lockfile pinning the
// exact set that will be installed — every dependency, at an exact version.
//
// Why this exists: without it, the generated Containerfile says
// `dnf -y install vim`, and podman's layer cache keys on that string. The
// string never changes, so the cache either always hits (and the layer goes
// stale) or is busted by an unrelated change — podman cannot tell whether
// today's `vim` resolves to the same package set as yesterday's. With a
// lockfile COPYied into the build, the cache keys on the *resolved set*: an
// unchanged upstream produces a byte-identical lockfile, the COPY hits cache,
// and the rebuild is a genuine no-op down to the image digest.
//
// Resolution is best-effort. Every Resolver is expected to be unavailable on
// some systems, and callers must fall back to the plain install path rather
// than failing the build — see Resolver.Available.
package resolve

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// LockFile is the lockfile's name inside the build context. It sits at the
// context root so the generated Containerfile can COPY it directly.
const LockFile = "remora.lock.yaml"

// CacheDir holds package-manager metadata between resolutions, so a scheduled
// rebuild does not re-download every repository's index just to discover
// nothing changed.
const CacheDir = "/var/cache/remora"

// Resolver produces a lockfile for one package manager.
type Resolver interface {
	// Name identifies the resolver in user-facing messages.
	Name() string
	// Available reports whether this resolver can run against base. It is
	// expected to return false routinely — the tooling may not be
	// installed in the base image, or podman may be missing — and callers
	// must treat false as "use the plain install path", not as an error.
	Available(base string) bool
	// Resolve writes LockFile into dir, pinning pkgs and their
	// dependencies as resolved against base.
	Resolve(base, dir string, pkgs []string) error
}

// For returns the resolver for a package manager, if one exists. Package
// managers without a resolver simply use the plain install path.
func For(pmName string) (Resolver, bool) {
	switch pmName {
	case "dnf":
		return dnfResolver{}, true
	}
	return nil, false
}

// Path returns the lockfile's path inside dir.
func Path(dir string) string { return filepath.Join(dir, LockFile) }

// Clear removes a stale lockfile from dir. Callers must do this whenever
// resolution does not produce a fresh one, so a lockfile left by an earlier
// build is never silently reused after the resolver becomes unavailable.
func Clear(dir string) error {
	err := os.Remove(Path(dir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// dnfResolver drives dnf5's manifest plugin (dnf5-plugin-manifest, which
// provides dnf5-command(manifest)). `manifest new` resolves a list of specs
// against the installed set and pins the result; `manifest install` in the
// build then installs exactly that.
//
// Both run inside a container started from the base image, never on the host:
// the base image's repositories, releasever, and installed set are what the
// build will actually see, and an explicit base: in the manifest may be a
// different distribution from the host entirely.
type dnfResolver struct{}

func (dnfResolver) Name() string { return "dnf5 manifest" }

func (dnfResolver) Available(base string) bool {
	if _, err := exec.LookPath("podman"); err != nil {
		return false
	}
	// The plugin is optional and marked experimental upstream, so probe for
	// the subcommand itself rather than for a dnf5 version.
	cmd := exec.Command("podman", "run", "--rm", "--entrypoint", "", base,
		"sh", "-c", "command -v dnf5 >/dev/null 2>&1 && dnf5 manifest --help >/dev/null 2>&1")
	return cmd.Run() == nil
}

func (d dnfResolver) Resolve(base, dir string, pkgs []string) error {
	if len(pkgs) == 0 {
		return fmt.Errorf("no packages to resolve")
	}
	cache := filepath.Join(CacheDir, "libdnf5")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", cache, err)
	}

	// The lockfile is written into a scratch directory rather than straight
	// into the state dir. Mounting a directory with :Z relabels it to a
	// container-private SELinux label, and doing that to /etc/remora would
	// leave the rest of the system unable to read it. Scratch dirs are ours
	// to relabel; the state dir is not, so it is never mounted at all —
	// `manifest new` takes its package specs as arguments.
	out, err := os.MkdirTemp("", "remora-resolve-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(out)

	args := []string{
		"run", "--rm", "--entrypoint", "",
		"--volume", out + ":/out:Z",
		"--volume", cache + ":/var/cache/libdnf5:Z",
		base,
		"dnf5", "manifest", "new",
		"--assumeyes",
		// Resolve against what the base image already has, so the lockfile
		// pins the packages being *added* plus their missing dependencies,
		// not a second copy of the base's own package set.
		"--use-system",
		// The base image's cached metadata can be older than the
		// repositories it points at; without this the lockfile could pin
		// versions that no longer exist.
		"--refresh",
		"--manifest", "/out/" + LockFile,
	}
	args = append(args, pkgs...)

	cmd := exec.Command("podman", args...)
	cmd.Stdout = os.Stderr // resolution progress is diagnostics, not output
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dnf5 manifest new: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(out, LockFile))
	if err != nil {
		return fmt.Errorf("resolver reported success but wrote no %s: %w", LockFile, err)
	}
	return writeAtomic(Path(dir), data)
}

// writeAtomic replaces path in one step, so an interrupted resolution can
// never leave a half-written lockfile that the next build would COPY.
func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
