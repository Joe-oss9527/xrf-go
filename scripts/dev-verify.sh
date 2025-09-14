#!/usr/bin/env bash
set -euo pipefail

echo "[dev-verify] Starting local checks..."

need() { command -v "$1" >/dev/null 2>&1 || { echo "[dev-verify] Missing: $1"; return 1; }; }

if ! need go; then
  echo "[dev-verify] Go is required. Install Go 1.25+ first." >&2
  exit 1
fi

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "[dev-verify] golangci-lint not found; attempting to install to ./.bin via 'go install'..."
  export GOBIN="$(pwd)/.bin"
  mkdir -p "$GOBIN"
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  export PATH="$GOBIN:$PATH"
fi

echo "[dev-verify] gofmt -s -w ."
gofmt -s -w .

echo "[dev-verify] go vet ./..."
go vet ./...

echo "[dev-verify] golangci-lint run"
golangci-lint run

echo "[dev-verify] go mod tidy (dry check)"
orig_mod=$(mktemp); orig_sum=$(mktemp)
cp go.mod "$orig_mod"; cp go.sum "$orig_sum" || true
go mod tidy >/dev/null 2>&1 || true
if ! diff -q "$orig_mod" go.mod >/dev/null || ! diff -q "$orig_sum" go.sum >/dev/null; then
  echo "[dev-verify] go.mod/go.sum changed. Please commit module tidy updates." >&2
  diff -u "$orig_mod" go.mod || true
  diff -u "$orig_sum" go.sum || true
  rm -f "$orig_mod" "$orig_sum"
  exit 1
fi
rm -f "$orig_mod" "$orig_sum"

echo "[dev-verify] go test (short) with validation skipped"
XRF_SKIP_VALIDATION=1 CGO_ENABLED=0 go test -v -race -short ./...

echo "[dev-verify] All checks passed."

