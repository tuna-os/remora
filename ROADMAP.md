# remora Roadmap

**Last updated**: 2026-10-04 | **Maintainer**: tuna-os (hanthor)

---

## Mission

Give every TunaOS user a container-native way to customize their immutable
desktop: a small manifest of packages and customizations, built into a local
derived image and rebased automatically whenever the base updates. remora is
the answer to `rpm-ostree install`, and the goal is that the customization
story is the same on every TunaOS variant — one manifest, six package-manager
families, plus support for package-manager-less bases via sysext/confext backends (#82). See [Package-manager support tiers](#package-manager-support-tiers)
for how far that goal has actually been carried today.

---

## Current Status

- **Latest release**: v0.4.3 (2026-09-24) — standalone Linux binaries for
  amd64/arm64 + `checksums.txt`, cut automatically by release-please and
  published via goreleaser. Adds bootc package integration and refactored
  provider architecture on top of v0.4.2's DNF lockfile resolver.
- **Preinstalled**: TunaOS images pin `REMORA_VERSION=v0.4.0` via
  `build_scripts/install-remora.sh`. An update to v0.4.3 is being evaluated
  post-release to capture the provider decoupling gains.
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
| P1 | Provider architecture decoupling & package-less base support (sysext/confext backends for Dakota/Tromsø) | #82, #91 | 🟡 In progress — provider refactor merged (#87), sysext/confext generation in scope for Phase 2 |
| P1 | Second package-manager family carried to full support (resolver + build-verified). apt is the strategic pick — it is the other family the org ships infrastructure for (tuna-os/debian-copr) | #56 | ⬜ Not started — needs maintainer sequencing |
| P1 | Runtime units drift from the manifest after initialization | #17 | 🟡 Open |
| ~~P0~~ | ~~Update `REMORA_VERSION` in tunaOS so images receive the digest-pinned rebase fix~~ — TunaOS now pins v0.4.0 | tunaOS#2083 | ✅ Done |
| ~~P2~~ | ~~Per-package-manager lockfile resolver, so a rebuild's cache key is the resolved package set rather than the spec list~~ — implemented for DNF via `dnf5 manifest` | #34 | ✅ Done |
| ~~P0~~ | ~~Cut a release carrying the 08-14→08-23 fixes~~ — shipped in v0.3.0 | #21 | ✅ Done |
| ~~P1~~ | ~~Explicit base images use the host package-manager contract~~ — the base image is now probed directly | #18 | ✅ Done |
| ~~P2~~ | ~~Release-cadence policy~~ — releases are cut by release-please from conventional commits; cadence is "whenever the release PR is merged" | #21 | ✅ Done |

---

## Quarterly Goals

### Current Quarter (2026 Q3 Exit & Q4 2026 Focus)

**Theme**: ship, stabilize, and expand customization backends

The "ship" half is done: v0.4.3 released, automated cutting, and a TunaOS pin pending evaluation. Provider architecture decoupling (#87 merged) unblocks package-less base support. Stabilizing dnf overlayfs interactions (#44) and CI build-verification (#55) remain P0 blockers.

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| `remora build` works on a shipped TunaOS dnf image | hanthor | #44 | 🔴 Open — blocking item for production use |
| Decouple providers & implement sysext/confext backend for package-less bases | hanthor | #82, #91, #87 | 🟡 Provider refactor merged; sysext/confext generation phase 2 |
| Add CI `podman build` matrix check | tuna-os | #55 | 🔴 Open — no build executed in CI today |
| Cut v0.4.3 release with provider decoupling | hanthor | #83 | ✅ Done (2026-09-24) |
| Evaluate TunaOS pin update to v0.4.3 | hanthor | tunaOS#TBD | ⬜ Pending — blocked on #44 fix |
| Resolve runtime-units drift (#17) | hanthor | #17 | ⬜ Not started |

### Next Quarter (2026 Q4)

**Theme**: stabilization, backends, and multi-distro coverage

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Fix dnf rpmdb overlayfs corruption — blockers both build (#44) and image adoption (#55) | hanthor | #44 | 🔴 Critical blocker |
| Implement sysext/confext output generation (Phase 2 of provider decoupling) | tuna-os | #82, #91 | ⬜ Planned for Q4 after Phase 1 lands |
| Add `podman build` verification to CI pipeline | tuna-os | #55 | 🔴 Open — needed before marking package-managers build-verified |
| Build-verified support for a second package-manager family (APT focus), resolver included | tuna-os | #56 | ⬜ Blocked on #55 CI tier |
| Adoption: remora usage surfaces in an org adoption snapshot | tuna-os | *needs a tracker* | ⬜ Blocked on #44/#55 resolution |
| Evaluate TunaOS image pin update to v0.4.3+ | tuna-os | tunaOS#TBD | ⬜ Pending #44 fix |

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
