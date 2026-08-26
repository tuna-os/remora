// Package generate renders the build context remora maintains under its
// state dir: a Containerfile derived from the manifest, per package manager.
// The generated file is overwritten on every build — users customize via the
// manifest, build_files/ scripts, and system_files/ overlay, never by editing
// the Containerfile.
//
// The Containerfile is deliberately split into three RUN layers — overlay +
// extra_run, then packages, then build scripts — so that editing a build
// script does not invalidate the expensive package layer. Combined with a
// digest-pinned FROM and `podman build --timestamp 0`, identical inputs
// produce an identical image digest, which is what lets remora skip the
// bootc switch when a scheduled rebuild changed nothing.
package generate

import (
	"fmt"
	"strings"

	"github.com/tuna-os/remora/internal/manifest"
)

// pm holds the per-package-manager fragments of the generated RUN script.
type pm struct {
	// cacheMounts are extra --mount flags on the package RUN line.
	cacheMounts []string
	// install renders the package installation command.
	install func(pkgs []string) string
	// installLock renders installation from a lockfile at path, when this
	// package manager has a resolver. nil means no lockfile support, and
	// generation always uses install.
	installLock func(path string) string
	// scrub lists paths to delete after the package transaction. Package
	// state that varies between otherwise-identical builds (logs, history
	// databases) would otherwise perturb the image digest and defeat the
	// no-op rebuild path. Cache directories mounted as type=cache are not
	// part of the layer; listing them here is harmless but redundant.
	scrub []string
}

// commonScrub is deleted after the package transaction regardless of package
// manager. Logs record timestamps and hostnames, so they differ on every
// build even when the installed package set is byte-identical.
var commonScrub = []string{"/var/log/*", "/var/tmp/*"}

var pms = map[string]pm{
	"dnf": {
		cacheMounts: []string{
			"--mount=type=cache,dst=/var/cache/libdnf5",
			"--mount=type=cache,dst=/var/cache/dnf",
		},
		install: func(pkgs []string) string {
			return "dnf -y install \\\n    " + strings.Join(pkgs, " \\\n    ")
		},
		installLock: func(path string) string {
			return "dnf5 manifest install --assumeyes --manifest " + path
		},
		// dnf's history database records transaction timestamps. The rpmdb
		// under /usr/lib/sysimage/rpm is deliberately NOT scrubbed — it is
		// the installed-package record, not a cache.
		scrub: []string{"/var/lib/dnf/history*"},
	},
	"zypper": {
		cacheMounts: []string{"--mount=type=cache,dst=/var/cache/zypp"},
		install: func(pkgs []string) string {
			return "zypper --non-interactive install --no-recommends \\\n    " + strings.Join(pkgs, " \\\n    ")
		},
	},
	"pacman": {
		cacheMounts: []string{"--mount=type=cache,dst=/var/cache/pacman/pkg"},
		install: func(pkgs []string) string {
			return "pacman -Sy --noconfirm --needed \\\n    " + strings.Join(pkgs, " \\\n    ")
		},
	},
	"apt": {
		cacheMounts: []string{
			"--mount=type=cache,dst=/var/cache/apt",
			"--mount=type=cache,dst=/var/lib/apt/lists",
		},
		install: func(pkgs []string) string {
			return "apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \\\n    " + strings.Join(pkgs, " \\\n    ")
		},
	},
	"portage": {
		// Distfiles + binpkgs survive across rebuilds; whether emerge
		// compiles or fetches binpkgs is the base image's binhost config.
		cacheMounts: []string{
			"--mount=type=cache,dst=/var/cache/distfiles",
			"--mount=type=cache,dst=/var/cache/binpkgs",
		},
		install: func(pkgs []string) string {
			return "emerge --verbose \\\n    " + strings.Join(pkgs, " \\\n    ")
		},
	},
	"apk": {
		cacheMounts: nil, // apk's fetch step is cheap; no cache dir by default
		install: func(pkgs []string) string {
			return "apk add --no-interactive \\\n    " + strings.Join(pkgs, " \\\n    ")
		},
	},
}

// SupportedPMs lists the package managers Containerfile generation handles.
func SupportedPMs() []string { return []string{"dnf", "zypper", "pacman", "apt", "portage", "apk"} }

