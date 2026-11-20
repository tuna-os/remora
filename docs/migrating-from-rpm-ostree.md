# Migrating from rpm-ostree to remora

If you have been customizing an image-based Fedora system with
`rpm-ostree install`, remora does the same job a different way. This guide
maps the rpm-ostree concepts and commands you know onto remora's, and ends
with a checklist for moving an existing system over.

For install steps and the full command reference, see the
[README](../README.md). For recovery, see the
[rollback runbook](../runbooks/rollback-a-bad-remora-build.md).

## The mental model

| | rpm-ostree | remora |
|---|---|---|
| What you edit | Layered packages recorded in the deployment | A manifest, `/etc/remora/remora.yaml` |
| How a change is applied | rpm-ostree composes a new deployment locally | remora builds a local derived container image and `bootc switch`es to it |
| Where the base comes from | The booted image's ref, followed by tag | The booted image, **pinned to a digest** in `/etc/remora/base` |
| Keeping up with the base | `rpm-ostree upgrade` | A systemd timer rebuilds automatically; `remora upgrade` moves the pin |
| Package managers | rpm only | dnf, zypper, pacman, apt, portage (emerge) and apk bases |

The practical difference: with rpm-ostree, your layered package list is state
attached to a deployment. With remora it is a plain file you can read, edit,
copy to another machine, and keep in version control. The image is rebuilt
from that file whenever the base updates.

## Workflow changes

