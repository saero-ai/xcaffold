<p align="center">
  <img src="assets/logo.svg" alt="xcaffold logo" height="150" />
  <img src="assets/xaff.svg" alt="Xaff" height="150" />
</p>

# xcaffold

[![CI](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml/badge.svg)](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/saero-ai/xcaffold)](https://goreportcard.com/report/github.com/saero-ai/xcaffold)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/dl/)

**deterministic agent configuration compiler for AI coding platforms.** Declare your AI agents in a single `.xcf` YAML file. `xcaffold` compiles it deterministically into native AI coding agent constraints (e.g., `.claude/`, `.cursor/rules/`, `.agents/`) — with drift detection, token budgeting, and sandboxed simulation.

```
                                    ──► (claude)       ──►  .claude/
scaffold.xcf  ──►  xcaffold apply   ──► (cursor)       ──►  .cursor/rules/
                                    ──► (antigravity)  ──►  .agents/
                                    ──► (agentsmd)     ──►  AGENTS.md
```

## Why xcaffold?

Most teams hand-edit their AI agent configurations manually. This means:
- **No drift detection** — edits made outside the IDE go unnoticed.
- **No token budgeting** — agents silently exceed context limits.
- **No auditability** — no SHA manifest, no CI/CD integration.
- **No testability** — agent behavior is never validated against declared assertions.

`xcaffold` applies GitOps discipline to agent configuration. The `.xcf` file is the only source of truth. Generated configurations are strict compilation artifacts.

## Installation

**Homebrew (macOS/Linux)**
```bash
brew install saero-ai/tap/xcaffold
```

**Scoop (Windows)**
```powershell
scoop bucket add saero-ai https://github.com/saero-ai/scoop-bucket.git
scoop install xcaffold
```

**Go install (All Platforms)**
```bash
go install github.com/saero-ai/xcaffold/cmd/xcaffold@latest
```

**Binary releases**

Pre-built binaries for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64) are available on the [Releases page](https://github.com/saero-ai/xcaffold/releases). Windows binaries are packaged as `.zip` while Unix binaries are `.tar.gz`.

**Build from source**
```bash
git clone https://github.com/saero-ai/xcaffold
cd xcaffold
make build
```

## Example Usage

```yaml
version: "1.0"
project:
  name: "acme-web-platform"

agents:
  developer:
    description: "Expert React developer."
    model: claude-3-7-sonnet-20250219
    tools: [Read, Write, Bash, Glob]
    blocked_tools: [WebFetch]
    skills: [git]
    instructions: |
      You are a frontend developer specializing in standard React.
      Always run tests before marking a task complete.
    assertions:
      - "The agent must not write files outside the project directory."
      - "The agent must run tests before marking a task complete."

skills:
  git:
    description: "Git commit conventions."
    instructions: "Always use conventional commits (feat:, fix:, chore:)."

rules:
  typescript:
    paths: ["src/**/*.ts"]
    instructions: "Prefer type aliases over interfaces. No any."

hooks:
  pre-commit:
    run: "make lint"

mcp:
  sqlite:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-sqlite", "data.db"]

test:
  judge_model: "claude-3-5-haiku-20241022"
```

Then run the full lifecycle:

```bash
xcaffold graph --tokens    # View maps & analyze token budgets
xcaffold apply             # Compile to targets (or --check to validate syntax)
xcaffold diff              # Check for manual drift in output dirs
xcaffold test --agent developer --judge  # Simulate + evaluate
```

## The Lifecycle

| Phase | Command | Output |
|---|---|---|
| Bootstrap | `xcaffold init` | `scaffold.xcf` starter template |
| Ingestion | `xcaffold import` | Migrated `.xcf` from existing targets, or translated from cross-platform workflows via `--source` |
| Audit | `xcaffold analyze` | `scaffold.xcf` + `audit.json` |
| Topology | `xcaffold graph` | terminal art, mermaid, JSON, or DOT graph (token estimates with `--tokens`) |
| Compilation | `xcaffold apply` | Target configuration + `scaffold.lock` |
| Drift Check | `xcaffold diff` | Exit 1 on drift |
| Validation | `xcaffold validate` | Exit 1 on errors; structural warnings with `--structural` |
| Simulation | `xcaffold test` | `trace.jsonl` + judge report |
| Registry | `xcaffold list` | Registered project inventory |
| Migration | `xcaffold migrate` | Upgraded layout + registry entry |

## Scopes

`--scope` is a global flag accepted by every xcaffold command.

| Scope | Config File | Output Directory | Flag |
|---|---|---|---|
| Project (default) | `scaffold.xcf` | Target directory (`.claude/`, etc.) | `--scope project` |
| Global | `~/.xcaffold/global.xcf` | `~/.claude/`, `~/.agents/`, etc. | `--scope global` |
| Both | — | Both directories | `--scope all` |

Run `xcaffold init --scope global` to create a `global.xcf` in `~/.xcaffold/`. Project configs can inherit from it with `extends: global` in their `scaffold.xcf`.

```bash
xcaffold apply --scope global   # compile global.xcf → ~/.claude/
xcaffold apply --scope all      # compile both scopes
xcaffold diff  --scope all      # drift check across both scopes
```

## Argument Reference

### Global flags

These flags are accepted by every xcaffold command.

| Flag | Default | Description |
|---|---|---|
| `--scope` | `project` | Compilation scope: `project`, `global`, or `all`. |
| `--config` | `./scaffold.xcf` | Path to `scaffold.xcf`. Useful in monorepo sub-directories. |

### `xcaffold validate`

| Flag | Default | Description |
|---|---|---|
| `--structural` | `false` | Run structural invariant checks: orphan resources, agents missing instructions, Bash tool without PreToolUse hook. Warnings are informational and do not fail the exit code. |

### `xcaffold apply`

| Flag/Argument | Default | Description |
|---|---|---|
| `--target` | `claude` | Compilation target platform (`claude`, `cursor`, `antigravity`, `agentsmd`). |
| `--check` | `false` | Validate your YAML syntax without compiling. |
| `--force` | `false` | Overwrite customized local files and bypass drift safeguard natively preventing data wiping. |
| `--backup` | `false` | Backup existing target directory before overwriting. Follows `BackupDir` metadata if provided. |
| `--project` | `""` | Apply to an external project directly via its registered global registry name without navigating directories. |

### `xcaffold plan`

| Flag | Default | Description |
|---|---|---|
| No specific flags | | Run a static dry-run parsing to preview the execution output AST paths and destinations before applying. |

### `xcaffold import`

| Flag/Argument | Default | Description |
|---|---|---|
| `--source` | `""` | File or directory of workflow markdown files to translate (e.g. `Antigravity` or `Cursor` docs). Enables Translation mode. |
| `--from` | `auto` | Expected intent format for static heuristics (`antigravity`, `claude`, `cursor`). Combined with `--source`. |
| `--plan` | `false` | Dry-run parsing; prints the generated primitive tree without modifying config. Combined with `--source`. |

### `xcaffold graph`

| Flag | Default | Description |
|---|---|---|
| `--format` | `terminal` | Format of output (`terminal`, `mermaid`, `dot`, `json`). |
| `--tokens`, `-t` | `false` | Analyze configuration bloat and provide estimated AST token counts. |
| `--agent`, `-a` | `""` | Target a specific agent (shows only its topology). |
| `--full`, `-f` | `false` | Show the fully expanded topology tree. |
| `--project`, `-p` | `""` | Target a registered project by name or path. |

### `xcaffold list`

No flags. Displays all registered projects from `~/.xcaffold/registry.xcf` with metadata (path, targets, resource counts, last applied timestamp).

### `xcaffold migrate`

No flags. Interactive command that detects and migrates:
- Legacy global config from `~/.claude/global.xcf` to `~/.xcaffold/global.xcf`
- Flat-layout `instructions_file` paths to reference-in-place paths
- Unregistered projects into the central registry

### `xcaffold test`

| Flag | Default | Description |
|---|---|---|
| `--agent`, `-a` | _(required)_ | Agent ID from `scaffold.xcf` to simulate. |
| `--judge` | `false` | Run LLM-as-a-Judge evaluation after simulation. |
| `--output`, `-o` | `trace.jsonl` | Path to write the execution trace. |
| `--cli-path` | `""` | Path to the CLI binary. Overrides `test.cli_path`. (Fallback to target defaults) |
| `--judge-model` | `""` | Anthropic model for the judge. Overrides `test.judge_model`. |

### `xcaffold analyze`

| Flag | Default | Description |
|---|---|---|
| `--model`, `-m` | `claude-3-7-sonnet-20250219` | Generative model to use for analysis. |
| `[directory]` | `.` | Directory to scan. |

## Schema Reference

The `scaffold.xcf` file supports the following top-level blocks:

* `project` - (Required) Object. Project identity. Contains `name` (string).
* `agents` - (Optional) Map. AI agent personas. Each entry supports:
  * `description` - (Optional) String. Shown in the CLI/IDE integration.
  * `instructions` - (Optional) String. The agent's inline system prompt.
  * `instructions_file` - (Optional) String. Path to external Markdown system prompt.
  * `model` - (Optional) String. Anthropic model ID.
  * `effort` - (Optional) String. Thinking effort level (`low`, `medium`, `high`).
  * `tools` - (Optional) List of strings. Allowed tools (e.g. `Read`, `Write`, `Bash`).
  * `blocked_tools` - (Optional) List of strings. Explicitly denied tools.
  * `skills` - (Optional) List of strings. Skill IDs to compose into this agent.
  * `rules` - (Optional) List of strings. Rule IDs to apply to this agent.
  * `mcp` - (Optional) List of strings. MCP server IDs available to this agent.
  * `assertions` - (Optional) List of strings. Behavioral constraints evaluated by `xcaffold test --judge`.
* `skills` - (Optional) Map. Reusable prompt packages. Each entry supports `description`, `instructions`, `instructions_file`, `paths`, `tools`, `references`.
* `rules` - (Optional) Map. Path-gated formatting guidelines. Each entry supports `paths`, `instructions`, `instructions_file`.
* `hooks` - (Optional) Map. Lifecycle event hooks. Each entry supports `event`, `match`, `run`.
* `mcp` - (Optional) Map. Local MCP server declarations. Each entry supports `command`, `args`, `env`.
* `test` - (Optional) Object. Simulator configuration. Supports `claude_path`, `judge_model`.
* `settings` - (Optional) Object. Full project environment configuration (env, statusLine, plugins).

## Attributes Reference

In addition to all arguments above, the following artifacts are generated on disk depending on the compilation target:

| Object | `claude` | `cursor` | `antigravity` | `agentsmd` |
|---|---|---|---|---|
| Agents | `.claude/agents/*.md` | `.cursor/rules/*.mdc` | `.agents/agents/*.yaml` | `AGENTS.md` (root) |
| Skills | `.claude/skills/*/SKILL.md` | `.cursor/rules/skills_*.mdc` | `.agents/skills/*/SKILL.md` | `AGENTS.md` (root) |
| Rules | `.claude/rules/*.md` | `.cursor/rules/*.mdc` | `.agents/rules/*.md` | `AGENTS.md` (root) + `{dir}/AGENTS.md` (directory-scoped rules) |

**Common output artifacts:**
* `scaffold.lock` - SHA-256 state manifest. Used by `xcaffold diff` for drift detection.
* `audit.json` - LLM compliance assessment produced by `xcaffold analyze`.
* `trace.jsonl` - Newline-delimited JSON execution trace produced by `xcaffold test`.

## Diagnostics

Use `xcaffold review [file]` to read any diagnostic artifact in your terminal:

```bash
xcaffold review              # Parse scaffold.xcf AST
xcaffold review audit.json   # Read compliance assessment
xcaffold review trace.jsonl  # Read simulation trace
```

## Import / Compatibility

`xcaffold import` scans existing platform directories (`.claude/`, `.cursor/`, `.agents/`) and generates a `scaffold.xcf` with metadata entries referencing instruction files where they already reside. No files are copied or duplicated.

`xcaffold apply` deterministically compiles `scaffold.xcf` into native target outputs. Any manually edited files in managed output paths will be overwritten.

**Recommended `.gitignore` entries:**
```
scaffold.lock
audit.json
trace.jsonl
```

Commit `scaffold.xcf` and your instruction files. CI validates the lock is not stale via `xcaffold diff`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
