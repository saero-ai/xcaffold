---
title: "Best Practices"
description: "Recommended configuration patterns and project layout conventions for xcaffold projects"
---

# Configuration Best Practices

Configuration patterns in xcaffold range from a single file to multi-directory domain splits. The right choice depends on project size and team structure. Each pattern below includes the directory layout and a working `.xcf` example.

**Note on naming:** This guide uses `project.xcf` as the manifest filename. The state files that track compiled output are stored in `.xcaffold/project.xcf.state` (for the default project) or `.xcaffold/<blueprint-name>.xcf.state` (for blueprint-specific compilations). There is no lock file — state files are the single record of what has been compiled.

---

## Minimal Layout

**Best for:** Personal projects, tutorials, and small teams with fewer than 5 agents.

```
project/
├── project.xcf
├── xcf/
│   └── agents/
│       └── developer.xcf
└── <target output directory>   # e.g. .claude/, .cursor/
```

A minimal project has one manifest and one `.xcf` file per agent:

`project.xcf`:

```yaml
kind: project
version: "1.0"
name: my-service
targets:
  - claude
```

`xcf/agents/developer.xcf`:

```
---
kind: agent
name: developer
description: "Implements features and writes tests"
model: claude-sonnet-4-5
---
You implement features for this project. Follow the existing code patterns
and write tests for all new logic before submitting a pull request.
```

`xcf/agents/reviewer.xcf`:

```
---
kind: agent
name: reviewer
description: "Reviews pull requests for correctness and style"
model: claude-sonnet-4-5
---
You review pull requests. Check for correctness, test coverage,
and adherence to project conventions. Leave clear, actionable feedback.
```

---

## Split by Resource Type

**Best for:** Medium projects where one developer manages all agents, and grouping by kind (agents, skills, rules) is natural.

By convention, keep `project.xcf` at the project root as the `kind: project` manifest. Place additional resources under `xcf/` subdirectories organized by resource type:

```
project/
├── project.xcf          ← kind: project (name, targets)
└── xcf/
    ├── agents/
    │   └── developer.xcf
    └── skills/
        └── git-workflow.xcf
```

`ParseDirectory` discovers all `.xcf` files recursively, so the directory structure is purely organizational. The `project.xcf` manifest declares the project name and targets — it does not need to enumerate the individual resource files:

```yaml
# project.xcf
kind: project
version: "1.0"
name: my-service
targets:
  - claude
```

`xcf/agents/developer.xcf`:

```
---
kind: agent
name: developer
description: "Implements features and writes tests"
model: claude-sonnet-4-5
---
You implement features for this project. Follow the existing code patterns
and write tests for all new logic. Run the test suite before declaring
a task complete.
```

`xcf/skills/git-workflow.xcf`:

```
---
kind: skill
name: git-workflow
description: "Branch, commit, and pull request conventions"
---
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
├── project.xcf          ← kind: project
└── xcf/
    ├── frontend/
    │   └── designer-agent.xcf
    └── backend/
        └── devops-agent.xcf
```

```yaml
# project.xcf
kind: project
version: "1.0"
name: my-platform
targets:
  - claude
```

`xcf/frontend/designer-agent.xcf`:

```
---
kind: agent
name: designer
description: "Implements frontend interfaces"
model: claude-sonnet-4-6
---
You build React components. Follow the established glass-morphic CSS guidelines
and ensure components are responsive.
```

`xcf/backend/devops-agent.xcf`:

```
---
kind: agent
name: devops
description: "Manages deployments and infrastructure changes"
model: claude-sonnet-4-5
---
You handle deployments for the backend services. Before applying any
infrastructure change, confirm the target environment. After deploying,
verify that health checks pass and surface any rollback steps if they fail.
```

> **Alternative:** Domain folders at the project root (outside `xcf/`) also work, but placing them under `xcf/` keeps configuration files cleanly separated from source code.

---

## Frontmatter Body vs `instructions-file`

Use frontmatter body as the default. Self-contained `.xcf` files are easier to review, move, and reason about. `xcaffold import` generates them in this format.

```
---
kind: agent
name: reviewer
---
You review pull requests. Check correctness, test coverage,
and adherence to project conventions.
```

```yaml
# Avoid for new configs — file reference
kind: agent
name: reviewer
instructions-file: docs/reviewer-instructions.md
```

`instructions-file:` exists for cases where long-form prose genuinely benefits from dedicated Markdown tooling. For most configurations, frontmatter body is simpler and more portable.

---

## Skill Organization

Skills range from a single `.xcf` file to a directory with four subdirectories. Match the structure to the skill's complexity — not every skill needs all subdirectories.

**Most skills need only the `.xcf` file.** A skill that provides instructions and nothing else does not need a directory at all. Keep it as `xcf/skills/git-workflow.xcf`.

**Add `references/` when the skill needs background knowledge.** If the AI needs to read an API spec, a coding standard, or a domain glossary to execute the skill correctly, place those files in `references/`. This is the most common subdirectory — most skills that graduate from a single file add references first.

