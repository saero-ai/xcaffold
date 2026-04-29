---
title: "Multi-Agent Workspace"
description: "Configure differentiated agents with distinct permissions, shared rules and skills, and validated output"
---

# Multi-Agent Workspace

This tutorial walks through configuring a team of differentiated AI agents. You will define two agents with distinct tool permissions, attach shared rules and skills, validate the workspace, visualize the topology, audit security field behavior across targets, and inspect the compiled output.

xcaffold uses a split-file layout: `project.xcf` (kind: project) at the root and individual `.xcf` files under `xcf/` for each resource. Body-bearing kinds (`agent`, `skill`, `rule`) use frontmatter format; structural kinds (`project`, `settings`, `hooks`, `policy`) use pure YAML.

**Time to complete:** ~15 minutes
**Prerequisites:** Completed the Getting Started tutorial. A fresh project directory with no existing `project.xcf`.

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

The `agents:` map uses each key as both the agent's internal ID and its output filename. `frontend-dev` compiles to `agents/frontend-dev.md`. Choose IDs that are lowercase, hyphenated, and unambiguous.

Start with the first agent only. Create two files:

`project.xcf`:

```yaml
kind: project
version: "1.0"
name: my-team
targets:
  - claude
```

`xcf/agents/frontend-dev/agent.xcf`:

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

**Expected output:**

```
syntax and cross-references: ok

validation passed
```

---

## Step 2 — Build the shared library

Rules and skills are defined in their own `.xcf` files as top-level resources (`kind: rule`, `kind: skill`). They form a shared library that agents reference by ID. They are compiled into separate files under `.claude/rules/` and `.claude/skills/` respectively.

**Rules** enforce behavioral constraints. A rule with `paths:` activates only when the agent reads or writes matching file patterns. A rule with `always-apply: true` is injected regardless of context.

**Skills** are reusable prompt packages. They are compiled to `.claude/skills/<id>/SKILL.md` and loaded when an agent invokes them.

Add the second agent, then define the shared library. Each resource is its own file under `xcf/`:

`project.xcf`:

```yaml
kind: project
version: "1.0"
name: my-team
targets:
  - claude
```

`xcf/agents/frontend-dev/agent.xcf`:

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

`xcf/agents/security-reviewer/agent.xcf`:

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

`xcf/rules/frontend-only/rule.xcf`:

```
---
kind: rule
version: "1.0"
name: frontend-only
paths: ["src/components/**", "src/pages/**"]
---
Only modify files in src/components/ and src/pages/.
```

`xcf/rules/security-review-protocol/rule.xcf`:

```
---
kind: rule
version: "1.0"
name: security-review-protocol
always-apply: true
---
Always output a structured JSON report.
[CRITICAL], [HIGH], [MEDIUM], [LOW] severity must be explicitly labeled.
```

`xcf/skills/component-patterns/skill.xcf`:

```
---
kind: skill
version: "1.0"
name: component-patterns
description: "React component pattern library reference."
instructions-file: "skills/component-patterns/SKILL.md"
---
```

Key points:
- `disallowed-tools` (lowercase `d`) is the YAML key. It corresponds to the `DisallowedTools` field in the Go AST.
- `skills:` and `rules:` on each agent are lists of IDs — the compiler resolves them from the top-level library of `kind: skill` and `kind: rule` documents.
- The `component-patterns` skill references `instructions-file:`. That file must exist on disk relative to `project.xcf` before you run `apply`.

### Layout reference

The directory layout for this workspace:

```
my-team/
  project.xcf                      # kind: project — metadata only
  xcf/
    agents/
      frontend-dev/
        agent.xcf                  # kind: agent
      security-reviewer/
        agent.xcf                  # kind: agent
    rules/
      frontend-only/
        rule.xcf                   # kind: rule
      security-review-protocol/
        rule.xcf                   # kind: rule
    skills/
      component-patterns/
        skill.xcf                  # kind: skill
```

`ParseDirectory` discovers all `.xcf` files recursively, parses each one, and merges the results into a single AST before compilation. The parser uses file discovery to find resources — no explicit ref lists needed in `project.xcf`.

See [Organizing Project Resources](../how-to/multi-file-projects.md) for best practices on directory organization as projects grow.

