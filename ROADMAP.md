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

- **Latest release**: v0.3.0 (2026-08-26) — standalone Linux binaries for
  amd64/arm64 + `checksums.txt`, cut automatically by release-please and
  published via goreleaser. Carries digest-pinned bases, reproducible layers
  and no-op rebuilds (#27), plus every fix stranded since v0.2.0: the
  install-snippet checksum hardening (#19/#20), installation docs, ROADMAP,
  CONTRIBUTING, and the unit-test additions.

  Note: the v0.3.0 release notes list only two entries. The generated
  changelog classifies conventional-commit subjects, and the stranded fixes
  landed with a bracket prefix (`[sec-check] fix: ...`), which is not
  classified. They shipped; they are just absent from the notes. CONTRIBUTING
  documents the prefix rule so this does not recur.
- **Preinstalled**: TunaOS images pin `REMORA_VERSION=v0.2.0` via
  `build_scripts/install-remora.sh` (deliberately no renovate marker — the
  sha256 check would fail on an unpinned bump). **The pin is now a release
  behind**; bumping it to v0.3.0 is what delivers the hardened install path
  to images.
- **Maturity**: active development since 07-11; factory, generate, host,
  manifest, and shim internals covered by unit tests; install snippet
  hardened to fail closed on bad checksums (#19/#20).
- **Docs**: published at docs/remora/index.md on the TunaOS docs site.

### Priorities

| Priority | Item | Tracking | Status |
|----------|------|----------|--------|
| P0 | Bump `REMORA_VERSION` in tunaOS `build_scripts/install-remora.sh` to v0.3.0 — images still ship the pre-hardening install path | #21 | 🔴 Open |
| P1 | Runtime units drift from the manifest after initialization | #17 | 🟡 Open |
| P2 | Per-package-manager lockfile resolver, so a rebuild's cache key is the resolved package set rather than the spec list | — | ⬜ Not started |
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
| Refresh the TunaOS image pin to v0.3.0 | hanthor | #21 | ⬜ Not started |
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
