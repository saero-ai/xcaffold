package ast

import "gopkg.in/yaml.v3"

const (
	ActivationModeAlways = "always"
	ActivationModePaths  = "paths"
)

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
	DefaultMode                  string   `yaml:"default-mode,omitempty"                 json:"defaultMode,omitempty"`
	AdditionalDirectories        []string `yaml:"additional-directories,omitempty"       json:"additionalDirectories,omitempty"`
	DisableBypassPermissionsMode string   `yaml:"disable-bypass-permissions-mode,omitempty" json:"disableBypassPermissionsMode,omitempty"`
}

// SandboxConfig configures OS-level process isolation for Bash commands.
type SandboxConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"                    json:"enabled,omitempty"`
	// AutoAllowBashIfSandboxed auto-approves bash commands when sandboxed, without prompting.
	// Named autoAllowBashIfSandboxed in Claude Code's settings.json.
	AutoAllowBashIfSandboxed *bool              `yaml:"auto-allow-bash-if-sandboxed,omitempty"   json:"autoAllowBashIfSandboxed,omitempty"`
	FailIfUnavailable        *bool              `yaml:"fail-if-unavailable,omitempty"          json:"failIfUnavailable,omitempty"`
	AllowUnsandboxedCommands *bool              `yaml:"allow-unsandboxed-commands,omitempty"   json:"allowUnsandboxedCommands,omitempty"`
	Filesystem               *SandboxFilesystem `yaml:"filesystem,omitempty"                 json:"filesystem,omitempty"`
	Network                  *SandboxNetwork    `yaml:"network,omitempty"                    json:"network,omitempty"`
	ExcludedCommands         []string           `yaml:"excluded-commands,omitempty"           json:"excludedCommands,omitempty"`
}

// SandboxFilesystem configures filesystem isolation boundaries.
type SandboxFilesystem struct {
	AllowWrite []string `yaml:"allow-write,omitempty" json:"allowWrite,omitempty"`
	DenyWrite  []string `yaml:"deny-write,omitempty"  json:"denyWrite,omitempty"`
	AllowRead  []string `yaml:"allow-read,omitempty"  json:"allowRead,omitempty"`
	DenyRead   []string `yaml:"deny-read,omitempty"   json:"denyRead,omitempty"`
}

// SandboxNetwork configures network isolation boundaries.
type SandboxNetwork struct {
	HTTPProxyPort           *int  `yaml:"http-proxy-port,omitempty"           json:"httpProxyPort,omitempty"`
	SOCKSProxyPort          *int  `yaml:"socks-proxy-port,omitempty"          json:"socksProxyPort,omitempty"`
	AllowManagedDomainsOnly *bool `yaml:"allow-managed-domains-only,omitempty" json:"allowManagedDomainsOnly,omitempty"`
	// AllowUnixSockets is a list of specific Unix domain socket paths permitted for
	// outbound connections. Use an empty list to deny all, or ["*"] to allow all.
	AllowUnixSockets []string `yaml:"allow-unix-sockets,omitempty"        json:"allowUnixSockets,omitempty"`
	// AllowLocalBinding permits the sandboxed process to bind to localhost ports.
	AllowLocalBinding *bool    `yaml:"allow-local-binding,omitempty"       json:"allowLocalBinding,omitempty"`
	AllowedDomains    []string `yaml:"allowed-domains,omitempty"          json:"allowedDomains,omitempty"`
}

