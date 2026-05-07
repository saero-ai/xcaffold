# Security Policy

## Reporting a Vulnerability

Do **not** open a public GitHub issue for security vulnerabilities.

Send a report to **security@saero.ai**. Include:

- `xcaffold version` output
- OS and Go version
- Steps to reproduce or a proof-of-concept
- Whether you believe the issue is remotely exploitable
- Your suggested severity: low / medium / high / critical

We will acknowledge your report within 48 hours, complete an initial triage within 7 business days, and coordinate a disclosure timeline with you before any public announcement. We ask that you do not disclose the issue publicly until a patch has been released.

## Supported Versions

Security updates are provided for the latest released version only.

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Older releases | No — upgrade to the latest release |

## Scope

**In scope:**

- Vulnerabilities in xcaffold's compilation pipeline (`xcaffold apply`, `xcaffold import`, `xcaffold validate`)
- CLI argument or flag handling that could enable path traversal, arbitrary file writes, or privilege escalation
- Lock file integrity issues that could cause silent output corruption

**Out of scope:**

- Vulnerabilities in the AI provider tools themselves (Claude Code, Gemini CLI, Cursor, GitHub Copilot, etc.) — report those directly to the respective vendor
- Vulnerabilities in third-party Go dependencies — report those to the upstream maintainer or via [GitHub's advisory database](https://github.com/advisories)
- Issues that require the attacker to already have write access to the user's `.xcaf` files or project directory
