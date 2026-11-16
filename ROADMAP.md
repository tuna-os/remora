# remora Roadmap

**Last updated**: 2026-11-16 | **Maintainer**: tuna-os (hanthor)

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

- **Latest release**: v0.4.6 (2026-10-17) — standalone Linux binaries for
  amd64/arm64 + `checksums.txt`, cut automatically by release-please and
  published via goreleaser. Builds on the digest-pinned bases, reproducible
  layers, and no-op rebuilds of v0.3.0 (#27) and the DNF lockfile resolver of
  v0.4.0 (#34). v0.4.4 fixed the dnf rpmdb failure (#98) and v0.4.6 reconciles
  runtime units with the manifest (#172).
- **Preinstalled**: TunaOS images pin `REMORA_VERSION=v0.4.6` via
  `build_scripts/install-remora.sh` in tuna-os/tunaOS. Renovate bumps that
  version but not the pinned checksums.
- **Maturity**: active development since 07-11; every package under
  `internal/` has unit tests; install snippet hardened to fail closed on bad
  checksums (#19/#20).
- **Docs**: published at docs/remora/index.md on the TunaOS docs site.
- **Fixed in v0.4.4**: on a dnf base the package transaction failed with
  `database disk image is malformed`, because the base image's rpmdb sits in a
  lower overlayfs layer where SQLite cannot write atomically (#44, reproduced
  on `ghcr.io/tuna-os/bonito:cosmic`). The generator now copies the rpmdb up
  before the install (#98). No CI job builds an image, so CI does not prove
  this fix (#55).
- **No build ever runs in CI.** Every check is text-level: `generate` writes a
  Containerfile and the workflow greps it (`install --no-build`). Nothing
  executes `podman build`, so #44's whole defect class is invisible to the
  pipeline (#55).

### Package-manager support tiers

The generator emits install commands and cache mounts for all six families and
the manifest accepts all six. The guarantees behind them are not equal (#56):

| Package manager | Command + cache mount | Lockfile resolver | Build-verified |
|-----------------|----------------------|-------------------|----------------|
| dnf | ✅ | ✅ `dnf5 manifest` (#34) | ❌ — no CI build (#55); the #44 rpmdb defect was fixed in #98 |
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
| P0 | A build tier in CI: one real `podman build` against a bootc base, plus a rebuild asserting the digest is stable | #55, #53 | 🔴 Open — no build runs today |
| P1 | Provider architecture decoupling & package-less base support (sysext/confext backends for Dakota/Tromsø) | #82, #111 | 🟡 Design merged in #82; scope lock open in #111; no backend code yet |
| P1 | Second package-manager family carried to full support (resolver + build-verified). apt is the strategic pick — it is the other family the org ships infrastructure for (tuna-os/debian-copr) | *needs a tracker* — #56 closed as a duplicate on 2026-11-10 without naming a successor | ⬜ Not started — needs maintainer sequencing |
| ~~P0~~ | ~~`remora build` fails on dnf bases — rpmdb on overlayfs~~ — the generator copies the rpmdb up before install | #44, #98 | ✅ Done — v0.4.4 |
| ~~P1~~ | ~~Runtime units drift from the manifest after initialization~~ — reconciled with the manifest | #17, #172 | ✅ Done — v0.4.6 |
| ~~P0~~ | ~~Update `REMORA_VERSION` in tunaOS so images receive the digest-pinned rebase fix~~ — TunaOS now pins v0.4.0 | tunaOS#2083 | ✅ Done |
| ~~P2~~ | ~~Per-package-manager lockfile resolver, so a rebuild's cache key is the resolved package set rather than the spec list~~ — implemented for DNF via `dnf5 manifest` | #34 | ✅ Done |
| ~~P0~~ | ~~Cut a release carrying the 08-14→08-23 fixes~~ — shipped in v0.3.0 | #21 | ✅ Done |
| ~~P1~~ | ~~Explicit base images use the host package-manager contract~~ — the base image is now probed directly | #18 | ✅ Done |
| ~~P2~~ | ~~Release-cadence policy~~ — releases are cut by release-please from conventional commits; cadence is "whenever the release PR is merged" | #21 | ✅ Done |

---

## Quarterly Goals

### Current Quarter (2026 Q3 Exit & Q4 2026 Focus)

**Theme**: ship, stabilize, and expand customization backends

The "ship" half is done: v0.4.6 released, automated cutting, and a TunaOS pin. The dnf overlayfs fix (#44, #98) shipped in v0.4.4. A CI build tier (#55) and provider decoupling (#82, #111) remain the bridge into Q4.

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| `remora build` works on a shipped TunaOS dnf image | hanthor | #44 | ✅ Done — fixed in #98 (v0.4.4) |
| Decouple providers & implement sysext/confext backend for package-less bases | hanthor | #82, #111 | 🟡 Design merged (#82); scope lock open (#111) |
| Add CI `podman build` matrix check | tuna-os | #55 | 🔴 Open |
| Cut v0.4.0 / v0.4.2 releases with accumulated fixes | hanthor | #21 | ✅ Done |
| Refresh the TunaOS image pin | hanthor | tunaOS#2083 | ✅ Done — v0.4.0 |
| Resolve runtime-units drift (#17) | hanthor | #17 | ✅ Done — #172 (v0.4.6) |

### Next Quarter (2026 Q4)

**Theme**: versioning, alternative backends, and multi-distro coverage

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Decoupled provider architecture & package-less base image support (sysext/confext backends) | tuna-os | #111 | ⬜ Scope lock open |
| Implement Phase 1 provider decoupling & sysext output generation | tuna-os | #82; implementation *needs a tracker* | ⬜ Not started — #91 tracked only the roadmap entry |
| Build-verified support for a second package-manager family (APT focus), resolver included | tuna-os | *needs a tracker* (#56 closed as a duplicate) | ⬜ Not started |
| Adoption: remora usage surfaces in an org adoption snapshot | tuna-os | *needs a tracker* | ⬜ Blocked on a decision |
| Release-cadence + versioning policy documented | tuna-os | #21 | ✅ Done — CONTRIBUTING "Releases" |

---

## Technical debt

| Item | Issue | Priority |
|------|-------|----------|
| ~~CLI package owns the build-plan state machine~~ — moved to `internal/buildplan` in #84 | ~~#49~~ | ~~P2~~ |
| ~~README install snippet runs `sudo install` even when the checksum check fails~~ — fixed in #20 (v0.2.7) | ~~#19~~ | ~~P2~~ |
| ~~Runtime units are written only by `init`; other paths rewrite the Containerfile alone~~ — fixed in #172 (v0.4.6) | ~~#17~~ | ~~P1~~ |

---

## Roadmap governance

A tracker cited in this file that closes must move its row in the same PR, or
the row must name a successor. The 08-26 revision carried a Q4 goal pointing at
tunaOS#1174 after it closed, which left the file's only forward-looking row
attached to work that no longer existed under that number.
