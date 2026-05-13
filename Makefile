.PHONY: setup lint test test-e2e build install clean generate verify-generate verify-markers

setup:
	@echo "=> Installing global dependencies & git hooks..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@chmod +x scripts/pre-commit.sh
	@ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
	@echo "=> Git hooks installed securely."

lint:
	@echo "=> Running golangci-lint..."
	@golangci-lint run ./... || go vet ./...

test:
	@echo "=> Running tests..."
	@go test -v ./...

# Automatically extract version from the release-please manifest
VERSION := $(shell grep '"\."' .release-please-manifest.json | cut -d '"' -f 4)

COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	@echo "=> Compiling standard binary..."
	@go build -ldflags="$(LDFLAGS)" -o xcaffold ./cmd/xcaffold/...

install:
	@echo "=> Installing global binary..."
	@go install -ldflags="$(LDFLAGS)" ./cmd/xcaffold/...

test-e2e:
	@echo "=> Running E2E tests..."
	@go test -tags=e2e -v -count=1 ./test/e2e/

generate:
	@echo "=> Generating schema registry and presence extractors..."
	@go run tools/gen-schema/main.go -output pkg/schema/registry_gen.go -presence-output internal/renderer/presence_gen.go

verify-generate:
	@echo "=> Verifying generated files are fresh..."
	@go run tools/gen-schema/main.go -output /tmp/registry_gen_check.go -presence-output /tmp/presence_gen_check.go
	@diff -q pkg/schema/registry_gen.go /tmp/registry_gen_check.go || \
		(echo "ERROR: registry_gen.go is stale. Run 'make generate' to update." && exit 1)
	@diff -q internal/renderer/presence_gen.go /tmp/presence_gen_check.go || \
		(echo "ERROR: presence_gen.go is stale. Run 'make generate' to update." && exit 1)
	@echo "=> Generated files are up to date."

verify-markers:
	@echo "=> Verifying +xcaf markers are complete..."
	@go run tools/gen-schema/main.go -validate-only

clean:
	@echo "=> Cleaning up build artifacts..."
	@rm -f xcaffold
	@rm -rf .claude/ .cursor/ .agents/ .gemini/
