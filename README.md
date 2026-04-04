<p align="center">
  <img src="docs/assets/logo.svg" alt="xcaffold logo" height="150" />
  <img src="docs/assets/xaff/xaff_animated.svg" alt="Xaff Mascot Animated" height="150" />
</p>

# xcaffold

[![CI](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml/badge.svg)](https://github.com/saero-ai/xcaffold/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/saero-ai/xcaffold)](https://goreportcard.com/report/github.com/saero-ai/xcaffold)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/dl/)

**agent configuration for Anthropic Claude Code.** Declare your AI agents in a single `.xcf` YAML file. `xcaffold` compiles it deterministically into `.claude/` — with drift detection, token budgeting, and sandboxed simulation.

```
scaffold.xcf  ──►  xcaffold apply   ──►  .claude/agents/*.md
                                    ──►  .claude/skills/*.md
                                    ──►  .claude/rules/*.md
                                    ──►  .claude/settings.json
                                    ──►  scaffold.lock
```

## Why xcaffold?

Most teams hand-edit `.claude/` markdown files. This means:
- **No drift detection** — edits made outside the IDE go unnoticed.
- **No token budgeting** — agents silently exceed context limits.
- **No auditability** — no SHA manifest, no CI/CD integration.
- **No testability** — agent behavior is never validated against declared assertions.

`xcaffold` applies GitOps discipline to agent configuration. The `.xcf` file is the only source of truth. Generated `.claude/` files are compilation artifacts.

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
xcaffold plan    # Analyze token budgets — writes plan.json
xcaffold apply   # Compile to .claude/ — writes scaffold.lock
xcaffold diff    # Check for manual drift in .claude/
xcaffold test --agent developer --judge  # Simulate + evaluate
```

## The Lifecycle

| Phase | Command | Output |
|---|---|---|
| Bootstrap | `xcaffold init` | `scaffold.xcf` starter template |
| Audit | `xcaffold analyze` | `scaffold.xcf` + `audit.json` |
| Token Budget | `xcaffold plan` | stdout report + `plan.json` |
| Topology | `xcaffold graph` | terminal art, mermaid, JSON, or DOT graph |
| Compilation | `xcaffold apply` | `.claude/**` + `scaffold.lock` |
| Drift Check | `xcaffold diff` | Exit 1 on drift |
| Validation | `xcaffold test` | `trace.jsonl` + judge report |

## Argument Reference

### `xcaffold apply`

| Argument | Default | Description |
|---|---|---|
| `[directory]` | `.` | Directory containing `scaffold.xcf`. |

### `xcaffold graph`

| Flag | Default | Description |
|---|---|---|
| `--format` | `terminal` | Format of output (`terminal`, `mermaid`, `dot`, `json`). |

### `xcaffold plan`

| Argument | Default | Description |
|---|---|---|
| `[file]` | `scaffold.xcf` | Path to the xcf file to analyze. |

### `xcaffold test`

| Flag | Default | Description |
|---|---|---|
| `--agent`, `-a` | _(required)_ | Agent ID from `scaffold.xcf` to simulate. |
| `--judge` | `false` | Run LLM-as-a-Judge evaluation after simulation. |
| `--output`, `-o` | `trace.jsonl` | Path to write the execution trace. |
| `--claude-path` | `""` | Path to the `claude` binary. Overrides `test.claude_path`. |
| `--judge-model` | `""` | Anthropic model for the judge. Overrides `test.judge_model`. |

### `xcaffold analyze`

| Flag | Default | Description |
|---|---|---|
| `--model`, `-m` | `claude-3-7-sonnet-20250219` | Generative model to use for analysis. |
| `[directory]` | `.` | Directory to scan. |

## Schema Reference

The `scaffold.xcf` file supports the following top-level blocks:

* `project` - (Required) Object. Project identity. Contains `name` (string).
* `agents` - (Optional) Map. Claude agent personas. Each entry supports:
  * `description` - (Optional) String. Shown in Claude Code agent picker.
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

In addition to all arguments above, the following artifacts are generated on disk:

* `.claude/agents/*.md` - Compiled agent persona definitions (Claude Code native format).
* `.claude/skills/*.md` - Compiled skill prompt packages.
* `.claude/rules/*.md` - Compiled path-gated rule definitions.
* `.claude/hooks.json` - Compiled lifecycle hook configuration.
* `.claude/settings.json` - Project-level settings including MCP server declarations.
* `scaffold.lock` - SHA-256 state manifest. Used by `xcaffold diff` for drift detection.
* `plan.json` - Token budget analysis report produced by `xcaffold plan`.
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

`xcaffold` enforces **One-Way Compilation**. It does not import or parse existing `.claude/` markdown files. Running `xcaffold apply` will overwrite any managed files in `.claude/`.

**Recommended `.gitignore` entries:**
```
.claude/
plan.json
audit.json
trace.jsonl
```

Commit `scaffold.xcf` and `scaffold.lock`. CI validates the lock is not stale via `xcaffold diff`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
