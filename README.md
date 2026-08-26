# remora 🐟

**User-friendly local layering for bootc systems, the container-native way.**

A remora rides along on a bigger fish. This remora rides along on your bootc
base image: it keeps a small manifest of packages and customizations, builds
them into a local derived image, and rebases your system to it — rebuilding
automatically whenever the base updates. It is the container-native answer to
`rpm-ostree install`, and it works on **dnf, zypper, pacman, apt, portage (emerge), and apk** bases.

Based on the local image factory pattern from
[renner0e/server](https://github.com/renner0e/server),
[zerolayer](https://github.com/akdev1l/zerolayer), and the Universal Blue
["locally built, automatically updating custom bootc image"](https://universal-blue.discourse.group/t/locally-built-automatically-updating-custom-bootc-image/11706)
thread.

## How it works

```
remora.yaml + base pin ──► Containerfile ──► podman quadlet (.build)
                                                  │ on schedule / on demand
                                                  ▼
                                    localhost/remora:latest
                                                  │ ExecStartPost: remora apply
                                                  ▼
                    bootc switch localhost/remora:latest@sha256:...
                              (skipped when the digest did not change)
```

- Your customizations live in `/etc/remora/`: a YAML manifest, optional
  `build_files/*.sh` scripts, and a `system_files/` overlay copied over `/`.
- remora generates a Containerfile from them (never edit it by hand) with
  per-package-manager cache mounts and a final `bootc container lint`. The
  overlay, the package transaction, and your build scripts each get their own
  layer, so editing a script does not invalidate the package install.
- The base image is **pinned to a digest** in `/etc/remora/base`. A tag moves
  under you; a digest does not, so two rebuilds a week apart produce the same
  image unless `remora upgrade` moved the pin on purpose.
- A podman-systemd `.build` unit refreshes the pin, rebuilds with
  `--timestamp 0`, and hands off to `remora apply`, which rebases only when
  the built image's digest actually changed. Reboot into the update as usual.

### Why the digest matters

Two details are easy to get wrong and both are load-bearing:

- **Switching to a bare tag is not enough.** `bootc switch
  localhost/remora:latest` hands bootc the same image specification on every
  run, so a rebuild that produced new content under an unchanged tag risks
  never being staged. remora resolves the freshly built digest and switches to
  `tag@sha256:...`.
- **Reproducible layers make the no-op path real.** `--timestamp 0` plus a
  deterministic Containerfile and a scrub of build-varying state (logs, dnf
  history) means identical inputs produce an identical digest. `remora apply`
  compares that against the staged/booted deployment and does nothing when
  they match — so the nightly timer stops minting a fresh deployment every
  night just to install the same packages again.

## Installation

Remora publishes standalone Linux binaries for amd64 and arm64 on the
[Releases](https://github.com/tuna-os/remora/releases/latest) page. Download
the binary and checksum file for the latest release, verify the download, and
install it in `/usr/local/bin`:

Run this as a script (`bash install-remora.sh`) rather than pasting it line by
line, so that a failed download or a failed checksum stops it before the
`sudo install`:

```bash
#!/usr/bin/env bash
set -euo pipefail

arch=$(uname -m)
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
esac

# -f makes an HTTP error a non-zero exit instead of a saved error page.
curl -fLO "https://github.com/tuna-os/remora/releases/latest/download/remora-linux-${arch}"
curl -fLO "https://github.com/tuna-os/remora/releases/latest/download/checksums.txt"

# Verify BEFORE installing, and stop here if the hash does not match. remora
# runs as root, installs shims into /usr/local/bin ahead of /usr/bin on PATH,
# and owns a timer that rebases the host image — a binary that failed this
# check must never reach /usr/local/bin.
sha256sum --check --ignore-missing checksums.txt

sudo install -m 0755 "remora-linux-${arch}" /usr/local/bin/remora
rm "remora-linux-${arch}" checksums.txt
```

`checksums.txt` is served from the same place as the binary, so it proves the
download was not corrupted in transit — it is not a signature and cannot tell
a genuine release from a tampered one. Signed releases are tracked in
[#19](https://github.com/tuna-os/remora/issues/19).

To build from source instead, install the Go version declared in `go.mod`,
clone this repository, and run:

```bash
go build -o remora ./cmd/remora
sudo install -m 0755 remora /usr/local/bin/remora
```

## Quickstart

```bash
sudo remora init                 # set up /etc/remora + quadlet + timer
sudo remora install htop vim     # layer packages, rebuild, rebase
sudo remora enable               # rebuild automatically (default: daily 04:00)
remora status
```

## Manifest (`/etc/remora/remora.yaml`)

```yaml
base: ""                  # empty = follow the booted image
package_manager: ""       # empty = auto-detect (dnf | zypper | pacman | apt | portage | apk)
packages:
  - htop
  - tailscale
extra_run:                # verbatim shell, runs BEFORE package install —
  - dnf config-manager addrepo --from-repofile=https://pkgs.tailscale.com/stable/fedora/tailscale.repo
image: localhost/remora:latest
schedule: "*-*-* 04:00:00"
```

- **`build_files/*.sh`** run in lexical order at the end of the build —
  enable services, tweak configs, or call exotic builders (this is where a
  [BuildStream](https://buildstream.build/) `bst build && bst checkout` step
  would go; remora doesn't wrap bst, it just gives it a home).
- **`system_files/`** is copied onto `/` verbatim (e.g.
  `system_files/etc/sysctl.d/99-tuning.conf`).

## Package-manager shims

```bash
sudo remora shims
```

installs interception shims in `/usr/local/bin` (machine-local and writable
on bootc systems, ahead of `/usr/bin` in PATH). `dnf install foo` — which
could never work against a read-only `/usr` anyway — now explains what's
going on and offers to run `remora install foo` for you. Read-only
subcommands (`search`, `info`, `pacman -Q`, …) pass through to the real
binary. `sudo remora shims --remove` uninstalls; foreign files are never
overwritten or deleted.

## uupd integration (zero dependencies)

If [uupd](https://github.com/ublue-os/uupd) is present, `remora init` drops
in a plain systemd override so every uupd run rebuilds the local image
*first* — uupd's own gating (battery, metered network, inhibitors) and
notification/reboot handling then apply the fresh build. No remora timer
needed on those systems, and no linkage in either direction: it's one
`Wants=`/`After=` drop-in.

## Commands

| Command | Effect |
|---|---|
| `remora init` | Create `/etc/remora`, install quadlet + timer (+ uupd hook) |
| `remora install PKG...` | Add to manifest, rebuild, rebase |
| `remora remove PKG...` | Remove from manifest, rebuild, rebase |
| `remora list` | Show layered packages |
| `remora build` | Rebuild + rebase now |
| `remora apply` | Rebase to the built image, but only if its digest changed |
| `remora upgrade` | Refresh the pinned base digest, then rebuild + rebase |
| `remora rebase IMAGE` | Point the manifest at a new base image, then rebuild |
| `remora enable` / `disable` | Toggle the automatic rebuild timer |
| `remora status` | Booted image, manifest summary, timer state |
| `remora generate` | Regenerate the Containerfile only |
| `remora shims [--remove]` | Install/remove package-manager interception |

`--no-build` on install/remove/upgrade/rebase edits state without triggering a
build; `--dir` overrides the state directory (useful for tests/CI).
`--apply` and `--soft-reboot auto|required` on `apply`/`build` pass through to
`bootc switch` to reboot immediately.

### `remora upgrade`, not `bootc upgrade`

Once remora has rebased the system onto its local image, `bootc upgrade` no
longer knows how to update you — it sees `containers-storage:localhost/remora:latest`
and cannot recognize the base image underneath the local layer. Use
`remora upgrade`, which refreshes the pinned base digest and rebuilds on top
of it. To go back to a system managed by bootc alone, run
`bootc rebase <base image>` and `remora disable`.

## Building

```bash
go build ./cmd/remora
go test ./...
```

## License

Apache-2.0
