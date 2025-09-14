# Repository Guidelines

## Project Structure & Module Organization
- Go module: `github.com/Joe-oss9527/xrf-go` (Go 1.25)
- CLI entry: `cmd/xrf/main.go`
- Packages: `pkg/config`, `pkg/system`, `pkg/tls`, `pkg/api`, `pkg/utils`
- Tests: `*_test.go` alongside packages; integration tests in `pkg/system`
- Scripts: `scripts/install.sh`
- CI/CD: `.github/workflows/ci.yml`, `.github/workflows/release.yml`

## Build, Test, and Development Commands
- Build CLI: `go build -trimpath -o xrf cmd/xrf/main.go`
- Run help: `./xrf --help`
- Unit tests: `go test -v -race ./...`
- Coverage: `go test -coverprofile=coverage.txt ./...`
- Lint: `golangci-lint run`
- Vet/Format: `go vet ./...` and `gofmt -s -w .`; tidy: `go mod tidy`

## Coding Style & Naming Conventions
- Go defaults: tabs for indentation; files `lower_snake_case.go`; packages lowercase
- Exported identifiers use CamelCase; unexported use lowerCamelCase
- Prefer error wrapping; return errors over panics (except in `main`)
- Logging: use `pkg/utils` logger in packages; CLI prints live in `cmd/xrf`
- Keep changes minimal and focused; avoid unnecessary dependencies

## Testing Guidelines
- Place tests next to source with `_test.go`; name tests `TestXxx`
- Avoid network and filesystem side effects; mock process execution where possible
- Use `-race` in CI; keep coverage healthy for changed areas
- Integration tests requiring system tools should be guarded or skipped in CI

## Commit & Pull Request Guidelines
- Commits: Conventional style preferred: `feat: ...`, `fix: ...`, `docs: ...`, `ci: ...`, `refactor: ...`, `test: ...`, `chore: ...` (scope optional)
- PRs must include: summary, rationale, test plan (commands run), and related issues
- Ensure `go vet`, `golangci-lint run`, tests, and `go mod tidy` are clean before review

## CI/CD Notes
- Actions: `actions/checkout@v5`, `actions/setup-go@v5 (go-version-file: go.mod)`, `golangci/golangci-lint-action@v8`
- Xray is used in CI for config validation; keep asset-name fallbacks intact

## Security & Configuration Tips
- No secrets in code or logs; use GitHub Encrypted Secrets
- Shell scripts: `set -euo pipefail`; prefer `sudo install` over `cp && chmod`

## Pre-Commit & Pre-Push Checklist
- Format: `gofmt -s -w .`
- Vet: `go vet ./...`
- Lint: `golangci-lint run`
- Test: `go test -v -race ./...` (locally run full tests; CI may use `-short`)
- Modules: `go mod tidy` and ensure `git status` is clean

## Using GitHub CLI (gh)
- Prefer `gh` for repository and CI interactions; keep `git` for local changes
- CI: `gh run list`, `gh run view <id> --log`, `gh run watch <id>`
- Releases: manage via tags + workflows; `gh release view <tag>`, `gh release create <tag> ...`
- Before any tag/release operation, run the Pre-Commit & Pre-Push Checklist (or `make verify`/`scripts/dev-verify.sh`)