---

## Step 3 — Validate the workspace

`xcaffold validate` checks YAML syntax and cross-reference integrity. The `--structural` flag adds a second pass that detects orphan resources, agents without instructions, and agents with `Bash` access but no `PreToolUse` hook.

Run without `--structural` first:

```
$ xcaffold validate
```

**Expected output:**

```
syntax and cross-references: ok

validation passed
```

Now add a rule that has no `paths:`, no `always-apply: true`, and is not referenced by any agent, to see what a structural warning looks like. Create `xcf/rules/orphan-rule/rule.xcf`:

```
---
kind: rule
version: "1.0"
name: orphan-rule
---
This rule is unreachable.
```

Run with `--structural`:

```
$ xcaffold validate --structural
```

**Expected output:**

```
syntax and cross-references: ok

structural warnings:
  - rule "orphan-rule" is defined but not referenced by any agent and has no paths or always-apply

validation passed
```

The exit code is still `0` — structural warnings are informational, not errors. Remove the orphan rule before continuing.

The warning format strings the compiler uses:

| Condition | Warning |
|---|---|
| Skill defined, no agent references it | `skill %q is defined but not referenced by any agent` |
| Rule with no paths, no always-apply, no agent reference | `rule %q is defined but not referenced by any agent and has no paths or always-apply` |
| Agent with no instructions or instructions-file | `agent %q has no instructions or instructions-file` |
| Agent with Bash, no PreToolUse hook | `agent %q has Bash tool but no PreToolUse hook for command validation` |

The `frontend-dev` agent has `Bash` in its tool list. Once you run `validate --structural` on the final config, you will see the hook warning. That is expected here; a production workspace should add a `PreToolUse` hook to validate Bash commands before execution.

---

## Step 4 — Visualize the topology

`xcaffold graph --full` renders the complete agent topology as an ASCII tree. It shows each agent's model, effort level, allowed tools, blocked tools, and library references.

```
$ xcaffold graph --full
```

**Expected output:**

```
┌──────────────────────────────────────────────────┐
│  my-team  •  2 agents  •  1 skills  •  2 rules  │
└──────────────────────────────────────────────────┘
  [ AGENTS ]
  ● frontend-dev [claude-sonnet-4-6 · high effort]
      │
      ├─▶ [Capabilities]
      │    ├─(tool)─▶ Read
      │    ├─(tool)─▶ Write
      │    ├─(tool)─▶ Edit
      │    ├─(tool)─▶ Bash
      │    ├─(tool)─▶ Glob
      │    └─(tool)─▶ Grep
      ├─▶ [Skills]
      │    └─▶ component-patterns
      └─▶ [Rules]
           └─▶ frontend-only

  ● security-reviewer [claude-sonnet-4-6 · high effort]
      │
      ├─▶ [Capabilities]
      │    ├─(tool)─▶ Read
      │    ├─(tool)─▶ Glob
      │    ├─(tool)─▶ Grep
      │    ├─(blocked)─▶ Write
      │    ├─(blocked)─▶ Edit
      │    └─(blocked)─▶ Bash
      └─▶ [Rules]
           └─▶ security-review-protocol

  [ LIBRARY ]
  ● rule: frontend-only
      └─(paths)─▶ src/components/**, src/pages/**
```

What to read in this output:

- The **header** shows the project name and resource counts.
- Under `[ AGENTS ]`, each agent node lists all tools under a single `[Capabilities]` section. Allowed tools use `(tool)` as their kind label; disallowed tools use `(blocked)`. Both appear in the same block.
- Under `[ LIBRARY ]`, only resources with sub-items (such as `paths:`) are rendered. `frontend-only` appears because it declares `paths:`. `component-patterns` and `security-review-protocol` have no paths or tool sub-items, so they are omitted from this section.

Use `--format mermaid` or `--format dot` to generate embeddable diagrams for documentation.

---

## Step 5 — Audit security permissions

`xcaffold apply --check-permissions` inspects which security fields your config declares and reports what the target platform will drop. This is a read-only check — no files are written.

### Checking the `cursor` target

The `cursor` target does not have a native concept of disallowed tools. Any `disallowed-tools` declared on an agent is silently dropped during compilation. `--check-permissions` surfaces this before you apply.

