---
title: "CLI Reference"
description: "Detailed breakdown of every xcaffold command, flag, and exit behavior"
---

# CLI Reference

Reference for all `xcaffold` commands and flags. Commands are organized by lifecycle phase, then utilities.

---

## Global Flags

Available on every subcommand via `rootCmd.PersistentFlags()`.

| Flag | Default | Description |
|---|---|---|
| `--config <path>` | `""` | Path to a `project.xcf` file or a directory containing `.xcf` files. If a directory is given, `parser.ParseDirectory()` scans all `*.xcf` files within it. Defaults to `./project.xcf` discovered via upward directory walk. Any filename is accepted as long as it contains `kind: project` frontmatter. |
| `--global` / `-g` | `false` | Operate on user-wide global config (`~/.xcaffold/global.xcf`). When omitted, the default scope is project. |
| `--blueprint <name>` | `""` | Compile only the named blueprint subset (resources selected by a `kind: blueprint` file with the matching name). Mutually exclusive with `--global`. |
| `--version` | — | Prints `<version> (commit: <sha>, date: <date>)` and exits. |

**Scope resolution (`resolveConfig` in `main.go`):**

When `--global` is set, config is resolved to `~/.xcaffold/global.xcf`. Otherwise, `./project.xcf` is discovered via `resolver.FindConfigDir` upward walk. `--global` is skipped for `init` and `import` (they bootstrap configs, so no pre-existing config is required).

---

## Lifecycle Commands

### `xcaffold init`

**File:** `cmd/xcaffold/init.go`

Bootstraps a new `project.xcf`. Idempotent: if `project.xcf` already exists, exits immediately with no changes.

On first run, detects existing `.claude/`, `.cursor/`, and `.agents/` platform directories. Presents a confirmation or interactive multi-select to import them. With `--yes`, imports all detected directories without prompting.

With `--global`, runs `registry.RebuildGlobalXCF()` to scan `~/.claude/`, `~/.cursor/`, and `~/.agents/` and write `~/.xcaffold/global.xcf`.

Also generates `xcf/skills/xcaffold/references/` �� an annotated, non-parsed field reference directory containing `agent.xcf.reference` and `skill.xcf.reference`. These are wired as companion files to the built-in xcaffold skill.

| Flag | Short | Default | Description |
|---|---|---|---|
| `--yes` | `-y` | `false` | Accept all defaults non-interactively. Suitable for CI/CD. |
| `--template <name>` | — | `""` | Use a pre-built topology template. Values: `rest-api`, `cli-tool`, `frontend-app`. |
| `--no-policies` | — | `false` | Skip generation of starter policy files. |

---

### `xcaffold import`

**File:** `cmd/xcaffold/import.go`

Two distinct modes:

**Native Import Mode (default):**
Scans a platform directory (`.claude/`, `.cursor/`, `.agents/`) and writes a `project.xcf` referencing the existing files via `instructions-file:` with zero file duplication. Reads `settings.json` and `hooks.json`. If multiple platform directories are detected, merges them using a larger-file-wins deduplication strategy.

With `--global`, scans all provider directories under the user's home (`~/.claude/`, `~/.cursor/`, `~/.agents/`) and writes `~/.xcaffold/global.xcf`. All discovered providers are merged.

**Cross-Platform Translation Mode (`--source`):**
Parses workflow Markdown files from another platform, detects functional intents via `internal/bir`, and decomposes them into xcaffold primitives (`skill`, `rule`, `permission`). Results are injected into an existing `project.xcf`. Requires `project.xcf` to already exist (run `xcaffold init` first).

| Flag | Default | Description |
|---|---|---|
| `--source <path>` | `""` | File or directory of workflow `.md` files to translate. Activates cross-platform translation mode. |
| `--from <platform>` | `auto` | Source platform format. Values: `antigravity`, `claude`, `cursor`, `gemini`, `copilot`, `auto`. |
| `--plan` | `false` | Dry-run: print the decomposition plan without writing any files. |
| `--with-memory` | `false` | Include memory `.md` files found in the platform directory. Copies them to `xcf/agents/<id>/memory/`. |
| `--auto-merge` | `""` | Merge strategy when multiple provider directories are detected. Currently supports: `union`. |

---

### `xcaffold analyze [directory]`

**File:** `cmd/xcaffold/analyze.go`

