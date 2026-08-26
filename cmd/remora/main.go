// remora — user-friendly local layering for bootc systems.
//
// remora keeps a small manifest of packages and customizations, generates a
// Containerfile from it, and maintains a podman quadlet + timer that rebuild
// the layered image on top of the newest base and rebase to it — the
// container-native answer to "rpm-ostree install" on any dnf, zypper,
// pacman, or apt based bootc image.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tuna-os/remora/internal/factory"
	"github.com/tuna-os/remora/internal/host"
	"github.com/tuna-os/remora/internal/manifest"
	"github.com/tuna-os/remora/internal/resolve"
	"github.com/tuna-os/remora/internal/shim"
)

const usage = `remora — local layering for bootc systems, the container-native way

Usage: remora <command> [args]

Commands:
  init                 Set up /etc/remora, the build quadlet, and the timer
  install PKG...       Add packages to the manifest and rebuild
  remove PKG...        Remove packages from the manifest and rebuild
  list                 Show layered packages
  build                Rebuild the image now (and rebase via bootc switch)
  apply                Rebase to the built image if its digest changed
  upgrade              Refresh the pinned base digest, then rebuild
  rebase IMAGE         Point the manifest at a new base image, then rebuild
  enable               Enable the automatic rebuild timer
  disable              Disable the automatic rebuild timer
  status               Show booted image, manifest, and timer state
  generate             Regenerate the Containerfile only (no build)
  shims                Install package-manager shims that redirect
                       'dnf install' etc. to remora (--remove to uninstall)

Flags:
  --dir DIR            State directory (default /etc/remora)
  --no-build           With install/remove/upgrade/rebase: update state only
  --remove             With shims: uninstall the shims
  --apply              With apply/build: reboot immediately after switching
  --soft-reboot MODE   With apply/build: soft reboot, MODE is auto or required

The base image remora builds FROM is pinned to a digest in /etc/remora/base,
so rebuilds are reproducible; 'remora upgrade' is what moves the pin.

Note: once remora has rebased the system to its local image, 'bootc upgrade'
no longer knows how to update you — it does not recognize the base underneath
the local layer. Use 'remora upgrade' instead. To go back to a system managed
by bootc alone, run 'bootc rebase <base image>' and 'remora disable'.

The manifest is /etc/remora/remora.yaml. Custom scripts go in
/etc/remora/build_files/*.sh, file overlays in /etc/remora/system_files/.
For extra repositories or exotic builders (e.g. BuildStream), use the
manifest's extra_run list or a build_files script.`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "remora:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	dir := manifest.DefaultDir
	noBuild := false
	removeFlag := false
	applyFlag := false
	softReboot := ""
	var rest []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				return fmt.Errorf("--dir needs a value")
			}
			i++
			dir = args[i]
		case "--no-build":
			noBuild = true
		case "--remove":
			removeFlag = true
		case "--apply":
			applyFlag = true
		case "--soft-reboot":
			if i+1 >= len(args) {
				return fmt.Errorf("--soft-reboot needs a value (auto or required)")
			}
			i++
			softReboot = args[i]
			if softReboot != "auto" && softReboot != "required" {
				return fmt.Errorf("--soft-reboot must be auto or required, got %q", softReboot)
			}
		case "-h", "--help", "help":
			fmt.Println(usage)
			return nil
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) == 0 {
		fmt.Println(usage)
		return nil
	}
	cmd, cmdArgs := rest[0], rest[1:]

	switch cmd {
	case "init":
		return cmdInit(dir)
	case "install":
		return cmdModify(dir, cmdArgs, nil, noBuild)
	case "remove":
		return cmdModify(dir, nil, cmdArgs, noBuild)
	case "list":
		return cmdList(dir)
	case "build":
		return cmdBuild(dir, true)
	case "apply":
		return cmdApply(dir, applyFlag, softReboot)
	case "upgrade":
		return cmdUpgrade(dir, noBuild)
	case "rebase":
		return cmdRebase(dir, cmdArgs, noBuild)
	case "generate":
		return cmdBuild(dir, false)
	case "enable":
		return host.Systemctl("enable", "--now", factory.TimerName)
	case "disable":
		return host.Systemctl("disable", "--now", factory.TimerName)
	case "status":
		return cmdStatus(dir)
	case "shims":
		return cmdShims(dir, removeFlag)
	default:
		return fmt.Errorf("unknown command %q (see remora --help)", cmd)
	}
}

