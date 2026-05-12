---
title: "xcaffold registry"
description: "List projects registered in the global registry."
---

> **Internal command** — This command is hidden from `xcaffold --help` output.

# `xcaffold registry`

Displays a list of all projects managed by `xcaffold` across your local system. `xcaffold` automatically registers projects during `init`, `import`, or `apply` to enable cross-project discovery and global management.

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
  MANAGED PROJECTS

  ● xcaffold-cli (/Users/user/dev/xcaffold)
    targets: claude
    resources: 3 agents, 5 skills, 2 rules
    last applied: today at 14:32

  ● my-new-app (/Users/user/dev/my-new-app)
    targets: cursor
    resources: 1 agents, 2 skills, 0 rules
    last applied: yesterday

  ● demo-skill (/Users/user/dev/demo-skill)
    targets: gemini
    resources: 0 agents, 1 skills, 0 rules
    last applied: 2026-05-03 09:15

  GLOBAL (~/.xcaffold/global.xcaf)
    resources: 2 agents, 4 skills, 1 rules

  3 projects registered.
```

When a registered project's `project.xcaf` is missing, the resources line shows `resources: not found (project.xcaf missing)`. When the manifest exists but fails to parse, it shows `resources: parse error`.

## Exit Codes

| Code | Meaning |
| :--- | :--- |
| `0` | Success |
| `1` | Failure (e.g., registry file unreadable) |
