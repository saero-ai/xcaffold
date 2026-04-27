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

echo "=> Checking for prohibited patterns..."
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ROOT="$(cd "$REPO_ROOT/.." && pwd)"

# Look for prohibited patterns file in the parent project
PATTERNS=""
for p in "$PROJECT_ROOT/.claude/hooks/prohibited-patterns.grep" \
         "$PROJECT_ROOT/.gemini/hooks/prohibited-patterns.grep"; do
  if [ -f "$p" ]; then
    PATTERNS="$p"
    break
  fi
done

if [ -n "$PATTERNS" ]; then
  VIOLATIONS=0
  for file in $(git diff --cached --name-only --diff-filter=ACM); do
    case "$file" in
      *.md|*.go|*.ts|*.tsx|*.json|*.yaml|*.yml|*.sh)
        while IFS= read -r pattern; do
          [[ -z "$pattern" || "$pattern" == \#* ]] && continue
          if git show ":$file" 2>/dev/null | grep -qiE "$pattern"; then
            echo "BLOCKED: '$file' contains prohibited pattern: $pattern" >&2
            VIOLATIONS=$((VIOLATIONS + 1))
          fi
        done < "$PATTERNS"
        ;;
    esac
  done
  if [ $VIOLATIONS -gt 0 ]; then
    echo "ERROR: $VIOLATIONS prohibited pattern(s) found in staged files." >&2
    echo "Fix the violations and try again." >&2
    exit 1
  fi
fi

echo "=> Pre-commit verifications passed!"
exit 0
