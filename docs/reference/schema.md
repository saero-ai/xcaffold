# Schema Reference

Exhaustive reference for the `.xcf` YAML configuration schema. Every field is verified against `internal/ast/types.go`.

> **Parsing**: Xcaffold uses a strict, fail-closed YAML parser (`KnownFields(true)`). Unknown fields cause an immediate parse error. All filesystem paths are resolved via `filepath.Clean()` — `..` traversal is rejected.

**Type conventions:**
- `*bool` / `*int` — A pointer type. When omitted from YAML, the field is `nil` (inherits platform default). When explicitly set to `false` / `0`, it overrides the default. This distinction matters: omitting a `*bool` field is **not** the same as setting it to `false`.
- `any` — A free-form value passed through to the target platform's settings file unchanged. Xcaffold does not validate the contents.

---

## Scopes

Xcaffold operates at two scopes, selected via the `--global / -g` flag.

| | Global (`-g`) | Project (default) |
|---|---|---|
| **Config dir** | `~/.xcaffold/` | Project root (`./`) |
| **Primary file** | `~/.xcaffold/global.xcf` | `./scaffold.xcf` |
| **Lock file** | `~/.xcaffold/scaffold.<target>.lock` | `./scaffold.<target>.lock` |
| **Compiled output** | `~/.claude/`, `~/.cursor/`, `~/.agents/` | `./.claude/`, `./.cursor/`, `./.agents/` |
| **Represents** | User-wide personal agent configuration | A specific codebase's agent setup |
| **Has `project:` block** | **No** — it is a user profile, not a project | **Yes** |
| **Has `test:`** | No | Yes |
| **Has `settings:`** | Yes — user-level defaults | Yes — project-level overrides |
| **Has `extends:`** | No — it is the root config | Yes — may declare `extends: "path/to/base.xcf"` |

### Implicit Global Inheritance

When evaluating a project configuration, xcaffold implicitly loads `~/.xcaffold/` first as a base layer. 

- **Cross-reference validation** — an agent in the project can reference a skill defined globally without a parse error.
- **Visualization** — `xcaffold graph` and `xcaffold plan` automatically show both inherited global resources and the local project resources.

> [!IMPORTANT]
> Because global configs are aggregated implicitly, you do **not** need to declare `extends: global` in your project's `.xcf` file.
> 
> Furthermore, during compilation (`xcaffold apply`), global resources are securely stripped from the AST. They are **not** physically copied into the project's compiled output directory (like `.claude/`). Global resources are already natively available to agentic runtimes independently (compiled separately via `xcaffold apply -g`). Duplication into project directories would create conflicting clones.

### Running both scopes

Global and project scopes are fully independent compilations. To compile both:

```bash
xcaffold apply -g   # compile ~/.xcaffold/global.xcf → ~/.claude/
xcaffold apply      # compile ./scaffold.xcf → ./.claude/
```

There is no `--all` flag — the two commands are independent (different sources, outputs, lock files) with no atomicity guarantee.

---

## `XcaffoldConfig`

