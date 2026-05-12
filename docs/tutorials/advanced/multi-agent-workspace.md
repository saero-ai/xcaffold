---
title: "Multi-Agent Workspace"
description: "Configure differentiated agents with distinct permissions, shared rules and skills, and validated output"
---

# Multi-Agent Workspace

This tutorial walks through configuring a team of differentiated AI agents. You will define two agents with distinct tool permissions, attach shared rules and skills, validate the workspace, visualize the topology, audit security field behavior across targets, and inspect the compiled output.

xcaffold uses a split-file layout: `project.xcaf` (kind: project) at the root and individual `.xcaf` files under `xcaf/` for each resource. Body-bearing kinds (`agent`, `skill`, `rule`) use frontmatter format; structural kinds (`project`, `settings`, `hooks`, `policy`) use pure YAML.

**Time to complete:** ~15 minutes
**Prerequisites:** Completed the Getting Started tutorial. A fresh project directory with no existing `project.xcaf`.

---

## Step 1 — Define agent roles

Before writing any YAML, answer two questions for each agent you need:

1. **What is its purpose?** A narrow, single-responsibility description prevents instruction drift.
2. **What tools does it need?** Grant the minimum set required. Tools not listed are unavailable.

For this tutorial, the team has two agents:

| Agent ID | Role | Allowed Tools | Blocked Tools |
|---|---|---|---|
| `frontend-dev` | Writes React and TypeScript components | `Read`, `Write`, `Edit`, `Bash`, `Glob`, `Grep` | — |
| `security-reviewer` | Read-only security audit | `Read`, `Glob`, `Grep` | `Write`, `Edit`, `Bash` |

Each agent directory name under `xcaf/agents/` is both the agent's ID and its output filename. `frontend-dev` compiles to `agents/frontend-dev.md`. Choose IDs that are lowercase, hyphenated, and unambiguous.

Start with the first agent only. Create two files:

`project.xcaf`:

```yaml
kind: project
version: "1.0"
name: my-team
targets:
  - claude
```

`xcaf/agents/frontend-dev/agent.xcaf`:

```
---
kind: agent
version: "1.0"
name: frontend-dev
description: "Frontend developer. React and TypeScript only."
model: "claude-sonnet-4-6"
effort: "high"
tools: [Read, Write, Edit, Bash, Glob, Grep]
---
You write React components and TypeScript.
Do not modify backend code.
```

Run a quick syntax check:

```
$ xcaffold validate
```

**Expected output (TTY glyphs shown as ASCII):**

```
my-team  .  never applied

  ok  syntax and schema

  structural warnings:
    **  agent "frontend-dev" has Bash tool but no PreToolUse hook for command validation

ok  Validation passed with 1 warning.  1 .xcaf files checked.
```

The structural warning is expected — `frontend-dev` has `Bash` in its tool list but no `PreToolUse` hook to validate commands. A production workspace should add one; for this tutorial, the warning is informational.

---

## Step 2 — Build the shared library

Rules and skills are defined in their own `.xcaf` files as top-level resources (`kind: rule`, `kind: skill`). They form a shared library that agents reference by ID. They are compiled into separate files under `.claude/rules/` and `.claude/skills/` respectively.

**Rules** enforce behavioral constraints. A rule with `paths:` activates only when the agent reads or writes matching file patterns. A rule with `activation: always` is injected regardless of context.

**Skills** are reusable prompt packages. They are compiled to `.claude/skills/<id>/SKILL.md` and loaded when an agent invokes them.

Add the second agent, then define the shared library. Each resource is its own file under `xcaf/`:

`project.xcaf`:

```yaml
kind: project
version: "1.0"
name: my-team
targets:
  - claude
```

`xcaf/agents/frontend-dev/agent.xcaf`:

```
---
kind: agent
version: "1.0"
name: frontend-dev
description: "Frontend developer. React and TypeScript only."
model: "claude-sonnet-4-6"
effort: "high"
tools: [Read, Write, Edit, Bash, Glob, Grep]
rules: ["frontend-only"]
skills: ["component-patterns"]
---
You write React components and TypeScript.
Do not modify backend code.
```