// Containerfile renders the full Containerfile for m. base is the resolved
// base image ref (never empty — the caller resolves "follow booted image",
// and pins it to a digest where it can), pmName the resolved package manager.
//
// lock is the name of a lockfile in the build context, or "" for none. When
// set, the package layer COPYies it and installs from it, which is what makes
// podman's cache key the resolved package set rather than the spec list: an
// unchanged upstream yields a byte-identical lockfile, so the COPY hits cache
// and the whole rebuild is a no-op. The caller is responsible for the lockfile
// actually existing — see internal/resolve.
func Containerfile(m *manifest.Manifest, base, pmName, lock string) (string, error) {
	p, ok := pms[pmName]
	if !ok {
		return "", fmt.Errorf("unsupported package manager %q (supported: %s)",
			pmName, strings.Join(SupportedPMs(), ", "))
	}

	var b strings.Builder
	b.WriteString("# Generated by remora — DO NOT EDIT.\n")
	b.WriteString("# Customize via remora.yaml, build_files/*.sh, and system_files/ instead.\n\n")

	// Build context stage: scripts + overlay referenced without ending up
	// as layers in the final image (renner0e/server pattern).
	b.WriteString("FROM scratch AS ctx\n")
	b.WriteString("COPY build_files /build_files\n")
	b.WriteString("COPY system_files /system_files\n\n")

	fmt.Fprintf(&b, "FROM %s\n\n", base)

	ctxMount := "--mount=type=bind,from=ctx,source=/,target=/ctx"
	tmpMount := "--mount=type=tmpfs,dst=/tmp"

	// Layer 1: overlay + extra_run. Cheap and changes rarely.
	writeRun(&b, "System files overlay and extra_run", []string{ctxMount, tmpMount}, func(s *strings.Builder) {
		s.WriteString("if [ -d /ctx/system_files ] && [ -n \"$(ls -A /ctx/system_files 2>/dev/null)\" ]; then\n")
		s.WriteString("    cp -avf /ctx/system_files/. /\n")
		s.WriteString("fi\n")
		if len(m.ExtraRun) > 0 {
			s.WriteString("\n# extra_run (manifest)\n")
			for _, line := range m.ExtraRun {
				s.WriteString(line + "\n")
			}
		}
	})

	// Layer 2: the package transaction, on its own so that editing a build
	// script or the overlay does not invalidate it.
	useLock := lock != "" && p.installLock != nil
	if len(m.Packages) > 0 {
		const lockDst = "/run/remora"
		if useLock {
			b.WriteString("# Lockfile: the resolved package set. Its checksum is this layer's\n")
			b.WriteString("# cache key, so an unchanged resolution rebuilds to the same digest.\n")
			fmt.Fprintf(&b, "COPY %s %s/%s\n\n", lock, lockDst, lock)
		}
		mounts := append([]string{tmpMount}, p.cacheMounts...)
		writeRun(&b, "Packages (manifest)", mounts, func(s *strings.Builder) {
			if useLock {
				s.WriteString(p.installLock(lockDst+"/"+lock) + "\n")
			} else {
				s.WriteString(p.install(m.Packages) + "\n")
			}
			scrub := append(append([]string{}, p.scrub...), commonScrub...)
			if useLock {
				// The lockfile is build input, not image content.
				scrub = append([]string{lockDst}, scrub...)
			}
			s.WriteString("\n# Scrub build-varying state so identical inputs yield an identical digest.\n")
			s.WriteString("rm -rf " + strings.Join(scrub, " ") + "\n")
		})
	}

	// Layer 3: user build scripts.
	writeRun(&b, "User build scripts, in lexical order", []string{ctxMount, tmpMount}, func(s *strings.Builder) {
		s.WriteString("for script in /ctx/build_files/*.sh; do\n")
		s.WriteString("    [ -e \"$script\" ] || continue\n")
		s.WriteString("    echo \"remora: running $script\"\n")
		s.WriteString("    bash \"$script\"\n")
		s.WriteString("done\n")
	})

	b.WriteString("RUN bootc container lint\n")
	return b.String(), nil
}

// writeRun emits one heredoc RUN layer with the given mounts and body.
func writeRun(b *strings.Builder, comment string, mounts []string, body func(*strings.Builder)) {
	fmt.Fprintf(b, "# %s\n", comment)
	fmt.Fprintf(b, "RUN %s <<'REMORA_EOF'\n", strings.Join(mounts, " \\\n    "))
	b.WriteString("set -euo pipefail\n\n")
	body(b)
	b.WriteString("REMORA_EOF\n\n")
}
