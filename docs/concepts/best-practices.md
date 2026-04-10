---
title: "Best Practices"
description: "Configuration and structural best practices"
---

# Xcaffold Configuration Best Practices

When building complex agentic environments, organizing your `.xcf` files efficiently becomes vital for maintaining team velocity and lowering cognitive load. Below are three recommended configuration patterns.

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

Split resources into separate files logically based on what they are.

```
project/
├── project.xcf    ← Project metadata and version
├── agents.xcf     ← Agents
├── skills.xcf     ← Skills
└── workflows.xcf  ← Workflows
```

## 3. Domain Layout (Feature Focus)
**Best for:** Large, cross-functional projects. (Highly Recommended)

Xcaffold recursively scans subdirectories. You can and should organize configurations into domain-driven folders. This maps directly to CODEOWNERS rules and keeps concerns isolated.

```
project/
├── project.xcf
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

## 4. Policy Configuration Patterns

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
