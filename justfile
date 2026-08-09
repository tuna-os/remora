# TunaOS local task surface. CI runs the same fast validation through `just check`.
set shell := ["bash", "-euo", "pipefail", "-c"]

default:
    @just --list

check: fmt-check lint test

fmt:
    gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
    test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))"

lint:
    go vet ./...

test:
    go test ./... -count=1

build:
    go build ./...