```
$ xcaffold apply --check-permissions --target cursor
```

**Expected output:**

```
[WARNING] cursor: agent "security-reviewer" disallowed-tools will be dropped — tool restrictions will NOT be enforced
[WARNING] cursor: settings.permissions will be dropped — no enforcement equivalent
[WARNING] cursor: settings.sandbox will be dropped — no sandbox model
```

The second and third warnings only appear if your config has `settings.permissions` or `settings.sandbox` blocks. In this tutorial's config they do not exist, so only the first warning appears. The output above shows all possible cursor warnings for reference.

The key warning for this config is the first one: the `security-reviewer` is declared as read-only via `disallowed-tools`, but that declaration has no effect when compiled for `cursor`. An agent that appears constrained in your YAML source has full tool access in the compiled output.

### Checking the `claude` target

```
$ xcaffold apply --check-permissions --target claude
```

**Expected output:**

```
[INFO]    claude: all security fields are supported
```

The `claude` target enforces `disallowed-tools` at runtime. The `security-reviewer`'s restrictions are compiled into the agent file and respected.

The `--check-permissions` flag exits `0` when only warnings are present, and non-zero when errors are found. Errors occur when a `settings.permissions.deny` rule conflicts with a tool in an agent's `tools:` list.

---

## Step 6 — Apply and inspect output

Apply the config to the `claude` target:

```
$ xcaffold apply --target claude
```

**Expected output:**

```
  [project] ✓ wrote .claude/agents/frontend-dev.md  (sha256:<hex>)
  [project] ✓ wrote .claude/agents/security-reviewer.md  (sha256:<hex>)

[project] ✓ Apply complete. .xcaffold/project.xcf.state updated.
```

Two agent files are written, one per agent ID. Each file is self-contained — it embeds the agent's model, effort, tools, instructions, and resolved rule content. The shared library resources were referenced during compilation but each agent receives only what it declared.

**`.claude/agents/frontend-dev.md`** (abbreviated):

```markdown
---
description: Frontend developer. React and TypeScript only.
model: claude-sonnet-4-6
effort: high
tools: [Read, Write, Edit, Bash, Glob, Grep]
skills: [component-patterns]
rules: [frontend-only]
---

You write React components and TypeScript.
Do not modify backend code.
```

**`.claude/agents/security-reviewer.md`** (abbreviated):

```markdown
---
description: Read-only security audit agent.
model: claude-sonnet-4-6
effort: high
tools: [Read, Glob, Grep]
disallowed-tools: [Write, Edit, Bash]
rules: [security-review-protocol]
---

You review code for security vulnerabilities.
Never modify files. Only read and report.
```

The two files share the same model and effort settings, but their tool lists are entirely different. `frontend-dev` has `Write`, `Edit`, and `Bash`; `security-reviewer` does not and additionally has those three tools listed under `disallowed-tools`. That field is enforced by the runtime, not just advisory. Rules are compiled to separate files under `.claude/rules/` — agent files reference them by ID in the `rules:` frontmatter field rather than inlining their content.

The SHA-256 hash on each write line is recorded in `.xcaffold/project.xcf.state`. On the next `apply`, xcaffold compares source hashes and skips compilation if nothing changed. If you manually edit a compiled file, the next `apply` will detect drift and abort unless you pass `--force`.

---

## What You Built

You configured a two-agent workspace with distinct tool permissions, defined shared rules and a skill in individual `.xcf` files under `xcf/`, validated structural integrity with `xcaffold validate --structural`, and audited how security fields behave across targets. You compiled the configuration to `.claude/` and verified that each agent file contains only the resources it declared.

---

## Next Steps

- **Drift remediation** — detect and restore managed files when compiled output has been modified directly: [Drift Remediation](drift-remediation.md)
- **Organize configurations** — structure resources into domain-scoped files under `xcf/`: [Organizing Project Resources](../how-to/multi-file-projects.md)
- **Policy enforcement** (Preview) — add `require` and `deny` constraints that block compilation when violated: [Policy Enforcement](../how-to/policy-enforcement.md)
- **CLI reference** — full command reference including all flags for `apply`, `diff`, `validate`, and `graph`: [CLI Reference](../reference/cli.md)
