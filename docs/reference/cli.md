# CLI Reference

Reference for all `xcaffold` commands and flags. Commands are organized by lifecycle phase, then utilities.

---

## Global Flags

Available on every subcommand via `rootCmd.PersistentFlags()`.

| Flag | Default | Description |
|---|---|---|
| `--config <path>` | `""` | Path to a `scaffold.xcf` file or a directory containing `.xcf` files. If a directory is given, `parser.ParseDirectory()` scans all `*.xcf` files within it. Defaults to `./scaffold.xcf` discovered via upward directory walk. |
| `--global` / `-g` | `false` | Operate on user-wide global config (`~/.xcaffold/global.xcf`). When omitted, the default scope is project. |
| `--version` | — | Prints `<version> (commit: <sha>, date: <date>)` and exits. |

**Scope resolution (`resolveConfig` in `main.go`):**

When `--global` is set, config is resolved to `~/.xcaffold/global.xcf`. Otherwise, `./scaffold.xcf` is discovered via `resolver.FindConfigDir` upward walk. `--global` is skipped for `init` and `import` (they bootstrap configs, so no pre-existing config is required).

---

## Lifecycle Commands

### `xcaffold init`

**File:** `cmd/xcaffold/init.go`

Bootstraps a new `scaffold.xcf`. Idempotent: if `scaffold.xcf` already exists, exits immediately with no changes.

On first run, detects existing `.claude/`, `.cursor/`, and `.agents/` platform directories. Presents a confirmation or interactive multi-select to import them. With `--yes`, imports all detected directories without prompting.

With `--global`, runs `registry.RebuildGlobalXCF()` to scan `~/.claude/`, `~/.cursor/`, and `~/.agents/` and write `~/.xcaffold/global.xcf`.

| Flag | Short | Default | Description |
|---|---|---|---|
| `--yes` | `-y` | `false` | Accept all defaults non-interactively. Suitable for CI/CD. |
| `--template <name>` | — | `""` | Use a pre-built topology template. Values: `rest-api`, `cli-tool`, `frontend-app`. |

---

### `xcaffold import`

**File:** `cmd/xcaffold/import.go`

Two distinct modes:

**Native Import Mode (default):**
Scans a platform directory (`.claude/`, `.cursor/`, `.agents/`) and writes a `scaffold.xcf` referencing the existing files via `instructions_file:` with zero file duplication. Reads `settings.json` and `hooks.json`. If multiple platform directories are detected, merges them using a larger-file-wins deduplication strategy.

**Cross-Platform Translation Mode (`--source`):**
Parses workflow Markdown files from another platform, detects functional intents via `internal/bir`, and decomposes them into xcaffold primitives (`skill`, `rule`, `permission`). Results are injected into an existing `scaffold.xcf`. Requires `scaffold.xcf` to already exist (run `xcaffold init` first).

| Flag | Default | Description |
|---|---|---|
| `--source <path>` | `""` | File or directory of workflow `.md` files to translate. Activates cross-platform translation mode. |
| `--from <platform>` | `auto` | Source platform format. Values: `antigravity`, `cursor`, `auto`. |
| `--plan` | `false` | Dry-run: print the decomposition plan without writing any files. |

---

### `xcaffold analyze [directory]`

**File:** `cmd/xcaffold/analyze.go`

Scans the repository to build a `ProjectSignature`, then calls an LLM to generate a `scaffold.xcf` and an `audit.json` compliance report.

**Auth resolution order:**
1. `ANTHROPIC_API_KEY` env var (direct Anthropic API)
2. `XCAFFOLD_LLM_API_KEY` + `XCAFFOLD_LLM_BASE_URL` (platform-agnostic API)
3. CLI binary on `$PATH` (subscription fallback)

| Flag | Short | Default | Description |
|---|---|---|---|
| `--model <model>` | `-m` | `claude-3-7-sonnet-20250219` | Generative model to use for scaffold generation. |

**Outputs:** `scaffold.xcf`, `audit.json`.

---

### `xcaffold graph [file]`

**File:** `cmd/xcaffold/graph.go`

Renders the agent dependency graph parsed from `scaffold.xcf`. Default terminal output shows a summary view; `--full` or `--agent` expands the full tree.

