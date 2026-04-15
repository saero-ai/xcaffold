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

	ResourceScope `yaml:",inline"` // Workspace-level resources
}

// AgentConfig defines an AI coding agent persona.
//
// Field ordering is canonical and mirrors the compiled markdown frontmatter:
//  1. Identity (name, description)
//  2. Model & Execution (model, effort, max-turns, mode)
//  3. Tool Access (tools, disallowed-tools, readonly)
//  4. Permissions & Invocation (permission-mode, disable-model-invocation, user-invocable)
//  5. Lifecycle (background, isolation, when)
//  6. Memory & Context (memory, color, initial-prompt)
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
}

// SkillConfig defines a reusable prompt package.
type SkillConfig struct {
	InstructionsFile string `yaml:"instructions-file,omitempty"`
	Instructions     string `yaml:"instructions,omitempty"`
	Description      string `yaml:"description,omitempty"`
	Name             string `yaml:"name,omitempty"`
	// References are docs/data files copied to skills/<id>/references/ at compile time.
	References []string `yaml:"references,omitempty"`
	// Scripts are executable helper files copied to skills/<id>/scripts/ at compile time.
	// Use for reusable code that skill invocations would otherwise re-implement each run.
	Scripts []string `yaml:"scripts,omitempty"`
	// Assets are output artifact files (templates, fonts, icons) copied to skills/<id>/assets/.
	Assets []string `yaml:"assets,omitempty"`
	Tools  []string `yaml:"tools,omitempty"`
	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`
}

// RuleConfig defines a path-gated formatting guideline.
type RuleConfig struct {
	AlwaysApply      *bool    `yaml:"always-apply,omitempty"`
	Description      string   `yaml:"description,omitempty"`
	Name             string   `yaml:"name,omitempty"`
	Instructions     string   `yaml:"instructions,omitempty"`
	InstructionsFile string   `yaml:"instructions-file,omitempty"`
	Paths            []string `yaml:"paths,omitempty"`
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
	StatusMessage  string            `yaml:"statusMessage,omitempty"    json:"statusMessage,omitempty"`
	AllowedEnvVars []string          `yaml:"allowedEnvVars,omitempty"   json:"allowedEnvVars,omitempty"`
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

// WorkflowConfig defines a named, reusable workflow (Antigravity Workflow).
// Each workflow maps to an entry under the `workflows:` key in scaffold.xcf.
type WorkflowConfig struct {
	Name             string `yaml:"name,omitempty"`
	Description      string `yaml:"description,omitempty"`
	Instructions     string `yaml:"instructions,omitempty"`
	InstructionsFile string `yaml:"instructions-file,omitempty"`
	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`
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
}
