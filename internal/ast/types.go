package ast

// ResourceScope contains all agentic primitives that can appear at both
// global scope (root of XcaffoldConfig) and workspace scope (inside ProjectConfig).
// Embedded with yaml:",inline" so fields appear at the same YAML level as the parent.
type ResourceScope struct {
	Agents    map[string]AgentConfig    `yaml:"agents,omitempty"`
	Skills    map[string]SkillConfig    `yaml:"skills,omitempty"`
	Rules     map[string]RuleConfig     `yaml:"rules,omitempty"`
	Hooks     HookConfig                `yaml:"hooks,omitempty"`
	MCP       map[string]MCPConfig      `yaml:"mcp,omitempty"`
	Workflows map[string]WorkflowConfig `yaml:"workflows,omitempty"`
	Policies  map[string]PolicyConfig   `yaml:"policies,omitempty"`
	Memory    map[string]MemoryConfig   `yaml:"memory,omitempty"` // NEW
}

// XcaffoldConfig is the root structure of a parsed .xcf YAML file.
type XcaffoldConfig struct {
	Kind    string `yaml:"-"` // Set by parser routing, not decoded from YAML
	Version string `yaml:"version"`
	Extends string `yaml:"extends,omitempty"`

	Settings SettingsConfig `yaml:"settings,omitempty"`

	ResourceScope `yaml:",inline"` // Global-level resources

	Project *ProjectConfig `yaml:"project,omitempty"` // nil for global configs
}

// ProjectConfig holds project-level metadata and workspace-scoped resources.
type ProjectConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Version     string `yaml:"version,omitempty"`
	Author      string `yaml:"author,omitempty"`
	Homepage    string `yaml:"homepage,omitempty"`
	Repository  string `yaml:"repository,omitempty"`
	License     string `yaml:"license,omitempty"`
	BackupDir   string `yaml:"backup-dir,omitempty"`

	// Targets lists the compilation targets (e.g. "claude", "antigravity").
	// Populated by the parser when decoding kind: project documents.
	Targets []string `yaml:"-"`

	// Reference lists: bare names linking to child resources in xcf/ subdirectories.
	// Populated by the parser when decoding kind: project documents.
	AgentRefs    []string `yaml:"-"`
	SkillRefs    []string `yaml:"-"`
	RuleRefs     []string `yaml:"-"`
	WorkflowRefs []string `yaml:"-"`
	MCPRefs      []string `yaml:"-"`
	PolicyRefs   []string `yaml:"-"`

	Test  TestConfig     `yaml:"test,omitempty"`
	Local SettingsConfig `yaml:"local,omitempty"`

	// Instructions fields — Group A: Root instructions.
	// Instructions and InstructionsFile are mutually exclusive.
	Instructions     string `yaml:"instructions,omitempty"`
	InstructionsFile string `yaml:"instructions-file,omitempty"`

	// InstructionsImports lists @-import targets preserved verbatim for providers
	// that support them (Claude, Gemini). Emitted as-is into the rendered output.
	InstructionsImports []string `yaml:"instructions-imports,omitempty"`

	// InstructionsScopes defines per-directory nested instruction files.
	// Order in this slice is authoritative (depth ascending, then alphabetical).
	InstructionsScopes []InstructionsScope `yaml:"instructions-scopes,omitempty"`

	// TargetOptions holds per-provider compile-time options for the project.
	// Keys are provider names (e.g. "copilot", "cursor"). Values are TargetOverride
	// instances. Only fields relevant to the named provider are examined.
	TargetOptions map[string]TargetOverride `yaml:"target-options,omitempty"`

	ResourceScope `yaml:",inline"` // Workspace-level resources
}

