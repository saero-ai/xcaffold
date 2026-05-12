---
title: "AI-Assisted Agent Authoring"
description: "Use xcaffold with an AI coding assistant to generate deterministic, provider-native agent configurations without hallucination"
---

# AI-Assisted Agent Authoring

xcaffold is designed to work alongside AI assistants — not just for human developers. When you give an AI assistant your `xcaf/` directory, it has a complete, annotated schema of every field your target providers support. The AI fills in the `.xcaf` files. You run `xcaffold apply`. No hallucinated frontmatter fields, no wrong directory structures, no provider-specific guessing.

This tutorial walks through two complementary workflows:

1. **You use AI to fill in your scaffold** — initialize with `xcaffold init`, hand the files to Claude or Gemini, get filled-in `.xcaf` resources back, then compile.
2. **AI uses xcaffold as a build tool** — the AI runs `xcaffold init` itself, edits the `.xcaf` files, and runs `xcaffold apply` to emit provider-native output.
3. **AI handles existing provider config** — the AI uses `xcaffold import` to reverse-engineer your `.claude/` or `.cursor/` directories into `.xcaf` source.

**Time to complete:** ~15 minutes  
**Prerequisites:** xcaffold installed, an AI assistant or IDE with Claude/Gemini/Copilot access, an empty project directory.

---

## Why xcaffold prevents AI hallucination

AI coding assistants know that Claude Code stores agents in `.claude/agents/*.md`, that Cursor uses `.cursor/rules/*.mdc`, that Gemini reads `.gemini/skills/*/SKILL.md`. But the exact field names, activation behaviors, and cross-field interactions differ significantly between providers — and between versions of the same provider.

The typical failure mode: an AI generates a `.claude/agents/dev.md` with `activation: always` in the frontmatter. That field does not exist for Claude Code agents — it belongs to rules. Claude Code silently ignores it. The agent ships broken with no error.

xcaffold breaks this failure mode in three steps:

1. `xcaffold init` generates `.xcaf` files for your selected targets, along with a set of **provider-aware reference files** in `xcaf/skills/xcaffold/references/`. Every field is annotated with which targets support it. The AI reads these references when editing `.xcaf` files.
2. The AI edits `.xcaf` files — not provider output. It can only set fields that the xcaffold schema accepts.
3. `xcaffold apply` validates the edited `.xcaf` against all configured policies before writing a single file. Policy violations are caught and reported before any output is written.

The AI never touches `.claude/`, `.cursor/`, or `.gemini/`. Those directories are write-only outputs owned by xcaffold.

---

## Workflow 1 — Human initializes, AI fills in the scaffold

### Step 1 — Scaffold a new project

Run `xcaffold init` with the targets your project will compile to. Select multiple targets in the interactive prompt, or pass them with `--target`:

```bash
mkdir my-project && cd my-project
xcaffold init --target claude,cursor
```

This generates:

```
my-project/
  project.xcaf                              # kind: project (at repo root)
  xcaf/
    agents/
      xaff/
        agent.xcaf                          # Xaff authoring agent (base)
        agent.claude.xcaf                   # Claude-specific overrides
        agent.cursor.xcaf                   # Cursor-specific overrides
    rules/
      xcaf-conventions/
        rule.xcaf                           # xcaffold conventions rule
    skills/
      xcaffold/
        skill.xcaf                          # self-referential xcaffold skill
        references/
          operating-guide.md
          authoring-guide.md
          agent-reference.md                # per-kind field support details
          skill-reference.md
          rule-reference.md
          workflow-reference.md
          mcp-reference.md
          hooks-reference.md
          memory-reference.md
          cli-cheatsheet.md
```

The `xcaf/agents/xaff/` directory contains Xaff, the built-in xcaffold authoring agent that knows the schema, CLI commands, and provider field support. Per-provider override files (e.g., `agent.claude.xcaf`, `agent.cursor.xcaf`) are generated alongside the base file — use these to customize the agent's behavior per target without duplicating the whole definition.

Provider field support details live in the reference files under `xcaf/skills/xcaffold/references/` (for example, `agent-reference.md`). When the AI reads these files, it has a ground-truth field catalog filtered to your selected targets.

### Step 2 — Describe your agent to the AI

Open your AI assistant and give it the contents of your agent's `.xcaf` file along with a plain-English description of what you want the agent to do. Example prompt:

> I have an xcaffold scaffold below. Fill in the `instructions` field with a focused backend API developer persona. The agent should enforce TypeScript strict mode, always prefer `async/await` over callbacks, and refuse to edit files outside `src/`. Refer to `xcaf/skills/xcaffold/references/agent-reference.md` for field support by target. Do not invent fields that are not listed in the reference. Do not include fields marked as unsupported for any of your targets.
>
> ```yaml
> [paste xcaf/agents/backend-dev.xcaf contents here]
> ```

The AI returns a completed `.xcaf` file. Because the reference file describes exactly which fields each target supports, a well-instructed AI will not place `effort:` in a way that breaks Cursor, and will not invent fields like `activation:` on an agent (that field belongs to rules).

> **Tip:** Before applying for the first time, use `xcaffold apply --backup` to save a snapshot of your existing provider output directories. This makes it easy to roll back if needed.

### Step 3 — Validate and apply

Run validate first to catch any schema errors before writing output:

```bash
xcaffold validate
```

Then apply to one or more targets:

```bash
xcaffold apply --target claude
xcaffold apply --target cursor
```

When validation passes, inspect the compiled output:

```bash
cat .claude/agents/backend-dev.md     # Claude Code native format
cat .cursor/agents/backend-dev.md     # Cursor native format
```

---

## Workflow 2 — AI runs xcaffold itself

This workflow is for agentic IDE sessions where the AI assistant has terminal access (Claude Code, Gemini CLI, Cursor with shell tools, GitHub Copilot agents). The AI treats xcaffold as a determinism layer — it uses it to generate configs that it cannot hallucinate its way around.

### What you tell the AI

Give the AI a single instruction. Example for Claude Code:

> Use `xcaffold` to initialize a new agent project in the current directory targeting `claude` and `gemini`. Then add two agents:
> 1. A **backend developer** agent with TypeScript expertise in `xcaf/agents/backend.xcaf`
> 2. A **code reviewer** agent that runs in read-only mode in `xcaf/agents/reviewer.xcaf`
>
> After writing the .xcaf files, validate with `xcaffold validate` and apply with `xcaffold apply --target claude --dry-run`. Report any policy violations.

### What the AI does

The AI runs a deterministic sequence of xcaffold commands:

```bash
# Step 1: Initialize. --json gives the AI a map of what was created.
xcaffold init --target claude,gemini --yes --json
```

This returns a machine-readable manifest of every generated file:

```json
{
  "project": "my-project",
  "targets": ["claude", "gemini"],
  "files": [
    "project.xcaf",
    "xcaf/agents/xaff/agent.xcaf",
    "xcaf/agents/xaff/agent.claude.xcaf",
    "xcaf/agents/xaff/agent.gemini.xcaf",
    "xcaf/skills/xcaffold/skill.xcaf",
    "xcaf/skills/xcaffold/references/operating-guide.md",
    "xcaf/skills/xcaffold/references/authoring-guide.md",
    "xcaf/rules/xcaf-conventions/rule.xcaf",
    "xcaf/skills/xcaffold/references/agent-reference.md",
    "xcaf/skills/xcaffold/references/skill-reference.md",
    "xcaf/skills/xcaffold/references/rule-reference.md",
    "xcaf/skills/xcaffold/references/workflow-reference.md",
    "xcaf/skills/xcaffold/references/mcp-reference.md",
    "xcaf/skills/xcaffold/references/hooks-reference.md",
    "xcaf/skills/xcaffold/references/memory-reference.md",
    "xcaf/skills/xcaffold/references/cli-cheatsheet.md"
  ]
}
```

The AI reads this manifest to know exactly which files were generated. It reads `xcaf/skills/xcaffold/references/agent-reference.md` for the field catalog, then creates new agent files (e.g., `xcaf/agents/backend.xcaf`, `xcaf/agents/reviewer.xcaf`) following the same structure as `xcaf/agents/xaff/agent.xcaf`.

```bash
# Step 2: The AI edits xcaf/agents/developer.xcaf and creates xcaf/agents/reviewer.xcaf
# (using Write or Edit tools in its tool belt)

# Step 3: Validate
xcaffold validate

# Step 4: Dry-run apply to preview output without writing files
xcaffold apply --target claude --dry-run
xcaffold apply --target gemini --dry-run
```

The AI reports the dry-run output back to you. Any policy violations from step 3 are reported with the exact field and agent, so the AI can fix and re-validate before any files are written.

### Why `--json` matters for AI workflows

Without `--json`, `xcaffold init` emits human-readable banners and interactive prompts. With `--json`, it emits only a machine-readable manifest suitable for `jq` or direct parsing inside an agentic loop. This is what makes xcaffold composable in AI tool chains:

```bash
# AI can pipeline: initialize → read manifest → determine files to edit
xcaffold init --target claude --yes --json | jq '.files[]'
```

---

## Workflow 3 — Handling existing configs

If your project already has a `.claude/` or `.cursor/` directory, use the AI to safely convert it to xcaffold source `xcaf/` directories using `xcaffold import`.

### Human prompt to AI

> This project has a `.claude/` directory. Initialize a new xcaffold project here targeting `claude` and `cursor`. Then import the existing Claude configuration into xcaffold. Show me what would be imported before actually running it.

### AI command sequence

```bash
# 1. Initialize empty scaffold
xcaffold init --target claude,cursor --yes

# 2. Preview what will be imported without writing files
xcaffold import --target claude --dry-run

# The AI reports the preview to the user. Once approved:

# 3. Import all resource types
xcaffold import --target claude

# Or import selectively by resource kind:
xcaffold import --target claude --agent --rule   # only agents and rules
xcaffold import --target claude --skill --hook   # only skills and hooks

# Imported resources are tagged with targets: [claude].
# Remove the targets field to make resources universal.

# 4. Verify the imported output
xcaffold validate
```

Once imported, the AI can now modify the configurations through the `.xcaf` files rather than editing the raw `.claude/` output.

---

## The `/xcaffold` AI skill

Every time you run `xcaffold init`, it automatically generates a self-referential skill at `xcaf/skills/xcaffold/skill.xcaf` along with reference files in `xcaf/skills/xcaffold/references/`. These include schema references, workflow cheat sheets, and provider field support details.

When you run `xcaffold apply`, this skill compiles to your project's `.claude/skills/xcaffold/` (or Cursor equivalent) directory. 

This gives you a magic loop: **your AI assistant instantly learns how to use xcaffold just by being inside an initialized project.**

### Keeping the skill up to date

When xcaffold releases a new version, the embedded reference files may be updated. Run `--upgrade` to refresh the toolkit files in place without reinitializing the whole project:

```bash
xcaffold init --upgrade --target claude,cursor --yes
```

This regenerates `xcaf/skills/xcaffold/references/*.md` and the skill scaffold with the latest content, leaving your custom agents, rules, and skills untouched. Run `xcaffold apply` afterward to compile the updated references to your provider directories.

---

## Adding a skill from scratch

Skills are the most complex resource type — they have their own directory structure in some providers (`skills/<id>/SKILL.md` in Antigravity), and their own frontmatter conventions. Ask the AI to create one using the reference:

### Human prompt to AI

> Read `xcaf/skills/xcaffold/references/skill-reference.md`. Create a new skill called `tdd` in `xcaf/skills/tdd/skill.xcaf`. The skill should:
>
> - Instruct the agent to write a failing test first, then minimal code to pass it, then refactor.
> - Apply to all selected targets (claude, cursor).
> - Set `allowed-tools` to `[Read, Write, Edit, Bash]`.
>
> Respect the provider matrix in the reference file. Do not include fields marked as `dropped` for either target. After writing the file, add `tdd` to the `skills:` list in `project.xcaf`.

### AI command sequence

The AI writes `xcaf/skills/tdd/skill.xcaf`, then runs:

```bash
xcaffold validate    # catch malformed YAML or schema violations
xcaffold apply --target claude --dry-run
```

If validation passes, you apply for real:

```bash
xcaffold apply --target claude
xcaffold apply --target cursor
```

The compiled output for Claude:

```
.claude/skills/tdd/SKILL.md     # Claude Code native skill directory
```

The compiled output for Cursor:

```
.cursor/skills/tdd/SKILL.md     # Cursor skills maintain the same directory structure
```

Note: rules compile differently — a `kind: rule` produces `.cursor/rules/<name>.mdc` for Cursor. Skills always compile to `skills/<name>/SKILL.md` for both targets.

Same source, two provider-native outputs.

---

> **Preview.** Policy enforcement is planned for a future release. The `kind: policy` resource and the `xcaf/policies/` directory are not yet supported by the compiler.

---

## What You Built

You have a project where:
- All agent configuration lives in `xcaf/` — the AI's editing surface.
- Provider-specific output is generated by xcaffold — the AI never touches `.claude/`, `.cursor/`, or `.gemini/` directly.
- The `--json` manifest flag makes the entire init workflow composable in agentic pipelines.

---

## Next Steps

- **Multi-agent workspace** — build a team of differentiated agents with shared skills: [`multi-agent-workspace.md`](../advanced/multi-agent-workspace.md)
- **CLI reference** — full flag documentation for every command: [CLI Reference](../../reference/commands/index.md)