`xcaf/agents/security-reviewer/agent.xcaf`:

```
---
kind: agent
version: "1.0"
name: security-reviewer
description: "Read-only security audit agent."
model: "claude-sonnet-4-6"
effort: "high"
tools: [Read, Glob, Grep]
disallowed-tools: [Write, Edit, Bash]
rules: ["security-review-protocol"]
---
You review code for security vulnerabilities.
Never modify files. Only read and report.
```

`xcaf/rules/frontend-only/rule.xcaf`:

```
---
kind: rule
version: "1.0"
name: frontend-only
paths: ["src/components/**", "src/pages/**"]
---
Only modify files in src/components/ and src/pages/.
```

`xcaf/rules/security-review-protocol/rule.xcaf`:

```
---
kind: rule
version: "1.0"
name: security-review-protocol
activation: always
---
Always output a structured JSON report.
[CRITICAL], [HIGH], [MEDIUM], [LOW] severity must be explicitly labeled.
```

`xcaf/skills/component-patterns/skill.xcaf`:

```
---
kind: skill
version: "1.0"
name: component-patterns
description: "React component pattern library reference."
---
Refer to the React component pattern library for naming conventions,
prop typing patterns, and composition strategies.
```

Key points:
- `disallowed-tools` blocks the listed tools at runtime. Both `tools` and `disallowed-tools` use lowercase YAML keys.
- `skills:` and `rules:` on each agent are lists of IDs — the compiler resolves them from the top-level library of `kind: skill` and `kind: rule` resources.
- Skill instructions go in the frontmatter body (below the closing `---`). xcaffold compiles them to `.claude/skills/<id>/SKILL.md`.

### Layout reference

The directory layout for this workspace:

```
my-team/
  project.xcaf                      # kind: project — metadata only
  xcaf/
    agents/
      frontend-dev/
        agent.xcaf                  # kind: agent
      security-reviewer/
        agent.xcaf                  # kind: agent
    rules/
      frontend-only/
        rule.xcaf                   # kind: rule
      security-review-protocol/
        rule.xcaf                   # kind: rule
    skills/
      component-patterns/
        skill.xcaf                  # kind: skill
```

xcaffold discovers all `.xcaf` files recursively under `xcaf/`, parses each one, and merges the results before compilation. No explicit resource lists are needed in `project.xcaf`.

See [Organizing Project Resources](../../best-practices/project-structure.md) for best practices on directory organization as projects grow.

---

## Step 3 — Validate the workspace

`xcaffold validate` checks YAML syntax, cross-reference integrity, and structural invariants in a single pass.

Run it on the full workspace:

```
$ xcaffold validate
```

**Expected output:**

```
my-team  .  never applied

  ok  syntax and schema
  ok  skill directories

  structural warnings:
    **  agent "frontend-dev" has Bash tool but no PreToolUse hook for command validation

ok  Validation passed with 1 warning.  5 .xcaf files checked.
```

The validator reports three tiers:

1. **Syntax and schema** — YAML parsing and field validation. An error here means a `.xcaf` file has invalid syntax or references an unknown field.
2. **Skill directories** — skill artifact structure matches `artifacts:` declarations.
3. **Structural warnings** — non-fatal invariant checks. The Bash-without-hook warning means `frontend-dev` has `Bash` tool access but no `PreToolUse` hook to validate commands before execution. A production workspace should add one.

The exit code is `0` when only warnings are present, and non-zero when errors are found.

The `frontend-dev` agent has `Bash` in its tool list. Once you see the hook warning, that is expected here; a production workspace should add a `PreToolUse` hook to validate Bash commands before execution.

---

## Step 4 — Visualize the topology

`xcaffold graph --full` renders the complete agent topology as an ASCII tree. It shows each agent's model, effort level, allowed tools, blocked tools, and library references.

