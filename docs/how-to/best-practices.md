---
title: "Best Practices"
description: "Recommended configuration patterns and project layout conventions for xcaffold projects"
---

# Configuration Best Practices

Configuration patterns in xcaffold range from a single file to multi-directory domain splits. The right choice depends on project size and team structure. Each pattern below includes the directory layout and a working `.xcf` example.

---

## Single File (Minimal)

**Best for:** Personal projects, tutorials, and small teams with fewer than 5 agents.

```
project/
├── scaffold.xcf
└── <target output directory>   # e.g. .claude/, .cursor/
```

A single `scaffold.xcf` can hold all resource kinds using YAML multi-document syntax:

```yaml
kind: project
version: "1.0"
name: my-service
targets:
  - claude
---
kind: agent
name: developer
description: "Implements features and writes tests"
model: claude-sonnet-4-5
instructions: |
  You implement features for this project. Follow the existing code patterns
  and write tests for all new logic before submitting a pull request.
---
kind: agent
name: reviewer
description: "Reviews pull requests for correctness and style"
model: claude-sonnet-4-5
instructions: |
  You review pull requests. Check for correctness, test coverage,
  and adherence to project conventions. Leave clear, actionable feedback.
```

---

## Split by Resource Type

**Best for:** Medium projects where one developer manages all agents, and grouping by kind (agents, skills, rules) is natural.

By convention, keep `scaffold.xcf` at the project root as the `kind: project` manifest. Place additional resources under `xcf/` subdirectories organized by resource type:

```
project/
├── scaffold.xcf          ← kind: project (name, targets)
└── xcf/
    ├── agents/
    │   └── developer.xcf
    └── skills/
        └── git-workflow.xcf
```

`ParseDirectory` discovers all `.xcf` files recursively, so the directory structure is purely organizational. The `scaffold.xcf` manifest declares the project name and targets — it does not need to enumerate the individual resource files:

```yaml
# scaffold.xcf
kind: project
version: "1.0"
name: my-service
targets:
  - claude
```

```yaml
# xcf/agents/developer.xcf
kind: agent
name: developer
description: "Implements features and writes tests"
model: claude-sonnet-4-5
instructions: |
  You implement features for this project. Follow the existing code patterns
  and write tests for all new logic. Run the test suite before declaring
  a task complete.
```

```yaml
# xcf/skills/git-workflow.xcf
kind: skill
name: git-workflow
description: "Branch, commit, and pull request conventions"
instructions: |
  ## Branch Naming
  Use feat/<name> for features, fix/<name> for bug fixes.

  ## Commit Format
  Follow Conventional Commits: type(scope): description

  ## Pull Requests
  Open a draft PR early, mark it ready when tests pass.
```

> **Alternative:** Flat files at the project root (e.g., `agents.xcf`, `skills.xcf`) also work since `ParseDirectory` discovers `.xcf` files recursively, but `xcf/` subdirectories are the recommended default.

---

## Split by Domain

**Best for:** Large cross-functional projects where different teams own different agent domains and CODEOWNERS rules apply.

Organize configurations into domain-driven folders under `xcf/`. This maps directly to CODEOWNERS rules and keeps concerns isolated:

```
project/
├── scaffold.xcf          ← kind: project
└── xcf/
    ├── frontend/
    │   └── designer-agent.xcf
    └── backend/
        └── devops-agent.xcf
```

```yaml
# scaffold.xcf
kind: project
version: "1.0"
name: my-platform
targets:
  - claude
```

```yaml
# xcf/frontend/designer-agent.xcf
kind: agent
name: designer
description: "Implements frontend interfaces"
model: claude-sonnet-4-6
instructions: |
  You build React components. Follow the established glass-morphic CSS guidelines
  and ensure components are responsive.
```

```yaml
# xcf/backend/devops-agent.xcf
kind: agent
name: devops
description: "Manages deployments and infrastructure changes"
model: claude-sonnet-4-5
instructions: |
  You handle deployments for the backend services. Before applying any
  infrastructure change, confirm the target environment. After deploying,
  verify that health checks pass and surface any rollback steps if they fail.
```

> **Alternative:** Domain folders at the project root (outside `xcf/`) also work, but placing them under `xcf/` keeps configuration files cleanly separated from source code.

---

## Inline Instructions vs `instructions-file`

Use inline `instructions:` as the default. Self-contained `.xcf` files are easier to review, move, and reason about. `xcaffold import` generates inline instructions by default.