Root structure of a parsed `.xcf` file. Used at both project scope (`./scaffold.xcf`) and global scope (`~/.xcaffold/global.xcf`). See [Scopes](#scopes) for field applicability at each level.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | Optional | File type discriminator. Values: `"config"` (compiler input) or omitted (treated as `config` for backward compatibility). Global configs should set `kind: config`. Non-config files (e.g. `registry.xcf`) use `kind: registry` and are skipped by the directory scanner. |
| `version` | `string` | **Required** | Schema version. Current: `"1.1"`. |
| `project` | `ProjectConfig` | Project scope only | Project-level metadata. Not present in global config. |
| `extends` | `string` | Optional (project only) | Path to a parent `.xcf` config. Use `"global"` to reference `~/.xcaffold/global.xcf` for validation and visualization. Does not affect compiled output. |
| `agents` | `map[string]AgentConfig` | Optional | Agent persona declarations keyed by ID. |
| `skills` | `map[string]SkillConfig` | Optional | Reusable prompt packages keyed by ID. |
| `rules` | `map[string]RuleConfig` | Optional | Path-gated formatting guidelines keyed by ID. |
| `hooks` | `HookConfig` | Optional | Lifecycle event handlers. |
| `mcp` | `map[string]MCPConfig` | Optional | MCP server definitions. Merged into `settings.mcpServers` during compilation; `settings.mcpServers` wins on key conflicts. |
| `workflows` | `map[string]WorkflowConfig` | Optional | Reusable workflows keyed by ID. **Antigravity-only**: silently ignored by Claude and Cursor renderers. |
| `test` | `TestConfig` | Project scope only | Configuration for `xcaffold test`. Not meaningful at global scope. |
| `settings` | `SettingsConfig` | Optional | Platform settings compiled to `settings.json`. At global scope, these become user-level defaults. |
| `local` | `SettingsConfig` | Project scope only | Local override settings compiled to `settings.local.json` (gitignored). |

---

## `ProjectConfig`

Project-level metadata. Present **only** in project-scope configs (`scaffold.xcf`). Global config (`global.xcf`) is a user profile, not a project — it has no `project:` block.

> [!NOTE]
> `project.name` is required in `scaffold.xcf`. It is used to register the project in `~/.xcaffold/registry.xcf`, prefix lock file entries, and label graph and plan output. It has no equivalent at global scope.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | `string` | **Required** | Canonical project name. Used in registry, graph output, and generated comments. |
| `description` | `string` | Optional | Human-readable project description. |
| `version` | `string` | Optional | Project version for tracking purposes. |
| `author` | `string` | Optional | Maintainer identifier. |
| `homepage` | `string` | Optional | Project URL. |
| `repository` | `string` | Optional | Source control URL. |
| `license` | `string` | Optional | SPDX license identifier. |
| `backup_dir` | `string` | Optional | Directory for `xcaffold apply --backup` output. Defaults to `.<target>_bak_<timestamp>` in the project root. |

---

## `AgentConfig`

Defines an agent persona. Compiled to `agents/<id>.md` with YAML frontmatter.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | `string` | Optional | Display name. Defaults to the map key if omitted. |
| `description` | `string` | Optional | Brief explanation of the agent's purpose. |
| `instructions` | `string` | Optional | Inline Markdown prompt body. Mutually exclusive with `instructions_file`. |
| `instructions_file` | `string` | Optional | Path to a Markdown file containing the prompt body. Resolved relative to `scaffold.xcf` directory. Mutually exclusive with `instructions`. |
| `model` | `string` | Optional | LLM model identifier. Supports aliases (`sonnet-4`, `opus-4`, `haiku-3.5`) which are resolved per-target by the model resolver. |
| `effort` | `string` | Optional | Resource utilization level (`"high"`, `"medium"`, `"low"`). |
| `memory` | `string` | Optional | Agent memory mode. |
| `maxTurns` | `int` | Optional | Maximum conversation turns before the agent stops. |
| `tools` | `[]string` | Optional | Runtime tools granted to the agent (e.g., `Bash`, `Read`, `Write`, `Edit`, `Glob`, `Grep`). |
| `disallowedTools` | `[]string` | Optional | Tools the agent is explicitly forbidden from using. |
| `skills` | `[]string` | Optional | Skill IDs to grant. Must match top-level `skills:` map keys. |
| `rules` | `[]string` | Optional | Rule IDs the agent must follow. Must match top-level `rules:` map keys. |
| `mcp` | `[]string` | Optional | MCP server IDs to load. Must match top-level `mcp:` map keys. |
| `permissionMode` | `string` | Optional | Default execution permission fallback. |
| `readonly` | `*bool` | Optional | When `true` and `tools` is empty, the compiler emits `tools: [Read, Grep, Glob]`. |
| `background` | `*bool` | Optional | Execute without blocking the UI. Compiled as `is_background` for Cursor. |
| `isolation` | `string` | Optional | Worktree or environment isolation preference. |
| `mode` | `string` | Optional | Agent execution mode. |
| `when` | `string` | Optional | Trigger condition for agent invocation. |
| `color` | `string` | Optional | Terminal UI color attribute. |
| `initialPrompt` | `string` | Optional | Default message sent on agent launch. |
| `assertions` | `[]string` | Optional | Behavioral constraints evaluated by `xcaffold test --judge`. |
| `targets` | `map[string]TargetOverride` | Optional | Per-target compilation overrides. Keys are target names (`cursor`, `antigravity`, `agentsmd`). |
| `mcpServers` | `map[string]MCPConfig` | Optional | Agent-scoped MCP server definitions (not merged with top-level `mcp:`). |
| `hooks` | `HookConfig` | Optional | Agent-scoped lifecycle hooks. |

> [!WARNING]
> **Cursor**: `effort`, `tools`, `disallowedTools`, `skills`, `rules`, `permissionMode`, `isolation`, `color`, `initialPrompt`, `memory`, `maxTurns`, `hooks`, `mcpServers` are silently dropped. `background` is renamed to `is_background`. Unmapped `model` values emit a stderr warning and are omitted.
>
> **Antigravity**: Agents are **not compiled** — only rules, skills, and workflows are emitted. Security fields (`permissionMode`, `disallowedTools`, `isolation`) emit stderr warnings unless `targets.antigravity.suppress_fidelity_warnings` is set.
>
> **AgentsMD**: Only `name`, `description`, `model`, and instruction body are preserved. All other fields emit fidelity warnings and are dropped.

---

## `TargetOverride`

Per-target compilation overrides. Used inside `agents.<id>.targets.<target>`.

| Field | Type | Required | Description |
|---|---|---|---|
| `hooks` | `map[string]string` | Optional | Target-specific hook overrides. |
| `suppress_fidelity_warnings` | `*bool` | Optional | When `true`, suppresses stderr warnings about dropped fields for this target. |
| `skip_synthesis` | `*bool` | Optional | When `true`, skips synthesis for this target. |
| `instructions_override` | `string` | Optional | Replacement instruction prompt used instead of the agent's default `instructions`. |

---

## `SkillConfig`

Defines a reusable prompt package. Compiled to `skills/<id>/SKILL.md`.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | `string` | Optional | Display name for the skill. |
| `description` | `string` | Optional | Human-readable description. Shown in listings and help text. |
| `instructions` | `string` | Optional | Inline Markdown prompt body. Mutually exclusive with `instructions_file`. |
| `instructions_file` | `string` | Optional | Path to a Markdown file. Mutually exclusive with `instructions`. |
| `tools` | `[]string` | Optional | Tools required or relevant during skill execution. |
| `references` | `[]string` | Optional | Supporting files or glob patterns. Resolved relative to `scaffold.xcf`. Copied to `skills/<id>/references/` during compilation. |
| `scripts` | `[]string` | Optional | Executable helper files. Resolved relative to `scaffold.xcf`. Copied to `skills/<id>/scripts/` during compilation. |
| `assets` | `[]string` | Optional | Output artifact files like templates or icons. Resolved relative to `scaffold.xcf`. Copied to `skills/<id>/assets/` during compilation. |

> [!WARNING]
> **Cursor**: Script and asset bundling is not supported. Those directories are dropped during compilation with a standard stderr warning.
>
> **Antigravity**: Frontmatter fields beyond `name` and `description` are dropped.
>
> **AgentsMD**: Only instruction bodies and `description` are emitted.

---

## `RuleConfig`

Path-gated formatting guideline. Compiled to `rules/<id>.md` (Claude), `rules/<id>.mdc` (Cursor), or `rules/<id>.md` (Antigravity).

| Field | Type | Required | Description |
|---|---|---|---|
| `description` | `string` | Optional | Summary of the rule. |
| `instructions` | `string` | Optional | Inline rule content. Mutually exclusive with `instructions_file`. |
| `instructions_file` | `string` | Optional | Path to a Markdown file. Mutually exclusive with `instructions`. |
| `paths` | `[]string` | Optional | Glob patterns restricting where the rule applies (e.g., `["src/**/*.go"]`). |
| `alwaysApply` | `*bool` | Optional | When `true`, the rule applies globally regardless of `paths`. |

> [!NOTE]
> **Cursor normalization**: `paths` is emitted as `globs:` in `.mdc` frontmatter. Rules without `paths` automatically receive `alwaysApply: true`.
>
> **Antigravity normalization**: No YAML frontmatter is emitted. `description` becomes a `# heading`. `paths` and `alwaysApply` are dropped — Antigravity handles rule activation via UI. Bodies exceeding 12,000 characters receive a warning comment.
>
> **AgentsMD**: Rules with `paths` are grouped into directory-scoped `AGENTS.md` files. `alwaysApply` is dropped.

---

## Hooks

### `HookConfig`

`HookConfig` is a `map[string][]HookMatcherGroup`. Keys are lifecycle event names:

`PreToolUse`, `PostToolUse`, `Notification`, `Stop`, `SubagentStop`, `InstructionsLoaded`, `PreCompact`, `SessionStart`, `ConfigChange`.

```yaml
hooks:
  PreToolUse:                  # ← event name (map key)
    - matcher: "Bash"          # ← HookMatcherGroup
      hooks:                   # ← []HookHandler
        - type: command
          command: "echo 'before bash'"
```

### `HookMatcherGroup`

Groups hook handlers under a matcher pattern within an event.

| Field | Type | Required | Description |
|---|---|---|---|
| `matcher` | `string` | Optional | Pattern to match against the tool name (e.g., `"Bash"`, `"Write"`). If empty, the hooks fire for all tools. |
| `hooks` | `[]HookHandler` | **Required** | List of handlers to execute when the matcher matches. |

### `HookHandler`

A single executable hook action.

| Field | Type | Required | Description |
|---|---|---|---|
| `type` | `string` | **Required** | Handler type. Values: `command`, `url`, `prompt`. |
| `command` | `string` | Optional | Shell command to execute (when `type: command`). |
| `url` | `string` | Optional | URL endpoint for webhook dispatch (when `type: url`). |
| `prompt` | `string` | Optional | LLM prompt to evaluate (when `type: prompt`). |
| `model` | `string` | Optional | LLM model for prompt hooks. |
| `shell` | `string` | Optional | Shell binary override (e.g., `/bin/bash`). |
| `if` | `string` | Optional | Conditional expression. Hook is skipped when the condition evaluates to false. |
| `statusMessage` | `string` | Optional | Status text displayed in the UI while the hook runs. |
| `headers` | `map[string]string` | Optional | HTTP headers for `url`-type hooks. |
| `allowedEnvVars` | `[]string` | Optional | Environment variables passed to the hook subprocess. |
| `async` | `*bool` | Optional | When `true`, the hook runs non-blocking. |
| `timeout` | `*int` | Optional | Maximum execution time in seconds before the hook is killed. |
| `once` | `*bool` | Optional | When `true`, the hook fires only once per session. |

> [!WARNING]
> **Cursor normalization**: Event names are converted to camelCase (`PreToolUse` → `preToolUse`). The three-level structure (`event → matcher group → handlers`) is flattened to two levels (`event → handlers` with `matcher` injected as a field on each handler). `${...}` interpolation patterns emit a warning — Cursor requires `${env:NAME}` syntax.
>
> **Antigravity**: Hooks are silently skipped — Antigravity has no hook system.
>
> **AgentsMD**: Hooks are dropped with fidelity warnings.

---

## `MCPConfig`

Defines a local or remote MCP (Model Context Protocol) server.

| Field | Type | Required | Description |
|---|---|---|---|
| `type` | `string` | Optional | Connection transport: `"stdio"` or `"sse"`. |
| `command` | `string` | Optional | Binary to spawn (for `stdio` transport). |
| `url` | `string` | Optional | Server endpoint URL (for `sse` transport). |
| `cwd` | `string` | Optional | Working directory for the spawned process. |
| `authProviderType` | `string` | Optional | Authentication method. |
| `args` | `[]string` | Optional | Command-line arguments for the spawned process. |
| `disabledTools` | `[]string` | Optional | Server-provided tools to mask. |
| `env` | `map[string]string` | Optional | Environment variables for the spawned process. |
| `headers` | `map[string]string` | Optional | HTTP headers for SSE connections. |
| `oauth` | `map[string]string` | Optional | OAuth authentication configuration. |
| `disabled` | `*bool` | Optional | When `true`, the server is defined but not loaded. |

> [!NOTE]
> **Claude**: MCP definitions from the top-level `mcp:` block are merged into `settings.json` under `mcpServers`. `settings.mcpServers` takes precedence on key conflicts.
>
> **Cursor normalization**: `url` is emitted as `serverUrl`. The `type` field is omitted — Cursor infers transport from the presence of `command` (stdio) or `serverUrl` (http/sse). Output file: `mcp.json`.
>
> **Antigravity**: MCP is silently skipped with `${...}` interpolation warnings on env values.

---

## `SettingsConfig`

Platform settings compiled to `settings.json` (Claude) or `settings.local.json` (from the `local:` block). These fields are passed through to the target platform's native settings file.

| Field | Type | Required | Description |
|---|---|---|---|
| `model` | `string` | Optional | Default LLM model for the project. Resolved through the model alias system per-target. |
| `effortLevel` | `string` | Optional | Default effort level: `"high"`, `"medium"`, `"low"`. |
| `defaultShell` | `string` | Optional | Shell binary for command execution (e.g., `/bin/bash`). |
| `language` | `string` | Optional | UI language code (e.g., `"en"`, `"ja"`). |
| `outputStyle` | `string` | Optional | Terminal output format (e.g., `"markdown"`, `"plain"`). |
| `plansDirectory` | `string` | Optional | Directory for persistent plan files. |
| `autoMemoryDirectory` | `string` | Optional | Directory for auto-memory context persistence. |
| `otelHeadersHelper` | `string` | Optional | OpenTelemetry header mapping helper command. |
| `agent` | `any` | Optional | Default agent configuration. Passed through unchanged. |
| `worktree` | `any` | Optional | Worktree settings. Passed through unchanged. |
| `autoMode` | `any` | Optional | Autonomous mode settings. Passed through unchanged. |
| `cleanupPeriodDays` | `*int` | Optional | Days before orphaned data is garbage-collected. |
| `includeGitInstructions` | `*bool` | Optional | When `true`, injects standard Git workflow instructions. |
| `skipDangerousModePermissionPrompt` | `*bool` | Optional | When `true`, suppresses the dangerous-mode confirmation dialog. |
| `autoMemoryEnabled` | `*bool` | Optional | When `true`, enables automatic context memory. |
| `disableAllHooks` | `*bool` | Optional | When `true`, globally disables all hook execution. |
| `attribution` | `*bool` | Optional | When `true`, adds attribution comments to generated code. |
| `disableSkillShellExecution` | `*bool` | Optional | When `true`, prevents skills from executing shell commands. |
| `alwaysThinkingEnabled` | `*bool` | Optional | When `true`, forces extended thinking mode for all interactions. |
| `respectGitignore` | `*bool` | Optional | When `true`, excludes `.gitignore`-matched files from scanning. |
| `permissions` | `*PermissionsConfig` | Optional | Permission rules for tool access. |
| `sandbox` | `*SandboxConfig` | Optional | OS-level process isolation for Bash commands. |
| `statusLine` | `*StatusLineConfig` | Optional | Custom status bar configuration. |
| `hooks` | `HookConfig` | Optional | Settings-level lifecycle hooks. |
| `mcpServers` | `map[string]MCPConfig` | Optional | MCP server definitions (takes precedence over top-level `mcp:` on key conflicts). |
| `env` | `map[string]string` | Optional | Global environment variables. |
| `enabledPlugins` | `map[string]bool` | Optional | Plugin enable/disable toggles. |
| `availableModels` | `[]string` | Optional | Models available for user selection. |
| `claudeMdExcludes` | `[]string` | Optional | File patterns excluded from context file loading. |

> [!IMPORTANT]
> **Claude only.** Settings are compiled to `settings.json` and `settings.local.json`. No other target renders settings. Cursor, Antigravity, and AgentsMD silently ignore the entire `settings:` and `local:` blocks.
>
> **Security fields**: `permissions` and `sandbox` emit explicit stderr warnings when compiled for Cursor or Antigravity, since those platforms have no enforcement mechanism. The warnings are: `"settings.permissions dropped — <target> has no permission enforcement"` and `"settings.sandbox dropped — <target> has no sandbox model"`.

---

## `PermissionsConfig`

Permission rules for tool access within `settings.permissions`.

| Field | Type | Required | Description |
|---|---|---|---|
| `allow` | `[]string` | Optional | Permitted tool operations (e.g., `"Bash(npm test *)"`, `"Read"`). |
| `deny` | `[]string` | Optional | Denied operations. Overrides `allow` when both match. |
| `ask` | `[]string` | Optional | Operations requiring interactive user confirmation. |

---

## `SandboxConfig`

OS-level process isolation within `settings.sandbox`.

| Field | Type | Required | Description |
|---|---|---|---|
| `enabled` | `*bool` | Optional | Master toggle for sandbox enforcement. |
| `autoAllow` | `*bool` | Optional | When `true`, auto-approves sandboxed commands without prompting. |
| `failIfUnavailable` | `*bool` | Optional | When `true`, commands fail if the sandbox daemon is unreachable. |
| `allowUnsandboxedCommands` | `*bool` | Optional | When `true`, permits unsandboxed execution as a fallback. |
| `filesystem` | `*SandboxFilesystem` | Optional | Filesystem isolation boundaries. |
| `network` | `*SandboxNetwork` | Optional | Network isolation boundaries. |
| `excludedCommands` | `[]string` | Optional | Shell commands that bypass sandbox restrictions. |

### `SandboxFilesystem`

Filesystem read/write boundaries within `sandbox.filesystem`.

| Field | Type | Required | Description |
|---|---|---|---|
| `allowWrite` | `[]string` | Optional | Paths where write access is permitted. |
| `denyWrite` | `[]string` | Optional | Paths where write access is denied (overrides `allowWrite`). |
| `allowRead` | `[]string` | Optional | Paths where read access is permitted. |
| `denyRead` | `[]string` | Optional | Paths where read access is denied (overrides `allowRead`). |

### `SandboxNetwork`

Network isolation boundaries within `sandbox.network`.

| Field | Type | Required | Description |
|---|---|---|---|
| `httpProxyPort` | `*int` | Optional | HTTP proxy port for network interception. |
| `socksProxyPort` | `*int` | Optional | SOCKS proxy port for network interception. |
| `allowManagedDomainsOnly` | `*bool` | Optional | When `true`, restricts connections to managed domains only. |
| `allowUnixSockets` | `*bool` | Optional | When `true`, permits Unix domain socket connections. |
| `allowedDomains` | `[]string` | Optional | Domains permitted for outbound connections. |

---

## `StatusLineConfig`

Custom status bar command within `settings.statusLine`.

| Field | Type | Required | Description |
|---|---|---|---|
| `type` | `string` | Optional | Status line type. Currently only `"command"` is supported. |
| `command` | `string` | Optional | Shell command whose output is displayed in the status bar. |

---

## `TestConfig`

Configuration for `xcaffold test`.

| Field | Type | Required | Description |
|---|---|---|---|
| `cli_path` | `string` | Optional | Path to the CLI binary used for simulation (e.g., `claude`, `cursor`). Defaults to `claude` on `$PATH`. |
| `claude_path` | `string` | Optional | **Deprecated.** Alias for `cli_path`. Migrated automatically by `xcaffold migrate` (schema `1.0` → `1.1`). |
| `judge_model` | `string` | Optional | LLM model used for `--judge` evaluation. Defaults to `claude-haiku-4-5-20251001`. |

---

## `WorkflowConfig`

Defines a named, reusable workflow. Compiled to `workflows/<id>.md`.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | `string` | Optional | Workflow title. Used as fallback `description` in Antigravity frontmatter. |
| `description` | `string` | Optional | Human-readable description of the workflow. |
| `instructions` | `string` | Optional | Inline Markdown workflow content. Mutually exclusive with `instructions_file`. |
| `instructions_file` | `string` | Optional | Path to a Markdown file. Mutually exclusive with `instructions`. |

> [!IMPORTANT]
> **Antigravity-only.** Workflows are compiled to `workflows/<id>.md` with YAML frontmatter containing only `description`. Claude and Cursor renderers silently ignore all workflow definitions.
>
> **AgentsMD**: Workflows are emitted under a `## Workflows` section.

---

## Provider Compatibility Matrix

Summary of which resource types compile for each target.

| Resource | Claude | Cursor | Antigravity | AgentsMD |
|---|---|---|---|---|
| **Agents** | ✅ `agents/<id>.md` | ✅ `agents/<id>.md` | ❌ skipped | ✅ `## Agents` section |
| **Skills** | ✅ `skills/<id>/SKILL.md` | ✅ `skills/<id>/SKILL.md` | ✅ `skills/<id>/SKILL.md` | ✅ `## Skills` section |
| **Rules** | ✅ `rules/<id>.md` | ✅ `rules/<id>.mdc` | ✅ `rules/<id>.md` | ✅ `## Rules` section |
| **Hooks** | ✅ `hooks.json` | ✅ `hooks.json` (flattened) | ❌ skipped | ❌ dropped |
| **MCP** | ✅ via `settings.json` | ✅ `mcp.json` | ❌ skipped | ❌ dropped |
| **Workflows** | ❌ ignored | ❌ ignored | ✅ `workflows/<id>.md` | ✅ `## Workflows` section |
| **Settings** | ✅ `settings.json` | ❌ ignored | ❌ ignored | ❌ ignored |
| **Local** | ✅ `settings.local.json` | ❌ ignored | ❌ ignored | ❌ ignored |

### Key normalizations by target

| Normalization | Source | Target |
|---|---|---|
| `background: true` | → `is_background: true` | Cursor agents |
| `paths:` | → `globs:` | Cursor rules (`.mdc` frontmatter) |
| `url:` | → `serverUrl:` | Cursor MCP (`mcp.json`) |
| MCP `type:` field | → omitted | Cursor (infers transport from `command` vs `serverUrl`) |
| Hook event casing | `PreToolUse` → `preToolUse` | Cursor hooks |
| Hook structure | 3-level (event → matcher group → handlers) → 2-level (event → handlers with inline matcher) | Cursor hooks |
| Rule frontmatter | `---` YAML frontmatter | → `# heading` (no frontmatter) | Antigravity rules |
| Rule `paths:` / `alwaysApply:` | → dropped | Antigravity rules |
| Skill frontmatter fields | all metadata | → only `name` + `description` | Antigravity skills |
| Model aliases | `sonnet-4`, `opus-4`, `haiku-3.5` | → resolved per-target via `renderer.ResolveModel()` |
