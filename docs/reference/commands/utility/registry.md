---
title: "xcaffold registry"
description: "List projects registered in the global registry."
---

# `xcaffold registry`

Displays a list of all projects managed by `xcaffold` across your local system. `xcaffold` automatically registers projects during `init` or `import` to enable cross-project discovery and global management.

## Usage

```bash
xcaffold registry [flags]
```

## Behavior

The `registry` command scans the global configuration directory (typically `~/.xcaffold/`) and prints a summary of:

-   **Managed Projects**: The name, filesystem path, compilation targets, and last applied timestamp for every registered project.
-   **Resource Summary**: A count of agents, skills, and rules discovered in each project's `project.xcaf`.
-   **Global Scope**: A summary of the user-wide global configuration.

## Examples

**List all registered projects:**

```bash
xcaffold registry
```

## Sample Output

```text
xcaffold  ·  global registry  ·  ~/.xcaffold/registry.xcf

PROJECT       PATH                          TARGETS    LAST APPLIED
xcaffold-cli  /Users/user/dev/xcaffold      claude     2h ago
my-new-app    /Users/user/dev/my-new-app    cursor     yesterday
demo-skill    /Users/user/dev/demo-skill    gemini     5d ago

→ 3 projects registered.
```

## Exit Codes

| Code | Meaning |
| :--- | :--- |
| `0` | Success |
| `1` | Failure (e.g., registry file unreadable) |