// resolveBase returns the base ref remora builds FROM, pinned to a digest.
//
// The manifest says what the user wants ("" = follow the booted image, or an
// explicit ref); dir/base records the digest that want resolved to. The pin
// is reused as long as it still refers to the same image name, so ordinary
// rebuilds are reproducible; `remora upgrade` is what moves it.
func resolveBase(dir string, m *manifest.Manifest) (string, error) {
	want := m.Base
	if pinned := manifest.LoadBase(dir); pinned != "" {
		name, _, _ := host.SplitDigest(pinned)
		// An explicit base: in the manifest overrides a stale pin for a
		// different image; "" (follow the booted image) keeps the pin,
		// which is what makes rebuilds reproducible between upgrades.
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
		// Pinning is an optimization for reproducibility, not a
		// precondition for building. A missing skopeo or an unreachable
		// registry degrades to the unpinned ref — which is what remora
		// did before pins existed — rather than making `remora generate`
		// unusable offline. `remora upgrade`, whose whole job is moving
		// the pin, still fails loudly.
		fmt.Fprintf(os.Stderr, "remora: could not pin %s to a digest (%v); building from the unpinned ref\n", want, err)
		return want, nil
	}
	if err := manifest.SaveBase(dir, pinned); err != nil {
		return "", err
	}
	return pinned, nil
}

// resolvePM picks the package manager for base. An explicit base: in the
// manifest can be any distribution, so the image itself is the authority —
// asking the host only works when the base is the booted image.
func resolvePM(m *manifest.Manifest, base string) (string, error) {
	if m.PackageManager != "" {
		return m.PackageManager, nil
	}
	if m.Base == "" {
		// Base is the booted image, so the host is the base.
		return host.DetectPM()
	}
	pm, err := host.DetectPMInImage(base)
	if err != nil {
		return "", fmt.Errorf("%w; set package_manager in remora.yaml", err)
	}
	return pm, nil
}

