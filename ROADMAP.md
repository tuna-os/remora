# remora Roadmap

**Last updated**: 2026-08-24 | **Maintainer**: tuna-os (hanthor)

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

- **Latest release**: v0.2.0 (2026-07-11) — standalone Linux binaries for
  amd64/arm64 + `SHA256SUMS`, published via goreleaser on `v*` tag push.
- **Preinstalled**: TunaOS images pin `REMORA_VERSION=v0.2.0` via
  `build_scripts/install-remora.sh` (deliberately no renovate marker — the
  sha256 check would fail on an unpinned bump).
- **Maturity**: active development since 07-11; factory, generate, host,
  manifest, and shim internals covered by unit tests; install snippet
  hardened to fail closed on bad checksums (#19/#20).
- **Docs**: published at docs/remora/index.md on the TunaOS docs site.

### Priorities

| Priority | Item | Tracking | Status |
|----------|------|----------|--------|
| P0 | Cut a release carrying the 08-14→08-23 fixes (install-snippet hardening, docs, tests) — currently stranded in unreleased tags | #21 | 🟡 In progress |
| P1 | Runtime units drift from the manifest after initialization | #17 | 🟡 Open |
| P1 | Explicit base images use the host package-manager contract | #18 | 🟡 Open |
| P2 | Release-cadence policy: define how often tags are cut and how the TunaOS image pin follows | #21 | ⬜ Not started |

---

## Quarterly Goals

### Current Quarter (2026 Q3)

**Theme**: ship and stabilize the customization channel

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Cut v0.3.0 with accumulated fixes + refresh TunaOS image pin | hanthor | #21 | ⬜ Not started |
| Resolve runtime-units drift (#17) | hanthor | #17 | ⬜ Not started |
| Define base-image contract (#18) | hanthor | #18 | ⬜ Not started |

### Next Quarter (2026 Q4)

**Theme**: versioning and coverage

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Release-cadence + versioning policy documented | tuna-os | #21 | ⬜ Not started |
| Adoption: remora usage surfaces in ADOPTION-METRICS snapshot | tuna-os | #1174 | ⬜ Not started |

---

*ROADMAP added by strategist agent (ACMM L6 — full mode). Signed-off-by: hanthor-hive-agent[bot] <290068839+hanthor-hive-agent[bot]@users.noreply.github.com>*
