# Multi-Kind Resource Manifests

xcaffold supports multi-kind YAML documents where each resource gets its own `kind:` discriminator. Resources can live in a single file (separated by `---`) or in individual `.xcf` files that `ParseDirectory` merges automatically.

> **Parsing**: All multi-kind documents are decoded with `KnownFields(true)`. Unknown fields cause an immediate parse error. This is the same fail-closed behavior as monolithic `kind: config` files.

---

## Supported Kinds

| Kind | Required | Singleton | Purpose |
|---|---|---|---|
| `project` | Yes | Yes | Project manifest: metadata, targets, child references |
| `agent` | No | No | Agent persona definition |
| `skill` | No | No | Reusable prompt package |
| `rule` | No | No | Path-gated formatting guideline |
| `workflow` | No | No | Named reusable workflow |
| `mcp` | No | No | MCP server definition |
| `hooks` | No | Yes | Lifecycle event handlers |
| `settings` | No | Yes | Platform settings |
| `config` | No | Yes | Legacy monolithic format (backward compatible) |

**Singleton** kinds do not carry a `name` field and can only appear once per merged config. Non-singleton kinds are keyed by `name` and support multiple instances.

---

## Envelope Fields

Every multi-kind document begins with envelope fields that identify the resource.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | Resource type discriminator. One of the values in the table above. When omitted or set to `config`, the document is decoded as a legacy monolithic `XcaffoldConfig`. |
| `version` | `string` | **Required** | Schema version. Current: `"1.0"`. |
| `name` | `string` | Required for non-singleton kinds | Unique identifier for the resource. Used as the map key when merging. Duplicate names within the same kind cause a parse error. |

Singleton kinds (`hooks`, `settings`) skip the `name` check. The `project` kind requires `name` (it identifies the project in the registry and graph output).

---

## kind: project

Project-level metadata and child resource references. Exactly one `project` document is expected per merged config.

> **YAML field collision note**: In a `kind: project` document, the fields `agents`, `skills`, `rules`, `workflows`, and `mcp` are `[]string` reference lists (bare names linking to child `.xcf` files). This differs from the monolithic `kind: config` format where these same YAML keys map to `map[string]<Type>` inline definitions. The parser uses a separate `projectDocFields` struct to avoid the type collision.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"project"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Canonical project name. Used in registry, graph output, and generated comments. |
| `description` | `string` | Optional | Human-readable project description. |
| `author` | `string` | Optional | Project author or maintainer. |
| `homepage` | `string` | Optional | Project homepage URL. |
| `repository` | `string` | Optional | Source repository URL. |
| `license` | `string` | Optional | SPDX license identifier. |
| `backup_dir` | `string` | Optional | Custom directory for `--backup` output. Default: `.<target>_bak_<timestamp>` in project root. |
| `targets` | `[]string` | Optional | Compilation targets (e.g. `["claude", "cursor"]`). |
| `agents` | `[]string` | Optional | Bare names of child agent `.xcf` files to include. |
| `skills` | `[]string` | Optional | Bare names of child skill `.xcf` files to include. |
| `rules` | `[]string` | Optional | Bare names of child rule `.xcf` files to include. |
| `workflows` | `[]string` | Optional | Bare names of child workflow `.xcf` files to include. |
| `mcp` | `[]string` | Optional | Bare names of child MCP `.xcf` files to include. |
| `test` | `TestConfig` | Optional | Configuration for `xcaffold test`. |
| `local` | `SettingsConfig` | Optional | Project-level settings overrides. |

### TestConfig

| Field | Type | Description |
|---|---|---|
| `cli_path` | `string` | Path to the CLI binary (e.g. `claude`, `cursor`). Defaults to `claude` on `$PATH`. |
| `claude_path` | `string` | **Deprecated.** Use `cli_path`. Retained for backward compatibility. |
| `judge_model` | `string` | Generative model for LLM-as-a-Judge evaluation. |

### Example

