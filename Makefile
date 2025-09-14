SHELL := /bin/bash

# Default target
.PHONY: verify
verify: fmt vet lint test tidy-check ## Run all local checks (fast path)

.PHONY: fmt
fmt: ## Format code (writes files)
	gofmt -s -w .

.PHONY: fmt-check
fmt-check: ## Fail if formatting is needed
	@if [ -n "$(shell gofmt -l .)" ]; then \
		echo "Go code is not formatted:"; \
		gofmt -d .; \
		exit 1; \
	fi

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: test
test: ## Run unit tests (short, with race; skips external validation)
	PATH="$(PWD)/scripts/stub-bin:$$PATH" XRF_SKIP_VALIDATION=1 CGO_ENABLED=1 go test -v -race -short ./...

.PHONY: test-full
test-full: ## Run full test suite (may require Xray installed)
	PATH="$(PWD)/scripts/stub-bin:$$PATH" CGO_ENABLED=1 go test -v -race ./...

.PHONY: tidy
tidy: ## Tidy modules (writes go.mod/go.sum)
	go mod tidy

.PHONY: tidy-check
tidy-check: ## Fail if go mod tidy would change files
	@orig_mod=$$(mktemp); orig_sum=$$(mktemp); \
	cp go.mod $$orig_mod; cp go.sum $$orig_sum; \
	go mod tidy >/dev/null 2>&1; \
	if ! diff -q $$orig_mod go.mod >/dev/null || ! diff -q $$orig_sum go.sum >/dev/null; then \
		echo "go mod tidy would make changes; please run 'go mod tidy' and commit"; \
		diff -u $$orig_mod go.mod || true; \
		diff -u $$orig_sum go.sum || true; \
		rm -f $$orig_mod $$orig_sum; \
		exit 1; \
	fi; \
	rm -f $$orig_mod $$orig_sum

.PHONY: build
build: ## Build CLI binary
	go build -trimpath -o xrf cmd/xrf/main.go

.PHONY: ci-local
ci-local: fmt-check vet lint test build ## Approximate CI locally

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | sed -E 's/:.*?## /\t- /'