| Flag | Short | Default | Description |
|---|---|---|---|
| `--format <fmt>` | — | `terminal` | Output format: `terminal`, `mermaid`, `dot`, `json`. |
| `--agent <id>` | `-a` | `""` | Focus on a single agent and its direct dependencies. Implies `--full`. |
| `--project <name>` | `-p` | `""` | Target a registered project by name or path instead of the current directory. |
| `--full` | `-f` | `false` | Show the fully expanded topology tree. Default view is a summary. |
| `--scan-output` | — | `false` | Also scan compiled output directories for artifacts not tracked in `scaffold.xcf`. |
| `--all` | — | `false` | Show global topology and all registered projects in one view. Mutually exclusive with `--global` and `--project`. |

**Format details:**

| Format | Output |
|---|---|
| `terminal` | ASCII art topology printed to stdout. |
| `mermaid` | Mermaid graph syntax. Pipe to a markdown file for embedding in docs. |
| `dot` | Graphviz DOT language. Pipe to `dot -Tsvg` to render an image. |
| `json` | Machine-readable JSON graph for programmatic use. |

---

### `xcaffold apply`

**File:** `cmd/xcaffold/apply.go`

Compiles `scaffold.xcf` (or a directory of `.xcf` files) into a target platform's native format. Writes a SHA-256 lock manifest to `scaffold.<target>.lock`. Automatically purges orphaned output files. **Evaluates the Policy Enforcement Engine against the source configuration, hard-blocking compilation on `error`-level violations.**

**Smart skip:** If `scaffold.<target>.lock` contains a `source_files` manifest and no source hashes have changed, compilation is skipped. Use `--force` to bypass.

**Drift guard:** Before writing, compares current output file hashes against the lock manifest. If manual edits are detected (drift), the command exits with an error. Use `--force` to override.

**Target → output directory mapping:**

| Target | Output Directory |
|---|---|
| `claude` | `.claude/` |
| `cursor` | `.cursor/` |
| `antigravity` | `.agents/` |
| `agentsmd` | `.agents/` |

