---
name: secure-coding
description: "Security coding standards enforced across all source files."
---
## Input Validation

All user-provided input must be validated at the system boundary. Never trust data from HTTP requests, CLI arguments, or external APIs without explicit validation.

- Sanitize file paths to prevent directory traversal (`filepath.Clean`, reject `..`)
- Use parameterized queries for all database operations — never string concatenation
- Validate content types before processing uploaded files
- Reject input exceeding expected length limits

## Secrets Management

Never store secrets in source code, environment variable defaults, or configuration files committed to version control. Use a secrets manager or injected environment variables at deploy time.

## Dependencies

Pin all dependency versions in go.mod. Review changelogs before upgrading. Run `go vet` and `govulncheck` as part of CI. Do not import packages that have known CVEs without a documented exception.

## Error Messages

Never expose internal state, stack traces, or database schemas in user-facing error messages. Log detailed errors server-side; return generic messages to clients.
