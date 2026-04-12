---
title: "Best Practices"
description: "Configuration and structural best practices"
---

# Xcaffold Configuration Best Practices

When building complex agentic environments, organizing your `.xcf` files efficiently is critical. Below are three recommended configuration patterns.

## 1. Minimal (Single File Baseline)
**Best for:** Tutorials, personal scripting, toy projects.

Everything lives in a `scaffold.xcf` file at the root.

```
project/
├── scaffold.xcf
└── .claude/
```

## 2. Taxonomic Layout (Resource Type)
**Best for:** Small to medium projects with a single developer.

Split resources into separate `.xcf` files under `xcf/` subdirectories organized by resource type.

```
project/
├── scaffold.xcf          ← kind: project (name, targets)
└── xcf/
    ├── agents/
    │   └── developer.xcf ← kind: agent
    ├── skills/
    │   └── deploy.xcf    ← kind: skill
    └── workflows/
        └── release.xcf   ← kind: workflow
```

> **Alternative:** Flat files at the project root (e.g. `agents.xcf`, `skills.xcf`) also work since `ParseDirectory` discovers `.xcf` files recursively, but `xcf/` subdirectories are the recommended default.

## 3. Domain Layout (Feature Focus)
**Best for:** Large, cross-functional projects. (Highly Recommended)

Xcaffold recursively scans subdirectories. You can organize configurations into domain-driven folders under `xcf/`. This maps directly to CODEOWNERS rules and keeps concerns isolated.

```
project/
├── scaffold.xcf          ← kind: project
└── xcf/
    ├── core/
    │   ├── rules.xcf
    │   └── mcp.xcf
    ├── frontend/
    │   ├── designer-agent.xcf
    │   └── ui-skills.xcf
    └── backend/
        ├── devops-agent.xcf
        └── deployments.xcf
```

> **Alternative:** Domain folders at the project root (outside `xcf/`) also work, but placing them under `xcf/` keeps configuration files cleanly separated from source code.

## 4. Multi-Kind Project Structure

### Use `scaffold.xcf` for the Project Manifest

By convention, the `kind: project` document should live in a file named `scaffold.xcf` at the project root. This is the filename used by `xcaffold init` and `xcaffold import`, and is the first file other developers will look for.

### Split Large Projects into `xcf/` Subdirectories

For projects with many resources, place each resource in its own `.xcf` file under `xcf/` subdirectories organized by type:

```
project/
├── scaffold.xcf          <- kind: project (name, targets, resource refs)
└── xcf/
    ├── agents/
    │   ├── developer.xcf
    │   └── reviewer.xcf
    ├── skills/
    │   └── git-workflow.xcf
    └── rules/
        └── code-review.xcf
```

`ParseDirectory` discovers all `.xcf` files recursively, so the directory structure is purely organizational.

### Inline Instructions in `.xcf` Files

Use inline `instructions:` as the default. Self-contained `.xcf` files are easier to review, move, and reason about. `xcaffold import` generates inline instructions by default. `instructions_file:` exists for backward compatibility but is not the recommended approach for new configurations.

### Declare Targets in `kind: project`

Define `targets:` in the project manifest rather than relying on the `--target` flag at apply time. This makes the project's intended targets explicit and reproducible:

```yaml
kind: project
version: "1.0"
name: my-service
targets:
  - claude
  - cursor
```

### Run Commands from the Project Root

Run `xcaffold` commands from the directory containing the `scaffold.xcf` file. The parser resolves all relative paths (instructions files, references, extends) from this directory.

## 5. Policy Configuration Patterns

This section covers how to organize and configure policies effectively.

### Severity Selection

Reserve `error` severity for constraints that protect sandbox integrity or prevent compiler output corruption (path traversal, schema violations). Use `warning` for configuration quality checks (missing descriptions, empty skills).

### Override Placement

Centralize policy overrides in a single `policies/` directory rather than scattering `kind: policy` files across domain folders. This makes overrides discoverable and easy to audit.

```
project/
├── scaffold.xcf
├── policies/
│   ├── approved-models.xcf         ← custom policy
│   └── allow-empty-skills.xcf      ← override of built-in
└── backend/
    └── devops-agent.xcf
```

### Name-Based Toggling

Override a built-in policy by creating a `.xcf` file with the same `name` and `severity: off`. Remove the override file to snap back to the default constraint. This avoids polluting the central schema with bypass flags.