Scans the repository to build a `ProjectSignature`, then calls an LLM to generate a `project.xcf` and an `audit.json` compliance report. Does not require an existing `project.xcf` — safe to run on any directory.

**Auth resolution order:**
1. `ANTHROPIC_API_KEY` env var (direct Anthropic API)
2. `XCAFFOLD_LLM_API_KEY` + `XCAFFOLD_LLM_BASE_URL` (platform-agnostic API)
3. CLI binary on `$PATH` (subscription fallback)

| Flag | Short | Default | Description |
|---|---|---|---|
| `--model <model>` | `-m` | `claude-3-7-sonnet-20250219` | Generative model to use for scaffold generation. |

**Outputs:** `project.xcf`, `audit.json`.

---

### `xcaffold graph [file]`

**File:** `cmd/xcaffold/graph.go`

Renders the agent dependency graph parsed from `project.xcf`. Default terminal output shows a summary view; `--full` or `--agent` expands the full tree.

| Flag | Short | Default | Description |
|---|---|---|---|
| `--format <fmt>` | — | `terminal` | Output format: `terminal`, `mermaid`, `dot`, `json`. |
| `--agent <id>` | `-a` | `""` | Focus on a single agent and its direct dependencies. Implies `--full`. |
| `--project <name>` | `-p` | `""` | Target a registered project by name or path instead of the current directory. |
| `--full` | `-f` | `false` | Show the fully expanded topology tree. Default view is a summary. |
| `--scan-output` | — | `false` | Also scan compiled output directories for artifacts not tracked in `project.xcf`. |
| `--all` | — | `false` | Show global topology and all registered projects in one view. Mutually exclusive with `--global` and `--project`. |
| `--blueprint <name>` | — | `""` | Show the graph for the named blueprint's resources only. |

The topology includes agents, skills, rules, MCP servers, hooks, and workflows. Hook nodes are labeled by event name; workflow nodes are labeled by workflow ID.

**Format details:**

| Format | Output |
|---|---|
| `terminal` | ASCII art topology printed to stdout. |
| `mermaid` | Mermaid graph syntax. Pipe to a markdown file for embedding in docs. |
| `dot` | Graphviz DOT language. Pipe to `dot -Tsvg` to render an image. |
| `json` | Machine-readable JSON graph for programmatic use. Field names use snake_case (`config_path`, `disk_entries`, `blocked_tools`). |

> **Breaking change (JSON consumers):** The `json` output uses snake_case field names. Any tooling that read the previous camelCase JSON output must be updated.

---

### `xcaffold apply`

**File:** `cmd/xcaffold/apply.go`

Compiles `project.xcf` (or a directory of `.xcf` files) into a target platform's native format. Writes a SHA-256 state manifest to `.xcaffold/<blueprint>.xcf.state`. Automatically purges orphaned output files.

**Smart skip:** If `.xcaffold/<blueprint>.xcf.state` contains a `source_files` manifest and no source hashes have changed, compilation is skipped. Use `--force` to bypass.

**Drift guard:** Before writing, compares current output file hashes against the state manifest. If manual edits are detected (drift), the command exits with an error. Use `--force` to override.

**Target → output directory mapping:**

| Target | Output Directory |
|---|---|
| `claude` | `.claude/` |
| `cursor` | `.cursor/` |
| `antigravity` | `.agents/` |
| `copilot` | `.github/` |
| `gemini` | `.gemini/` |