// SettingsConfig represents the full platform settings.json structure.
// The mcp: block at the top level of XcaffoldConfig is a convenience shorthand
// that gets merged into the mcpServers key during compilation. Fields defined
// here take precedence over the shorthand for mcpServers if both are set.
type SettingsConfig struct {
	// Unique identifier for this settings block.
	// +xcaf:required
	// +xcaf:group=Identity
	Name string `yaml:"name,omitempty" json:"-"`

	// Human-readable purpose of this settings block.
	// +xcaf:optional
	// +xcaf:group=Identity
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Agent configuration object passed through to the provider.
	// +xcaf:optional
	// +xcaf:group=Platform
	Agent any `yaml:"agent,omitempty" json:"agent,omitempty"`

	// Worktree configuration object passed through to the provider.
	// +xcaf:optional
	// +xcaf:group=Platform
	Worktree any `yaml:"worktree,omitempty" json:"worktree,omitempty"`

	// Auto-mode behavior configuration.
	// +xcaf:optional
	// +xcaf:group=Platform
	AutoMode any `yaml:"auto-mode,omitempty" json:"autoMode,omitempty"`

	// Days before unused sessions are cleaned up.
	// +xcaf:optional
	// +xcaf:group=Platform
	CleanupPeriodDays *int `yaml:"cleanup-period-days,omitempty" json:"cleanupPeriodDays,omitempty"`

	// Whether to include git instructions in agent context.
	// +xcaf:optional
	// +xcaf:group=Platform
	IncludeGitInstructions *bool `yaml:"include-git-instructions,omitempty" json:"includeGitInstructions,omitempty"`

	// Skip the dangerous mode permission confirmation prompt.
	// +xcaf:optional
	// +xcaf:group=Permissions
	SkipDangerousModePermissionPrompt *bool `yaml:"skip-dangerous-mode-permission-prompt,omitempty" json:"skipDangerousModePermissionPrompt,omitempty"`

	// Tool permission rules (allow, deny, ask lists).
	// +xcaf:optional
	// +xcaf:group=Permissions
	Permissions *PermissionsConfig `yaml:"permissions,omitempty" json:"permissions,omitempty"`

	// OS-level process isolation configuration.
	// +xcaf:optional
	// +xcaf:group=Permissions
	Sandbox *SandboxConfig `yaml:"sandbox,omitempty" json:"sandbox,omitempty"`

	// Enable automatic memory persistence.
	// +xcaf:optional
	// +xcaf:group=Memory
	AutoMemoryEnabled *bool `yaml:"auto-memory-enabled,omitempty" json:"autoMemoryEnabled,omitempty"`

	// Disable all lifecycle hooks globally.
	// +xcaf:optional
	// +xcaf:group=Hooks
	DisableAllHooks *bool `yaml:"disable-all-hooks,omitempty" json:"disableAllHooks,omitempty"`

	// Enable commit attribution metadata.
	// +xcaf:optional
	// +xcaf:group=Platform
	Attribution *bool `yaml:"attribution,omitempty" json:"attribution,omitempty"`

	// Inline MCP server definitions.
	// +xcaf:optional
	// +xcaf:group=MCP
	// +xcaf:type=map
	MCPServers map[string]MCPConfig `yaml:"mcp-servers,omitempty" json:"mcpServers,omitempty"`

	// Lifecycle hook definitions.
	// +xcaf:optional
	// +xcaf:group=Hooks
	Hooks HookConfig `yaml:"hooks,omitempty" json:"hooks,omitempty"`

	// Status line display configuration.
	// +xcaf:optional
	// +xcaf:group=Platform
	StatusLine *StatusLineConfig `yaml:"status-line,omitempty" json:"statusLine,omitempty"`

	// Whether to respect .gitignore when scanning files.
	// +xcaf:optional
	// +xcaf:group=Platform
	RespectGitignore *bool `yaml:"respect-gitignore,omitempty" json:"respectGitignore,omitempty"`

	// Environment variables injected into agent sessions.
	// +xcaf:optional
	// +xcaf:group=Environment
	// +xcaf:type=map
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`

	// Plugin enable/disable flags keyed by plugin name.
	// +xcaf:optional
	// +xcaf:group=Platform
	// +xcaf:type=map
	EnabledPlugins map[string]bool `yaml:"enabled-plugins,omitempty" json:"enabledPlugins,omitempty"`

	// Prevent skills from executing shell commands.
	// +xcaf:optional
	// +xcaf:group=Permissions
	DisableSkillShellExecution *bool `yaml:"disable-skill-shell-execution,omitempty" json:"disableSkillShellExecution,omitempty"`

	// Force extended thinking on every turn.
	// +xcaf:optional
	// +xcaf:group=Model
	AlwaysThinkingEnabled *bool `yaml:"always-thinking-enabled,omitempty" json:"alwaysThinkingEnabled,omitempty"`

	// Default reasoning effort level.
	// +xcaf:optional
	// +xcaf:group=Model
	EffortLevel string `yaml:"effort-level,omitempty" json:"effortLevel,omitempty"`

	// Default shell for command execution.
	// +xcaf:optional
	// +xcaf:group=Platform
	DefaultShell string `yaml:"default-shell,omitempty" json:"defaultShell,omitempty"`

	// UI language preference.
	// +xcaf:optional
	// +xcaf:group=Platform
	Language string `yaml:"language,omitempty" json:"language,omitempty"`

	// Output verbosity style.
	// +xcaf:optional
	// +xcaf:group=Platform
	OutputStyle string `yaml:"output-style,omitempty" json:"outputStyle,omitempty"`

	// Directory for storing plan files.
	// +xcaf:optional
	// +xcaf:group=Platform
	PlansDirectory string `yaml:"plans-directory,omitempty" json:"plansDirectory,omitempty"`

	// Default model for the platform session.
	// +xcaf:optional
	// +xcaf:group=Model
	Model string `yaml:"model,omitempty" json:"model,omitempty"`

	// Helper command for generating OpenTelemetry headers.
	// +xcaf:optional
	// +xcaf:group=Platform
	OtelHeadersHelper string `yaml:"otel-headers-helper,omitempty" json:"otelHeadersHelper,omitempty"`

	// Directory for automatic memory file storage.
	// +xcaf:optional
	// +xcaf:group=Memory
	AutoMemoryDirectory string `yaml:"auto-memory-directory,omitempty" json:"autoMemoryDirectory,omitempty"`

	// List of model IDs available for selection.
	// +xcaf:optional
	// +xcaf:group=Model
	// +xcaf:type=[]string
	AvailableModels []string `yaml:"available-models,omitempty" json:"availableModels,omitempty"`

	// Glob patterns for paths excluded from root context file scanning.
	// +xcaf:optional
	// +xcaf:group=Platform
	// +xcaf:type=[]string
	MdExcludes []string `yaml:"md-excludes,omitempty" json:"mdExcludes,omitempty"`

	// Per-provider behavioral overrides for this settings block.
	// +xcaf:optional
	// +xcaf:group=Multi-Target
	// +xcaf:type=map
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this settings block was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}

// TestConfig holds project-level configuration for `xcaffold test`.
type TestConfig struct {
	// CliPath is the path to the CLI binary (e.g., claude, cursor). Required for CLI auth mode.
	CliPath string `yaml:"cli-path,omitempty"`

	// JudgeModel is the generative model used for LLM-as-a-Judge evaluation.
	JudgeModel string `yaml:"judge-model,omitempty"`

	// Task is the user prompt sent to the agent during simulation.
	// Defaults to a generic capability discovery prompt when empty.
	Task string `yaml:"task,omitempty"`

	// MaxTurns caps the number of simulated conversation turns.
	// Reserved for future multi-turn support; currently unused beyond recording.
	MaxTurns *int `yaml:"max-turns,omitempty"`
}

// Activation controls when a workflow is applied: always (all contexts) or paths (file-scoped).
// It accepts either a scalar "always" or a sequence of path globs ["*.go", "*.ts"].
type Activation struct {
	Mode  string   // "always" or "paths"
	Paths []string // populated when Mode == "paths"
}

// UnmarshalYAML implements yaml.Unmarshaler for Activation.
// Accepts scalar "always" → Mode="always", Paths=nil
// Accepts sequence ["*.go", ...] → Mode="paths", Paths=[...]
// Rejects all other values with clear error messages.
func (a *Activation) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Value == ActivationModeAlways {
			a.Mode = ActivationModeAlways
			a.Paths = nil
			return nil
		}
		return &yaml.TypeError{
			Errors: []string{
				"invalid activation \"" + value.Value + "\": expected \"always\" or a list of path globs",
			},
		}
	case yaml.SequenceNode:
		a.Mode = ActivationModePaths
		return value.Decode(&a.Paths)
	default:
		return &yaml.TypeError{
			Errors: []string{
				"invalid activation: expected string or list, got " + value.Tag,
			},
		}
	}
}

// MarshalYAML implements yaml.Marshaler for Activation.
// Round-trip: Mode="always" → scalar "always", Mode="paths" → sequence of paths.
func (a Activation) MarshalYAML() (interface{}, error) {
	if a.Mode == ActivationModeAlways {
		return ActivationModeAlways, nil
	}
	return a.Paths, nil
}

// WorkflowConfig defines a named, reusable, multi-step procedure.
// Each workflow maps to an entry under the `workflows:` key in project.xcaf.
// api-version: workflow/v1 is the current stable shape; workflow/v2 will add
// parameterized steps and DAG ordering without breaking v1 schemas.
type WorkflowConfig struct {
	// Schema shape discriminator for workflow versioning.
	// +xcaf:optional
	// +xcaf:group=Identity
	// +xcaf:enum=workflow/v1
	// +xcaf:default=workflow/v1
	// +xcaf:provider=antigravity:optional
	ApiVersion string `yaml:"api-version,omitempty"`

	// Unique identifier for this workflow within the project.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this workflow.
	// +xcaf:optional
	// +xcaf:group=Identity
	// +xcaf:provider=antigravity:optional
	Description string `yaml:"description,omitempty"`

	// Ordered procedural steps for multi-step workflows.
	// +xcaf:optional
	// +xcaf:group=Steps
	// +xcaf:type=[]WorkflowStep
	// +xcaf:provider=antigravity:optional
	Steps []WorkflowStep `yaml:"steps,omitempty"`

	// Per-provider overrides and lowering-strategy directives.
	// +xcaf:optional
	// +xcaf:group=Multi-Target
	// +xcaf:type=map
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Activation mode: "always" (all contexts) or a list of path globs for conditional triggering.
	// +xcaf:optional
	// +xcaf:group=Activation
	// +xcaf:type=Activation
	// +xcaf:provider=claude:optional,cursor:optional,gemini:optional,copilot:optional,antigravity:optional
	Activation *Activation `yaml:"activation,omitempty"`

	// Named subdirectories to copy from xcaf/workflows/<id>/ to provider output.
	// +xcaf:optional
	// +xcaf:group=Composition
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional,cursor:optional,gemini:optional,copilot:optional,antigravity:optional
	Artifacts []string `yaml:"artifacts,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. Never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}

// WorkflowStep is one named step in a workflow's ordered body.
type WorkflowStep struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Skill       string `yaml:"skill,omitempty"`
	// Inline procedural content for this step.
	// Used when the step defines its own instructions rather than referencing an external skill.
	// +xcaf:optional
	// +xcaf:group=Steps
	// +xcaf:provider=antigravity:optional
	Instructions string `yaml:"instructions,omitempty"`
}

