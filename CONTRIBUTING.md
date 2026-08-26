# Contributing to remora

Thank you for helping improve `remora`! This guide covers local setup, validation workflows, code style, and submission requirements.

## Development Requirements

- **Go**: 1.22+ (as specified in `go.mod`)
- **just**: Command runner (optional, but recommended)
- **git**: For version control and signing commits

## Getting Started

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/tuna-os/remora.git
   cd remora
   ```

2. Verify your setup:
   ```bash
   go build ./...
   go test ./...
   ```

## Local Validation Workflow

Before submitting a pull request, ensure all local checks pass. You can run the checks using `just` or raw `go` commands:

### Using `just` (Recommended)

```bash
just check
```

`just check` runs formatting verification (`gofmt`), static analysis (`go vet`), and unit testing (`go test`).

### Using `go` CLI directly

- **Formatting**:
  ```bash
  test -z "$(gofmt -l $(find . -name '*.go' -not -path './vendor/*'))"
  ```
- **Linting**:
  ```bash
  go vet ./...
  ```
- **Testing**:
  ```bash
  go test ./... -count=1
  ```

## Pull Request Guidelines

1. **Signed Commits**: All commits must include a Developer Certificate of Origin (DCO) sign-off:
   ```bash
   git commit -s -m "feat: add support for custom package flag"
   ```
2. **Target Branch**: Submit pull requests against the `main` branch.
3. **Tests**: Add unit tests for new functionality or bug fixes in `internal/`.
4. **Documentation**: Update `README.md` or `ROADMAP.md` if changing user-facing behaviors or CLI arguments.
5. **Conventional commit titles**: pull requests are squash-merged, so the PR
   title becomes the commit subject on `main` — and that subject is what
   decides the next version and the changelog entry. Use a conventional
   prefix:

   | Prefix | Effect on the next release |
   |---|---|
   | `feat:` | minor bump (`0.2.0` → `0.3.0`), listed under **Features** |
   | `fix:` | patch bump (`0.2.0` → `0.2.1`), listed under **Bug Fixes** |
   | `perf:` / `refactor:` / `docs:` | patch bump, listed under their section |
   | `test:` / `ci:` / `build:` / `chore:` | kept out of the changelog |
   | `feat!:` or a `BREAKING CHANGE:` footer | minor bump while below 1.0 |

   The minor-not-major behavior for breaking changes is deliberate while
   remora is below 1.0, and is set by `bump-minor-pre-major` in
   `release-please-config.json`.

   A subject with no recognized prefix — or with something before it, like
   `[agent] fix: ...` — is not classified, so it lands on `main` without
   appearing in the changelog.

## Releases

Releases are automated with
[release-please](https://github.com/googleapis/release-please). There is
nothing to tag by hand.

1. Merge your PR to `main` with a conventional title (see above).
2. release-please opens or updates a pull request titled
   `chore(main): release X.Y.Z`, containing the computed version bump and the
   generated `CHANGELOG.md` entry. It stays open and accumulates changes.
3. Merging that release PR cuts the release: release-please pushes the `vX.Y.Z`
   tag and creates the GitHub release from the changelog entry, then
   goreleaser builds the amd64/arm64 binaries and attaches them along with
   `checksums.txt`.

Do not edit `CHANGELOG.md` or `.release-please-manifest.json` by hand — both
are generated, and hand edits are overwritten on the next run.

The release pull request is opened by a bot using `GITHUB_TOKEN`, and events
from that token do not start workflow runs — so it would arrive with no checks
on it. The release workflow works around this by dispatching CI onto the
release branch explicitly (`workflow_dispatch` is the documented exception
that always creates a run), which attaches the checks to the pull request's
head commit. If those checks are ever missing, the release is still safe to
cut when CI is green on `main` for the commit being released: the release PR
only touches generated metadata.

`.github/workflows/release.yml` still fires on a hand-pushed `v*` tag. That is
the escape hatch for releasing outside this flow; the normal path is merging
the release PR.
