---
title: "Getting Started with xcaffold"
description: "Initialize a project, compile your first agent, and understand the .xcaf to output pipeline"
---

# Getting Started with xcaffold

This tutorial walks through creating your first harness configuration with xcaffold. By the end, you will have a working `project.xcaf` file compiled to your target platform's native format. This is Harness-as-Code in action — declare once, compile everywhere.

**Time to complete:** ~10 minutes  
**Prerequisites:** xcaffold installed and available on `$PATH`, any text editor, an empty project directory.

No AI subscription or API key is required to run `init` and `apply`.

---

## Step 1 — Initialize a new project

Open a terminal in your empty project directory and run:

```bash
xcaffold init --yes
```

The `--yes` flag accepts all defaults non-interactively. xcaffold infers the project name from the directory name — it never prompts for one.

**Expected output:**

```
xcaf-doc-test-getting-started  .  never applied

  Initializing xcaffold project.

  ok project.xcaf
  ok xcaf/agents/xaff/                     base + 1 override
  ok xcaf/skills/xcaffold/
  ok xcaf/rules/xcaf-conventions/
  ok xcaf/skills/xcaffold/references/    10 references

-> Run 'xcaffold validate' then 'xcaffold apply'.
  Includes Xaff agent, xcaffold skill, and xcaf-conventions rule.
```

xcaffold detects the first available agent runtime on your `$PATH` and uses it as the suggested default target. All five targets — `claude`, `cursor`, `antigravity`, `copilot`, and `gemini` — are fully supported and selected via `--target` on apply.

The `init` command generates a scalable multi-file directory layout for your project. This prevents configuration bloat as your agent ecosystem grows:

```
my-project/
  project.xcaf              # kind: project
  xcaf/
    agents/
      xaff/
        agent.xcaf           # kind: agent (base)
        agent.antigravity.xcaf  # provider override
    rules/
      xcaf-conventions/
        rule.xcaf            # kind: rule
    skills/
      xcaffold/
        skill.xcaf           # kind: skill
        references/          # 10 reference .md files
```

Open `project.xcaf` in your editor. It acts as the project manifest:

```yaml
kind: project
version: "1.0"
name: my-project
targets:
  - claude
```

`project.xcaf` declares the project identity and compilation targets. Resources — agents, rules, skills — are defined in separate files under `xcaf/` and discovered automatically when you run `xcaffold apply`. You do not list them here.

The starter agent is scaffolded in `xcaf/agents/xaff/`. Xaff is an xcaffold authoring agent: it knows the `.xcaf` schema and can author and maintain your xcaffold resources.

This structure is the source of truth. Every file xcaffold produces traces back to this `xcaf/` configuration layout. You never edit generated output directly.

---

## Step 1b — Validate before applying

Before compiling, run the validator to check syntax and structure:

```bash
xcaffold validate
```

A clean project produces no output and exits 0. Fix any reported errors before proceeding to apply.

---

## Step 2 — Compile to your target