**Add `scripts/` when the skill needs to execute helpers.** Build scripts, linting wrappers, and data migration tools belong here. If the skill's instructions say "run this script," the script lives in `scripts/`.

**Add `assets/` when the skill generates output from templates.** Boilerplate files, JSON schemas, and stub generators are production artifacts that become part of the output. The distinction from `references/` is that assets are transformed or copied into the final result, while references are read-only context.

**Add `examples/` when the AI needs to see what "good" looks like.** Golden files and correct-format demonstrations help the AI match an expected output style. Use `examples/` sparingly — a few well-chosen samples are more effective than an exhaustive collection.

**Choosing between `references/` and `assets/`:**

| Question | If yes | If no |
|---|---|---|
| Does the AI read this file to inform decisions? | `references/` | — |
| Does this file become part of the compiled output? | `assets/` | — |
| Is this a template the AI fills in? | `assets/` | — |
| Is this a spec or standard the AI should follow? | `references/` | — |

**Choosing between `references/` and `examples/`:**

| Question | If yes | If no |
|---|---|---|
| Does this file describe rules or conventions? | `references/` | — |
| Does this file show a finished output sample? | `examples/` | — |

---

## Blueprint Design

Blueprints are opt-in. A project with fewer than 5 agents typically doesn't need them — `xcaffold apply` compiles everything by default.

**When to introduce blueprints:**
- Multiple developers work on different subsystems (backend vs. frontend vs. infra)
- Context switching between roles (reviewing code vs. writing features vs. debugging)
- Different environments need different agent configurations

**Granularity guidance:**
- One blueprint per role or workflow, not per individual agent
- If a blueprint has fewer than 2 agents, it's probably too granular
- If a blueprint has more than 10 agents, consider splitting by domain

**Keep blueprints lean:**
- Prefer transitive dependencies — selecting an agent auto-includes its skills and rules
- Only list skills/rules explicitly in the blueprint when you need to override the agent's defaults
- Use `xcaffold list --blueprint <name> --resolved` to see what actually gets compiled

**Named settings and hooks:**
- Use the default (unnamed) settings for shared configuration
- Create named settings only when a blueprint needs materially different platform behavior
- Same principle for hooks — most projects need one set of hooks, not one per blueprint

**Drift hygiene:**
- Run `xcaffold status --blueprint <name>` before switching between blueprints
- Each blueprint maintains independent state — drifting in one doesn't affect another

---

## Policy Organization

Declare `kind: policy` files alongside other resource files under `xcf/` and reference them from the `policies:` list in the `kind: project` manifest. This keeps policy definitions co-located with the resources they constrain:

```
project/
├── project.xcf                    ← kind: project (includes policies: list)
└── xcf/
    └── policies/
        ├── approved-models.xcf     ← kind: policy (custom constraint)
        └── allow-empty-skills.xcf  ← kind: policy (built-in override)
```

```yaml
# project.xcf
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

Use `xcaffold translate` for one-shot conversions when onboarding a new AI coding platform or sharing tools with team members using different IDEs. For ongoing management across multiple providers, use `xcaffold import` to establish a central `project.xcf` project, then `xcaffold apply` to multiple targets from that single source:

```yaml
# project.xcf — one source, multiple targets
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

- **Starting a new project and deciding on a layout** — the minimal two-file baseline (`project.xcf` + one agent file) is sufficient for personal or tutorial projects; the domain layout becomes relevant when multiple teams own different agent configurations and CODEOWNERS rules apply.
- **Choosing between frontmatter body and `instructions-file:`** — frontmatter body is the default because it keeps the resource definition self-contained and is what `xcaffold import` generates; `instructions-file:` is retained for cases where long-form prose benefits from dedicated Markdown tooling.
- **Organizing policies in a large project** — placing `kind: policy` files under `xcf/policies/` and listing them in the `policies:` field of the project manifest makes the full policy surface visible in one location and keeps overrides (`severity: off`) co-located with the resources they affect.
- **Deciding when to introduce blueprints** — most projects don't need them; if you find yourself managing separate agent sets for different environments or teams, blueprints keep state separate and prevent drift in one from affecting another.
- **Deciding between `xcaffold translate` and `xcaffold import`** — `translate` is appropriate for one-shot cross-provider migrations; `import` establishes a managed `project.xcf` source intended for ongoing management via `xcaffold apply`.

---

## Related

- [Getting Started](../tutorials/getting-started.md) — creating a project using the minimal layout
- [Split Configs](../how-to/multi-file-projects.md) — step-by-step guide to splitting a single-file project into the taxonomic or domain layout
- [Import Existing Config](../how-to/import-existing-config.md) — converting an existing provider config (`.claude/`, `.cursor/`, etc.) to a managed project
- [Policy Enforcement](../how-to/policy-enforcement.md) — configuring and overriding built-in policies
- [Schema Reference](../reference/schema.md) — `kind: project`, `kind: policy`, and `lifecycle:` field definitions