// InstructionsScope defines instructions for a specific directory path within the project.
type InstructionsScope struct {
	// Path is the directory this scope applies to, relative to the project root.
	// Required. Duplicate paths are a parse error.
	Path string `yaml:"path"`

	// Instructions and InstructionsFile are mutually exclusive.
	Instructions     string `yaml:"instructions,omitempty"`
	InstructionsFile string `yaml:"instructions-file,omitempty"`

	// MergeStrategy is load-bearing: preserves runtime nesting semantic across round-trips.
	// Valid values: concat | closest-wins | flat. Defaults to "concat" if omitted.
	MergeStrategy string `yaml:"merge-strategy,omitempty"`

	// SourceProvider and SourceFilename are provenance metadata only.
	// xcaffold never reads these fields after import.
	SourceProvider string `yaml:"source-provider,omitempty"`
	SourceFilename string `yaml:"source-filename,omitempty"`

	// Variants holds divergent content for the same path across providers.
	Variants map[string]InstructionsVariant `yaml:"variants,omitempty"`

	// Reconciliation records the strategy and state for divergent variants.
	Reconciliation *ReconciliationConfig `yaml:"reconciliation,omitempty"`

	// Inherited is set by the parser when this scope originates from an
	// extends: global base config. It is never serialized and causes StripInherited
	// to remove it from the local project during compilation.
	Inherited bool `yaml:"-"`
}

// InstructionsVariant holds the per-provider sidecar path when two providers
// have divergent content for the same scope path.
type InstructionsVariant struct {
	InstructionsFile string `yaml:"instructions-file,omitempty"`
	SourceFilename   string `yaml:"source-filename,omitempty"`
}

// ReconciliationConfig records the strategy and state for divergent variants.
type ReconciliationConfig struct {
	// Strategy: per-target | union | manual
	Strategy string `yaml:"strategy,omitempty"`
	// LastReconciled is an RFC3339 timestamp set by the importer.
	LastReconciled string `yaml:"last-reconciled,omitempty"`
	// Notes is a human-readable explanation set by the importer.
	Notes string `yaml:"notes,omitempty"`
}

// AgentConfig defines an AI coding agent persona.
//
// Field ordering is canonical and mirrors the compiled markdown frontmatter:
//  1. Identity (name, description)
//  2. Model & Execution (model, effort, maxTurns, mode)
//  3. Tool Access (tools, disallowedTools, readonly)
//  4. Permissions & Invocation (permissionMode, disableModelInvocation, userInvocable)
//  5. Lifecycle (background, isolation, when)
//  6. Memory & Context (memory, color, initialPrompt)
//  7. Composition references (skills, rules, mcp, assertions)
//  8. Inline composition (mcpServers, hooks)
//  9. Multi-Target (targets)
//  10. Instructions (always last)
type AgentConfig struct {
	// Group 1: Identity
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`

	// Group 2: Model & Execution
	Model    string `yaml:"model,omitempty"`
	Effort   string `yaml:"effort,omitempty"`
	MaxTurns int    `yaml:"max-turns,omitempty"`
	Mode     string `yaml:"mode,omitempty"`

	// Group 3: Tool Access
	Tools           []string `yaml:"tools,omitempty"`
	DisallowedTools []string `yaml:"disallowed-tools,omitempty"`
	Readonly        *bool    `yaml:"readonly,omitempty"`

	// Group 4: Permissions & Invocation
	PermissionMode         string `yaml:"permission-mode,omitempty"`
	DisableModelInvocation *bool  `yaml:"disable-model-invocation,omitempty"`
	UserInvocable          *bool  `yaml:"user-invocable,omitempty"`

	// Group 5: Lifecycle
	Background *bool  `yaml:"background,omitempty"`
	Isolation  string `yaml:"isolation,omitempty"`
	When       string `yaml:"when,omitempty"`

	// Group 6: Memory & Context
	Memory        string `yaml:"memory,omitempty"`
	Color         string `yaml:"color,omitempty"`
	InitialPrompt string `yaml:"initial-prompt,omitempty"`

	// Group 7: Composition references
	Skills     []string `yaml:"skills,omitempty"`
	Rules      []string `yaml:"rules,omitempty"`
	MCP        []string `yaml:"mcp,omitempty"`
	Assertions []string `yaml:"assertions,omitempty"`

	// Group 8: Inline composition
	MCPServers map[string]MCPConfig `yaml:"mcp-servers,omitempty"`
	Hooks      HookConfig           `yaml:"hooks,omitempty"`

	// Group 9: Multi-Target
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Group 10: Instructions (always last)
	Instructions     string `yaml:"instructions,omitempty"`
	InstructionsFile string `yaml:"instructions-file,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized and causes renderers
	// to skip the resource during project-scope compilation.
	Inherited bool `yaml:"-"`
}

