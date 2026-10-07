# remora Production Readiness Guide

**Version**: 1.0  
**Applicable Releases**: v0.4.0 and later  
**Last Updated**: 2026-10-07

---

## Overview

This guide defines what "production-ready" means for remora across different package managers and deployment scales. It helps you assess whether remora is suitable for your use case and what limitations to expect.

---

## Readiness Criteria by Package Manager

### ✅ dnf (Fedora, RHEL, CentOS Stream) — Recommended for Production

**Maturity Level**: Production-Ready (with known limitation)

**What's Covered**:
- ✅ Package installation + layering (via cache mounts)
- ✅ Lockfile resolver (`dnf5 manifest` — #34)
- ✅ Reproducible builds (`--timestamp 0` + deterministic Containerfile)
- ✅ No-op rebuilds (digest pinning, cache hit detection)
- ✅ Unit tests on core logic
- ✅ Installation snippet hardened to fail closed on bad checksums
- ✅ Shipped on TunaOS images (v0.4.0+)

**Known Limitation** (#44):
- ⚠️ On dnf bases, `remora build` fails when the base image's rpmdb sits in a lower overlayfs layer (SQLite cannot write atomically). **Workaround**: Use `extra_run` in your manifest to move the rpmdb. **Status**: Fix proposed, pending merge.

**When to Use**:
- Fedora 39+ or CentOS Stream 10+ with bootc or greenboot
- Immutable desktop/server with local customization needs
- CI/CD pipelines building derived bootc images

**When to Avoid**:
- You need a fully tested, blocking CI gate (see #55 — build tier in CI not yet implemented)
- Your base image has rpmdb-on-overlayfs layout and you cannot use the #44 workaround

---

### 🟡 zypper (openSUSE, SUSE) — Beta

**Maturity Level**: Supported, Text-Level Only

**What's Covered**:
- ✅ Package installation + layering (via cache mounts)
- ✅ Containerfile generation

**Not Yet Covered**:
- ❌ Lockfile resolver (no zypper equivalent to `dnf5 manifest` yet)
- ❌ Build verification in CI (see #55)

**Consequence**: Rebuilds are cache-keyed by the manifest spec list, not the resolved package set. After upstream publishes a newer package version, your layer may reuse the old version from cache.

**When to Use**:
- openSUSE Tumbleweed / MicroOS with bootc (evaluate with your team)
- Experimental/dev environments where package freshness can be manually controlled

**When to Avoid**:
- Production environments requiring guaranteed package freshness
- You need upstream package updates to be picked up automatically

---

### 🟡 pacman (Arch, EndeavourOS) — Beta

**Maturity Level**: Supported, Text-Level Only

**Same as zypper** — command generation works, no lockfile resolver or build verification. Use for evaluation only.

---

### 🟡 apt (Debian, Ubuntu) — Beta

**Maturity Level**: Supported, Text-Level Only

**Status**: apt is the strategic second package manager (see ROADMAP.md #56); support will be elevated to production in Q4 2026.

**Same limitations as zypper/pacman above.**

---

### 🟡 portage (Gentoo) — Beta

**Maturity Level**: Supported, Text-Level Only

**Same limitations.**

---

### 🟡 apk (Alpine) — Beta

**Maturity Level**: Supported, Text-Level Only

**Same limitations.**

---

## Deployment Scale Readiness

| Scale | Readiness | Notes |
|-------|-----------|-------|
| **Single Host** | ✅ Production | dnf: fully supported; others: feature-complete, limitations noted above |
| **Small Fleet (< 50)** | ✅ Production (dnf) / 🟡 Beta (others) | Per-host builds work; consider CI-based builds for consistency |
| **Large Fleet (50+)** | 🟡 Evaluate | Consider delegating builds to CI; homogeneous package-manager ecosystem strongly recommended |
| **CI/CD Pipeline** | ⚠️ Partial (dnf only) | Containerfile generation works; full `podman build` gate in CI pending (#55) |

---

## Before You Deploy

### Checklist

- [ ] **Package Manager**: Confirm remora supports your base's package manager (see tiers above)
- [ ] **Blocking Issues**: Review [ROADMAP.md — Current Status](./ROADMAP.md#current-status); confirm open P0/P1 issues don't affect your use case
- [ ] **Base Image Layout**: If using dnf, verify your base image doesn't have rpmdb-on-overlayfs (#44); if it does, test the workaround
- [ ] **Local Build Capacity**: Ensure your host has sufficient disk/CPU for `podman build` (first build: ~2–5 min depending on package count)
- [ ] **Version Pin Strategy**: Read how remora handles base image digest pinning (README.md — "Why the digest matters")
- [ ] **Customization Scope**: Keep manifest scope small (< 50 packages recommended); remora is for *layering*, not full system rebuilds

### Testing Before Production

1. **Single host trial**: Deploy remora with a small manifest (3–5 packages) and let it run 2–3 build cycles
2. **Verify no-op behavior**: Check that rebuilds with no manifest changes produce the same digest and skip rebase
3. **Verify update behavior**: Pin a package to an older version, let remora rebuild, then remove the pin — confirm the newer version is picked up (dnf) or manually trigger `remora upgrade`
4. **Rollback test**: Deploy an older base digest, then roll forward — confirm seamless rebase

---

## Known Limitations & Workarounds

| Issue | Workaround | Tracking |
|-------|-----------|----------|
| **dnf rpmdb-on-overlayfs** (#44) | Use `extra_run: ["mv /var/lib/rpm /root/; ln -s /root /var/lib/rpm"]` in manifest | #44 |
| **No build verification in CI** (#55) | Test locally; review Containerfile by hand; plan full CI gate rollout | #55 |
| **apt/zypper/pacman cache reuse on upstream updates** (#56) | Manually trigger `remora upgrade` to pin latest versions; plan per-package-manager resolvers in Q4 | #56 |
| **Runtime units drift after init** (#17) | Avoid manual edits to `/etc/remora/` after init; use `remora upgrade` to transition manifests | #17 |

---

## Upgrading Between Versions

remora follows semantic versioning (0.x until 1.0 ships breaking changes as minor bumps). Check [CHANGELOG.md](./CHANGELOG.md) and [ROADMAP.md](./ROADMAP.md) before upgrading.

**Safe to upgrade within**: v0.4.x (patch releases only; no breaking changes)  
**Recommended approach**: Evaluate v0.4.2 first on single host; roll to fleet after 1–2 weeks of no-incident operation

---

## Support & Feedback

- **Report a bug**: [GitHub Issues](https://github.com/tuna-os/remora/issues/new)
- **Request a feature**: [GitHub Issues](https://github.com/tuna-os/remora/issues/new?labels=enhancement)
- **Add your deployment to ADOPTERS.md**: [Pull Request](https://github.com/tuna-os/remora/pulls)
- **Discuss with the community**: [GitHub Discussions](https://github.com/tuna-os/remora/discussions)

---

## Roadmap Alignment

This guide reflects remora's maturity as of 2026-10-07. Key upcoming changes:

- **Q4 2026**: Full support tier for apt (resolver + build verification); package-less base support (sysext/confext)
- **Q1 2027** (planned): zypper/pacman brought to build-verified status

See [ROADMAP.md](./ROADMAP.md) for the full quarterly plan.

---

*Maintained by: [tuna-os/remora](https://github.com/tuna-os/remora) team*
