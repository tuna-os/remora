# Release and Rollback Safety — remora

remora is a root-privileged tool that modifies the host system and manages bootc deployments. This runbook documents the safety expectations and deployment constraints for releases.

## Pre-release checklist

Before tagging a release:

- [ ] `just check` passes locally (gofmt, go vet, go test)
- [ ] CI passes: all gates green on main
- [ ] At least one human has reviewed and approved the PR merging the intended commit
- [ ] No breaking changes to the manifest schema without documenting migration steps
- [ ] Changes to the Containerfile generation logic have been tested on at least one target base (dnf, zypper, pacman, apt, portage, apk)
- [ ] If package resolution changed: tested on a base with and without `dnf5-plugin-manifest`
- [ ] If boot/rebase logic changed: tested a full bootc switch cycle
- [ ] CHANGELOG entries are present and accurate (release-please handles this)
- [ ] Git history is clean: all commits are signed (`-s`) and have DCO trailers

## Why each check matters

**gofmt, go vet, go test:** remora runs as root with access to the bootloader, ESP, and system packages. A formatting or type error can corrupt the host. All three checks must pass.

**CI gates:** The three smoke tests in `ci.yml` verify the generated Containerfile contains the digest-pinned `FROM`, the three `RUN` layers in the right order, and the no-op rebuild invariant. These are not cosmetic: if any fail, the no-op rebuild will quietly rebase to the same image every night.

**Human review:** Root tools need human judgment. An automated merge or auto-tag defeats the purpose.

**Manifest schema stability:** remora users upgrade incrementally. A schema break without a documented migration makes old manifests permanently unusable.

**Multi-base testing:** remora is documented to work on six package managers. If resolution changes, test on at least one base per manager, or note the narrowed scope in the release notes.

**Bootc switch testing:** The most dangerous code path. A bootc switch failure can leave the system unbootable. Do not ship rebase logic changes without testing a full cycle.

**Release-please history:** release-please maintains version consistency across releases. Hand-editing bypasses that: a version mismatch between binary and tag causes silent confusion.

**DCO and signing:** The Developer Certificate of Origin attests that code is original and licensed. Signed commits are auditable.

## The release process

1. Merge a PR to main that increments the version in `go.mod` or updates `CHANGELOG.md` (or both).
2. `release-please` detects this and opens a Release PR with a new tag.
3. A human reviews and merges the Release PR.
4. GitHub merges the tag.
5. `release.yml` detects the tag and runs `goreleaser`, which builds the binary, creates the GitHub release, and signs it.

Do not manually `git tag` or push tags directly; that bypasses release-please.

## Deployment safety

remora releases are published as:
- **GitHub releases:** source code and binaries
- **Checksums:** `checksums.txt` with SHA256 hashes
- **Binary signature:** signed with the release key (see [#19](https://github.com/tuna-os/remora/issues/19) for provenance signing)

Users install via:

```bash
# Verify checksums before installing
sha256sum -c checksums.txt
# Then install
sudo install remora /usr/local/bin/
```

The README deliberately documents this two-step process. Do not suggest skipping the checksum verification: the binary is privileged.

## Rollback procedure

If a release introduces a serious bug (boot failures, data corruption, security issues):

1. **Do not delete the tag.** Deleting a release tag makes it hard to trace what happened.
2. **Open an incident issue** using the [incident template](incident-template.md).
3. **Follow [rollback-a-bad-remora-build.md](rollback-a-bad-remora-build.md)** for step-by-step instructions.
4. **Tag a hotfix release** once the root cause is fixed and tested.
5. **Schedule a postmortem** using the [postmortem template](postmortem-template.md).

## What to watch for

**Silent failures:** The no-op rebuild invariant is critical. If it breaks, nightly timers will quietly rebase to the same image every night. The CI smoke tests detect this, but they are greps of the generated output. If the generation logic changes, re-run the smoke tests locally to verify.

**Package manager drift:** remora supports six package managers, each with different resolution and layering behavior. A change that passes CI on Fedora may fail silently on Arch. If a change touches `internal/resolve` or the package install phase, test on multiple bases.

**Bootc behavior changes:** bootc itself changes between releases. A remora release should note the minimum bootc version required. If a bootc update introduces a breaking change, document it in release notes and consider a major version bump.

**Rollback complexity:** Once users upgrade to a broken release, rolling back requires them to either use an older release or rebuild with a pinned version. Document the rollback procedure clearly in release notes if issues arise.

## Versioning

remora uses semantic versioning (MAJOR.MINOR.PATCH):

- **MAJOR:** Breaking changes to the manifest schema or bootc integration
- **MINOR:** New features or non-breaking enhancements
- **PATCH:** Bug fixes

When in doubt, ask in the issue or PR: root tools should err on the side of conservatism.

## Related resources

- [Rollback procedure](rollback-a-bad-remora-build.md)
- [Incident template](incident-template.md)
- [Postmortem template](postmortem-template.md)
- AGENTS.md: core invariants and CI enforcement
- `.release-please-manifest.json`: version state