| Flag | Default | Description |
|---|---|---|
| `--target <target>` | `claude` | Compilation target. One of: `claude`, `cursor`, `antigravity`, `agentsmd`. |
| `--dry-run` | `false` | Preview changes as a colored unified diff without writing any files. |
| `--check` | `false` | Validate YAML syntax and cross-references only. Does not compile. |
| `--check-permissions` | `false` | Report security fields that will be dropped for the active `--target`. Exits non-zero if contradictions are found (e.g., a tool in `permissions.deny` also appears in an agent's `tools` list). |
| `--force` | `false` | Overwrite even if drift is detected or sources are unchanged. |
| `--backup` | `false` | Copy the existing output directory to a timestamped backup before overwriting. Backup directory name: `.<target>_bak_<timestamp>`. Custom location via `project.backup_dir` in `scaffold.xcf`. |
| `--project <name>` | `""` | Apply to a different project registered in the global registry by name. Resolves the project's path and uses it as the config root. |

---

### `xcaffold diff`

**File:** `cmd/xcaffold/diff.go`

Compares SHA-256 hashes of all tracked output files against the lock manifest (`scaffold.<target>.lock`). Also compares source file hashes if `SourceFiles` is present in the lock.

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
| `--target <target>` | `""` (defaults to `claude`) | Target lock file to inspect. One of: `claude`, `cursor`, `antigravity`. |

---

### `xcaffold test`

**File:** `cmd/xcaffold/test.go`

Spawns an HTTP intercept proxy, launches the target CLI subprocess with `HTTPS_PROXY` and `HTTP_PROXY` set, and records all tool calls to a JSONL trace file.

With `--judge`, sends the trace and the agent's `assertions` list to an LLM for evaluation.

**Prerequisites:**
- The target CLI binary (e.g., `claude`) must be available on `$PATH`, or `test.cli_path` must be set in `scaffold.xcf`.
- `ANTHROPIC_API_KEY` or `XCAFFOLD_LLM_API_KEY` must be set for `--judge`.

**CLI path resolution priority:** `--cli-path` flag > `test.cli_path` in `scaffold.xcf` > `test.claude_path` (deprecated) > `claude` on `$PATH`.

**Auth for judge (resolution order):**
1. `XCAFFOLD_LLM_API_KEY` + `XCAFFOLD_LLM_BASE_URL`
2. `ANTHROPIC_API_KEY`
3. CLI binary subscription fallback

| Flag | Short | Default | Description |
|---|---|---|---|
| `--agent <id>` | `-a` | — | **Required.** Agent ID to simulate. Must exist in `scaffold.xcf`. |
| `--judge` | — | `false` | Run LLM-as-a-Judge evaluation after simulation. Evaluates against `agents.<id>.assertions`. |
| `--output <path>` | `-o` | `trace.jsonl` | Path for the execution trace output. |
| `--cli-path <path>` | — | `""` | Path to the CLI binary. Overrides `test.cli_path` in `scaffold.xcf`. |
| `--judge-model <model>` | — | `""` | Model for judge evaluation. Overrides `test.judge_model`. Falls back to `claude-haiku-4-5-20251001`. |

---

### `xcaffold export`

**File:** `cmd/xcaffold/export.go`

Compiles `scaffold.xcf` and packages the output as a distributable plugin directory.

| Flag | Default | Description |
|---|---|---|
| `--output <path>` | — | **Required.** Destination directory for the exported plugin. |
| `--format <fmt>` | `plugin` | Export format. Only `plugin` is currently supported. |
| `--target <target>` | `""` (defaults to `claude`) | Compilation target for the export. |

---

## Utility Commands

### `xcaffold validate`

**File:** `cmd/xcaffold/validate.go`

Validates `scaffold.xcf` without compiling. Checks:
1. YAML syntax and known fields (fail-closed parser — unknown fields are an error)
2. Cross-reference integrity: agent `skills:`, `rules:`, `mcp:` IDs must resolve to top-level map keys
3. File existence: `instructions_file` and `references` paths must resolve on disk
4. Plugin validation: `enabledPlugins` keys checked against a known registry
5. **Project Policies:** Evaluates all constraints using the Policy Enforcement Engine.

With `--structural`, additionally checks:
- Orphan skills (defined but not referenced by any agent)
- Orphan rules (defined, not referenced, no `paths`, no `alwaysApply: true`)
- Agents with no `instructions` or `instructions_file`
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

Universal parser for xcaffold diagnostic artifacts. Does not require a `scaffold.xcf` to be present (`resolveConfig` is skipped for `review`).

**Supported file types:**

| File | Output |
|---|---|
| `scaffold.xcf` | AST tree: project name, agents with model/tools/assertions. |
| `audit.json` | Compliance scores: `security`, `prompt_quality`, `tool_restrictions` (each `/100`) and feedback. |
| `plan.json` | Pretty-printed JSON. |
| `trace.jsonl` | Timestamp and tool name for each recorded event. |

**Special argument:** `all` — loops through all four file types in the current directory (or the global config directory with `--global`).

No command-specific flags.

---

### `xcaffold list`

**File:** `cmd/xcaffold/list.go`

Lists all projects registered in the global registry (`~/.xcaffold/registry.json`). Registry entries are created automatically by `xcaffold apply` and `xcaffold init`.

For each project, displays: name, path, targets, resource counts (agents, skills, rules), and last applied timestamp.

No flags.

---

### `xcaffold migrate`

**File:** `cmd/xcaffold/migrate.go`

Applies schema version upgrades and layout migrations. Safe to run repeatedly (idempotent).

**Operations (in order):**

1. **Schema `1.0 → 1.1`**: Copies `test.claude_path` to `test.cli_path` and clears the deprecated field. Writes a `.bak` backup before overwriting.
2. **Global scope migration**: Moves `~/.claude/global.xcf` → `~/.xcaffold/global.xcf` (and the accompanying lock file). Requires interactive confirmation.
3. **Project scope migration**: Rewrites flat `instructions_file` paths (e.g., `"developer.md"`) to full reference-in-place paths (e.g., `".claude/agents/developer.md"`). Registers the project. Requires interactive confirmation.

No flags.