// TargetOverride specifies overrides for multi-provider targets.
type TargetOverride struct {
	Hooks                    map[string]string `yaml:"hooks,omitempty"`
	SuppressFidelityWarnings *bool             `yaml:"suppress-fidelity-warnings,omitempty"`
	SkipSynthesis            *bool             `yaml:"skip-synthesis,omitempty"`
	InstructionsOverride     string            `yaml:"instructions-override,omitempty"`
	Provider                 map[string]any    `yaml:"provider,omitempty"`
	// InstructionsMode controls how project instructions-scopes are emitted.
	// Valid values: flat (default) | nested. Only used by the Copilot renderer.
	// flat: all scopes merged into a single .github/copilot-instructions.md file.
	// nested: scopes emitted as per-directory AGENTS.md files (closest-wins class).
	InstructionsMode string `yaml:"instructions-mode,omitempty"`
}

// SkillConfig defines a reusable prompt package.
//
// Field ordering follows the canonical 6-group structure from
// docs/reference/schema.md:
//
//	Group 1 — Identity (name, description, when-to-use, license)
//	Group 3 — Tool Access (allowed-tools)
//	Group 4 — Permissions & Invocation Control (disable-model-invocation, user-invocable, argument-hint)
//	Group 7 — Composition / Supporting Files (references, scripts, assets)
//	Group 9 — Multi-Target (targets — per-provider overrides and provider: pass-through)
//	Group 10 — Instructions (instructions, instructions_file) — ALWAYS last
type SkillConfig struct {
	// Group 1 — Identity
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`
	WhenToUse   string `yaml:"when-to-use,omitempty"`
	License     string `yaml:"license,omitempty"`

	// Group 3 — Tool Access
	AllowedTools []string `yaml:"allowed-tools,omitempty"`

	// Group 4 — Permissions & Invocation Control
	DisableModelInvocation *bool  `yaml:"disable-model-invocation,omitempty"`
	UserInvocable          *bool  `yaml:"user-invocable,omitempty"`
	ArgumentHint           string `yaml:"argument-hint,omitempty"`

	// Group 7 — Composition / Supporting Files (agentskills.io folder convention)
	// References are docs/data files copied to skills/<id>/references/ at compile time.
	References []string `yaml:"references,omitempty"`
	// Scripts are executable helper files copied to skills/<id>/scripts/ at compile time.
	Scripts []string `yaml:"scripts,omitempty"`
	// Assets are output artifact files (templates, fonts, icons) copied to skills/<id>/assets/.
	Assets []string `yaml:"assets,omitempty"`

	// Group 9 — Multi-Target (per-provider overrides + provider: pass-through)
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Group 10 — Instructions (mutually exclusive — enforced by parser)
	Instructions     string `yaml:"instructions,omitempty"`
	InstructionsFile string `yaml:"instructions-file,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`
}

// Rule activation mode values. These are the canonical cross-provider activation
// modes for the rule kind. Renderers map these to provider-native expressions.
const (
	RuleActivationAlways         = "always"
	RuleActivationPathGlob       = "path-glob"
	RuleActivationModelDecided   = "model-decided"
	RuleActivationManualMention  = "manual-mention"
	RuleActivationExplicitInvoke = "explicit-invoke"
)

// RuleConfig defines a path-gated formatting guideline.
type RuleConfig struct {
	AlwaysApply      *bool    `yaml:"always-apply,omitempty"`
	Description      string   `yaml:"description,omitempty"`
	Activation       string   `yaml:"activation,omitempty"`
	Name             string   `yaml:"name,omitempty"`
	Instructions     string   `yaml:"instructions,omitempty"`
	InstructionsFile string   `yaml:"instructions-file,omitempty"`
	Paths            []string `yaml:"paths,omitempty"`
	// ExcludeAgents is a Copilot-specific list of agent types that should NOT
	// receive this rule. Valid values: code-review | cloud-agent.
	// Silently ignored by all non-Copilot renderers.
	ExcludeAgents []string `yaml:"exclude-agents,omitempty"`
	// Targets holds per-provider overrides including provider-native pass-through fields.
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`
	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`
}

