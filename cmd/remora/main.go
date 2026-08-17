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
	"strings"

	"github.com/tuna-os/remora/internal/factory"
	"github.com/tuna-os/remora/internal/host"
	"github.com/tuna-os/remora/internal/manifest"
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
  enable               Enable the automatic rebuild timer
  disable              Disable the automatic rebuild timer
  status               Show booted image, manifest, and timer state
  generate             Regenerate the Containerfile only (no build)
  shims                Install package-manager shims that redirect
                       'dnf install' etc. to remora (--remove to uninstall)

Flags:
  --dir DIR            State directory (default /etc/remora)
  --no-build           With install/remove: update the manifest only
  --remove             With shims: uninstall the shims

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

// regenerate resolves base + package manager and rewrites the build context.
func regenerate(dir string, m *manifest.Manifest) error {
	base := m.Base
	if base == "" {
		b, err := host.BootedImage()
		if err != nil {
			return err
		}
		base = b
	}
	pm := m.PackageManager
	if pm == "" {
		p, err := host.DetectPM()
		if err != nil {
			return err
		}
		pm = p
	}
	if err := factory.WriteContext(dir, m, base, pm); err != nil {
		return err
	}
	fmt.Printf("generated %s/Containerfile (base=%s, pm=%s, %d packages)\n",
		dir, base, pm, len(m.Packages))
	return nil
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
	if err := regenerate(dir, m); err != nil {
		return err
	}
	if err := factory.InstallUnits(m, dir, "/", func() error {
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
	if err := regenerate(dir, m); err != nil {
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
	if err := regenerate(dir, m); err != nil {
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
