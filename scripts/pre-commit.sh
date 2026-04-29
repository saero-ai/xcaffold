#!/usr/bin/env bash

# xcaffold pre-commit hook
# Verifies formatting, static analysis, and unit tests before allowing a commit.

set -e
exec 1>&2

echo "=> Running go fmt..."
if [ -n "$(go fmt ./...)" ]; then
    echo "ERROR: 'go fmt' found formatting issues. Please format your code before committing."
    exit 1
fi

echo "=> Running go vet..."
go vet ./... || {
    echo "ERROR: 'go vet' failed. Please fix static analysis errors."
    exit 1
}

echo "=> Running go test..."
go test ./... || {
    echo "ERROR: 'go test' failed. All unit tests must pass before committing."
    exit 1
}

echo "=> Pre-commit verifications passed!"
exit 0
