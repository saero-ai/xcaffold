---
name: Provider incompatibility
about: Report when xcaffold output does not work with a provider CLI version
title: '[provider] '
labels: 'provider-compat'
assignees: ''
---

**Provider**
Name and version (e.g., Claude Code 1.0.20, Cursor 0.50, Gemini CLI 1.2.0):

**xcaffold version**
Output of `xcaffold version`:

**What happened**
Describe the incompatibility. What did xcaffold produce, and how did the provider CLI respond?

**Reproduction**
1. Create a minimal `.xcaf` file that triggers the issue
2. Run `xcaffold apply --target <provider>`
3. Open the provider CLI or IDE
4. Describe what went wrong

**Minimal .xcaf file**
```yaml
# Paste the smallest .xcaf file that reproduces the issue
```

**xcaffold output**
```
# Paste the contents of the generated file(s) that the provider rejects or ignores
```

**Provider error or behavior**
```
# Paste any error messages, or describe the unexpected behavior
```

**Environment**
- OS: [e.g. macOS 15, Ubuntu 24.04]
- Go version (if building from source): [e.g. 1.24.2]
