---
title: "One Kind Per File — Resource Format Reference"
description: "Reference for xcaffold resource file formats: frontmatter for body-bearing kinds, pure YAML for structural kinds"
---

# One Kind Per File — Resource Format Reference

xcaffold uses a single-kind-per-file layout. Each `.xcf` file declares exactly one `kind:`. Resources under `xcf/` are discovered automatically by `ParseDirectory`, which scans recursively and merges results before compilation.

> **Parsing**: All `.xcf` files are decoded with `KnownFields(true)`. Unknown fields cause an immediate parse error.

---

## File Format by Kind

xcaffold uses two distinct file formats depending on whether a resource has an instruction body.

### Body-Bearing Kinds (Frontmatter Format)

`agent`, `skill`, `rule`, and `workflow` embed their instruction body as Markdown after a YAML frontmatter block.

```
---
kind: agent
version: "1.0"
name: developer
description: "Senior backend developer."
model: claude-sonnet-4-6
---
You are a senior backend developer.
Follow TDD. Write tests before implementation.
```

The `---` delimiters mark the frontmatter block. Everything after the closing `---` is the instruction body. The `instructions:` field is not used when frontmatter format is in effect.

### Non-Body Kinds (Pure YAML)

`project`, `settings`, `hooks`, `policy`, `mcp`, `global`, and `memory` use plain YAML with no `---` delimiters.

```yaml
kind: project
version: "1.0"
name: my-api
targets:
  - claude
```

---

## Supported Kinds

| Kind | Format | Singleton | Purpose |
|---|---|---|---|
| `project` | Pure YAML | Yes | Project manifest: metadata, targets, and project-wide instructions. |
| `agent` | Frontmatter | No | Agent persona definition |
| `skill` | Frontmatter | No | Reusable prompt package |
| `rule` | Frontmatter | No | Path-gated formatting guideline |
| `workflow` | Frontmatter | No | Named reusable workflow |
| `mcp` | Pure YAML | No | MCP server definition |
| `hooks` | Pure YAML | Yes | Lifecycle event handlers |
| `settings` | Pure YAML | Yes | Platform settings |
| `global` | Pure YAML | Yes | Global-scope configuration (resources + settings, no project metadata) |
| `policy` | Pure YAML | No | Declarative constraint (require/deny rules) |

**Singleton** kinds do not carry a `name` field and can only appear once per merged config. Non-singleton kinds are keyed by `name` and support multiple instances.

---

## Envelope Fields

Every `.xcf` file begins with envelope fields that identify the resource.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | Required. Must be one of the supported kind values listed above. |
| `version` | `string` | **Required** | Schema version. Current: `"1.0"`. |
| `name` | `string` | Required for non-singleton kinds | Unique identifier for the resource. Used as the map key when merging. Duplicate names within the same kind cause a parse error. |

Singleton kinds (`hooks`, `settings`) skip the `name` check. The `project` kind requires `name` (it identifies the project in the registry and graph output).

---

## kind: project

Project-level metadata and child resource references. Exactly one `project` document is expected per merged config. Uses pure YAML format.

> **YAML field collision note**: In a `kind: project` document, the fields `agents`, `skills`, `rules`, `workflows`, and `mcp` are decoded as `[]string` (bare name lists), not as resource definition maps. These lists are advisory only — `ParseDirectory` discovers resources by scanning `xcf/` recursively, not by reading these lists. The separate decode struct (`projectDocFields`) prevents type collision with resource definition kinds; it does not make the lists authoritative.

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
| `backup-dir` | `string` | Optional | Custom directory for `--backup` output. Default: `.<target>_bak_<timestamp>` in project root. |
| `targets` | `[]string` | Optional | Compilation targets (e.g. `["claude", "cursor"]`). |
| `agents` | `[]string` | Optional | Advisory. Bare names of agent resources in `xcf/`. Not used by `ParseDirectory` for resource loading. `xcaffold validate` warns if out of sync with the filesystem. |
| `skills` | `[]string` | Optional | Advisory. Bare names of skill resources in `xcf/`. Not used by `ParseDirectory` for resource loading. `xcaffold validate` warns if out of sync with the filesystem. |
| `rules` | `[]string` | Optional | Advisory. Bare names of rule resources in `xcf/`. Not used by `ParseDirectory` for resource loading. `xcaffold validate` warns if out of sync with the filesystem. |
| `workflows` | `[]string` | Optional | Advisory. Bare names of workflow resources in `xcf/`. Not used by `ParseDirectory` for resource loading. `xcaffold validate` warns if out of sync with the filesystem. |
| `mcp` | `[]string` | Optional | Advisory. Bare names of MCP resources in `xcf/`. Not used by `ParseDirectory` for resource loading. `xcaffold validate` warns if out of sync with the filesystem. |
| `policies` | `[]string` | Optional | Advisory. Bare names of policy resources in `xcf/`. Not used by `ParseDirectory` for resource loading. `xcaffold validate` warns if out of sync with the filesystem. |
| `test` | `TestConfig` | Optional | Configuration for `xcaffold test`. |
| `local` | `SettingsConfig` | Optional | Project-level settings overrides. |