// PolicyConfig defines a declarative constraint evaluated against the AST
// and compiled output during apply and validate.
type PolicyConfig struct {
	// Unique identifier for this policy.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name"`

	// Human-readable purpose of this policy.
	// +xcaf:optional
	// +xcaf:group=Identity
	Description string `yaml:"description,omitempty"`

	// Violation severity level when the policy fails.
	// +xcaf:required
	// +xcaf:group=Evaluation
	// +xcaf:enum=error,warning,off
	Severity string `yaml:"severity"`

	// Resource kind this policy evaluates.
	// +xcaf:required
	// +xcaf:group=Evaluation
	// +xcaf:enum=agent,skill,rule,hook,settings,output
	Target string `yaml:"target"`

	// Filter conditions selecting which resources to evaluate.
	// +xcaf:optional
	// +xcaf:group=Match Filter
	Match *PolicyMatch `yaml:"match,omitempty"`

	// Field constraints applied to matched resources.
	// +xcaf:optional
	// +xcaf:group=Requirements
	// +xcaf:type=[]PolicyRequire
	Require []PolicyRequire `yaml:"require,omitempty"`

	// Forbidden content or path patterns in compiled output.
	// +xcaf:optional
	// +xcaf:group=Deny Rules
	// +xcaf:type=[]PolicyDeny
	Deny []PolicyDeny `yaml:"deny,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
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
