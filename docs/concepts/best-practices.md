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
