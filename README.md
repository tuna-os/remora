# remora 🐟

**User-friendly local layering for bootc systems, the container-native way.**

A remora rides along on a bigger fish. This remora rides along on your bootc
base image: it keeps a small manifest of packages and customizations, builds
them into a local derived image, and rebases your system to it — rebuilding
automatically whenever the base updates. It is the container-native answer to
`rpm-ostree install`, and it works on **dnf, zypper, pacman, and apt** bases.

Based on the local image factory pattern from
[renner0e/server](https://github.com/renner0e/server),
[zerolayer](https://github.com/akdev1l/zerolayer), and the Universal Blue
["locally built, automatically updating custom bootc image"](https://universal-blue.discourse.group/t/locally-built-automatically-updating-custom-bootc-image/11706)
thread.

## How it works

```
remora.yaml ──► Containerfile ──► podman quadlet (.build, Pull=newer)
                                        │ on schedule / on demand
                                        ▼
                          localhost/remora:latest
                                        │ ExecStartPost
                                        ▼
              bootc switch --transport=containers-storage
```

- Your customizations live in `/etc/remora/`: a YAML manifest, optional
  `build_files/*.sh` scripts, and a `system_files/` overlay copied over `/`.
- remora generates a Containerfile from them (never edit it by hand) with
  per-package-manager cache mounts and a final `bootc container lint`.
- A podman-systemd `.build` unit rebuilds with `Pull=newer` — new base
  published upstream ⇒ your layers are rebuilt on top of it — then
  `bootc switch`es to the result. Reboot into the update as usual.

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
package_manager: ""       # empty = auto-detect (dnf | zypper | pacman | apt)
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
| `remora enable` / `disable` | Toggle the automatic rebuild timer |
| `remora status` | Booted image, manifest summary, timer state |
| `remora generate` | Regenerate the Containerfile only |
| `remora shims [--remove]` | Install/remove package-manager interception |

`--no-build` on install/remove edits the manifest without triggering a build;
`--dir` overrides the state directory (useful for tests/CI).

## Building

```bash
go build ./cmd/remora
go test ./...
```

## License

Apache-2.0