// HookConfig maps lifecycle event names to their matcher groups.
// Event names: PreToolUse, PostToolUse, Notification, Stop,
// SubagentStop, InstructionsLoaded, PreCompact, SessionStart, ConfigChange.
type HookConfig map[string][]HookMatcherGroup

// HookMatcherGroup defines a matcher pattern and its associated hook handlers.
type HookMatcherGroup struct {
	Matcher string        `yaml:"matcher,omitempty" json:"matcher,omitempty"`
	Hooks   []HookHandler `yaml:"hooks"             json:"hooks"`
}

// HookHandler defines a single executable hook action.
type HookHandler struct {
	Async          *bool             `yaml:"async,omitempty"            json:"async,omitempty"`
	Headers        map[string]string `yaml:"headers,omitempty"          json:"headers,omitempty"`
	Timeout        *int              `yaml:"timeout,omitempty"          json:"timeout,omitempty"`
	Once           *bool             `yaml:"once,omitempty"             json:"once,omitempty"`
	Command        string            `yaml:"command,omitempty"          json:"command,omitempty"`
	URL            string            `yaml:"url,omitempty"              json:"url,omitempty"`
	Prompt         string            `yaml:"prompt,omitempty"           json:"prompt,omitempty"`
	Model          string            `yaml:"model,omitempty"            json:"model,omitempty"`
	If             string            `yaml:"if,omitempty"               json:"if,omitempty"`
	Type           string            `yaml:"type"                       json:"type"`
	Shell          string            `yaml:"shell,omitempty"            json:"shell,omitempty"`
	StatusMessage  string            `yaml:"status-message,omitempty"    json:"statusMessage,omitempty"`
	AllowedEnvVars []string          `yaml:"allowed-env-vars,omitempty"   json:"allowedEnvVars,omitempty"`
}

// MCPConfig defines a local or remote MCP server context.
type MCPConfig struct {
	Env              map[string]string `yaml:"env,omitempty"     json:"env,omitempty"`
	Headers          map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Disabled         *bool             `yaml:"disabled,omitempty"         json:"disabled,omitempty"`
	OAuth            map[string]string `yaml:"oauth,omitempty"            json:"oauth,omitempty"`
	Name             string            `yaml:"name,omitempty"             json:"name,omitempty"`
	Type             string            `yaml:"type,omitempty"    json:"type,omitempty"`
	Command          string            `yaml:"command,omitempty" json:"command,omitempty"`
	URL              string            `yaml:"url,omitempty"     json:"url,omitempty"`
	Cwd              string            `yaml:"cwd,omitempty"              json:"cwd,omitempty"`
	AuthProviderType string            `yaml:"authProviderType,omitempty" json:"authProviderType,omitempty"`
	Args             []string          `yaml:"args,omitempty"    json:"args,omitempty"`
	DisabledTools    []string          `yaml:"disabledTools,omitempty"    json:"disabledTools,omitempty"`
	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-" json:"-"`
}

// StatusLineConfig defines the statusLine setting for the platform.
// The original format is {"type": "command", "command": "<shell cmd>"}.
type StatusLineConfig struct {
	Type    string `yaml:"type,omitempty"    json:"type,omitempty"`
	Command string `yaml:"command,omitempty" json:"command,omitempty"`
}

// PermissionsConfig defines strongly-typed permission rules.
// Each field is a list of permission rule strings (e.g. "Bash(npm test *)").
type PermissionsConfig struct {
	Allow                        []string `yaml:"allow,omitempty"                       json:"allow,omitempty"`
	Deny                         []string `yaml:"deny,omitempty"                        json:"deny,omitempty"`
	Ask                          []string `yaml:"ask,omitempty"                         json:"ask,omitempty"`
	DefaultMode                  string   `yaml:"defaultMode,omitempty"                 json:"defaultMode,omitempty"`
	AdditionalDirectories        []string `yaml:"additionalDirectories,omitempty"       json:"additionalDirectories,omitempty"`
	DisableBypassPermissionsMode string   `yaml:"disableBypassPermissionsMode,omitempty" json:"disableBypassPermissionsMode,omitempty"`
}

