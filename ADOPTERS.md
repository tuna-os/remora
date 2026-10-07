# remora Adopters & Use Cases

This file tracks known production and pre-production deployments of remora. Adding your use case here signals to the community that remora is ready for your workload and helps the project understand adoption patterns.

## Format

Each entry includes:
- **Organization/User**: Name of adopter
- **Deployment**: Scale and context (individual, fleet, CI/CD, etc.)
- **Use Case**: What remora solves for you
- **Base Image**: dnf/zypper/pacman/apt/portage/apk (helps prioritize package-manager support)
- **Since**: First production deployment date (or "in-progress" for evaluation)

---

## In-Production Deployments

### TunaOS / Universal Blue ecosystem

| Organization | Deployment | Use Case | Base | Since |
|---|---|---|---|---|
| TunaOS | Pre-installation on shipped images | Reference implementation; on-host layer management for dnf bases | dnf (Fedora) | 2026-09-03 (v0.4.0) |

---

## Evaluation / Pre-Production

| Organization | Deployment | Use Case | Base | Since |
|---|---|---|---|---|
| *Waiting for first external adopter* | — | — | — | — |

---

## Contributing an Entry

To add your deployment:

1. **Confirm production readiness** — remora [v0.4.2](https://github.com/tuna-os/remora/releases/tag/v0.4.2) or later on your base
2. **Note known limitations** — see ROADMAP.md [Current Status](./ROADMAP.md#current-status) for package-manager support tiers and blocking issues (#44, #55)
3. **Open an issue or PR** linking to this file with your entry
4. **Include your contact** (email, GitHub handle, or org contact) so the team can reach you for feedback or coordination

---

## Maturity by Package Manager

remora maturity varies by package manager. See [ROADMAP.md - Package-manager support tiers](./ROADMAP.md#package-manager-support-tiers) for current guarantees:

- **dnf** (Fedora, RHEL, CentOS): ✅ Build-verified (with known #44 overlayfs limitation on dnf bases)
- **zypper, pacman, apt, portage, apk**: ⚠️ Command + cache mount, but no lockfile resolver or build verification yet

If you are evaluating remora on a package manager not yet in production use, your feedback accelerates support — [file an issue](https://github.com/tuna-os/remora/issues/new) with your experience.

---

## FAQ

**Q: How is remora different from rpm-ostree install?**  
A: remora is the bootc-native answer: it treats your customizations as a container, pins the base image by digest, and rebases on every boot if the build changed. The manifest is declarative and version-controllable; the image is reproducible and can be built locally or in CI.

**Q: Can I use remora on non-Fedora bases?**  
A: Yes — remora supports dnf, zypper, pacman, apt, portage, and apk. See ROADMAP.md for which are build-verified (dnf only today) and which are tested text-level only.

**Q: Is remora production-ready?**  
A: On dnf bases, yes — subject to [known issues](./ROADMAP.md#current-status). On other package managers, command generation works but build verification is missing (#55). See the support tiers table above.

---

*Last updated: 2026-10-07*  
*Maintained by: [tuna-os/remora](https://github.com/tuna-os/remora) team*
