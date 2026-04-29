---
title: "AI-Assisted Agent Authoring"
description: "Use xcaffold with an AI assistant  to generate deterministic, provider-native agent configurations without hallucination"
---

# AI-Assisted Agent Authoring

xcaffold is designed to work alongside AI assistants — not just for human developers. When you give an AI assistant your `xcf/` directory, it has a complete, annotated schema of every field your target providers support. The AI fills in the `.xcf` files. You run `xcaffold apply`. No hallucinated frontmatter fields, no wrong directory structures, no provider-specific guessing.

This tutorial walks through two complementary workflows:

1. **You use AI to fill in your scaffold** — initialize with `xcaffold init`, hand the files to Claude or Gemini, get filled-in `.xcf` resources back, then compile.
2. **AI uses xcaffold as a build tool** — the AI runs `xcaffold init` itself, edits the `.xcf` files, and runs `xcaffold apply` to emit provider-native output.
3. **AI handles existing provider config** — the AI uses `xcaffold import` to reverse-engineer your `.claude/` or `.cursor/` directories into `.xcf` source.

**Time to complete:** ~15 minutes  
**Prerequisites:** xcaffold installed, an AI assistant or IDE with Claude/Gemini/Copilot access, an empty project directory.

---

## Why xcaffold prevents AI hallucination

AI coding assistants know that Claude Code stores agents in `.claude/agents/*.md`, that Cursor uses `.cursor/rules/*.mdc`, that Gemini reads `.gemini/skills/*/SKILL.md`. But the exact field names, activation behaviors, and cross-field interactions differ significantly between providers — and between versions of the same provider.

The typical failure mode: an AI generates a `.claude/agents/dev.md` with `activation: always` in the frontmatter. That field does not exist for Claude Code agents — it belongs to rules. Claude Code silently ignores it. The agent ships broken with no error.

xcaffold breaks this failure mode in three steps:

1. `xcaffold init` generates `.xcf` files with an **embedded provider matrix** for your selected targets. Every field is annotated with which targets support it. The AI has a ground-truth reference baked into the file it is editing.
2. The AI edits `.xcf` files — not provider output. It can only set fields that the xcaffold schema accepts.
3. `xcaffold apply` validates the edited `.xcf` against all configured policies before writing a single file. Policy violations are caught and reported before any output is written.

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
  project.xcf              # kind: project
  xcf/
    agents/
      developer.xcf         # kind: agent (with provider matrix)
    rules/
      conventions.xcf       # kind: rule (with provider matrix)
    settings.xcf            # kind: settings (MCP, permissions)
    references/
      agent.xcf.reference   # full annotated field catalog
      skill.xcf.reference   # full annotated field catalog
```

Every `.xcf` file opens with a commented **provider support matrix** filtered to your selected targets (`claude`, `cursor` in this example):

```yaml
# kind: agent - provider field support for your selected targets
#
#  Field                 claude    cursor
#  name / description    YES       YES
#  model                 YES       dropped
#  effort                YES       dropped
#  permission-mode       YES       dropped
#  tools                 YES       YES
#  skills / rules / mcp  YES       YES
#  hooks                 YES       dropped
#  memory                YES       dropped
#  targets: overrides    YES       YES
```

This matrix is the single source of truth the AI will use when filling in the config.

### Step 2 — Describe your agent to the AI

Open your AI assistant and give it the contents of `xcf/agents/developer.xcf` along with a plain-English description of what you want the agent to do. Example prompt:

> I have an xcaffold scaffold below. Fill in the `instructions` field with a focused backend API developer persona. The agent should enforce TypeScript strict mode, always prefer `async/await` over callbacks, and refuse to edit files outside `src/`. Respect the provider matrix — if a field says `dropped` for `cursor`, leave it as-is or remove it. Do not invent fields that are not already present in the YAML.
>
> ```yaml
> [paste xcf/agents/developer.xcf contents here]
> ```

The AI returns a completed `xcf/agents/developer.xcf`. Because the matrix is embedded in the file, a well-instructed AI will not place `effort:` in a way that breaks Cursor, and will not invent fields like `activation:` on an agent (that field belongs to rules).

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
cat .claude/agents/developer.md      # Claude Code native format
cat .cursor/agents/developer.md      # Cursor native format
```

---

## Workflow 2 — AI runs xcaffold itself

This workflow is for agentic IDE sessions where the AI assistant has terminal access (Claude Code, Gemini CLI, Cursor with shell tools, GitHub Copilot agents). The AI treats xcaffold as a determinism layer — it uses it to generate configs that it cannot hallucinate its way around.

### What you tell the AI

Give the AI a single instruction. Example for Claude Code:

> Use `xcaffold` to initialize a new agent project in the current directory targeting `claude` and `gemini`. Then add two agents:
> 1. A **backend developer** agent with TypeScript expertise in `xcf/agents/backend.xcf`
> 2. A **code reviewer** agent that runs in read-only mode in `xcf/agents/reviewer.xcf`
>
> After writing the .xcf files, validate with `xcaffold validate` and apply with `xcaffold apply --target claude --dry-run`. Report any policy violations.

