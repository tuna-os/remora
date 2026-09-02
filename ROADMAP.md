# remora Roadmap

**Last updated**: 2026-08-26 | **Maintainer**: tuna-os (hanthor)

---

## Mission

Give every TunaOS user a container-native way to customize their immutable
desktop: a small manifest of packages and customizations, built into a local
derived image and rebased automatically whenever the base updates. remora is
the answer to `rpm-ostree install` that works identically on **dnf, zypper,
pacman, apt, portage (emerge), and apk** bases — so the customization story
is the same on every TunaOS variant.

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

### Priorities

| Priority | Item | Tracking | Status |
|----------|------|----------|--------|
| ~~P0~~ | ~~Update `REMORA_VERSION` in tunaOS so images receive the digest-pinned rebase fix~~ — TunaOS now pins v0.4.0 | tunaOS#2083 | ✅ Done |
| P1 | Runtime units drift from the manifest after initialization | #17 | 🟡 Open |
| ~~P2~~ | ~~Per-package-manager lockfile resolver, so a rebuild's cache key is the resolved package set rather than the spec list~~ — implemented for DNF via `dnf5 manifest` | #34 | ✅ Done |
| ~~P0~~ | ~~Cut a release carrying the 08-14→08-23 fixes~~ — shipped in v0.3.0 | #21 | ✅ Done |
| ~~P1~~ | ~~Explicit base images use the host package-manager contract~~ — the base image is now probed directly | #18 | ✅ Done |
| ~~P2~~ | ~~Release-cadence policy~~ — releases are cut by release-please from conventional commits; cadence is "whenever the release PR is merged" | #21 | ✅ Done |

---

## Quarterly Goals

### Current Quarter (2026 Q3)

**Theme**: ship and stabilize the customization channel

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Cut v0.3.0 with accumulated fixes | hanthor | #21 | ✅ Done |
| Refresh the TunaOS image pin | hanthor | tunaOS#2083 | ✅ Done — v0.4.0 |
| Resolve runtime-units drift (#17) | hanthor | #17 | ⬜ Not started |
| Define base-image contract (#18) | hanthor | #18 | ✅ Done |

### Next Quarter (2026 Q4)

**Theme**: versioning and coverage

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Release-cadence + versioning policy documented | tuna-os | #21 | ✅ Done — CONTRIBUTING "Releases" |
| Adoption: remora usage surfaces in ADOPTION-METRICS snapshot | tuna-os | #1174 | ⬜ Not started |

---

*ROADMAP added by strategist agent (ACMM L6 — full mode). Signed-off-by: hanthor-hive-agent[bot] <290068839+hanthor-hive-agent[bot]@users.noreply.github.com>*