// SandboxConfig configures OS-level process isolation for Bash commands.
type SandboxConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"                    json:"enabled,omitempty"`
	// AutoAllowBashIfSandboxed auto-approves bash commands when sandboxed, without prompting.
	// Named autoAllowBashIfSandboxed in Claude Code's settings.json.
	AutoAllowBashIfSandboxed *bool              `yaml:"autoAllowBashIfSandboxed,omitempty"   json:"autoAllowBashIfSandboxed,omitempty"`
	FailIfUnavailable        *bool              `yaml:"failIfUnavailable,omitempty"          json:"failIfUnavailable,omitempty"`
	AllowUnsandboxedCommands *bool              `yaml:"allowUnsandboxedCommands,omitempty"   json:"allowUnsandboxedCommands,omitempty"`
	Filesystem               *SandboxFilesystem `yaml:"filesystem,omitempty"                 json:"filesystem,omitempty"`
	Network                  *SandboxNetwork    `yaml:"network,omitempty"                    json:"network,omitempty"`
	ExcludedCommands         []string           `yaml:"excludedCommands,omitempty"           json:"excludedCommands,omitempty"`
}

// SandboxFilesystem configures filesystem isolation boundaries.
type SandboxFilesystem struct {
	AllowWrite []string `yaml:"allowWrite,omitempty" json:"allowWrite,omitempty"`
	DenyWrite  []string `yaml:"denyWrite,omitempty"  json:"denyWrite,omitempty"`
	AllowRead  []string `yaml:"allowRead,omitempty"  json:"allowRead,omitempty"`
	DenyRead   []string `yaml:"denyRead,omitempty"   json:"denyRead,omitempty"`
}

// SandboxNetwork configures network isolation boundaries.
type SandboxNetwork struct {
	HTTPProxyPort           *int  `yaml:"httpProxyPort,omitempty"           json:"httpProxyPort,omitempty"`
	SOCKSProxyPort          *int  `yaml:"socksProxyPort,omitempty"          json:"socksProxyPort,omitempty"`
	AllowManagedDomainsOnly *bool `yaml:"allowManagedDomainsOnly,omitempty" json:"allowManagedDomainsOnly,omitempty"`
	// AllowUnixSockets is a list of specific Unix domain socket paths permitted for
	// outbound connections. Use an empty list to deny all, or ["*"] to allow all.
	AllowUnixSockets []string `yaml:"allowUnixSockets,omitempty"        json:"allowUnixSockets,omitempty"`
	// AllowLocalBinding permits the sandboxed process to bind to localhost ports.
	AllowLocalBinding *bool    `yaml:"allowLocalBinding,omitempty"       json:"allowLocalBinding,omitempty"`
	AllowedDomains    []string `yaml:"allowedDomains,omitempty"          json:"allowedDomains,omitempty"`
}

