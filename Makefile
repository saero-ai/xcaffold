.PHONY: setup lint test build clean

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

VERSION ?= 1.0.0-dev
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	@echo "=> Compiling standard binary..."
	@go build -ldflags="$(LDFLAGS)" -o xcaffold ./cmd/xcaffold/...

clean:
	@echo "=> Cleaning up build artifacts..."
	@rm -f xcaffold
	@rm -rf .claude/