```yaml
kind: project
version: "1.0"
name: my-api
description: REST API backend
author: team@example.com
repository: https://github.com/example/my-api
targets:
  - claude
  - cursor
agents:
  - developer
  - reviewer
skills:
  - tdd
rules:
  - testing
test:
  cli_path: claude
  judge_model: claude-haiku-4-5-20251001
```

---

## kind: agent

Agent persona definition. Each agent is keyed by `name` and compiled to a markdown file under the target's agents directory.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"agent"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Agent identifier. Used as the map key and output filename. |
| `description` | `string` | Optional | Short description shown in graph output and agent headers. |
| `instructions` | `string` | Optional | Inline instruction body. Mutually exclusive with `instructions_file`. |
| `instructions_file` | `string` | Optional | Path to a file containing instructions. Mutually exclusive with `instructions`. |
| `model` | `string` | Optional | Model override (e.g. `claude-sonnet-4-20250514`). |
| `effort` | `string` | Optional | Effort level override. |
| `memory` | `string` | Optional | Memory directory path for this agent. |
| `maxTurns` | `int` | Optional | Maximum conversation turns. |
| `permissionMode` | `string` | Optional | Permission mode for the agent session. |
| `isolation` | `string` | Optional | Isolation mode. |
| `mode` | `string` | Optional | Agent mode. |
| `when` | `string` | Optional | Condition for when the agent is available. |
| `color` | `string` | Optional | Display color in UI. |
| `initialPrompt` | `string` | Optional | Prompt sent automatically when the agent starts. |
| `readonly` | `*bool` | Optional | When `true`, agent cannot write files. |
| `background` | `*bool` | Optional | When `true`, agent runs in the background. |
| `tools` | `[]string` | Optional | Allowed tool names. |
| `disallowedTools` | `[]string` | Optional | Explicitly denied tool names. |
| `skills` | `[]string` | Optional | Skill IDs this agent can invoke. Must resolve to defined skills. |
| `rules` | `[]string` | Optional | Rule IDs applied to this agent. Must resolve to defined rules. |
| `mcp` | `[]string` | Optional | MCP server IDs available to this agent. Must resolve to defined MCP servers. |
| `assertions` | `[]string` | Optional | Test assertions for `xcaffold test --judge`. |
| `hooks` | `HookConfig` | Optional | Agent-scoped lifecycle hooks. |
| `targets` | `map[string]TargetOverride` | Optional | Per-target overrides. See [TargetOverride](#targetoverride). |
| `mcpServers` | `map[string]MCPConfig` | Optional | Agent-scoped MCP server definitions. |

### TargetOverride

| Field | Type | Description |
|---|---|---|
| `instructions_override` | `string` | Replacement instructions for this target. |
| `suppress_fidelity_warnings` | `*bool` | Suppress warnings about features unsupported by this target. |
| `skip_synthesis` | `*bool` | Skip synthesis step for this target. |
| `hooks` | `map[string]string` | Target-specific hook overrides. |

### Example

```yaml
kind: agent
version: "1.0"
name: developer
description: Senior backend developer
model: claude-sonnet-4-20250514
instructions: |
  You are a senior backend developer.
  Follow TDD. Write tests before implementation.
tools:
  - Read
  - Edit
  - Bash
  - Write
skills:
  - tdd
rules:
  - testing
maxTurns: 25
```

---

## kind: skill

Reusable prompt package. Compiled to a directory with a `SKILL.md` file and optional reference/script/asset subdirectories.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"skill"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Skill identifier. |
| `description` | `string` | Optional | Short description of the skill's purpose. |
| `instructions` | `string` | Optional | Inline skill instructions. Mutually exclusive with `instructions_file`. |
| `instructions_file` | `string` | Optional | Path to a file containing instructions. Mutually exclusive with `instructions`. |
| `tools` | `[]string` | Optional | Tools this skill requires. |
| `references` | `[]string` | Optional | Doc/data files copied to `skills/<id>/references/` at compile time. |
| `scripts` | `[]string` | Optional | Executable helpers copied to `skills/<id>/scripts/` at compile time. |
| `assets` | `[]string` | Optional | Output artifacts (templates, fonts, icons) copied to `skills/<id>/assets/`. |

### Example

```yaml
kind: skill
version: "1.0"
name: tdd
description: Test-driven development workflow
instructions: |
  Follow the Red-Green-Refactor cycle.
  1. Write a failing test.
  2. Write minimal code to pass.
  3. Refactor.
tools:
  - Bash
  - Edit
  - Read
```

---

## kind: rule

Path-gated formatting guideline. Compiled to a markdown file under the target's rules directory.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"rule"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Rule identifier. |
| `description` | `string` | Optional | Short description of the rule. |
| `instructions` | `string` | Optional | Inline rule body. Mutually exclusive with `instructions_file`. |
| `instructions_file` | `string` | Optional | Path to a file containing instructions. Mutually exclusive with `instructions`. |
| `alwaysApply` | `*bool` | Optional | When `true`, rule applies to all files regardless of `paths`. When omitted, inherits platform default. |
| `paths` | `[]string` | Optional | Glob patterns restricting which files this rule applies to. |

### Example

```yaml
kind: rule
version: "1.0"
name: testing
description: Go testing conventions
alwaysApply: false
paths:
  - "**/*_test.go"
instructions: |
  Use table-driven tests. Name tests TestFunc_Scenario.
  Always add negative test cases.
```

---

## kind: workflow

Named reusable workflow. Compiled for targets that support workflows (e.g. Antigravity). Silently ignored by Claude and Cursor renderers.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"workflow"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Workflow identifier. |
| `description` | `string` | Optional | Short description. |
| `instructions` | `string` | Optional | Inline workflow instructions. Mutually exclusive with `instructions_file`. |
| `instructions_file` | `string` | Optional | Path to a file containing instructions. Mutually exclusive with `instructions`. |

### Example

```yaml
kind: workflow
version: "1.0"
name: deploy
description: Production deployment checklist
instructions: |
  1. Run full test suite.
  2. Build release binary.
  3. Tag version.
  4. Push to registry.
```

---

## kind: mcp

MCP (Model Context Protocol) server definition. Compiled into the `mcpServers` key of `settings.json`.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"mcp"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Server identifier. Used as the key in `mcpServers`. |
| `type` | `string` | Optional | Server type (e.g. `"stdio"`, `"sse"`). |
| `command` | `string` | Optional | Command to start a stdio-based server. |
| `args` | `[]string` | Optional | Arguments passed to `command`. |
| `url` | `string` | Optional | URL for SSE-based servers. |
| `cwd` | `string` | Optional | Working directory for the server process. |
| `env` | `map[string]string` | Optional | Environment variables passed to the server. |
| `headers` | `map[string]string` | Optional | HTTP headers for SSE connections. |
| `disabled` | `*bool` | Optional | When `true`, server is defined but not activated. |
| `disabledTools` | `[]string` | Optional | Tools from this server that should be disabled. |
| `oauth` | `map[string]string` | Optional | OAuth configuration for authenticated servers. |
| `authProviderType` | `string` | Optional | Authentication provider type. |

### Example

```yaml
kind: mcp
version: "1.0"
name: github
type: stdio
command: npx
args:
  - -y
  - "@modelcontextprotocol/server-github"
env:
  GITHUB_PERSONAL_ACCESS_TOKEN: "${GITHUB_TOKEN}"
```

---

## kind: hooks

Lifecycle event handlers. Singleton: no `name` field. Hook events are wrapped under an `events:` key because `HookConfig` is a map type that cannot be inlined.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"hooks"` |
| `version` | `string` | **Required** | Schema version. |
| `events` | `HookConfig` | **Required** | Map of event names to matcher groups. |

### HookConfig

`HookConfig` is `map[string][]HookMatcherGroup`. Keys are event names:

`PreToolUse`, `PostToolUse`, `Notification`, `Stop`, `SubagentStop`, `InstructionsLoaded`, `PreCompact`, `SessionStart`, `ConfigChange`.

### HookMatcherGroup

| Field | Type | Description |
|---|---|---|
| `matcher` | `string` | Pattern to match against the event payload. |
| `hooks` | `[]HookHandler` | Handlers to execute when the matcher matches. |

### HookHandler

| Field | Type | Description |
|---|---|---|
| `type` | `string` | **Required.** Handler type: `"command"`, `"url"`, `"prompt"`. |
| `command` | `string` | Shell command to execute (for `type: command`). |
| `url` | `string` | URL to call (for `type: url`). |
| `prompt` | `string` | Prompt to inject (for `type: prompt`). |
| `model` | `string` | Model to use for prompt hooks. |
| `shell` | `string` | Shell to use for command hooks (e.g. `bash`, `zsh`). |
| `if` | `string` | Conditional expression. Hook runs only when this evaluates truthy. |
| `statusMessage` | `string` | Message displayed while the hook runs. |
| `async` | `*bool` | When `true`, hook runs without blocking. |
| `timeout` | `*int` | Timeout in milliseconds. |
| `once` | `*bool` | When `true`, hook fires only once per session. |
| `headers` | `map[string]string` | HTTP headers for URL hooks. |
| `allowedEnvVars` | `[]string` | Environment variables the hook is allowed to read. |

### Example

```yaml
kind: hooks
version: "1.0"
events:
  PreToolUse:
    - matcher: Bash
      hooks:
        - type: command
          command: echo "Bash tool invoked"
          timeout: 5000
  SessionStart:
    - hooks:
        - type: command
          command: echo "Session started"
          once: true
```

---

## kind: settings

Platform settings compiled to `settings.json`. Singleton: no `name` field.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"settings"` |
| `version` | `string` | **Required** | Schema version. |
| `model` | `string` | Optional | Default model. |
| `effortLevel` | `string` | Optional | Default effort level. |
| `defaultShell` | `string` | Optional | Default shell for Bash commands. |
| `language` | `string` | Optional | Language preference. |
| `outputStyle` | `string` | Optional | Output formatting style. |
| `plansDirectory` | `string` | Optional | Directory for plan files. |
| `autoMemoryDirectory` | `string` | Optional | Directory for auto-memory files. |
| `otelHeadersHelper` | `string` | Optional | OpenTelemetry headers helper command. |
| `includeGitInstructions` | `*bool` | Optional | Include git instructions in context. |
| `autoMemoryEnabled` | `*bool` | Optional | Enable automatic memory. |
| `disableAllHooks` | `*bool` | Optional | Disable all hooks globally. |
| `respectGitignore` | `*bool` | Optional | Respect `.gitignore` patterns. |
| `attribution` | `*bool` | Optional | Enable attribution in output. |
| `skipDangerousModePermissionPrompt` | `*bool` | Optional | Skip the dangerous mode confirmation prompt. |
| `alwaysThinkingEnabled` | `*bool` | Optional | Enable always-on thinking. |
| `disableSkillShellExecution` | `*bool` | Optional | Disable shell execution within skills. |
| `cleanupPeriodDays` | `*int` | Optional | Days before automatic cleanup of old sessions. |
| `agent` | `any` | Optional | Free-form agent configuration. Passed through unchanged. |
| `worktree` | `any` | Optional | Free-form worktree configuration. Passed through unchanged. |
| `autoMode` | `any` | Optional | Free-form auto-mode configuration. Passed through unchanged. |
| `permissions` | `*PermissionsConfig` | Optional | Permission rules. See below. |
| `sandbox` | `*SandboxConfig` | Optional | Sandbox configuration. See below. |
| `statusLine` | `*StatusLineConfig` | Optional | Status line configuration. |
| `hooks` | `HookConfig` | Optional | Settings-level hooks. |
| `mcpServers` | `map[string]MCPConfig` | Optional | MCP servers defined at settings level. |
| `env` | `map[string]string` | Optional | Environment variables. |
| `enabledPlugins` | `map[string]bool` | Optional | Plugin enable/disable toggles. |
| `availableModels` | `[]string` | Optional | List of available models. |
| `claudeMdExcludes` | `[]string` | Optional | Glob patterns for CLAUDE.md files to exclude. |

### PermissionsConfig

| Field | Type | Description |
|---|---|---|
| `defaultMode` | `string` | Default permission mode. |
| `disableBypassPermissionsMode` | `string` | Disable bypass permissions mode. |
| `allow` | `[]string` | Allowed permission rules (e.g. `"Bash(npm test *)"`, `"Read"`). |
| `deny` | `[]string` | Denied permission rules. |
| `ask` | `[]string` | Rules that require user confirmation. |
| `additionalDirectories` | `[]string` | Extra directories the agent may access. |

### SandboxConfig

| Field | Type | Description |
|---|---|---|
| `enabled` | `*bool` | Enable OS-level process isolation. |
| `autoAllowBashIfSandboxed` | `*bool` | Auto-approve bash when sandboxed. |
| `failIfUnavailable` | `*bool` | Fail if sandboxing is unavailable on the platform. |
| `allowUnsandboxedCommands` | `*bool` | Allow specific commands to run outside the sandbox. |
| `excludedCommands` | `[]string` | Commands excluded from sandboxing. |
| `filesystem` | `*SandboxFilesystem` | Filesystem isolation rules. |
| `network` | `*SandboxNetwork` | Network isolation rules. |

### Example

```yaml
kind: settings
version: "1.0"
model: claude-sonnet-4-20250514
permissions:
  allow:
    - "Bash(npm test *)"
    - "Read"
    - "Glob"
  deny:
    - "Bash(rm -rf *)"
sandbox:
  enabled: true
  autoAllowBashIfSandboxed: true
```

---

## kind: config

Legacy monolithic format. A single document containing all resources inline under their respective map keys (`agents:`, `skills:`, `rules:`, etc.) alongside `project:` and `settings:` blocks. This is the original xcaffold format and remains fully supported.

When `kind` is omitted or set to `"config"`, the document is decoded as a full `XcaffoldConfig` struct. All fields documented in the [Schema Reference](schema.md) are valid.

```yaml
# kind: config is implied when kind is omitted
version: "1.0"
project:
  name: my-project
agents:
  developer:
    description: Backend developer
    instructions: Write clean Go code.
skills:
  tdd:
    description: TDD workflow
    instructions: Red-Green-Refactor.
```

---

## Merge Behavior

### Multi-document files

A single `.xcf` file can contain multiple YAML documents separated by `---`. The parser iterates through each document, extracts the `kind:` discriminator, and routes to the appropriate kind-specific decoder.

```yaml
kind: project
version: "1.0"
name: my-api
agents:
  - developer
---
kind: agent
version: "1.0"
name: developer
description: Backend developer
instructions: Write clean Go code.
---
kind: rule
version: "1.0"
name: testing
instructions: Use table-driven tests.
```

### ParseDirectory

`ParseDirectory` scans a directory for all `*.xcf` files, parses each independently, and merges the results. Merge rules:

- **Strict dedup**: Duplicate resource IDs within the same kind cause a parse error (`duplicate agent ID "foo"`).
- **Singleton merge**: A second `project`, `hooks`, or `settings` document overwrites the first. For `hooks`, events are appended (not replaced) per event key.
- **Version propagation**: If a resource document sets `version` and the merged config has no version yet, the resource's version is propagated.

### Project scope routing

In a `kind: project` document, the fields `agents`, `skills`, `rules`, `workflows`, and `mcp` are `[]string` reference lists. These bare names tell `ParseDirectory` which child `.xcf` files to include. The actual resource definitions live in separate files (e.g. `xcf/agents/developer.xcf`), each with their own `kind: agent` header.

---

## Backward Compatibility

`kind: config` (and documents with `kind` omitted) remain permanently supported. Existing monolithic `scaffold.xcf` files parse identically to previous versions. The multi-kind format is additive -- it does not deprecate or remove any existing functionality.

The two formats can coexist in the same directory: `ParseDirectory` handles both `kind: config` files and individual `kind: <resource>` files, merging them into a single AST with the same strict dedup rules.