| You used to run | With remora |
|---|---|
| `rpm-ostree install htop` | `sudo remora install htop` |
| `rpm-ostree uninstall htop` | `sudo remora remove htop` |
| `rpm-ostree status` (layered packages) | `remora list`, `remora status` |
| `rpm-ostree upgrade` | `sudo remora upgrade` (or let the timer do it) |
| `rpm-ostree rebase <image>` | `sudo remora rebase <image>` |
| `rpm-ostree rollback` | See [Rollback](#rollback) |

Notes:

- `remora install` edits the manifest, rebuilds the image, and rebases. Pass
  `--no-build` to edit the manifest only, and run `sudo remora build` later.
- A rebuild runs in `remora-build.service`. `remora build` starts the unit and
  returns, so follow it with `journalctl -fu remora-build.service`.
- Reboot to enter the new deployment, as with rpm-ostree. `remora apply`
  accepts `--apply` and `--soft-reboot auto|required` to pass through to
  `bootc switch`.
- Once remora owns the system, use `remora upgrade`, not `bootc upgrade`.
  `bootc upgrade` cannot see through the local `localhost/remora:latest`
  image to the base underneath it.
- remora can install shims for the package manager so a habitual
  `dnf install foo` explains the situation and offers to run
  `remora install foo`. Enable them with `sudo remora shims`.

## State: where things live

Everything remora owns is under `/etc/remora/`:

| Path | Contents |
|---|---|
| `remora.yaml` | The manifest: packages, `extra_run`, image tag, schedule |
| `base` | The pinned base image digest the build is `FROM` |
| `base.previous` | The pin before the last `upgrade` or `rebase` (one generation) |
| `build_files/*.sh` | Scripts run in lexical order at the end of the build |
| `system_files/` | An overlay copied onto `/` verbatim |
| `Containerfile` | Generated. Never edit it by hand; it is overwritten |

To inspect the current state:

```bash
remora status            # booted image, manifest summary, timer state
remora list              # layered packages
cat /etc/remora/base     # the digest the build starts from
sudo bootc status        # staged and booted deployments
```

## Common patterns

### Layer a few packages

```bash
sudo remora install htop vim
```

or put them in the manifest:

```yaml
packages:
  - htop
  - vim
```

### Add a third-party repository

rpm-ostree users drop a `.repo` file into `/etc/yum.repos.d/` before
installing. In remora, add the repository from `extra_run`, which runs
**before** the package install:

```yaml
packages:
  - tailscale
extra_run:
  - dnf config-manager addrepo --from-repofile=https://pkgs.tailscale.com/stable/fedora/tailscale.repo
```

### Enable a service or change a config file

Layered packages that need a service enabled used to be a manual
`systemctl enable` after reboot. Put it in a build script instead:

```bash
#!/usr/bin/env bash
# /etc/remora/build_files/10-enable-podman.sh
set -euo pipefail
systemctl enable podman.socket
```

Files that belong in `/etc` or elsewhere go in the overlay, for example
`/etc/remora/system_files/etc/sysctl.d/99-tuning.conf` lands at
`/etc/sysctl.d/99-tuning.conf`.

## Digest pinning

rpm-ostree follows the tag of the image you booted. When the tag moves, your
next `rpm-ostree upgrade` moves with it, and two machines upgraded a week
apart can end up on different bases.

remora records the base as a digest in `/etc/remora/base`. A tag moves; a
digest does not. Two rebuilds a week apart produce the same image unless
`remora upgrade` moved the pin on purpose. This is also what makes rebuilds
cheap: builds are reproducible (`--timestamp 0`, a deterministic
Containerfile), and `remora apply` rebases only when the built image's digest
changed, so a nightly rebuild with nothing new does not create a new
deployment.

For dnf bases, remora also resolves your package list to an exact set in
`remora.lock.yaml`, so an unchanged resolution gives a byte-identical layer.
It is best-effort and falls back to a plain install when it cannot run. See
the README's Lockfiles section.

## Rollback

rpm-ostree keeps deployments and `rpm-ostree rollback` swaps back to the
previous one. On a remora system, `bootc rollback` does the same swap, but it
**does not stick** by itself: it does not change `/etc/remora`, so the next
scheduled rebuild builds the same manifest from the same base and stages the
broken image again.

The short version:

1. `sudo remora disable` to stop the timer (and the uupd hook, if present).
2. `sudo bootc rollback`, then reboot.
3. Undo the bad input. For a base that moved, `sudo remora rebase --previous
   --no-build` restores `/etc/remora/base.previous`. For a package, use
   `sudo remora remove PKG --no-build`.
4. `sudo remora build`, verify, then `sudo remora enable`.

The full procedure, including recovery when the system will not boot, is in
[runbooks/rollback-a-bad-remora-build.md](../runbooks/rollback-a-bad-remora-build.md).

## Migration checklist

1. **Record what you have.** Run `rpm-ostree status` and note the layered
   packages. Copy any custom `.repo` files you added under
   `/etc/yum.repos.d/`.
2. **Confirm you are on a bootc image.** remora rebases with `bootc switch`,
   so the host must be bootc-managed. Check with `sudo bootc status`.
3. **Install remora.** Follow the README's Installation section, which
   verifies `checksums.txt` before installing.
4. **Initialize.** `sudo remora init` creates `/etc/remora`, the quadlet and
   the timer.
5. **Write the manifest.** Run `sudo remora install --no-build PKG...` for your
   layered packages, or edit `/etc/remora/remora.yaml`. Move repository setup
   into `extra_run`, service enablement into `build_files/*.sh`, and config
   files into `system_files/`.
6. **Remove the old layers.** Reset your rpm-ostree layers so the same
   packages are not installed twice, and reboot once to confirm the system is
   healthy before continuing.
7. **Build and apply.** `sudo remora build`, watch
   `journalctl -fu remora-build.service`, then reboot into the new
   deployment.
8. **Verify.** `remora status`, `remora list`, and `sudo bootc status`.
9. **Turn on automation.** `sudo remora enable`. Optionally `sudo remora shims`.
10. **Rehearse the rollback** once, using the runbook, while the system is
    healthy.

## Not covered by remora

remora layers packages, files and build scripts. The README does not document
equivalents for other rpm-ostree features such as `override replace`, kernel
arguments, or initramfs regeneration. If you rely on those, express them in a
`build_files/*.sh` script or check the issue tracker before migrating.
