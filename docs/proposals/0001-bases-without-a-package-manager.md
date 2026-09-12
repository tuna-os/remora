# Proposal 0001 — remora on bases without a package manager

**Status**: draft, for discussion
**Targets**: [projectbluefin/dakota](https://github.com/projectbluefin/dakota),
[tuna-os/tromso](https://github.com/hanthor/tromso)
**Touches**: `internal/host` (detection), `internal/generate` (Containerfile),
`internal/resolve` (lockfile), `internal/shim`, `ROADMAP.md` (#56, #55)

---

## The situation

Bluefin Dakota and Aurora Tromsø are assembled from source with BuildStream on
top of freedesktop-sdk and published directly as bootc images. They ship no
dnf, no rpm, no apt — the platform *is* the fdsdk runtime plus Flathub, and
Bluefin's documentation is explicit that there is no package layering on
Dakota the way there is on the RPM images. Dakota's own contributor guide goes
further and forbids "Containerfile package overlays, or post-build package
installation" as a way of building Dakota itself.

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
- `remora apply` comparing that digest against the staged deployment and doing
  nothing when they match;
- the quadlet `.build` unit, the timer, and the uupd drop-in that makes the
  rebuild part of the system's normal update pass;
- `remora upgrade` moving the base pin on purpose rather than by drift;
- `bootc switch tag@sha256:…` and the rollback runbook behind it.

Every one of those is package-manager agnostic. The package transaction is the
*only* part of remora that needs dnf, and it is one of three layers. What
remora actually sells is a durable, reproducible, automatically-rebuilt local
derivation of an image the user cannot otherwise modify — and on Dakota and
Tromsø, that is not a nice-to-have, it is the only mechanism of its kind.

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

## Proposal

Four phases, independently shippable, in value order. Phase 1 is small and
unblocking; phase 2 is where the strategic value is and pays back on the six
package-manager families remora already supports.

### Phase 1 — `package_manager: none`

Make "this base has no package manager" a first-class, *expected* answer
rather than an error.

- `detectPM` returns `"none"` instead of an error when neither a binary probe
  nor the os-release table identifies a package manager. The binary probe
  stays authoritative and the os-release table stays as the tie-breaker for
  derivatives; only the final `return "", fmt.Errorf(...)` changes. The
  diagnostic does not disappear — it moves to the point of use, where it can
  say something useful.
- `pms["none"]` is added to the generator with a nil `install`. Layers 1 and 3
  render unchanged; the package layer is simply not emitted, exactly as it is
  not emitted today when `packages` is empty.
- `Manifest.Validate` rejects a non-empty `packages` list under `none` with a
  message that names the alternatives (`oci_copy`, `flatpaks`, `build_files`,
  `extra_run`) rather than a generic parse failure.
- `remora install` on a `none` base refuses the same way, and suggests
  `remora install --from-image` (phase 2) or the manifest.

**On the CI smoke greps.** `ci.yml` asserts exactly three `<<'REMORA_EOF'`
heredocs, and `AGENTS.md` says to treat that grep as a spec. A `none` base
produces two. The invariant behind the grep is *"the overlay, the package
transaction, and the build scripts are separate layers"* — not *"there are
always three heredocs"*. The right change is to keep the existing three-heredoc
assertion on its dnf fixture untouched and **add** a `none` fixture asserting
two heredocs and no install command. That extends the spec; it does not
loosen it.

### Phase 2 — `oci_copy:`, and a lockfile that works everywhere

The container-native unit of installation is an image, not a package.

```yaml
package_manager: none
oci_copy:
  - image: ghcr.io/tuna-os/tailscale:1.80.0
    paths:
      - /usr/bin/tailscale
      - /usr/bin/tailscaled
      - /usr/lib/systemd/system/tailscaled.service
```

renders into the package-layer slot as:

```dockerfile
COPY --from=ghcr.io/tuna-os/tailscale@sha256:… /usr/bin/tailscale /usr/bin/tailscale
```

remora resolves each `image:` tag to a digest before the build and records it
in `remora.lock.yaml` — the same file, the same `COPY`-hits-cache mechanism,
the same contract that resolution is best-effort and a failure falls back
rather than failing the build.

This is the part worth arguing for on its own merits, independent of Dakota.
The roadmap's standing complaint (#56) is that only dnf has an instrument that
makes a rebuild's cache key the *resolved set* rather than the *spec list*,
so on the other five families the no-op-rebuild guarantee and the freshness
guarantee are the same knob turned opposite ways. A digest is a resolved set,
exactly and by construction, on every base including ones with no package
manager at all. `oci_copy` resolution is a stronger version of the dnf
resolver that happens to need no resolver in the base image, no
`dnf5-plugin-manifest`, and no distribution cooperation of any kind.

`remora upgrade` re-resolves these pins alongside the base pin, so freshness
stays a deliberate act.

**Honest limits.** `COPY --from` does no dependency resolution. A binary that
links against libraries the base does not carry will not run, and remora
cannot tell you that at generate time. Documentation should be blunt: use
static binaries, or binaries built against the same freedesktop-sdk runtime
the base provides. `bootc container lint` catches some shape problems; it does
not catch a missing `.so`. A `ldd`-in-the-build-container check in the
generated script is a possible later refinement, not a phase-2 requirement.

### Phase 3 — `flatpaks:` via preinstall drop-ins

Dakota declares Flathub as its application platform and ships Flatpak 1.16.6,
which supports `/usr/share/flatpak/preinstall.d/*.preinstall`. So a declared
app list belongs in the image as a preinstall drop-in, written into the
overlay layer:

```yaml
flatpaks:
  - org.mozilla.firefox
  - com.visualstudio.code
```

This keeps the build hermetic and reproducible — remora writes a deterministic
text file, it does not run `flatpak install` inside a build container, which
would need privileges the build does not have and would put per-machine state
into the image. Flatpak's own preinstall machinery does the work on each
machine that boots the image. Cheap to implement, and it makes the manifest a
complete description of the system rather than a description of half of it.

### Phase 4 — BuildStream bridge (optional, possibly never)

The README already says `build_files/` is where a `bst build && bst artifact
checkout` step "would go". For Dakota and Tromsø that is the *native* way to
produce something that must be compiled against fdsdk. Formalising it:

```yaml
bst:
  project: https://github.com/projectbluefin/dakota
  ref: <commit>
  elements: [core/foo.bst]
```

→ a build stage that checks out the project at a pinned commit, builds with
upstream's remote artifact cache configured, and `COPY --from=bst`es the
checkout into the image.

Pinning to a commit preserves the digest invariant, and a warm remote cache
means most elements are downloaded rather than compiled. Both of those are
"means", not "guarantees": a cache miss turns a nightly rebuild into a
multi-hour compile on the user's laptop, which is a categorically different
proposition from every other thing remora does. **Recommend deferring this
until phases 1–3 are in users' hands** and someone has actually measured a
cold-cache build. It is listed here so the manifest design in phases 1–2 does
not accidentally foreclose it.

---

## Alternatives considered

**systemd-sysext.** The obvious competitor: layer `/usr` content without
rebuilding or rebasing anything, and it is designed for exactly this kind of
image. It is genuinely better for ephemeral or development-time additions. It
is worse for remora's target case because the layered content is not part of
the system's identity: `bootc status` does not see it, `bootc rollback` does
not unwind it, the digest that rolls back is not the digest that was running,
and the extension has to be independently version-matched against the base on
every update. remora's whole proposition is that the customised system *is* an
image with a digest, and that property is the one worth keeping.

**Fork the BuildStream project.** Upstream's answer, and correct for people
who intend to distribute an image. It turns a user who wants a VPN client into
a distributor with a CI pipeline. Not a customisation story.

**Flatpak / Homebrew / Distrobox alone.** Already the right answer for
applications, and remora should say so loudly rather than competing. They do
not touch `/usr`, boot-time state, kernel modules, or anything that must exist
before a user session. That remainder is the scope of this proposal, and it
should stay the scope.

---

## What must be verified before any of this ships

The roadmap's own P0 is that **no build ever runs in CI** (#55) — every check
is a grep over generated text. Targeting a base that shares nothing with the
dnf images makes that gap sharper, not softer. This proposal should not merge
past phase 1 on the strength of reasoning alone. Four things need a real
machine:

1. **podman is present on Dakota and Tromsø.** remora builds locally; without
   a container runtime on the host there is no proposal. This is the first
   question to ask upstream and it gates everything else. (Dakota is aimed at
   cloud-native practitioners, so the expected answer is yes — expected is not
   verified.)
2. **`bootc switch --transport=containers-storage` is not refused by
   signature policy.** Dakota is cosign-signed and its documented switch
   commands carry `--enforce-container-sigpolicy`. A locally built image is
   unsigned by construction. If the host policy rejects it, remora needs to
   either drop a `localhost/remora` entry into the policy at `init` time — a
   deliberate, documented weakening that must be surfaced to the user, not
   done quietly — or this proposal stops at phase 1 being useless. **Highest
   risk item in the document.**
3. **The no-op rebuild holds on a composefs base.** Build twice, assert an
   identical digest, assert `remora apply` skips the switch. This invariant
   has never been exercised against composefs-backed storage or a
   systemd-boot/UKI deployment, and it is the invariant everything else rests
   on.
4. **A boot test.** `remora init`, a `system_files` overlay plus one
   `oci_copy`, `remora build`, reboot, confirm the file is present and
   `bootc status` reports the local image.

Two known limitations to document rather than solve: a layered **kernel
module** will not be in the UKI's initrd, so anything needed at early boot is
out of scope; and Dakota is compiled for **x86-64-v3**, so `oci_copy` sources
must be too.

## Recommended sequencing

Phase 1 is roughly a day's work — two guard clauses, a manifest validation
message, a CI fixture — and on its own it converts remora from "crashes on
Dakota" into a complete story for configuration, systemd units, certificates,
udev rules and overlay files, using `system_files/` and `build_files/` exactly
as they exist today. Ship that first, with the teaching shims, and get it in
front of a Dakota alpha user.

Phase 2 is the one to argue for on its own merits even if Dakota never
materialises, because a digest-pinned `oci_copy` lockfile is the resolved-set
cache key that #56 has been waiting for, on all six families at once.

Phases 3 and 4 follow real user demand, in that order.

## Open questions

- Is `package_manager: none` the right spelling, or does the manifest want a
  separate `sources:` concept with package managers as one source among
  several? `none` is the smaller change and reuses an existing field; a
  `sources:` redesign is more honest about what the manifest has become once
  `oci_copy` and `flatpaks` exist. Deliberately left open — phases 1 and 2 do
  not foreclose either.
- Should `remora shims` install *teaching* shims on a PM-less base — a `dnf`
  that exists only to say "this system has no package manager, here is what to
  do instead"? The shim package already refuses to overwrite foreign files, so
  on a base with no real binary the shim is a pure explainer with nothing to
  pass through to. Low cost, and it catches the single most likely first
  mistake a new Dakota user makes. Recommend yes, bundled with phase 1.
- Does upstream want to know? A local-derivation story that upstream considers
  out of bounds is a worse outcome than one they endorse or at least tolerate.
  Worth raising in the Dakota repo before phase 2 rather than after.