### What the AI does

The AI runs a deterministic sequence of xcaffold commands:

```bash
# Step 1: Initialize. --json gives the AI a map of what was created.
xcaffold init --target claude,gemini --yes --json
```

This returns a machine-readable manifest of every generated file:

```json
{
  "targets": ["claude", "gemini"],
  "files": [
    "project.xcf",
    "xcf/agents/developer.xcf",
    "xcf/rules/conventions.xcf",
    "xcf/settings.xcf",
    "xcf/skills/xcaffold/references/agent.xcf.reference",
    "xcf/skills/xcaffold/references/skill.xcf.reference"
  ],
  "provider_notes": {
    "claude": "full feature set",
    "gemini": "hooks, memory, permission-mode unsupported — will be dropped at apply"
  }
}
```

The AI reads this manifest to know exactly which files to edit. It reads `xcf/skills/xcaffold/references/agent.xcf.reference` for the complete field catalog, and `xcf/agents/developer.xcf` for the provider matrix.

```bash
# Step 2: The AI edits xcf/agents/developer.xcf and creates xcf/agents/reviewer.xcf
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

If your project already has a `.claude/` or `.cursor/` directory, use the AI to safely convert it to xcaffold source `xcf/` directories using `xcaffold import`.

### Human prompt to AI

> This project has a `.claude/` directory. Initialize a new xcaffold project here targeting `claude` and `cursor`. Then import the existing Claude configuration into xcaffold. Show me the `import --plan` output before actually importing.

### AI command sequence

```bash
# 1. Initialize empty scaffold
xcaffold init --target claude,cursor --yes

# 2. Preview import plan
xcaffold import --from claude --plan

# The AI reports the plan to the user. Once approved:

# 3. Import
xcaffold import --from claude

# 4. Verify the imported output
xcaffold validate
```

Once imported, the AI can now modify the configurations through the `.xcf` files rather than editing the raw `.claude/` output.

---

## The `/xcaffold` AI skill

Every time you run `xcaffold init`, it automatically generates a self-referential skill in `xcf/skills/xcaffold.xcf`. This file includes a built-in schema reference, workflow cheat sheets, and provider matrices.

When you run `xcaffold apply`, this skill compiles to your project's `.claude/skills/xcaffold/` (or Cursor equivalent) directory. 

This gives you a magic loop: **your AI assistant instantly learns how to use xcaffold just by being inside an initialized project.**

### Using it globally

If you want the `/xcaffold` skill available to your AI assistant everywhere (not just inside specific initialized projects), run:

```bash
# Initialize the user-wide environment
xcaffold init --global --target claude,cursor

# Compile the skill to your global ~/.claude/skills/ directories
xcaffold apply --global --target claude
xcaffold apply --global --target cursor
```

Once compiled, you can type `/xcaffold` in Claude Code or Cursor, or simply ask Gemini *"scaffold a new agent for me using xcaffold"* from any directory, and it will autonomously know the exact commands and schema to use.

---

## Adding a skill from scratch

Skills are the most complex resource type — they have their own directory structure in some providers (`skills/<id>/SKILL.md` in Antigravity), and their own frontmatter conventions. Ask the AI to create one using the reference:

### Human prompt to AI

> Read `xcf/skills/xcaffold/references/skill.xcf.reference`. Create a new skill called `tdd` in `xcf/skills/tdd.xcf`. The skill should:
>
> - Instruct the agent to write a failing test first, then minimal code to pass it, then refactor.
> - Apply to all selected targets (claude, cursor).
> - Set `allowed-tools` to `[Read, Write, Edit, Bash]`.
>
> Respect the provider matrix in the reference file. Do not include fields marked as `dropped` for either target. After writing the file, add `tdd` to the `skills:` list in `project.xcf`.

### AI command sequence

The AI writes `xcf/skills/tdd.xcf`, then runs:

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
.cursor/rules/tdd.mdc           # Cursor uses .mdc rule files for skills
```

Same source, two provider-native outputs.

---

> **Preview.** Policy enforcement is planned for a future release. The `kind: policy` resource and the `xcf/policies/` directory are not yet supported by the compiler.

---

## What You Built

You have a project where:
- All agent configuration lives in `xcf/` — the AI's editing surface.
- Provider-specific output is generated by xcaffold — the AI never touches `.claude/`, `.cursor/`, or `.gemini/` directly.
- The `--json` manifest flag makes the entire init workflow composable in agentic pipelines.

---

## Next Steps

- **Multi-agent workspace** — build a team of differentiated agents with shared skills: [`multi-agent-workspace.md`](multi-agent-workspace.md)
- **Target overrides** — fine-tune behavior per provider without duplicating resources: [`../how-to/target-overrides.md`](../how-to/target-overrides.md)
- **CLI reference** — full flag documentation for every command: [`../reference/cli.md`](../reference/cli.md)
