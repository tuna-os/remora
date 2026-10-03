# remora Roadmap

**Last updated**: 2026-10-03 | **Maintainer**: tuna-os (hanthor)

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

- **Latest release**: v0.4.2 (2026-09-03) — standalone Linux binaries for
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
| P1 | Provider architecture decoupling & package-less base support (sysext/confext backends for Dakota/Tromsø) | #82, #91 | 🟡 Open — design proposed in #82 |
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

**Theme**: stabilize, verify, and plan decoupling

The "ship" half is done: v0.4.2 released, automated cutting, and a TunaOS pin. **Critical blockers**: P0 verification gaps (no CI build) and known defects on shipped images (#44) now block Q4 expansion. Pending decisions on provider decoupling scope (#111) and adoption tracking (#110) must be made before workload can be sequenced.

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Resolve P0 defect: `remora build` fails on dnf bases with overlayfs rpmdb | hanthor | #44 | 🔴 Blocking — known on TunaOS v0.4.0 images |
| Add CI verification: `podman build` matrix against multiple package-manager families | tuna-os | #55, #107 | 🔴 P0 — currently no build runs in CI |
| Lock scope for provider decoupling (Phase 1 vs. full sysext/confext) | hanthor | #82, #91, #111 | 🟡 Design exists, approval pending |
| Resolve runtime-units drift from manifest after initialization | hanthor | #17, #112 | 🟡 Stalled 8+ weeks — needs prioritization |
| Clarify Q4 adoption tracking decision | tuna-os | #110 | 🟡 Blocked — decision required |
| Cut v0.4.0 / v0.4.2 releases with accumulated fixes | hanthor | #21 | ✅ Done |
| Refresh the TunaOS image pin | hanthor | tunaOS#2083 | ✅ Done — v0.4.0 |

### Next Quarter (2026 Q4)

**Theme**: unblock Q3 defects, lock decisions, prepare Phase 1

Q4 planning is contingent on resolving three decision points:
1. **Provider decoupling scope** (#111) — full decoupling or Phase 1 partial?
2. **Adoption tracking** (#110) — is remora included in Q4 org metrics?
3. **Priority clarification** (#112) — is #17 (drift) P1 or backlog?

Once locked, the unblocking sequence is: fix #44 (dnf), add CI build (#107), then begin Phase 1 decoupling.

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| **BLOCKER**: Decide provider-decoupling scope and file Phase 1 design spec | tuna-os | #111 | 🔴 Must lock before implementation |
| **BLOCKER**: Resolve or defer #17 (runtime-units drift) — clarify priority | hanthor | #112 | 🔴 Must decide before Q4 kickoff |
| **BLOCKER**: Clarify adoption-tracking decision and create tracking issue if approved | tuna-os | #110 | 🔴 Blocks adoption coordination |
| Fix #44: dnf rpmdb atomicity on overlayfs | hanthor | #44 | 🔴 After blockers cleared, target v0.5.0 |
| Add CI `podman build` matrix (dnf/apt/zypper, amd64/arm64) | tuna-os | #107 | 🔴 After #44 fix, verify fix in CI |
| Implement Phase 1 provider decoupling & sysext output | tuna-os | #82, #91, #111 | ⬜ After scope decision (#111) |
| Build-verified support for APT resolver | tuna-os | #56 | ⬜ After Phase 1 launches |

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
