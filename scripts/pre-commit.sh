#!/usr/bin/env bash

# xcaffold Open-Source Pre-commit Hook
# Ensures that code formatting, static analysis, and unit tests pass before allowing a commit.

set -e

# Redirect output to stderr.
exec 1>&2

echo "=> Checking index integrity..."
if git fsck --no-dangling 2>&1 | grep -q "missing blob"; then
    echo "ERROR: Git index references missing objects. The index may be corrupted."
    echo "Fix: rm .git/worktrees/<name>/index && git read-tree HEAD"
    exit 1
fi

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
