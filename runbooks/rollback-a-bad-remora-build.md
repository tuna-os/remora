# Runbook: roll back a bad remora build

**Applies to**: any host where `remora init` has been run — including every
TunaOS image, which ships remora preinstalled.

**Use this when** a rebuild has left the system in a worse state than before:
a package added to `remora.yaml` breaks the desktop, a `build_files/` script
misbehaves, or `remora upgrade` moved the base pin onto a base image that is
broken (see #44 for a live example of a base-dependent build defect).

The important thing to understand first: **`bootc rollback` alone is not
enough on a remora-managed system.** Rolling back changes which deployment
boots, but it does not change any remora state. The next rebuild — from
`remora-build.timer`, or from uupd on hosts where remora hooked itself into
it — rebuilds from the same manifest and the same pinned base, and its
`ExecStartPost=remora apply` stages that image again. Stopping the automation
is therefore step 1, not an afterthought.

---

## Step 1 — stop the automation

Do this before anything else, so a scheduled rebuild cannot undo the recovery
while it is in progress.

```bash
sudo remora disable
```

That disables `remora-build.timer`. **If uupd is installed on this host, that
is not sufficient.** `remora init` writes a drop-in that makes every uupd run
pull in the rebuild:

```bash
sudo systemctl cat -- uupd.service | grep -A2 'remora'
```

If `/etc/systemd/system/uupd.service.d/10-remora.conf` exists, uupd still
drives rebuilds on its own schedule regardless of the timer's state. Move it
aside and reload:

```bash
sudo mv /etc/systemd/system/uupd.service.d/10-remora.conf /root/10-remora.conf.disabled
sudo systemctl daemon-reload
```

Confirm nothing is left armed:

```bash
systemctl is-enabled remora-build.timer        # expect: disabled
systemctl is-active  remora-build.service      # expect: inactive
systemctl cat -- uupd.service | grep -c remora # expect: 0
```

Restoring the drop-in at the end (step 5) is what re-enables the uupd path.

## Step 2 — get back onto a working system

Two options, depending on how bad the damage is.

**Option A — back to the previous deployment** (the system booted fine before
the last rebase):

```bash
sudo bootc rollback
sudo systemctl reboot
```

**Option B — back to the plain base image**, abandoning the local layer
entirely (the local layer itself is what is broken, or several bad rebuilds
have stacked up):

```bash
cat /etc/remora/base            # the pinned base ref remora last built FROM
sudo bootc rebase <that ref>    # e.g. quay.io/fedora/fedora-bootc@sha256:...
sudo systemctl reboot
```

Option B gives up all layered packages until the build is fixed. That is the
correct trade when the machine is someone's only workstation.

## Step 3 — work out which input broke it

Recovery is not finished until the bad input is identified, because step 5
re-arms the same build.

```bash
journalctl -u remora-build.service -b -1 --no-pager   # the failed/bad build
cat /etc/remora/remora.yaml                           # packages, extra_run
cat /etc/remora/base                                  # current base pin
ls /etc/remora/build_files/                           # custom scripts
git -C /etc/remora log -1 2>/dev/null || true         # if the dir is tracked
```

The three inputs that can change under you, in rough order of likelihood:

| Input | Changed by | Symptom |
|---|---|---|
| Base image pin | `remora upgrade` (also run as the quadlet's `ExecStartPre`) | Build worked yesterday, no local edits |
| Package set | `remora install` / `remove` | Broke right after a package change |
| Build scripts / overlay | Hand edits under `/etc/remora` | Broke right after an edit |

Note that `remora build` and `remora apply` will happily stage an image built
from a *successful* podman build that is nonetheless broken at runtime. A
green `remora-build.service` does not clear the build as the cause.

## Step 4 — undo the bad input

**Base pin moved.** `remora upgrade` overwrites `/etc/remora/base` in place
and keeps no history, so the previous known-good digest is not recorded
anywhere on the host. Recover it from the deployment you rolled back to:

```bash
sudo bootc status --json | jq -r '.status.rollback.image.image.image, .status.rollback.image.imageDigest'
```

On a remora-managed system that reports the *local* image, not the base, so
it only helps when the rollback deployment predates remora's first rebase.
Otherwise recover the digest from wherever the base is published (the
registry's tag history, or the TunaOS image build that produced it), then pin
back to it explicitly:

```bash
sudo remora rebase <base>@sha256:<known-good-digest> --no-build
```

Passing an explicit `@sha256:` digest is what makes this a pin rather than a
re-resolve — `remora rebase` on a bare tag resolves whatever the tag points at
now, which is the thing that just broke.

**Package or script.** Remove the offending package (`sudo remora remove PKG
--no-build`) or revert the script, then rebuild in the foreground and watch it:

```bash
sudo remora build
sudo journalctl -fu remora-build.service
```

**To avoid needing this section next time**, record the pin whenever the
system is known good — for example before an upgrade:

```bash
sudo cp /etc/remora/base /etc/remora/base.last-good
```

## Step 5 — verify, then re-arm

Verify *before* re-enabling the automation, not after:

```bash
remora status                    # booted image, manifest, timer state
sudo remora build                # foreground rebuild, must exit 0
sudo bootc status                # staged/booted deployments look right
```

Reboot and use the machine normally before re-arming. Then:

```bash
sudo remora enable
# and, if it was moved aside in step 1:
sudo mv /root/10-remora.conf.disabled /etc/systemd/system/uupd.service.d/10-remora.conf
sudo systemctl daemon-reload
```

---

## Reference: where the state lives, and what expires

| Path / object | What it is | Lifetime |
|---|---|---|
| `/etc/remora/remora.yaml` | Manifest: packages, `extra_run`, image tag, schedule | Persistent |
| `/etc/remora/base` | Pinned base ref (`image@sha256:…`) actually built FROM | **Overwritten in place** by `remora upgrade` / `rebase` — no history |
| `/etc/remora/Containerfile` | Generated; rewritten on every regenerate | Regenerated |
| `/etc/remora/build_files/`, `system_files/` | Custom scripts and file overlay | Persistent |
| `/etc/containers/systemd/remora.build` | Quadlet build unit | Rewritten by `remora init` |
| `/etc/systemd/system/remora-build.timer` | Rebuild schedule (default `*-*-* 04:00:00`) | Rewritten by `remora init` |
| `/etc/systemd/system/uupd.service.d/10-remora.conf` | uupd hook — **not** removed by `remora disable` | Written by `remora init` when uupd is present |
| `localhost/remora:latest` in podman storage | The current local build | Tag moves to each new build |
| Older local builds (untagged, in podman storage) | Previous images, addressable by digest | **Pruned after 168h** by the quadlet's `ExecStartPost=podman image prune --filter=label=containers.bootc=1 --filter=until=168h` |

Two consequences worth internalizing:

- Re-staging a specific older *local* build by digest
  (`sudo bootc switch --transport=containers-storage localhost/remora@sha256:…`)
  only works inside that 7-day window. bootc's own deployment storage is
  separate and is not what the prune touches, so `bootc rollback` itself is
  unaffected — but a rebuild from three weeks ago is simply gone.
- Nothing on the host records *why* a rollback happened. Leave a note (an
  issue, or a comment at the top of `remora.yaml`) when a base digest is
  pinned back deliberately, or the next `remora upgrade` will walk straight
  onto it again.

## Escalation

If the system will not boot the current deployment, recovery is a bootc/ostree
problem rather than a remora one: interrupt the bootloader and select the
previous entry, then start again from step 1.

If no entry boots, the machine needs to be recovered from live media. Mount
the root filesystem and copy `/etc/remora` off it before reinstalling — the
manifest, the base pin, and any `build_files/` scripts all live there, and
they are the only record of what the machine was configured to build.
