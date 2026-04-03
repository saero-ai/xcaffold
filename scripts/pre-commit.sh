#!/usr/bin/env bash

# xcaffold Open-Source Pre-commit Hook
# Ensures that code formatting, static analysis, and unit tests pass before allowing a commit.

set -e

# Redirect output to stderr.
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
