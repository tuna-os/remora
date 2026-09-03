# AGENTS.md — agent guide for tuna-os/remora

`remora` layers packages onto a bootc host by building a local derived image
and rebasing onto it. Go, single binary, runs as **root** on the user's host.

The domain is documented unusually well in the code itself — start with the
package comments in [`internal/generate`](internal/generate/containerfile.go)
and [`internal/resolve`](internal/resolve/resolve.go), and
[`README.md`](README.md) for the user-facing model. This file covers what
those do *not* say: what CI actually enforces, and what breaks quietly.

## The invariant that has no unit test: the no-op rebuild

remora's whole value rests on *identical inputs producing an identical image
digest*, so a nightly rebuild that changed nothing skips the `bootc switch`.
Four things hold that up, and each is easy to break with a change that looks
harmless and passes `go test`:

1. **Digest-pinned `FROM`.** A moving tag reintroduces drift.
2. **Three separate `RUN` layers** — overlay + `extra_run`, then packages,
   then build scripts — so editing a build script does not invalidate the
   expensive package layer.
3. **`--timestamp 0` plus scrubbing build-varying state** (logs, dnf history).
   The rpmdb under `/usr/lib/sysimage/rpm` is deliberately *not* scrubbed: it
   is the installed-package record, not a cache.
4. **`bootc switch tag@sha256:…`, never a bare tag.** A bare tag hands bootc
   the same image specification every run, so new content under an unchanged
   tag may never be staged.

Lose any of these and nothing fails — the timer just quietly mints a fresh
deployment every night to install the same packages again. That is why
`ci.yml` carries three shell smoke blocks asserting the *shape* of generated
output: a `sha256:` in `FROM`, exactly three `<<'REMORA_EOF'` heredocs, no
`remora.lock.yaml` reference when resolution did not run, and `apply` exiting
non-zero when nothing has been built. **Treat those greps as a spec.** If a
change makes one fail, the question is whether the invariant still holds, not
how to adjust the grep.

The stale-lockfile assertion is the subtlest: a `remora.lock.yaml` left by an
earlier build must not survive a `generate` that did not resolve, or it pins
an old package set indefinitely and the Containerfile `COPY`s a file
generation never vouched for.

Resolution itself is **best-effort by contract**. `Resolver.Available` is
expected to return false routinely — no podman, no `dnf5-plugin-manifest` in
the base image — and callers must fall back to the plain install path rather
than failing the build.

## Checks

```bash
just check      # gofmt -l must be empty, go vet ./..., go test ./... -count=1
go build ./...
```

`ci.yml` runs exactly that plus the three smoke blocks, and gates on a
`required-checks` job with `if: always()`, so a skipped or cancelled `test`
job reports failure rather than an absent check.

> **Configured but not enforced.** `.golangci.yml` exists, but no workflow
> runs `golangci-lint` — `just lint` is `go vet` alone. `codecov.yml` sets
> coverage targets, but nothing uploads coverage. Neither is a gate today.

`CONTRIBUTING.md` requires a DCO sign-off (`git commit -s`) on every commit;
no workflow checks it, so it is a convention a human must honour rather than
something CI will catch.

## Don'ts

- **Don't hand-edit a generated `Containerfile`.** It is overwritten on every
  build. User customisation goes in the manifest, `build_files/*.sh`, or the
  `system_files/` overlay.
- **Don't unpin the actionlint image.** `actionlint.yml` uses a digest, not a
  tag, so a lint result is reproducible from the commit alone. The workflow
  comment carries the upgrade recipe (`docker buildx imagetools inspect`) —
  follow it and update the version comment alongside the digest.
- **Don't make lockfile resolution mandatory.** Package managers without a
  resolver, and hosts without podman, must keep working.
- **Don't lower the install bar in the README.** The documented install
  verifies `checksums.txt` *before* `sudo install`, deliberately: this binary
  runs as root, puts shims in `/usr/local/bin` ahead of `/usr/bin`, and owns a
  timer that rebases the host image. (That file proves transit integrity only,
  not provenance — signing is tracked in
  [#19](https://github.com/tuna-os/remora/issues/19).)

## Releases

Tag-driven: pushing `v*` runs goreleaser. Version bumps and the changelog come
from release-please (`release-please-config.json`,
`.release-please-manifest.json`), so hand-editing `CHANGELOG.md` or a version
string fights the automation.