// SettingsConfig represents the full platform settings.json structure.
// The mcp: block at the top level of XcaffoldConfig is a convenience shorthand
// that gets merged into the mcpServers key during compilation. Fields defined
// here take precedence over the shorthand for mcpServers if both are set.
type SettingsConfig struct {
	Agent                             any                  `yaml:"agent,omitempty" json:"agent,omitempty"`
	Worktree                          any                  `yaml:"worktree,omitempty" json:"worktree,omitempty"`
	AutoMode                          any                  `yaml:"autoMode,omitempty" json:"autoMode,omitempty"`
	CleanupPeriodDays                 *int                 `yaml:"cleanupPeriodDays,omitempty" json:"cleanupPeriodDays,omitempty"`
	IncludeGitInstructions            *bool                `yaml:"includeGitInstructions,omitempty" json:"includeGitInstructions,omitempty"`
	SkipDangerousModePermissionPrompt *bool                `yaml:"skipDangerousModePermissionPrompt,omitempty" json:"skipDangerousModePermissionPrompt,omitempty"`
	Permissions                       *PermissionsConfig   `yaml:"permissions,omitempty" json:"permissions,omitempty"`
	Sandbox                           *SandboxConfig       `yaml:"sandbox,omitempty" json:"sandbox,omitempty"`
	AutoMemoryEnabled                 *bool                `yaml:"autoMemoryEnabled,omitempty" json:"autoMemoryEnabled,omitempty"`
	DisableAllHooks                   *bool                `yaml:"disableAllHooks,omitempty" json:"disableAllHooks,omitempty"`
	Attribution                       *bool                `yaml:"attribution,omitempty" json:"attribution,omitempty"`
	MCPServers                        map[string]MCPConfig `yaml:"mcpServers,omitempty" json:"mcpServers,omitempty"`
	Hooks                             HookConfig           `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	StatusLine                        *StatusLineConfig    `yaml:"statusLine,omitempty" json:"statusLine,omitempty"`
	RespectGitignore                  *bool                `yaml:"respectGitignore,omitempty" json:"respectGitignore,omitempty"`
	Env                               map[string]string    `yaml:"env,omitempty" json:"env,omitempty"`
	EnabledPlugins                    map[string]bool      `yaml:"enabledPlugins,omitempty" json:"enabledPlugins,omitempty"`
	DisableSkillShellExecution        *bool                `yaml:"disableSkillShellExecution,omitempty" json:"disableSkillShellExecution,omitempty"`
	AlwaysThinkingEnabled             *bool                `yaml:"alwaysThinkingEnabled,omitempty" json:"alwaysThinkingEnabled,omitempty"`
	EffortLevel                       string               `yaml:"effortLevel,omitempty" json:"effortLevel,omitempty"`
	DefaultShell                      string               `yaml:"defaultShell,omitempty" json:"defaultShell,omitempty"`
	Language                          string               `yaml:"language,omitempty" json:"language,omitempty"`
	OutputStyle                       string               `yaml:"outputStyle,omitempty" json:"outputStyle,omitempty"`
	PlansDirectory                    string               `yaml:"plansDirectory,omitempty" json:"plansDirectory,omitempty"`
	Model                             string               `yaml:"model,omitempty" json:"model,omitempty"`
	OtelHeadersHelper                 string               `yaml:"otelHeadersHelper,omitempty" json:"otelHeadersHelper,omitempty"`
	AutoMemoryDirectory               string               `yaml:"autoMemoryDirectory,omitempty" json:"autoMemoryDirectory,omitempty"`
	AvailableModels                   []string             `yaml:"availableModels,omitempty" json:"availableModels,omitempty"`
	ClaudeMdExcludes                  []string             `yaml:"claudeMdExcludes,omitempty" json:"claudeMdExcludes,omitempty"`
}

// TestConfig holds project-level configuration for `xcaffold test`.
type TestConfig struct {
	// CliPath is the path to the CLI binary (e.g., claude, cursor). Defaults to "claude" on $PATH.
	CliPath string `yaml:"cli-path,omitempty"`

	// ClaudePath is deprecated in favor of CliPath but retained for backward compatibility.
	ClaudePath string `yaml:"claude-path,omitempty"`

	// JudgeModel is the generative model used for LLM-as-a-Judge evaluation.
	JudgeModel string `yaml:"judge-model,omitempty"`

	// Task is the user prompt sent to the agent during simulation.
	// Defaults to a generic capability discovery prompt when empty.
	Task string `yaml:"task,omitempty"`

	// MaxTurns caps the number of simulated conversation turns.
	// Reserved for future multi-turn support; currently unused beyond recording.
	MaxTurns int `yaml:"max-turns,omitempty"`
}

// WorkflowConfig defines a named, reusable, multi-step procedure.
// Each workflow maps to an entry under the `workflows:` key in scaffold.xcf.
// api-version: workflow/v1 is the current stable shape; workflow/v2 will add
// parameterized steps and DAG ordering without breaking v1 schemas.
type WorkflowConfig struct {
	// api-version discriminates the schema shape. Default: "workflow/v1".
	ApiVersion string `yaml:"api-version,omitempty"`

	// Identity
	Name        string `yaml:"name,omitempty"`
	Description string `yaml:"description,omitempty"`

	// Steps is the ordered procedural body.
	// Mutually exclusive with Instructions / InstructionsFile at the top level
	// (top-level body is permitted only for single-step legacy configs).
	Steps []WorkflowStep `yaml:"steps,omitempty"`

	// Targets holds per-provider overrides and lowering-strategy directives.
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Instructions and InstructionsFile are the top-level body for single-step
	// or legacy workflows. Mutually exclusive with each other; deprecated in
	// favor of Steps when more than one step is needed.
	Instructions     string `yaml:"instructions,omitempty"`
	InstructionsFile string `yaml:"instructions-file,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. Never serialized.
	Inherited bool `yaml:"-"`
}