| Flag | Default | Description |
|---|---|---|
| `--target <target>` | `claude` | Compilation target. One of: `claude`, `cursor`, `antigravity`, `copilot`, `gemini`. |
| `--dry-run` | `false` | Preview changes as a colored unified diff without writing any files. |
| `--check` | `false` | Validate YAML syntax and cross-references only. Does not compile. Returns non-zero exit code if any errors are found. |
| `--check-permissions` | `false` | Report security fields that will be dropped for the active `--target`. Exits non-zero if contradictions are found (e.g., a tool in `permissions.deny` also appears in an agent's `tools` list). |
| `--force` | `false` | Overwrite even if drift is detected or sources are unchanged. |
| `--backup` | `false` | Copy the existing output directory to a timestamped backup before overwriting. Backup directory name: `.<target>_bak_<timestamp>`. Custom location via `project.backup-dir` in `project.xcf`. |
| `--project <name>` | `""` | Apply to a different project registered in the global registry by name. Resolves the project's path and uses it as the config root. |

---

### `xcaffold diff`

**File:** `cmd/xcaffold/diff.go`

Compares SHA-256 hashes of all tracked output files against the state manifest (`.xcaffold/<blueprint>.xcf.state`). Also compares source file hashes if `SourceFiles` is present in the state.

**Status codes per file:**

| Status | Meaning |
|---|---|
| `clean` | File hash matches lock. |
| `DRIFTED` | File content has changed since last `apply`. |
| `MISSING` | File is in the lock but does not exist on disk. |
| `SRC DELETED` | Source `.xcf` file tracked in lock no longer exists. |
| `SRC DRIFTED` | Source `.xcf` file content changed since last `apply`. |
| `SRC ADDED` | A new `.xcf` file exists that was not present at last `apply`. |

Exits non-zero with a count of drifted files if any drift is found.

| Flag | Default | Description |
|---|---|---|
| `--target <target>` | `""` (defaults to `claude`) | Target state file to inspect. One of: `claude`, `cursor`, `antigravity`, `copilot`, `gemini`. |
| `--blueprint <name>` | `""` | Check drift for the named blueprint's state file (`.xcaffold/<name>.xcf.state`). |

---

### `xcaffold status`

**File:** `cmd/xcaffold/status.go`

Reads state from `.xcaffold/` and displays a summary of project compilation state, active blueprint, per-target freshness, and any detected drift.

**Output fields:**

| Field | Description |
|-------|-------------|
| `Blueprint` | Active blueprint name (the blueprint with `active: true`). Shows `none` if no blueprint is active. |
| `Targets` | Compilation targets recorded in the active state file. |
| Per-target status | Last applied timestamp and artifact count. Shows `drifted` if artifact hashes differ from state. |
| `Sources` | Count of tracked source `.xcf` files. Lists changed files individually. |

**Example output:**

```
Blueprint: backend (active)
Targets:   claude, cursor

  claude:  applied 2 hours ago, 9 artifacts (all clean)
  cursor:  applied 2 hours ago, 9 artifacts (1 drifted)

Sources:   7 files
  changed  xcf/skills/tdd/tdd.xcf    <- re-apply needed

Run 'xcaffold apply --blueprint backend' to sync.
```

| Flag | Default | Description |
|---|---|---|
| `--blueprint <name>` | `""` | Show status for the named blueprint's state file. |
| `--global` | `false` | Show status for the global config state. |

**Exit codes:**

| Code | Meaning |
|------|---------|
| `0` | No drift detected. |
| `1` | Drift detected (source changed or artifact modified). |
| `2` | No state file found (project has never been applied). |

For a walkthrough, see [Checking Project Status](../how-to/checking-project-status.md).

---

### `xcaffold test`

**File:** `cmd/xcaffold/test.go`

Simulates a compiled agent by reading its system prompt from `.claude/agents/<id>.md`, sending a task to the LLM API directly, and recording all tool calls declared in the response to a JSONL trace file.

Requires `xcaffold apply` to be run first — the agent must be compiled to `.claude/agents/` before testing.

With `--judge`, sends the trace and the agent's `assertions` list to an LLM for evaluation.

**Prerequisites:**
- Run `xcaffold apply` before testing — the agent system prompt must be compiled.
- `ANTHROPIC_API_KEY` or `XCAFFOLD_LLM_API_KEY` must be set for simulation and `--judge`.

**Task resolution:** Uses `test.task` from `project.xcf`. If unset, defaults to `"Describe what tools you have available and what you would do first."`.

**CLI path resolution (judge fallback only):** `--cli-path` flag > `test.cli-path` in `project.xcf` > `claude` on `$PATH`.

**Auth resolution order (simulation and judge):**
1. `XCAFFOLD_LLM_API_KEY` + `XCAFFOLD_LLM_BASE_URL`
2. `ANTHROPIC_API_KEY`
3. CLI binary subscription fallback

| Flag | Short | Default | Description |
|---|---|---|---|
| `--agent <id>` | `-a` | — | **Required.** Agent ID to simulate. Must exist in `project.xcf`. |
| `--judge` | — | `false` | Run LLM-as-a-Judge evaluation after simulation. Evaluates against `agents.<id>.assertions`. |
| `--output <path>` | `-o` | `trace.jsonl` | Path for the execution trace output. |
| `--cli-path <path>` | — | `""` | Path to the CLI binary used as judge subscription fallback. Overrides `test.cli-path` in `project.xcf`. |
| `--judge-model <model>` | — | `""` | Model for judge evaluation. Overrides `test.judge-model`. Falls back to `claude-haiku-4-5-20251001`. |

---

### `xcaffold export`

**File:** `cmd/xcaffold/export.go`

Compiles `project.xcf` and packages the output as a distributable plugin directory.

| Flag | Default | Description |
|---|---|---|
| `--output <path>` | — | **Required.** Destination directory for the exported plugin. |
| `--format <fmt>` | `plugin` | Export format. Only `plugin` is currently supported. |
| `--target <target>` | `""` (defaults to `claude`) | Compilation target for the export. One of: `claude`, `cursor`, `antigravity`, `copilot`, `gemini`. |

---

## Utility Commands

### `xcaffold validate`

**File:** `cmd/xcaffold/validate.go`

Validates `project.xcf` without compiling. Checks:
1. YAML syntax and known fields (fail-closed parser — unknown fields are an error)
2. Cross-reference integrity: agent `skills:`, `rules:`, `mcp:` IDs must resolve to top-level map keys
3. File existence: `instructions-file` and `references` paths must resolve on disk
4. Tool validation: agent `tools` and `disallowed-tools` entries checked against a known registry (includes `Task`, `Computer`, `AskUserQuestion`, `Agent`, `ExitPlanMode`, `EnterPlanMode`)

With `--structural`, additionally checks:
- Orphan skills (defined but not referenced by any agent)
- Orphan rules (defined, not referenced, no `paths`, no `always-apply: true`)
- Agents with no `instructions` or `instructions-file`
- Agents with `Bash` in `tools` but no `PreToolUse` hook

Exit code `0` means valid. Non-zero means errors found.

| Flag | Default | Description |
|---|---|---|
| `--structural` | `false` | Run structural invariant checks (orphan resources, missing instructions, missing hooks). |
| `--global` | `false` | Validate the global config at `~/.xcaffold/global.xcf` instead of the project config. |

**Example:** `$ xcaffold validate --global`

---

### `xcaffold review [file]`

**File:** `cmd/xcaffold/review.go`

Universal parser for xcaffold diagnostic artifacts. Does not require a `project.xcf` to be present (`resolveConfig` is skipped for `review`).

**Supported file types:**

| File | Output |
|---|---|
| `project.xcf` | AST tree: project name, agents (model/tools/assertions), skills, rules, hooks, MCP servers, and workflows. |
| `audit.json` | Compliance scores: `security`, `prompt_quality`, `tool_restrictions` (each `/100`) and feedback. |
| `plan.json` | Pretty-printed JSON. |
| `trace.jsonl` | Timestamp and tool name for each recorded event. |

**Special argument:** `all` — loops through all four file types in the current directory (or the global config directory with `--global`).

No command-specific flags.

---

### `xcaffold list`

**File:** `cmd/xcaffold/list.go`

Scans the current project and displays all discovered resources grouped by type, plus all defined blueprints. Does not require flags to use.

**Output sections:**

- **Resources** — all agents, skills, rules, workflows, MCP servers, policies, and memory entries discovered in `xcf/`. Memory entries are discovered via filesystem scan of `xcf/agents/<id>/memory/` directories.
- **Blueprints** — all `kind: blueprint` definitions found in `xcf/`, with their resource counts and active status.

| Flag | Default | Description |
|---|---|---|
| `--blueprint <name>` | `""` | (Hidden) Filter output to the named blueprint's resources. |
| `--resolved` | `false` | (Hidden, use with `--blueprint`) Expand transitive `extends:` dependencies before listing. |

---

### `xcaffold migrate`

**File:** `cmd/xcaffold/migrate.go`

Restructures project layouts to align with xcaffold conventions. Safe to run repeatedly (idempotent).

**Operations performed:**

1. **Project manifest rename:** Detects `scaffold.xcf` and renames to `project.xcf` (creates a `.backup` copy first)
2. **State file migration:** Converts `scaffold.<target>.lock` files in the project root to `.xcaffold/project.xcf.state` (original files renamed to `*.lock.migrated`)
3. **Schema version upgrade:** Updates project configuration to the current schema version

No flags. Run from a directory containing xcaffold configuration files.