```yaml
# Recommended — inline
kind: agent
name: reviewer
instructions: |
  You review pull requests. Check correctness, test coverage,
  and adherence to project conventions.
```

```yaml
# Avoid for new configs — file reference
kind: agent
name: reviewer
instructions-file: docs/reviewer-instructions.md
```

`instructions-file:` exists for backward compatibility and for cases where long-form prose genuinely benefits from dedicated Markdown tooling. For most configurations, inline instructions are simpler and more portable.

---

## Policy Organization

Declare `kind: policy` files alongside other resource files under `xcf/` and reference them from the `policies:` list in the `kind: project` manifest. This keeps policy definitions co-located with the resources they constrain:

```
project/
├── scaffold.xcf                    ← kind: project (includes policies: list)
└── xcf/
    └── policies/
        ├── approved-models.xcf     ← kind: policy (custom constraint)
        └── allow-empty-skills.xcf  ← kind: policy (built-in override)
```

```yaml
# scaffold.xcf
kind: project
version: "1.0"
name: my-platform
targets:
  - claude
policies:
  - approved-models
  - allow-empty-skills
```

```yaml
# xcf/policies/approved-models.xcf
kind: policy
name: approved-models
description: "Restrict agents to approved model identifiers"
match:
  kind: agent
require:
  - field: model
    values:
      - claude-sonnet-4-5
      - claude-opus-4-5
severity: error
```

```yaml
# xcf/policies/allow-empty-skills.xcf
kind: policy
name: allow-empty-skills
severity: "off"
```

Override a built-in policy by creating a `.xcf` file with the same `name` and `severity: off`. Remove the override file to snap back to the default. Reserve `error` severity for constraints that protect compiler output integrity (path traversal, schema violations). Use `warning` for configuration quality checks (missing descriptions, empty skills).

---

## Cross-Provider Translation

Use `xcaffold translate` for one-shot conversions when onboarding a new AI coding platform or sharing tools with team members using different IDEs. For ongoing management across multiple providers, use `xcaffold import` to establish a central `scaffold.xcf` project, then `xcaffold apply` to multiple targets from that single source:

```yaml
# scaffold.xcf — one source, multiple targets
kind: project
version: "1.0"
name: my-service
targets:
  - claude
  - cursor
```

Running `xcaffold apply` from this manifest compiles provider-specific output for each listed target. Running `xcaffold translate` against an existing provider directory is appropriate for a one-time migration only.

---

## Memory Lifecycle Selection

Define memory `lifecycle: seed-once` for user personalization profiles or knowledge entries where the agent is expected to learn and update the content over time. The compiler will not overwrite those entries on subsequent `xcaffold apply` runs:

```yaml
kind: agent
name: developer
memory:
  - name: user-preferences
    lifecycle: seed-once
    content: |
      Initial preferences — the agent will update this over time.
```

Use `lifecycle: tracked` only for memories that represent strictly managed documentation where the repository is the source of truth. Tracked memories are overwritten on every apply.

---

## When This Matters

- **Starting a new project and deciding on a layout** — the minimal single-file baseline is sufficient for personal or tutorial projects; the domain layout becomes relevant when multiple teams own different agent configurations and CODEOWNERS rules apply.
- **Choosing between `instructions:` and `instructions-file:`** — inline instructions are recommended by default because they keep the resource definition self-contained and are what `xcaffold import` generates; `instructions-file:` is retained for cases where long-form prose benefits from dedicated Markdown tooling.
- **Organizing policies in a large project** — placing `kind: policy` files under `xcf/policies/` and listing them in the `policies:` field of the project manifest makes the full policy surface visible in one location and keeps overrides (`severity: off`) co-located with the resources they affect.
- **Deciding between `xcaffold translate` and `xcaffold import`** — `translate` is appropriate for one-shot cross-provider migrations; `import` establishes a managed `scaffold.xcf` source intended for ongoing management via `xcaffold apply`.

---

## Related

- [Getting Started](../tutorials/getting-started.md) — creating a project using the minimal layout
- [Split Configs](../how-to/multi-file-projects.md) — step-by-step guide to splitting a single-file project into the taxonomic or domain layout
- [Import Existing Config](../how-to/import-existing-config.md) — converting an existing provider config (`.claude/`, `.cursor/`, etc.) to a managed project
- [Policy Enforcement](../how-to/policy-enforcement.md) — configuring and overriding built-in policies
- [Schema Reference](../reference/schema.md) — `kind: project`, `kind: policy`, and `lifecycle:` field definitions