### TestConfig

| Field | Type | Description |
|---|---|---|
| `cli-path` | `string` | Path to the CLI binary (e.g. `claude`, `cursor`). Defaults to `claude` on `$PATH`. |
| `judge-model` | `string` | Generative model for LLM-as-a-Judge evaluation. |

### Example

```yaml
# project.xcf — project manifest (metadata only)
kind: project
version: "1.0"
name: my-api
description: REST API backend
author: team@example.com
repository: https://github.com/example/my-api
targets:
  - claude
  - cursor
test:
  cli-path: claude
  judge-model: claude-haiku-4-5-20251001
```

Resources (`agents`, `skills`, `rules`, etc.) are discovered automatically by scanning the `xcf/` directory. You do not list them in `project.xcf`. The ref list fields (`agents`, `skills`, etc.) remain valid YAML and the parser accepts them — they are treated as advisory documentation and do not change compilation behavior.

---

## kind: agent

Agent persona definition. Each agent is a separate file under `xcf/agents/`, keyed by `name` and compiled to a markdown file under the target's agents directory. Uses frontmatter format.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"agent"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Agent identifier. Used as the map key and output filename. |
| `description` | `string` | Optional | Short description shown in graph output and agent headers. |
| `instructions-file` | `string` | Optional | Path to a file containing instructions. Cannot be combined with a frontmatter body. |
| `model` | `string` | Optional | Model override (e.g. `claude-sonnet-4-20250514`). |
| `effort` | `string` | Optional | Effort level override. |
| `memory` | `string` | Optional | Memory directory path for this agent. |
| `max-turns` | `int` | Optional | Maximum conversation turns. |
| `permission-mode` | `string` | Optional | Permission mode for the agent session. |
| `isolation` | `string` | Optional | Isolation mode. |
| `mode` | `string` | Optional | Agent mode. |
| `when` | `string` | Optional | Condition for when the agent is available. |
| `color` | `string` | Optional | Display color in UI. |
| `initial-prompt` | `string` | Optional | Prompt sent automatically when the agent starts. |
| `readonly` | `*bool` | Optional | When `true`, agent cannot write files. |
| `background` | `*bool` | Optional | When `true`, agent runs in the background. |
| `tools` | `[]string` | Optional | Allowed tool names. |
| `disallowed-tools` | `[]string` | Optional | Explicitly denied tool names. |
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

```
---
kind: agent
version: "1.0"
name: developer
description: Senior backend developer
model: claude-sonnet-4-20250514
tools:
  - Read
  - Edit
  - Bash
  - Write
skills:
  - tdd
rules:
  - testing
max-turns: 25
---
You are a senior backend developer.
Follow TDD. Write tests before implementation.
```

---

## kind: skill

Reusable prompt package. Compiled to a directory with a `SKILL.md` file and optional reference/script/asset subdirectories. Uses frontmatter format.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"skill"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Skill identifier. |
| `description` | `string` | Optional | Short description of the skill's purpose. |
| `instructions-file` | `string` | Optional | Path to a file containing instructions. Cannot be combined with a frontmatter body. |
| `tools` | `[]string` | Optional | Tools this skill requires. |
| `references` | `[]string` | Optional | Doc/data files copied to `skills/<id>/references/` at compile time. |
| `scripts` | `[]string` | Optional | Executable helpers copied to `skills/<id>/scripts/` at compile time. |
| `assets` | `[]string` | Optional | Output artifacts (templates, fonts, icons) copied to `skills/<id>/assets/`. |

### Example

```
---
kind: skill
version: "1.0"
name: tdd
description: Test-driven development workflow
allowed-tools:
  - Bash
  - Edit
  - Read
---
Follow the Red-Green-Refactor cycle.
1. Write a failing test.
2. Write minimal code to pass.
3. Refactor.
```

---

## kind: rule

Path-gated formatting guideline. Compiled to a markdown file under the target's rules directory. Uses frontmatter format.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"rule"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Rule identifier. |
| `description` | `string` | Optional | Short description of the rule. |
| `instructions-file` | `string` | Optional | Path to a file containing instructions. Cannot be combined with a frontmatter body. |
| `always-apply` | `*bool` | Optional | When `true`, rule applies to all files regardless of `paths`. When omitted, inherits platform default. |
| `paths` | `[]string` | Optional | Glob patterns restricting which files this rule applies to. |

### Example