// regenerate resolves base + package manager and rewrites the build context.
//
// wantLock asks for a lockfile to be resolved as part of this pass. It is
// false whenever no build follows, because resolution starts a container from
// the base image — which would pull a multi-gigabyte bootc image as a side
// effect of a command like `remora generate` or `remora install --no-build`
// that has no other reason to touch the network. Every build path regenerates
// first, so the Containerfile is still lockfile-aware by the time it matters.
func regenerate(dir string, m *manifest.Manifest, wantLock bool) error {
	base, err := resolveBase(dir, m)
	if err != nil {
		return err
	}
	pm, err := resolvePM(m, base)
	if err != nil {
		return err
	}
	lock := ""
	if wantLock {
		lock = resolveLock(dir, m, base, pm)
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
	return nil
}

// resolveLock produces a lockfile pinning the exact package set, and returns
// its name for the Containerfile to COPY — or "" to fall back to installing
// from the spec list.
//
// Every failure path here is a fallback, not an error. The resolver needs
// podman, a reachable base image, and package-manager tooling that may simply
// not be installed; none of that is a reason to refuse to build, since the
// spec-list path is exactly what remora did before lockfiles existed. What it
// must never do is leave a stale lockfile in place — that would pin an old
// package set forever once the resolver stopped being available.
func resolveLock(dir string, m *manifest.Manifest, base, pm string) string {
	fallback := func(format string, args ...any) string {
		if format != "" {
			fmt.Fprintf(os.Stderr, "remora: "+format+"\n", args...)
		}
		if err := resolve.Clear(dir); err != nil {
			fmt.Fprintf(os.Stderr, "remora: could not remove a stale %s: %v\n", resolve.LockFile, err)
		}
		return ""
	}

	if len(m.Packages) == 0 {
		return fallback("")
	}
	if m.Lockfile != nil && !*m.Lockfile {
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

// exePath is the absolute path of the running remora binary, for the units
// to call back into. It falls back to a PATH lookup if the executable path
// cannot be determined.
func exePath() string {
	if p, err := os.Executable(); err == nil {
		if abs, err := filepath.Abs(p); err == nil {
			return abs
		}
	}
	return "remora"
}

func cmdInit(dir string) error {
	m, err := manifest.Load(dir)
	if os.IsNotExist(err) {
		m = &manifest.Manifest{Packages: []string{}}
		if err := m.Save(dir); err != nil {
			return err
		}
		fmt.Printf("created %s\n", manifest.Path(dir))
	} else if err != nil {
		return err
	}
	if err := regenerate(dir, m, false); err != nil {
		return err
	}
	if err := factory.InstallUnits(m, dir, "/", exePath(), func() error {
		return host.Systemctl("daemon-reload")
	}); err != nil {
		return err
	}
	fmt.Printf("installed %s and %s\n", factory.QuadletPath, factory.TimerPath)
	if host.UupdPresent() {
		if err := factory.InstallUupdHook("/"); err != nil {
			return err
		}
		if err := host.Systemctl("daemon-reload"); err != nil {
			return err
		}
		fmt.Printf("uupd detected — hooked rebuilds into it (%s);\n"+
			"uupd's schedule and reboot handling now drive updates, no separate timer needed\n",
			factory.UupdDropinPath)
	}
	fmt.Printf(`
Next steps:
  remora install <pkg>...   layer packages and rebuild
  remora enable             rebuild on remora's own timer (%s)
`, m.OnCalendar())
	return nil
}

func cmdModify(dir string, add, remove []string, noBuild bool) error {
	m, err := manifest.Load(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no manifest at %s — run `remora init` first", manifest.Path(dir))
		}
		return err
	}
	if len(add) == 0 && len(remove) == 0 {
		return fmt.Errorf("no packages given")
	}
	if added := m.AddPackages(add); len(added) > 0 {
		fmt.Println("added:", strings.Join(added, " "))
	}
	if removed := m.RemovePackages(remove); len(removed) > 0 {
		fmt.Println("removed:", strings.Join(removed, " "))
	}
	if err := m.Save(dir); err != nil {
		return err
	}
	if err := regenerate(dir, m, !noBuild); err != nil {
		return err
	}
	if noBuild {
		fmt.Println("manifest updated; run `remora build` to apply")
		return nil
	}
	return startBuild()
}

func cmdList(dir string) error {
	m, err := manifest.Load(dir)
	if err != nil {
		return err
	}
	if len(m.Packages) == 0 {
		fmt.Println("no layered packages")
		return nil
	}
	for _, p := range m.Packages {
		fmt.Println(p)
	}
	return nil
}

func cmdBuild(dir string, build bool) error {
	m, err := manifest.Load(dir)
	if err != nil {
		return err
	}
	if err := regenerate(dir, m, build); err != nil {
		return err
	}
	if !build {
		return nil
	}
	return startBuild()
}

func startBuild() error {
	fmt.Println("starting", factory.ServiceName, "(follow with: journalctl -fu "+factory.ServiceName+")")
	return host.Systemctl("start", factory.ServiceName)
}

func cmdShims(dir string, remove bool) error {
	m, err := manifest.Load(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	pm := ""
	if m != nil {
		pm = m.PackageManager
	}
	if pm == "" {
		p, err := host.DetectPM()
		if err != nil {
			return err
		}
		pm = p
	}
	if remove {
		removed, err := shim.Remove(shim.Dir, pm)
		for _, p := range removed {
			fmt.Println("removed", p)
		}
		return err
	}
	installed, err := shim.Install(shim.Dir, pm)
	for _, p := range installed {
		fmt.Println("installed", p)
	}
	if err == nil {
		fmt.Println("mutating package commands now redirect to remora; read-only ones pass through")
	}
	return err
}

func cmdStatus(dir string) error {
	if booted, err := host.BootedImage(); err == nil {
		fmt.Println("booted image: ", booted)
	} else {
		fmt.Println("booted image:  (not a bootc system?)")
	}
	m, err := manifest.Load(dir)
	if err != nil {
		fmt.Printf("manifest:      none (%v)\n", err)
		return nil
	}
	fmt.Println("local image:  ", m.ImageTag())
	fmt.Printf("packages:      %d layered\n", len(m.Packages))
	fmt.Println("schedule:     ", m.OnCalendar())
	_ = host.Systemctl("--no-pager", "status", factory.TimerName)
	return nil
}

// cmdApply rebases onto the locally built image — but only when doing so
// would actually change anything.
//
// Two things are going on here. First, the switch target carries a digest
// (tag@sha256:...) rather than a bare tag: handing bootc the same image
// specification on every run risks a rebuild that produced new content under
// an unchanged tag never being staged. Second, because the generated
// Containerfile is layered deterministically and podman builds it with
// --timestamp 0, a rebuild whose inputs did not change reproduces the same
// digest — so comparing against the staged/booted deployment turns the daily
// timer into a genuine no-op instead of a fresh deployment every night.
func cmdApply(dir string, apply bool, softReboot string) error {
	m, err := manifest.Load(dir)
	if err != nil {
		return err
	}
	tag := m.ImageTag()
	digest, err := host.LocalDigest(tag)
	if err != nil {
		return err
	}
	if digest == "" {
		return fmt.Errorf("no local image %s — run `remora build` first", tag)
	}
	current, err := host.StagedOrBootedDigest()
	if err != nil {
		return err
	}
	if current != "" && current == digest {
		fmt.Printf("%s is already staged or booted (%s) — nothing to do\n", tag, digest)
		return nil
	}
	ref := tag + "@" + digest
	fmt.Println("switching to", ref)
	return host.BootcSwitch(ref, apply, softReboot)
}

// cmdUpgrade refreshes the pinned base digest from the registry and, unless
// --no-build was given, rebuilds on top of it. This is the replacement for
// `bootc upgrade` on a remora-managed system: bootc cannot see past the
// local layer to the base image underneath, but remora tracks it explicitly.
func cmdUpgrade(dir string, noBuild bool) error {
	m, err := manifest.Load(dir)
	if err != nil {
		return err
	}
	old, err := resolveBase(dir, m)
	if err != nil {
		return err
	}
	name, _, _ := host.SplitDigest(old)
	digest, err := host.LatestDigest(name)
	if err != nil {
		return err
	}
	updated := name + "@" + digest
	if updated == old {
		fmt.Println("base image is up to date:", old)
	} else {
		if err := manifest.SaveBase(dir, updated); err != nil {
			return err
		}
		fmt.Printf("base image updated:\n  from %s\n  to   %s\n", old, updated)
	}
	// Regenerate even with --no-build. The pin just moved, so the
	// Containerfile's FROM is now stale, and the quadlet's ExecStartPre runs
	// exactly this command immediately before building — leaving the context
	// unregenerated would rebuild the old base forever. --no-build means
	// "do not start the build service", not "leave the context inconsistent".
	if err := regenerate(dir, m, true); err != nil {
		return err
	}
	if noBuild {
		return nil
	}
	return startBuild()
}

// cmdRebase points the manifest at a different base image and rebuilds. The
// ref is pinned to a digest, resolving it from the registry when the caller
// did not supply one.
func cmdRebase(dir string, args []string, noBuild bool) error {
	if len(args) != 1 {
		return fmt.Errorf("rebase takes exactly one image ref")
	}
	m, err := manifest.Load(dir)
	if err != nil {
		return err
	}
	pinned, err := host.PinBase(args[0])
	if err != nil {
		return err
	}
	m.Base, _, _ = host.SplitDigest(pinned)
	if err := m.Save(dir); err != nil {
		return err
	}
	if err := manifest.SaveBase(dir, pinned); err != nil {
		return err
	}
	fmt.Println("base image set to", pinned)
	// As in cmdUpgrade: the base changed, so the context must be rewritten
	// whether or not a build follows.
	if err := regenerate(dir, m, true); err != nil {
		return err
	}
	if noBuild {
		fmt.Println("run `remora build` to apply")
		return nil
	}
	return startBuild()
}