Run apply with an explicit target. This example uses `claude` (Claude Code's native directory format):

```bash
xcaffold apply --target claude --yes
```

The `--yes` flag skips the confirmation prompt. Without it, apply lists the files it will write and asks for confirmation before proceeding.

**Terminal output:**

```
my-project  .  claude  .  never applied


  NEW (13 files):
    +  skills/xcaffold/references/cli-cheatsheet.md
    +  skills/xcaffold/references/rule-reference.md
    +  skills/xcaffold/references/mcp-reference.md
    +  skills/xcaffold/references/memory-reference.md
    +  agents/xaff.md
    +  skills/xcaffold/references/skill-reference.md
    +  skills/xcaffold/references/workflow-reference.md
    +  skills/xcaffold/references/agent-reference.md
    +  skills/xcaffold/references/hooks-reference.md
    +  skills/xcaffold/references/operating-guide.md
    +  rules/xcaf-conventions.md
    +  skills/xcaffold/SKILL.md
    +  skills/xcaffold/references/authoring-guide.md

ok  Apply complete. 13 files written to .claude/
  Run 'xcaffold import' to sync manual edits back to .xcaf sources.
```

Inspect the compiled agent:

```bash
cat .claude/agents/xaff.md
```

To also compile for a second target, run apply again with a different target:

```bash
xcaffold apply --target gemini --yes
```

---

## Step 3 — Inspect the compiled output

`.claude/agents/xaff.md` contains exactly what `xcaf/agents/xaff/agent.xcaf` described, in YAML frontmatter + Markdown body format:

```markdown
---
name: xaff
description: xcaffold authoring agent. Knows the xcaffold schema, CLI commands, and provider field support.
tools: [Read, Write, Edit, Glob, Grep]
skills: [xcaffold]
---

You are Xaff, the xcaffold authoring agent.

xcaffold is a deterministic agent configuration compiler. It compiles `.xcaf` source
files into native AI provider output (.claude/, .cursor/, .gemini/, etc.).
...
```

Every line maps back to `xcaf/agents/xaff/agent.xcaf`:

| Output field | Source field |
|---|---|
| `name: xaff` | `name: xaff` |
| `description:` | `description:` |
| `tools:` list | `tools:` array |
| `skills:` list | `skills:` array |
| Body text | File body (below the `---` separator) |

> **Note on fidelity:** Different targets support different fields. The `claude` target preserves all fields. Targets with a narrower feature set drop fields they have no equivalent for. The `agent.antigravity.xcaf` file in `xcaf/agents/xaff/` provides a provider-specific override that is merged in automatically when compiling for the `antigravity` target.

---

## Step 4 — Examine the state file

`xcaffold apply` writes a state file alongside your configuration:

```
.xcaffold/project.xcaf.state
```

Its content:

```yaml
version: 1
xcaffold-version: 0.2.0-dev
source-files:
    - path: project.xcaf
      hash: sha256:ac46b79d...
    - path: xcaf/agents/xaff/agent.antigravity.xcaf
      hash: sha256:b670e087...
    - path: xcaf/agents/xaff/agent.xcaf
      hash: sha256:cd12527d...
    - path: xcaf/rules/xcaf-conventions/rule.xcaf
      hash: sha256:53483cef...
    - path: xcaf/skills/xcaffold/skill.xcaf
      hash: sha256:50e447ec...
targets:
    claude:
        last-applied: "2026-05-12T11:02:11Z"
        artifacts:
            - path: agents/xaff.md
              hash: sha256:76ab6baa...
            - path: rules/xcaf-conventions.md
              hash: sha256:03f08936...
            - path: skills/xcaffold/SKILL.md
              hash: sha256:97669815...
            # ... (13 total artifacts)
```

The state file records SHA-256 hashes of both source files and compiled output. It enables deterministic recompilation: `xcaffold apply` skips recompilation when source hashes are unchanged.

**What this enables:**

- `xcaffold apply` skips recompilation when source hashes are unchanged:

  ```
  Sources unchanged — skipping compilation. Use --force to recompile.
  ```

- `xcaffold status` compares the current file hashes against the state to detect manual edits. Any change to compiled output made outside xcaffold registers as drift.

To inspect drift between source and compiled output, run:

```bash
xcaffold status
```

---

## Step 5 — Make a change and re-apply

Open `xcaf/agents/xaff/agent.xcaf` in your editor. Add a line to the body section (below the `---` separator):

```yaml
---
kind: agent
version: "1.0"
name: xaff
description: xcaffold authoring agent. Knows the xcaffold schema, CLI commands, and provider field support.
tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
skills:
  - xcaffold
rules:
  - xcaf-conventions
---
You are Xaff, the xcaffold authoring agent.
Always run 'xcaffold validate' before 'xcaffold apply'.
```

Run apply again:

```bash
xcaffold apply --target claude --yes
```

The new instruction line appears in `.claude/agents/xaff.md`. The state file's artifact hash updates. The source `xcaf/agents/xaff/agent.xcaf` is the only file you edited — xcaffold propagates the change to the compiled output.

---

## What You Built

You initialized a project with `xcaffold init`, producing a multi-file `xcaf/` layout with the Xaff authoring agent, the xcaffold skill, and the xcaf-conventions rule. You compiled the project to `.claude/` with `xcaffold apply`, inspected the 13 generated files, and verified that the state file records SHA-256 hashes of both source and output. You then edited the agent source and reapplied, confirming that xcaffold recompiles only the files whose sources changed.

---

## Next Steps

- **Drift remediation** — learn how `xcaffold status` detects and resolves manual edits in compiled output: [`drift-remediation.md`](../advanced/drift-remediation.md)
- **Multi-agent workspaces** — define multiple agents, skills, and rules: [`multi-agent-workspace.md`](../advanced/multi-agent-workspace.md)
- **Organize configurations** — structure resources into domain-scoped files under `xcaf/`: [Project Structure](../../best-practices/project-structure.md)
- **Import existing config** — adopt xcaffold on an existing `.claude/` project: [import command reference](../../reference/commands/lifecycle/import.md)
- **Other targets** — compile the same configuration to `.cursor/` (`--target cursor`), `.gemini/` (`--target gemini`), `.github/` (`--target copilot`), or `.agents/` (`--target antigravity`) without changing `project.xcaf`.
