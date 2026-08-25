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