```
---
kind: rule
version: "1.0"
name: testing
description: Go testing conventions
always-apply: false
paths:
  - "**/*_test.go"
---
Use table-driven tests. Name tests TestFunc_Scenario.
Always add negative test cases.
```

---

## kind: workflow

Named reusable workflow. Compiled for targets that support workflows (Antigravity). Claude Code lacks standalone workflows. Cursor encodes workflows via Rules. GitHub Copilot and Gemini CLI have no native workflow support. Uses frontmatter format.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"workflow"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Workflow identifier. |
| `description` | `string` | Optional | Short description. |
| `instructions-file` | `string` | Optional | Path to a file containing instructions. Cannot be combined with a frontmatter body. |

### Example

```
---
kind: workflow
version: "1.0"
name: deploy
description: Production deployment checklist
---
1. Run full test suite.
2. Build release binary.
3. Tag version.
4. Push to registry.
```

---

## kind: mcp

MCP (Model Context Protocol) server definition. Compiled into the `mcpServers` key of `settings.json`. Uses pure YAML format.

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

Lifecycle event handlers. Singleton: no `name` field. Hook events are wrapped under an `events:` key because `HookConfig` is a map type that cannot be inlined. Uses pure YAML format.

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

Platform settings compiled to `settings.json`. Singleton: no `name` field. Uses pure YAML format.

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

## kind: global

Global-scope configuration for `~/.xcaffold/global.xcf`. Contains shared resources
(agents, skills, rules, workflows, MCP, hooks, policies) and settings that apply
across all projects. Does not contain project metadata. Uses pure YAML format.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"global"` |
| `version` | `string` | **Required** | Schema version. |
| `extends` | `string` | Optional | Path to parent config for inheritance. |
| `settings` | `SettingsConfig` | Optional | Global settings. |
| `agents` | `map[string]AgentConfig` | Optional | Inline global agent definitions. |
| `skills` | `map[string]SkillConfig` | Optional | Inline global skill definitions. |
| `rules` | `map[string]RuleConfig` | Optional | Inline global rule definitions. |
| `workflows` | `map[string]WorkflowConfig` | Optional | Inline global workflow definitions. |
| `mcp` | `map[string]MCPConfig` | Optional | Inline global MCP definitions. |
| `hooks` | `HookConfig` | Optional | Global lifecycle hooks. |
| `policies` | `map[string]PolicyConfig` | Optional | Inline global policy definitions. |

---

## kind: policy

Declarative constraint evaluated against the AST and compiled output. Uses pure YAML format.

| Field | Type | Required | Description |
|---|---|---|---|
| `kind` | `string` | **Required** | `"policy"` |
| `version` | `string` | **Required** | Schema version. |
| `name` | `string` | **Required** | Unique policy identifier. |
| `description` | `string` | Optional | Human-readable explanation. |
| `severity` | `string` | **Required** | `"error"`, `"warning"`, or `"off"`. |
| `target` | `string` | **Required** | `"agent"`, `"skill"`, `"rule"`, `"hook"`, `"settings"`, or `"output"`. |
| `match` | `PolicyMatch` | Optional | Filter conditions (AND-ed). |
| `require` | `[]PolicyRequire` | Optional | Field value constraints. |
| `deny` | `[]PolicyDeny` | Optional | Forbidden content/path patterns. |

---

## Directory Layout

A typical project uses this layout:

```
my-project/
  project.xcf                    # kind: project (pure YAML)
  xcf/
    agents/
      developer.xcf              # kind: agent (frontmatter)
      reviewer.xcf               # kind: agent (frontmatter)
    rules/
      conventions.xcf            # kind: rule (frontmatter)
    skills/
      tdd.xcf                    # kind: skill (frontmatter)
    settings.xcf                 # kind: settings (pure YAML)
    policies/
      safety.xcf                 # kind: policy (pure YAML)
```

`ParseDirectory` scans `xcf/` recursively, parses each file by `kind`, and merges the results into a single `XcaffoldConfig` before compilation.

### Merge Behavior

`ParseDirectory` scans a directory for all `*.xcf` files, parses each independently, and merges the results. Merge rules:

- **Strict dedup**: Duplicate resource IDs within the same kind cause a parse error (`duplicate agent ID "foo"`).
- **Singleton merge**: A second `project`, `hooks`, or `settings` document overwrites the first. For `hooks`, events are appended (not replaced) per event key.
- **Version propagation**: If a resource document sets `version` and the merged config has no version yet, the resource's version is propagated.

### Project scope routing

`ParseDirectory` discovers resources by recursively scanning `xcf/` for all `*.xcf` files and routing each by `kind:`. The `agents`, `skills`, `rules`, `workflows`, and `mcp` lists in a `kind: project` document are advisory; they do not filter or gate resource loading. `xcaffold validate` emits warnings when these lists diverge from the actual filesystem contents, helping teams keep documentation current.
