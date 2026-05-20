---
name: Provider incompatibility
about: Report when xcaffold output does not work with a provider CLI version
title: '[provider] '
labels: provider-compat
---

## Provider
<!-- Name and version (e.g., Claude Code 1.0.20, Cursor 0.50, Gemini CLI 1.2.0) -->

## Environment

- **OS**: <!-- e.g. macOS 15.3, Ubuntu 24.04, Windows 11 -->
- **xcaffold version**: <!-- output of `xcaffold --version` -->
- **Go version** (if building from source): <!-- output of `go version` -->

## Description
<!-- Describe the incompatibility. What did xcaffold produce, and how did the provider CLI respond? -->

## Steps to Reproduce

1. Create a minimal `.xcaf` file that triggers the issue
2. Run `xcaffold apply --target <provider>`
3. Open the provider CLI or IDE
4. <!-- Describe what went wrong -->

## Minimal .xcaf File

```yaml
# Paste the smallest .xcaf file that reproduces the issue
```

## xcaffold Output

```
# Paste the contents of the generated file(s) that the provider rejects or ignores
```

## Provider Error or Behavior

```
# Paste any error messages, or describe the unexpected behavior
```