```
$ xcaffold graph --full
```

**Expected output:**

```
my-team  .  2 agents (project)  .  2 rules

══════════════════════════════════════════
  GLOBAL
══════════════════════════════════════════

══════════════════════════════════════════
  PROJECT: my-team
══════════════════════════════════════════

  ● frontend-dev
  │   tools    Read  Write  Edit  Bash  Glob  Grep  
  │
  ├── skills
  │     └── component-patterns
  │
  └── rules
        └── frontend-only

  ● security-reviewer
  │   tools    Read  Glob  Grep  
  │
  └── rules
        └── security-review-protocol

  ──────────────────────────────────────────
  RULES  (2)
    (root)   frontend-only  security-review-protocol
```

What to read in this output:

- The **header** shows the project name, agent count, and rule count.
- Under `PROJECT`, each agent node shows its allowed tools as an inline list, then its skills and rules as sub-trees.
- The `RULES` footer lists all rules in the project with their scope (`(root)` means project-level).
- `security-reviewer` has no skills, so only the `rules` sub-tree appears.

Use `--format mermaid` or `--format dot` to generate embeddable diagrams for documentation.

---

## Step 5 — Apply and inspect output

Apply the config to the `claude` target:

```
$ xcaffold apply --target claude
```

**Expected output:**

```
my-team  .  claude  .  never applied


  NEW (5 files):
    +  agents/frontend-dev.md
    +  agents/security-reviewer.md
    +  skills/component-patterns/SKILL.md
    +  rules/frontend-only.md
    +  rules/security-review-protocol.md


ok  Apply complete. 5 files written to .claude/
  Run 'xcaffold import' to sync manual edits back to .xcaf sources.
```

Five files are written: two agent files, one skill file, and two rule files. Each agent file embeds the agent's model, effort, tools, and instructions. Skills and rules are compiled to separate files under `.claude/skills/` and `.claude/rules/` respectively.

**`.claude/agents/frontend-dev.md`** (abbreviated):

```markdown
---
name: frontend-dev
description: Frontend developer. React and TypeScript only.
effort: high
tools: [Read, Write, Edit, Bash, Glob, Grep]
skills: [component-patterns]
model: claude-sonnet-4-6
---

You write React components and TypeScript.
Do not modify backend code.
```

**`.claude/agents/security-reviewer.md`** (abbreviated):

```markdown
---
name: security-reviewer
description: Read-only security audit agent.
effort: high
tools: [Read, Glob, Grep]
disallowed-tools: [Write, Edit, Bash]
model: claude-sonnet-4-6
---

You review code for security vulnerabilities.
Never modify files. Only read and report.
```

The two agent files share the same model and effort settings, but their tool lists are entirely different. `frontend-dev` has `Write`, `Edit`, and `Bash`; `security-reviewer` does not and additionally has those three tools listed under `disallowed-tools`. That field is enforced by the runtime, not just advisory. Rules are compiled to separate files under `.claude/rules/` and are not referenced in agent frontmatter.

xcaffold records source hashes in `.xcaffold/project.xcaf.state`. On the next `apply`, it compares hashes and skips compilation if nothing changed. If you manually edit a compiled file, the next `apply` will detect drift and abort unless you pass `--force`.

---

## What You Built

You configured a two-agent workspace with distinct tool permissions, defined shared rules and a skill in individual `.xcaf` files under `xcaf/`, validated structural integrity with `xcaffold validate`, and visualized the topology with `xcaffold graph --full`. You compiled the configuration to `.claude/` and verified that each agent file contains only the resources it declared.

---

## Next Steps

- **Drift remediation** — detect and restore managed files when compiled output has been modified directly: [Drift Remediation](drift-remediation.md)
- **Organize configurations** — structure resources into domain-scoped files under `xcaf/`: [Organizing Project Resources](../../best-practices/project-structure.md)
- **CLI reference** — full command reference including all flags for `apply`, `diff`, `validate`, and `graph`: [CLI Reference](../../reference/commands/index.md)