// WorkflowStep is one named step in a workflow's ordered body.
type WorkflowStep struct {
	Name             string `yaml:"name"`
	Description      string `yaml:"description,omitempty"`
	Instructions     string `yaml:"instructions,omitempty"`
	InstructionsFile string `yaml:"instructions-file,omitempty"`
}

// PolicyConfig defines a declarative constraint evaluated against the AST
// and compiled output during apply and validate.
type PolicyConfig struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description,omitempty"`
	Severity    string          `yaml:"severity"`
	Target      string          `yaml:"target"`
	Match       *PolicyMatch    `yaml:"match,omitempty"`
	Require     []PolicyRequire `yaml:"require,omitempty"`
	Deny        []PolicyDeny    `yaml:"deny,omitempty"`
}

// PolicyMatch filters which resources a policy evaluates. All conditions
// are AND-ed. An empty or nil PolicyMatch matches all resources.
type PolicyMatch struct {
	HasTool        string `yaml:"has-tool,omitempty"`
	HasField       string `yaml:"has-field,omitempty"`
	NameMatches    string `yaml:"name-matches,omitempty"`
	TargetIncludes string `yaml:"target-includes,omitempty"`
}

// PolicyRequire defines a field constraint on a matched resource.
type PolicyRequire struct {
	Field     string   `yaml:"field"`
	IsPresent *bool    `yaml:"is-present,omitempty"`
	MinLength *int     `yaml:"min-length,omitempty"`
	MaxCount  *int     `yaml:"max-count,omitempty"`
	OneOf     []string `yaml:"one-of,omitempty"`
}

// PolicyDeny defines forbidden content or path patterns in compiled output.
type PolicyDeny struct {
	ContentContains []string `yaml:"content-contains,omitempty"`
	ContentMatches  string   `yaml:"content-matches,omitempty"`
	PathContains    string   `yaml:"path-contains,omitempty"`
}

// MemoryConfig defines a named memory entry that is snapshot-imported from a
// provider's agent-written memory store and seeded into a target provider on apply.
// Field ordering mirrors the MemoryConfig canonical group ordering:
//
//  1. Identity (name, type, description)
//  2. Lifecycle (lifecycle)
//  3. Multi-Target (targets)
//  4. Body (instructions, instructions-file)
type MemoryConfig struct {
	// Group 1: Identity
	Name        string `yaml:"name,omitempty"`
	Type        string `yaml:"type,omitempty"` // user | feedback | project | reference
	Description string `yaml:"description,omitempty"`

	// Group 2: Lifecycle
	Lifecycle string `yaml:"lifecycle,omitempty"` // seed-once (default) | tracked

	// Group 3: Multi-Target
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Group 4: Body (mutually exclusive)
	Instructions     string `yaml:"instructions,omitempty"`
	InstructionsFile string `yaml:"instructions-file,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`
}

// StripInherited removes all top-level resources that are marked as Inherited=true.
// This is called before compilation to ensure that resources loaded from
// extends: global are not physically generated into local project directories.
// It modifies the XcaffoldConfig in place.
func (c *XcaffoldConfig) StripInherited() {
	for k, v := range c.Agents {
		if v.Inherited {
			delete(c.Agents, k)
		}
	}
	for k, v := range c.Skills {
		if v.Inherited {
			delete(c.Skills, k)
		}
	}
	for k, v := range c.Rules {
		if v.Inherited {
			delete(c.Rules, k)
		}
	}
	for k, v := range c.MCP {
		if v.Inherited {
			delete(c.MCP, k)
		}
	}
	for k, v := range c.Workflows {
		if v.Inherited {
			delete(c.Workflows, k)
		}
	}
	for k, m := range c.Memory {
		if m.Inherited {
			delete(c.Memory, k)
		}
	}
	if c.Project != nil {
		filtered := c.Project.InstructionsScopes[:0]
		for _, scope := range c.Project.InstructionsScopes {
			if !scope.Inherited {
				filtered = append(filtered, scope)
			}
		}
		c.Project.InstructionsScopes = filtered
	}
}
