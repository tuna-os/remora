# Proposal 0001 — remora on bases without a package manager

**Status**: draft, for discussion
**Targets**: [projectbluefin/dakota](https://github.com/projectbluefin/dakota),
[tuna-os/tromso](https://github.com/hanthor/tromso)
**Touches**: `internal/host` (detection), `internal/generate` (Containerfile),
`internal/resolve` (lockfile), `internal/shim`, a new artifact backend,
`ROADMAP.md` (#56, #55)

---

## The situation

Bluefin Dakota and Aurora Tromsø are assembled from source with BuildStream on
top of freedesktop-sdk and published directly as bootc images. They ship no
dnf, no rpm, no apt — the platform *is* the fdsdk runtime plus Flathub, and
Bluefin's documentation is explicit that there is no package layering on
Dakota the way there is on the RPM images. Dakota's own contributor guide goes
further and forbids "Containerfile package overlays, or post-build package
installation" as a way of building Dakota itself.

There is a second fact that matters at least as much, and it points somewhere
different from where this document originally went. **Dakota is already
building sysexts.** BuildStream produces sysext artifacts and bootc images
from the same element definitions, the project is experimenting with a pure
DDI Bluefin alongside the bootc one, and the stated long-horizon goal is for
Bluefin to become a systemd system extension deployable on stock GNOME OS —
`updatectl enable bluefin`, or a sysupdate `.transfer` file dropped into
`/etc/sysupdate.d`, layering the Bluefin experience without re-spinning an OS
image. Alpha 5 still ships the bootc image with Flatpak as the user-facing
story, so this is direction rather than delivery, but it is *funded*
direction in the build system today.

So the obvious reading is that remora has nothing to do here. That reading is
wrong, and it is worth being precise about why.

## The thesis: the package manager was never the point

Delete the `dnf -y install` line from a generated Containerfile and count what
remains standing:

- the digest-pinned `FROM`, so a rebuild a week later is the same rebuild;
- three separate `RUN` layers, so editing a script does not invalidate the
  expensive layer;
- `--timestamp 0` plus the scrub of build-varying state, so identical inputs
  produce an identical digest;
- an apply step that compares that digest against what is already live and
  does nothing when they match;
- the quadlet `.build` unit, the timer, and the uupd drop-in that makes the
  rebuild part of the system's normal update pass;
- `remora upgrade` moving the base pin on purpose rather than by drift;
- the rollback runbook.

Every one of those is package-manager agnostic. The package transaction is the
*only* part of remora that needs dnf, and it is one of three layers. What
remora actually sells is:

> a manifest, built reproducibly into an artifact, re-applied automatically,
> and kept in sync with a base image that moves underneath it.

Note what that sentence does **not** say: it does not say the artifact is a
derived bootc image. That is today's only backend, not the thesis.

Today a Dakota user who needs a file in `/usr` has three options: fork the
BuildStream project and become a distributor; abandon the requirement; or
reinstall. Flatpak, Homebrew and Distrobox — the answer Bluefin gives for
*applications* — cannot place a udev rule, a kernel module, a systemd unit, a
CA certificate, a VPN client that needs to exist before login, or a
`/usr/lib/firmware` blob. That gap is remora-shaped.

> **Framing that matters for adoption.** This is not a proposal to layer
> packages onto Dakota, and it should not be pitched upstream as one. It is a
> proposal to give Dakota users a *supported local derivation* with the same
> update and rollback semantics as the image itself. Dakota's rule is about
> how Dakota is built. remora operates one level down, on the user's machine,
> on an image Dakota already published.

## What actually breaks today

Concretely, on a host with no package-manager binary:

1. **`host.detectPM` fails.** With no binary on `PATH` it falls through to the
   os-release table, and Dakota's `ID` is not in it, so detection returns
   `could not detect package manager … set package_manager in remora.yaml`.
   `remora init` and `remora generate` stop there. Setting `package_manager`
   by hand does not help, because —
2. **`generate.Containerfile` rejects any name outside the six-entry `pms`
   map.** There is no legal value to write.
3. **`remora install PKG` has no referent.** The verb needs either a different
   unit of installation or an honest refusal.
4. **`remora shims` has nothing to shim** — `names(pm)` returns nil, so the
   subcommand is a no-op on exactly the systems where a confused user is most
   likely to type `dnf install`.

Nothing here is deep. The Containerfile generator already emits a perfectly
good PM-less build: layer 1 is the `system_files` overlay plus `extra_run`,
layer 3 is `build_files/*.sh`, and the final `bootc container lint` needs only
the `bootc` binary, which a bootc image has by definition. Two guard clauses
are standing between remora and a working Dakota story.

---

## Proposal: separate *providers* from *backends*

The manifest currently conflates two independent questions. Pulling them apart
is what makes the Dakota story fall out cleanly:

| Axis | Question | Values |
|---|---|---|
| **Provider** | Where does the content come from? | package manager · `oci_copy` · `flatpaks` · overlay + build scripts · `bst` |
| **Backend** | What artifact is produced and how is it applied? | derived bootc image (today) · **sysext/confext** |

They compose. `oci_copy` content can land in a derived image or in a sysext;
an overlay can feed either. Today remora has six providers and one backend,
and the backend is the part Dakota actually constrains.

### Backend A — derived bootc image (today's behaviour)

Unchanged. `podman build` → digest → `bootc switch --transport=containers-storage
tag@sha256:…` → reboot. Right when the customisation must be part of the
system's identity: visible in `bootc status`, rolled back by `bootc rollback`,
one digest describing the whole machine.

### Backend B — sysext (recommended default on Dakota)

Same manifest, same pinned base, same build; different artifact. remora builds
the content against the digest-pinned base, exports only the `/usr` and `/opt`
payload, stamps a generated
`/usr/lib/extension-release.d/extension-release.remora` with the host's `ID`,
`VERSION_ID` (or `SYSEXT_LEVEL`) and `ARCHITECTURE`, drops it in
`/var/lib/extensions/remora/`, and runs `systemd-sysext refresh`.

I originally listed sysext under "alternatives considered" and argued it
loses. That was reasoning in a vacuum against an upstream already moving
toward it, and on Dakota specifically it gets the two biggest things right:

- **The signing problem disappears.** Dakota is cosign-signed and its
  documented switch commands carry `--enforce-container-sigpolicy`. A locally
  built image is unsigned by construction, which makes backend A's viability
  on Dakota an open question (see Verification). Backend B never calls `bootc
  switch`, never rebases off the signed image, and never disturbs the
  composefs-sealed root. The signed system stays the signed system.
- **No reboot.** `systemd-sysext refresh` merges the overlay live. For
  configuration, units, certificates and userspace binaries — which is most of
  what this is for — the rebase-and-reboot cycle was always the heaviest part
  of the story.

And the argument *for remora* is stronger on this backend than on the other
one. A sysext's `extension-release` must match the host's `ID` and
`VERSION_ID` or `SYSEXT_LEVEL`, or systemd silently refuses to merge it.
Dakota's testing stream **publishes daily**. So a hand-built sysext stops
loading on the next base update, quietly, with no error the user will see
until something is missing. Keeping an extension rebuilt and re-stamped
against a base that moves underneath it is *precisely* the machine remora
already is: a pinned base, a timer, a rebuild, and an apply step that no-ops
when nothing changed. The version-matching treadmill is sysext's best-known
pain point and it is the one problem remora was built to solve.

There is a further strategic payoff. If Bluefin itself becomes a sysext on
GNOME OS, a remora-built user extension sits *alongside* the Bluefin
extension as a peer — same mechanism, same tooling, no special case. That is a
much better position than "we rebase you off the image upstream signed."

**Mechanics, concretely.** A final `FROM scratch` stage `COPY`s just the
payload paths plus the generated extension-release, so the exported tree is
deterministic and diffable. First implementation should use a **directory**
extension under `/var/lib/extensions/remora/` — systemd supports these
natively and it needs no `mkfs.erofs`, no loopback, no verity tooling. A
`.raw` DDI with dm-verity is the later, *signable* form and should not gate
phase one. The no-op invariant carries over intact: hash the exported tree,
skip the refresh when it is unchanged.

**Honest limits, which are real:**

- **`/usr` and `/opt` only.** `/etc` needs `systemd-confext`, a separate
  mechanism with its own extension-release. remora's `system_files/` overlay
  writes anywhere today, so this backend must *split* it — `/usr` and `/opt`
  to the sysext, `/etc` to a confext — and must **error** on paths it cannot
  place (`/var`, `/home`) rather than silently dropping them. This is the
  largest piece of design work in the proposal.
- **Not part of the system's identity.** `bootc status` does not see it,
  `bootc rollback` does not unwind it, and the digest that rolls back is not
  the digest that was running. Versioning and rollback become sysupdate's
  problem, not bootc's. This is a genuine cost, not a technicality — it is
  why backend A should stay, and stay the default everywhere else.
- **Unsigned extensions may still be refused** on a locked-down system via
  `systemd.image_policy` / `SYSEXT_IMAGE_POLICY`. Softer than backend A's
  version of this problem — a per-extension knob rather than the whole root —
  but not nothing.
- **No merge-time scriptlets.** `build_files/` still run at build time, which
  covers most of what scriptlets did.

### Providers

**`package_manager: none`** — make "no package manager" an expected answer
rather than an error.

- `detectPM` returns `"none"` instead of an error when neither a binary probe
  nor the os-release table identifies a package manager. The binary probe
  stays authoritative and the os-release table stays as the tie-breaker for
  derivatives; only the final `return "", fmt.Errorf(...)` changes. The
  diagnostic does not disappear — it moves to the point of use, where it can
  say something useful.
- `pms["none"]` is added with a nil `install`. Layers 1 and 3 render
  unchanged; the package layer is simply not emitted, exactly as it is not
  emitted today when `packages` is empty.
- `Manifest.Validate` rejects a non-empty `packages` list under `none` with a
  message naming the alternatives rather than a generic parse failure.
- `remora install` on a `none` base refuses the same way.

**On the CI smoke greps.** `ci.yml` asserts exactly three `<<'REMORA_EOF'`
heredocs, and `AGENTS.md` says to treat that grep as a spec. A `none` base
produces two. The invariant behind the grep is *"the overlay, the package
transaction, and the build scripts are separate layers"* — not *"there are
always three heredocs"*. Keep the existing three-heredoc assertion on its dnf
fixture untouched and **add** a `none` fixture asserting two heredocs and no
install command. That extends the spec; it does not loosen it.

**`oci_copy:`** — the container-native unit of installation is an image.

```yaml
package_manager: none
oci_copy:
  - image: ghcr.io/tuna-os/tailscale:1.80.0
    paths:
      - /usr/bin/tailscale
      - /usr/lib/systemd/system/tailscaled.service
```

→ `COPY --from=ghcr.io/tuna-os/tailscale@sha256:… /usr/bin/tailscale /usr/bin/tailscale`

remora resolves each tag to a digest before the build and records it in
`remora.lock.yaml` — the same file, the same `COPY`-hits-cache mechanism, the
same best-effort contract where a resolution failure falls back rather than
failing the build.

This one is worth arguing for independent of Dakota. The roadmap's standing
complaint (#56) is that only dnf has an instrument making a rebuild's cache key
the *resolved set* rather than the *spec list*, so on the other five families
the no-op-rebuild guarantee and the freshness guarantee are the same knob
turned opposite ways. A digest is a resolved set, exactly and by construction,
on every base — including ones with no package manager, no
`dnf5-plugin-manifest`, and no distribution cooperation of any kind.
`remora upgrade` re-resolves these pins alongside the base pin, so freshness
stays a deliberate act.

*Limit:* `COPY --from` does no dependency resolution, and remora cannot tell
you at generate time that a binary will fail to link. Documentation should be
blunt — static binaries, or binaries built against the same freedesktop-sdk
runtime the base provides. `bootc container lint` catches some shape problems;
it does not catch a missing `.so`.

**`flatpaks:`** — Dakota declares Flathub as its application platform and
ships Flatpak 1.18. A declared app list belongs in the image as a
`/usr/share/flatpak/preinstall.d/*.preinstall` drop-in written into the
overlay layer. remora writes a deterministic text file; it does not run
`flatpak install` inside a build container, which would need privileges the
build does not have and would bake per-machine state into the artifact.

**`bst:`** — the README already says `build_files/` is where a `bst build &&
bst artifact checkout` step "would go", and for Dakota and Tromsø that is the
native way to produce something that must be compiled against fdsdk. Pinning
the project to a commit preserves the digest invariant, and a warm remote
artifact cache means most elements are downloaded rather than compiled. Both
are *means*, not guarantees: a cache miss turns a nightly rebuild into a
multi-hour compile on a laptop, which is categorically different from
everything else remora does. **Recommend deferring** until someone has
measured a cold-cache build.

---

## Alternatives considered

**Fork the BuildStream project.** Upstream's answer, and correct for people who
intend to distribute an image. It turns a user who wants a VPN client into a
distributor with a CI pipeline. Not a customisation story.

**Flatpak / Homebrew / Distrobox alone.** Already the right answer for
applications, and remora should say so loudly rather than competing. They do
not touch `/usr`, boot-time state, kernel modules, or anything that must exist
before a user session. That remainder is the scope of this proposal and should
stay the scope.

**Sysext only — drop backend A.** Tempting once you accept the argument above,
and wrong. The image backend is the one that makes the customised system *be*
an image with a digest: `bootc status` sees it, `bootc rollback` unwinds it,
one identity describes the whole machine. That property is worth keeping where
it is available, which is everywhere remora runs today. Dakota is the case
where it may not be available; it is not the general case.

**`systemd-sysupdate` as the delivery mechanism** rather than remora's own
timer. Worth revisiting once Dakota's sysupdate story lands — if the system
already runs `updatectl`, remora hooking into it is the direct analogue of the
existing uupd drop-in, and the same argument applies (no linkage in either
direction, one `Wants=`/`After=`). Premature while it is still direction.

---

## What must be verified before any of this ships

The roadmap's own P0 is that **no build ever runs in CI** (#55) — every check
is a grep over generated text. Targeting bases that share nothing with the dnf
images makes that gap sharper. This should not merge past phase 1 on the
strength of reasoning alone.

1. **podman is present on Dakota and Tromsø.** remora builds locally; without
   a container runtime on the host there is no proposal, on either backend.
   First question to ask upstream, gates everything else. (Dakota targets
   cloud-native practitioners, so the expected answer is yes — expected is not
   verified.)
2. **Backend A: does signature policy refuse a local image?** Dakota is
   cosign-signed and its documented switch commands carry
   `--enforce-container-sigpolicy`. If the host policy rejects
   `bootc switch --transport=containers-storage`, backend A is unavailable on
   Dakota unless remora drops a `localhost/remora` policy entry at `init` —
   a deliberate, documented weakening that must be surfaced to the user, never
   done quietly. **This is the single finding that most changes the plan**: if
   it comes back "refused", backend B stops being the recommended option on
   Dakota and becomes the only one.
3. **Backend B: does an unsigned directory extension merge?** Check
   `systemd.image_policy` / `SYSEXT_IMAGE_POLICY` on a stock Dakota install,
   and confirm `systemd-sysext.service` is enabled.
4. **The no-op invariant holds on both backends.** Build twice, assert an
   identical digest (A) or an identical exported tree hash (B), assert the
   apply step skips. Never exercised against composefs-backed storage or a
   systemd-boot/UKI deployment.
5. **A boot test**, per backend: overlay plus one `oci_copy`, build, apply,
   confirm the file is live and the system is in the expected state.

Two limitations to document rather than solve: a layered **kernel module**
will not be in the UKI's initrd on either backend, so anything needed at early
boot is out of scope; and Dakota is compiled for **x86-64-v3**, so `oci_copy`
sources must be too.

## Recommended sequencing

**Phase 1 — `package_manager: none`.** Roughly a day: two guard clauses, a
validation message, a CI fixture. Unblocks both backends and, on its own,
makes remora a complete story for configuration, units, certificates, udev
rules and overlay files using `system_files/` and `build_files/` exactly as
they exist today. Ship with the teaching shims (below) and get it in front of
a Dakota alpha user.

**Phases 2 and 3 are independent and can go in either order.** If the goal is
Dakota, do the **sysext backend** first — it is what makes remora viable there
regardless of how verification item 2 lands, and it is aligned with where
upstream is going. If the goal is remora's existing six families, do
**`oci_copy`** first — it is the resolved-set cache key #56 has been waiting
for, on all six at once.

**Phases 4 and 5** (`flatpaks:`, then the bst bridge) follow real user demand.

## Open questions

- Is `package_manager: none` the right spelling, or does the manifest want an
  explicit `sources:` concept with package managers as one source among
  several? Once `oci_copy` and `flatpaks` exist, `package_manager` is a
  misnomer. `none` is the smaller change; a `sources:` redesign is more honest.
  Deliberately left open — nothing in phases 1–3 forecloses either.
- How is the backend selected? An explicit `backend: image | sysext` key is
  clearest, but a base with no package manager *and* an enforced signature
  policy has exactly one workable answer, so detection could pick it. Explicit
  with a sensible default is probably right; silent backend switching between
  rebuilds definitely is not.
- **Should the sysext backend be offered on the dnf bases too?** Nothing about
  it is Dakota-specific, and "add a binary to `/usr` without a reboot" is
  attractive on every image remora supports. Scope risk: it doubles the
  support surface. Recommend building it Dakota-first and deciding afterwards
  on evidence.
- Should `remora shims` install *teaching* shims on a PM-less base — a `dnf`
  that exists only to say "this system has no package manager, here is what to
  do instead"? The shim package already refuses to overwrite foreign files, so
  on a base with no real binary the shim is a pure explainer with nothing to
  pass through to. Low cost, catches the most likely first mistake a new
  Dakota user makes. Recommend yes, bundled with phase 1.
- Does upstream want to know? A local-derivation story upstream considers out
  of bounds is a worse outcome than one they endorse or tolerate — and the
  sysext backend is a far easier conversation to open than the rebase one.
  Worth raising in the Dakota repo before phase 2.
