# remora Roadmap

**Last updated**: 2026-09-02 | **Maintainer**: tuna-os (hanthor)

---

## Mission

Give every TunaOS user a container-native way to customize their immutable
desktop: a small manifest of packages and customizations, built into a local
derived image and rebased automatically whenever the base updates. remora is
the answer to `rpm-ostree install`, and the goal is that the customization
story is the same on every TunaOS variant — one manifest, six package-manager
families. See [Package-manager support tiers](#package-manager-support-tiers)
for how far that goal has actually been carried today.

---

## Current Status

- **Latest release**: v0.4.0 (2026-08-26) — standalone Linux binaries for
  amd64/arm64 + `checksums.txt`, cut automatically by release-please and
  published via goreleaser. Adds the DNF lockfile resolver (#34) on top of the
  digest-pinned bases, reproducible layers, and no-op rebuilds shipped in
  v0.3.0 (#27).
- **Preinstalled**: TunaOS images pin `REMORA_VERSION=v0.4.0` via
  `build_scripts/install-remora.sh`. The update from v0.2.0 was completed in
  tuna-os/tunaOS#2083.
- **Maturity**: active development since 07-11; factory, generate, host,
  manifest, and shim internals covered by unit tests; install snippet
  hardened to fail closed on bad checksums (#19/#20).
- **Docs**: published at docs/remora/index.md on the TunaOS docs site.
- **Known broken on shipped images**: on a dnf base the package transaction
  fails with `database disk image is malformed` — the base image's rpmdb sits
  in a lower overlayfs layer where SQLite cannot write atomically. Reproduced
  on `ghcr.io/tuna-os/bonito:cosmic` (#44), which pins this very version. A
  manifest-level `extra_run` workaround is recorded on the issue; the fix
  belongs in the generator.
- **No build ever runs in CI.** Every check is text-level: `generate` writes a
  Containerfile and the workflow greps it (`install --no-build`). Nothing
  executes `podman build`, so #44's whole defect class is invisible to the
  pipeline (#55).

### Package-manager support tiers

The generator emits install commands and cache mounts for all six families and
the manifest accepts all six. The guarantees behind them are not equal (#56):

| Package manager | Command + cache mount | Lockfile resolver | Build-verified |
|-----------------|----------------------|-------------------|----------------|
| dnf | ✅ | ✅ `dnf5 manifest` (#34) | ❌ — but #44 is a known dnf-base defect |
| zypper | ✅ | ❌ | ❌ |
| pacman | ✅ | ❌ | ❌ |
| apt | ✅ | ❌ | ❌ |
| portage | ✅ | ❌ | ❌ |
| apk | ✅ | ❌ | ❌ |

Without a resolver a rebuild's cache key is the **spec list**, not the resolved
package set: `packages: [htop]` stays unchanged, so the layer is reused even
after upstream publishes a newer `htop`. The no-op-rebuild guarantee and the
freshness guarantee are the same knob turned opposite ways, and only dnf
currently has the instrument that resolves the tension.

### Priorities

| Priority | Item | Tracking | Status |
|----------|------|----------|--------|
| P0 | `remora build` fails on dnf bases — rpmdb on overlayfs; fix in the generator, not in user manifests | #44 | 🔴 Open — reproduced on an image that pins v0.4.0 |
| P0 | A build tier in CI: one real `podman build` against a bootc base, plus a rebuild asserting the digest is stable | #55, #53 | 🔴 Open — no build runs today |
| P1 | Second package-manager family carried to full support (resolver + build-verified). apt is the strategic pick — it is the other family the org ships infrastructure for (tuna-os/debian-copr) | #56 | ⬜ Not started — needs maintainer sequencing |
| P1 | Runtime units drift from the manifest after initialization | #17 | 🟡 Open |
| ~~P0~~ | ~~Update `REMORA_VERSION` in tunaOS so images receive the digest-pinned rebase fix~~ — TunaOS now pins v0.4.0 | tunaOS#2083 | ✅ Done |
| ~~P2~~ | ~~Per-package-manager lockfile resolver, so a rebuild's cache key is the resolved package set rather than the spec list~~ — implemented for DNF via `dnf5 manifest` | #34 | ✅ Done |
| ~~P0~~ | ~~Cut a release carrying the 08-14→08-23 fixes~~ — shipped in v0.3.0 | #21 | ✅ Done |
| ~~P1~~ | ~~Explicit base images use the host package-manager contract~~ — the base image is now probed directly | #18 | ✅ Done |
| ~~P2~~ | ~~Release-cadence policy~~ — releases are cut by release-please from conventional commits; cadence is "whenever the release PR is merged" | #21 | ✅ Done |

---

## Quarterly Goals

### Current Quarter (2026 Q3) — closes 2026-09-30

**Theme**: ship and stabilize the customization channel

The "ship" half is done: four tagged releases, automated cutting, and a
TunaOS pin. The "stabilize" half is where the quarter's remaining time goes —
#44 means the shipped channel does not currently carry a package onto the
images that ship it.

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| `remora build` works on a shipped TunaOS dnf image | hanthor | #44 | 🔴 Open — added 09-02, the quarter's blocking item |
| Cut v0.3.0 with accumulated fixes | hanthor | #21 | ✅ Done |
| Refresh the TunaOS image pin | hanthor | tunaOS#2083 | ✅ Done — v0.4.0 |
| Resolve runtime-units drift (#17) | hanthor | #17 | ⬜ Not started |
| Define base-image contract (#18) | hanthor | #18 | ✅ Done |

### Next Quarter (2026 Q4)

**Theme**: versioning and coverage

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Build-verified support for a second package-manager family, resolver included | tuna-os | #56 | ⬜ Not started |
| Adoption: remora usage surfaces in an org adoption snapshot | tuna-os | *needs a tracker* | ⬜ Blocked on a decision — tunaOS#1174 is closed, so this row has no live tracker. Either a successor exists org-wide or remora needs its own instrument |
| Release-cadence + versioning policy documented | tuna-os | #21 | ✅ Done — CONTRIBUTING "Releases" |

---

## Technical debt

| Item | Issue | Priority |
|------|-------|----------|
| CLI package owns the build-plan state machine | #49 | P2 |
| README install snippet runs `sudo install` even when the checksum check fails | #19 | P2 |
| Runtime units are written only by `init`; other paths rewrite the Containerfile alone | #17 | P1 |

---

## Roadmap governance

A tracker cited in this file that closes must move its row in the same PR, or
the row must name a successor. The 08-26 revision carried a Q4 goal pointing at
tunaOS#1174 after it closed, which left the file's only forward-looking row
attached to work that no longer existed under that number.
