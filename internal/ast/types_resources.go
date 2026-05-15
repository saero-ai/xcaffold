package ast

// AgentConfig defines an AI coding agent persona.
//
// Field ordering is canonical and mirrors the compiled markdown frontmatter:
//  1. Identity (name, description)
//  2. Model & Execution (model, effort, maxTurns)
//  3. Tool Access (tools, disallowedTools, readonly)
//  4. Permissions & Invocation (permissionMode, disableModelInvocation, userInvocable)
//  5. Lifecycle (background, isolation)
//  6. Memory & Context (memory, color, initialPrompt)
//  7. Composition references (skills, rules, mcp, assertions)
//  8. Inline composition (mcpServers, hooks)
//  9. Multi-Target (targets)
//  10. Instructions (always last)
type AgentConfig struct {
	// Unique identifier for this agent within the project.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	// +xcaf:role=identity
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this agent.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:provider=claude:required,gemini:required,copilot:required,cursor:optional,antigravity:optional
	// +xcaf:role=rendering
	Description string `yaml:"description"`

	// LLM model identifier or alias resolved at compile time.
	// +xcaf:optional
	// +xcaf:group=Model & Execution
	// +xcaf:example=sonnet
	// +xcaf:provider=claude:optional,gemini:optional,copilot:optional,cursor:optional,antigravity:optional
	// +xcaf:role=rendering
	Model string `yaml:"model,omitempty"`

	// Reasoning effort level hint for the model provider.
	// +xcaf:optional
	// +xcaf:group=Model & Execution
	// +xcaf:provider=claude:optional,cursor:optional
	// +xcaf:role=rendering
	Effort string `yaml:"effort,omitempty"`

	// Maximum conversation turns before the agent exits.
	// +xcaf:optional
	// +xcaf:group=Model & Execution
	// +xcaf:provider=claude:optional,gemini:optional
	// +xcaf:role=rendering
	MaxTurns *int `yaml:"max-turns,omitempty"`

	// Ordered list of tools this agent may invoke.
	// +xcaf:optional
	// +xcaf:group=Tool Access
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional,gemini:optional,copilot:optional,cursor:unsupported,antigravity:unsupported
	// +xcaf:role=rendering
	Tools ClearableList `yaml:"tools,omitempty"`

	// Tools explicitly denied to this agent.
	// +xcaf:optional
	// +xcaf:group=Tool Access
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional
	// +xcaf:role=rendering
	DisallowedTools ClearableList `yaml:"disallowed-tools,omitempty"`

	// When true, restricts the agent to read-only tool access.
	// +xcaf:optional
	// +xcaf:group=Tool Access
	// +xcaf:provider=claude:optional,cursor:optional
	// +xcaf:role=rendering
	Readonly *bool `yaml:"readonly,omitempty"`

	// Security mode controlling tool authorization behavior.
	// +xcaf:optional
	// +xcaf:group=Permissions & Invocation
	// +xcaf:provider=claude:optional
	// +xcaf:role=rendering
	PermissionMode string `yaml:"permission-mode,omitempty"`

	// Prevents the agent from spawning sub-agents.
	// +xcaf:optional
	// +xcaf:group=Permissions & Invocation
	// +xcaf:provider=claude:optional
	// +xcaf:role=rendering
	DisableModelInvocation *bool `yaml:"disable-model-invocation,omitempty"`

	// Whether users can invoke this agent directly via slash command.
	// +xcaf:optional
	// +xcaf:group=Permissions & Invocation
	// +xcaf:provider=claude:optional
	// +xcaf:role=rendering
	UserInvocable *bool `yaml:"user-invocable,omitempty"`

	// Runs the agent in background mode without interactive prompts.
	// +xcaf:optional
	// +xcaf:group=Lifecycle
	// +xcaf:provider=claude:optional,cursor:optional
	// +xcaf:role=rendering
	Background *bool `yaml:"background,omitempty"`

	// Process isolation level for the agent session.
	// +xcaf:optional
	// +xcaf:group=Lifecycle
	// +xcaf:provider=claude:optional
	// +xcaf:role=rendering
	Isolation string `yaml:"isolation,omitempty"`

	// Named memory banks attached to this agent.
	// +xcaf:optional
	// +xcaf:group=Memory & Context
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional
	// +xcaf:role=rendering
	Memory FlexStringSlice `yaml:"memory,omitempty"`

	// Display color for terminal output differentiation.
	// +xcaf:optional
	// +xcaf:group=Memory & Context
	// +xcaf:role=metadata
	Color string `yaml:"color,omitempty"`

	// System prompt prepended to every conversation.
	// +xcaf:optional
	// +xcaf:group=Memory & Context
	// +xcaf:provider=claude:optional
	// +xcaf:role=rendering
	InitialPrompt string `yaml:"initial-prompt,omitempty"`

	// Skill resource IDs attached to this agent.
	// +xcaf:optional
	// +xcaf:group=Composition
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional
	// +xcaf:role=composition,rendering
	Skills ClearableList `yaml:"skills,omitempty"`

	// Rule resource IDs governing this agent.
	// +xcaf:optional
	// +xcaf:group=Composition
	// +xcaf:type=[]string
	// +xcaf:role=composition
	Rules ClearableList `yaml:"rules,omitempty"`

	// MCP server resource IDs available to this agent.
	// +xcaf:optional
	// +xcaf:group=Composition
	// +xcaf:type=[]string
	// +xcaf:role=composition
	MCP ClearableList `yaml:"mcp,omitempty"`

	// Policy assertion IDs evaluated post-compilation.
	// +xcaf:optional
	// +xcaf:group=Composition
	// +xcaf:type=[]string
	// +xcaf:role=composition
	Assertions ClearableList `yaml:"assertions,omitempty"`

	// Inline MCP server definitions keyed by server name.
	// +xcaf:optional
	// +xcaf:group=Inline Composition
	// +xcaf:type=map
	// +xcaf:provider=claude:optional,gemini:optional,copilot:optional
	// +xcaf:role=rendering
	MCPServers map[string]MCPConfig `yaml:"mcp-servers,omitempty"`

	// Inline lifecycle hook definitions for this agent.
	// +xcaf:optional
	// +xcaf:group=Inline Composition
	// +xcaf:provider=claude:optional
	// +xcaf:role=rendering
	Hooks HookConfig `yaml:"hooks,omitempty"`

	// Per-provider override configuration keyed by provider name.
	// +xcaf:optional
	// +xcaf:group=Multi-Target
	// +xcaf:type=map
	// +xcaf:role=filtering
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Populated by the parser from the frontmatter body section.
	Body string `yaml:"-"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized and causes renderers
	// to skip the resource during project-scope compilation.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}

// TargetOverride specifies overrides for multi-provider targets.
type TargetOverride struct {
	Hooks                    map[string]string `yaml:"hooks,omitempty"`
	SuppressFidelityWarnings *bool             `yaml:"suppress-fidelity-warnings,omitempty"`
	SkipSynthesis            *bool             `yaml:"skip-synthesis,omitempty"`

	Provider map[string]any `yaml:"provider,omitempty"`
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
	// Unique identifier for this skill within the project.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	// +xcaf:role=identity
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this skill.
	// +xcaf:optional
	// +xcaf:group=Identity
	// +xcaf:provider=claude:optional,cursor:optional,gemini:optional,copilot:optional,antigravity:optional
	// +xcaf:role=rendering
	Description string `yaml:"description,omitempty"`

	// Guidance for the model on when to invoke this skill.
	// +xcaf:optional
	// +xcaf:group=Identity
	// +xcaf:provider=claude:optional
	// +xcaf:role=metadata
	WhenToUse string `yaml:"when-to-use,omitempty"`

	// SPDX license identifier for open-source skills.
	// +xcaf:optional
	// +xcaf:group=Identity
	// +xcaf:provider=claude:optional,copilot:optional
	// +xcaf:role=metadata
	License string `yaml:"license,omitempty"`

	// Tools this skill is permitted to use. Skill-specific field.
	// +xcaf:optional
	// +xcaf:group=Tool Access
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional,copilot:optional
	// +xcaf:role=rendering
	AllowedTools ClearableList `yaml:"allowed-tools,omitempty"`

	// Prevents the skill from spawning sub-agents.
	// +xcaf:optional
	// +xcaf:group=Permissions & Invocation
	// +xcaf:provider=claude:optional
	// +xcaf:role=rendering
	DisableModelInvocation *bool `yaml:"disable-model-invocation,omitempty"`

	// Whether users can invoke this skill directly via slash command.
	// +xcaf:optional
	// +xcaf:group=Permissions & Invocation
	// +xcaf:provider=claude:optional
	// +xcaf:role=metadata
	UserInvocable *bool `yaml:"user-invocable,omitempty"`

	// Hint text shown to the user when invoking the skill.
	// +xcaf:optional
	// +xcaf:group=Permissions & Invocation
	// +xcaf:provider=claude:optional
	// +xcaf:role=metadata
	ArgumentHint string `yaml:"argument-hint,omitempty"`

	// Named subdirectories to copy from xcaf/skills/<id>/ to provider output.
	// +xcaf:optional
	// +xcaf:group=Composition
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional,cursor:optional,gemini:optional,copilot:optional,antigravity:optional
	// +xcaf:role=composition
	Artifacts []string `yaml:"artifacts,omitempty"`

	// Per-provider override configuration keyed by provider name.
	// +xcaf:optional
	// +xcaf:group=Multi-Target
	// +xcaf:type=map
	// +xcaf:role=filtering
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Populated by the parser from the frontmatter body section.
	Body string `yaml:"-"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
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
	// When true, applies this rule to all files unconditionally.
	// +xcaf:optional
	// +xcaf:group=Activation
	// +xcaf:provider=cursor:optional
	// +xcaf:role=rendering
	AlwaysApply *bool `yaml:"always-apply,omitempty"`

	// Human-readable purpose of this rule.
	// +xcaf:optional
	// +xcaf:group=Identity
	// +xcaf:provider=claude:optional,cursor:optional,copilot:optional,antigravity:optional
	// +xcaf:role=rendering
	Description string `yaml:"description,omitempty"`

	// Cross-provider activation mode for this rule.
	// +xcaf:optional
	// +xcaf:group=Activation
	// +xcaf:enum=always,path-glob,model-decided,manual-mention,explicit-invoke
	// +xcaf:provider=cursor:optional
	// +xcaf:role=rendering
	Activation string `yaml:"activation,omitempty"`

	// Unique identifier for this rule within the project.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	// +xcaf:role=identity
	Name string `yaml:"name,omitempty"`

	// Populated by the parser from the frontmatter body section.
	Body string `yaml:"-"`

	// Glob patterns for path-based activation.
	// +xcaf:optional
	// +xcaf:group=Activation
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional,cursor:optional,copilot:optional,antigravity:optional
	// +xcaf:role=rendering
	Paths ClearableList `yaml:"paths,omitempty"`

	// Agent types excluded from receiving this rule.
	// +xcaf:optional
	// +xcaf:group=Provider-Specific
	// +xcaf:type=[]string
	// +xcaf:enum=code-review,cloud-agent
	// +xcaf:provider=copilot:optional
	// +xcaf:role=rendering
	ExcludeAgents ClearableList `yaml:"exclude-agents,omitempty"`

	// Per-provider override configuration keyed by provider name.
	// +xcaf:optional
	// +xcaf:group=Multi-Target
	// +xcaf:type=map
	// +xcaf:role=filtering
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
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

	// SourceProvider identifies the provider this hook was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}

// NamedHookConfig is a named lifecycle hook block.
type NamedHookConfig struct {
	// Unique identifier for this hook block.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this hook block.
	// +xcaf:optional
	// +xcaf:group=Identity
	Description string `yaml:"description,omitempty"`

	// Named subdirectories to copy from xcaf/hooks/<name>/ to provider hook dirs.
	// +xcaf:optional
	// +xcaf:group=Composition
	// +xcaf:type=[]string
	Artifacts []string `yaml:"artifacts,omitempty"`

	// Lifecycle event handlers keyed by event name.
	// +xcaf:optional
	// +xcaf:group=Events
	// +xcaf:provider=claude:optional
	Events HookConfig `yaml:"events,omitempty"`

	// Per-provider behavioral overrides for this hook block.
	// +xcaf:optional
	// +xcaf:group=Multi-Target
	// +xcaf:type=map
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}

// MCPConfig defines a local or remote MCP server context.
type MCPConfig struct {
	// Environment variables passed to the server process.
	// +xcaf:optional
	// +xcaf:group=Environment & Headers
	// +xcaf:type=map
	// +xcaf:provider=claude:optional,gemini:optional,copilot:optional
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`

	// HTTP headers for remote server connections.
	// +xcaf:optional
	// +xcaf:group=Environment & Headers
	// +xcaf:type=map
	// +xcaf:provider=claude:optional
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// When true, the server is registered but not started.
	// +xcaf:optional
	// +xcaf:group=Control
	// +xcaf:provider=claude:optional
	Disabled *bool `yaml:"disabled,omitempty" json:"disabled,omitempty"`

	// OAuth configuration key-value pairs for authentication.
	// +xcaf:optional
	// +xcaf:group=Authentication
	// +xcaf:type=map
	// +xcaf:provider=claude:optional
	OAuth map[string]string `yaml:"oauth,omitempty" json:"oauth,omitempty"`

	// Unique identifier for this MCP server.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty" json:"-"`

	// Human-readable purpose of this MCP server.
	// +xcaf:optional
	// +xcaf:group=Identity
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Server type (stdio or sse).
	// +xcaf:optional
	// +xcaf:group=Connection
	// +xcaf:provider=claude:optional,gemini:optional,copilot:optional
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Shell command to start the MCP server process.
	// +xcaf:optional
	// +xcaf:group=Connection
	// +xcaf:provider=claude:optional,gemini:optional,copilot:optional
	Command string `yaml:"command,omitempty" json:"command,omitempty"`

	// URL for remote MCP servers.
	// +xcaf:optional
	// +xcaf:group=Connection
	// +xcaf:provider=claude:optional,gemini:optional,copilot:optional
	URL string `yaml:"url,omitempty" json:"url,omitempty"`

	// Working directory for the MCP server process.
	// +xcaf:optional
	// +xcaf:group=Connection
	// +xcaf:provider=claude:optional
	Cwd string `yaml:"cwd,omitempty" json:"cwd,omitempty"`

	// Authentication provider type for remote servers.
	// +xcaf:optional
	// +xcaf:group=Authentication
	// +xcaf:provider=claude:optional
	AuthProviderType string `yaml:"auth-provider-type,omitempty" json:"authProviderType,omitempty"`

	// Arguments passed to the server command.
	// +xcaf:optional
	// +xcaf:group=Connection
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional,gemini:optional,copilot:optional
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`

	// Tool names to disable on this MCP server.
	// +xcaf:optional
	// +xcaf:group=Control
	// +xcaf:type=[]string
	// +xcaf:provider=claude:optional
	DisabledTools []string `yaml:"disabled-tools,omitempty" json:"disabledTools,omitempty"`

	// Per-provider behavioral overrides for this MCP server.
	// +xcaf:optional
	// +xcaf:group=Multi-Target
	// +xcaf:type=map
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-" json:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}

// MemoryConfig defines a named memory entry scoped to an agent.
// Memory is convention-based: the compiler discovers .md files under
// xcaf/agents/<agentID>/memory/ and populates Content from the file body.
type MemoryConfig struct {
	// Unique identifier for this memory entry.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this memory entry.
	// +xcaf:optional
	// +xcaf:group=Identity
	Description string `yaml:"description,omitempty"`

	// Markdown body read from the .md file at compile time.
	Content string `yaml:"-"`

	// Owning agent derived from directory placement at parse time.
	AgentRef string `yaml:"-"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}

// ContextConfig defines a named context block — shared prompt context that can be
// selectively included in compiled output. Contexts are blueprint-selectable and
// may be scoped to specific provider targets.
type ContextConfig struct {
	// Unique identifier for this context block.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this context block.
	// +xcaf:optional
	// +xcaf:group=Identity
	// +xcaf:provider=cursor:optional
	Description string `yaml:"description,omitempty"`

	// Marks this context as tie-breaker when multiple match the same target.
	// +xcaf:optional
	// +xcaf:group=Behavior
	// +xcaf:provider=cursor:optional
	Default bool `yaml:"default,omitempty"`

	// Markdown content of the context block.
	Body string `yaml:"-"`

	// Restricts this context to specific provider targets.
	// +xcaf:optional
	// +xcaf:group=Targeting
	// +xcaf:type=[]string
	Targets []string `yaml:"targets,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}

// TemplateConfig defines a named template resource. Templates are structural
// scaffolds that generate project files during init. Overrides allow
// provider-specific template customization.
type TemplateConfig struct {
	// Unique identifier for this template within the project.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name"`

	// Human-readable purpose of this template.
	// +xcaf:optional
	// +xcaf:group=Identity
	Description string `yaml:"description,omitempty"`

	// Default compilation target provider for this template.
	// +xcaf:optional
	// +xcaf:group=Compilation
	DefaultTarget string `yaml:"default-target,omitempty"`

	Body string `yaml:"-"`

	Inherited bool `yaml:"-"`

	SourceProvider string `yaml:"-" json:"-"`
}

// BlueprintConfig defines a named resource subset selector.
// A blueprint selects which agents, skills, rules, workflows, MCP servers,
// policies, memory entries, contexts, settings, and hooks to compile.
type BlueprintConfig struct {
	// Unique identifier for this blueprint.
	// +xcaf:required
	// +xcaf:group=Identity
	// +xcaf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name"`

	// Human-readable purpose of this blueprint.
	// +xcaf:optional
	// +xcaf:group=Identity
	Description string `yaml:"description,omitempty"`

	// Name of another blueprint to inherit selections from.
	// +xcaf:optional
	// +xcaf:group=Inheritance
	Extends string `yaml:"extends,omitempty"`

	// Agent resource IDs to include in this blueprint.
	// +xcaf:optional
	// +xcaf:group=Resource Selectors
	// +xcaf:type=[]string
	Agents []string `yaml:"agents,omitempty"`

	// Skill resource IDs to include in this blueprint.
	// +xcaf:optional
	// +xcaf:group=Resource Selectors
	// +xcaf:type=[]string
	Skills []string `yaml:"skills,omitempty"`

	// Rule resource IDs to include in this blueprint.
	// +xcaf:optional
	// +xcaf:group=Resource Selectors
	// +xcaf:type=[]string
	Rules []string `yaml:"rules,omitempty"`

	// Workflow resource IDs to include in this blueprint.
	// +xcaf:optional
	// +xcaf:group=Resource Selectors
	// +xcaf:type=[]string
	Workflows []string `yaml:"workflows,omitempty"`

	// MCP server resource IDs to include in this blueprint.
	// +xcaf:optional
	// +xcaf:group=Resource Selectors
	// +xcaf:type=[]string
	MCP []string `yaml:"mcp,omitempty"`

	// Policy resource IDs to include in this blueprint.
	// +xcaf:optional
	// +xcaf:group=Resource Selectors
	// +xcaf:type=[]string
	Policies []string `yaml:"policies,omitempty"`

	// Memory resource IDs to include in this blueprint.
	// +xcaf:optional
	// +xcaf:group=Resource Selectors
	// +xcaf:type=[]string
	Memory []string `yaml:"memory,omitempty"`

	// Context resource IDs to include in this blueprint.
	// +xcaf:optional
	// +xcaf:group=Resource Selectors
	// +xcaf:type=[]string
	Contexts []string `yaml:"contexts,omitempty"`

	// Name of the settings block to use.
	// +xcaf:optional
	// +xcaf:group=Singleton Selectors
	Settings string `yaml:"settings,omitempty"`

	// Name of the hooks block to use.
	// +xcaf:optional
	// +xcaf:group=Singleton Selectors
	Hooks string `yaml:"hooks,omitempty"`

	// Compilation targets for this blueprint.
	// When set, controls which providers this blueprint compiles for.
	// When empty, falls through to project.targets or --target flag.
	// +xcaf:optional
	// +xcaf:group=Multi-Target
	// +xcaf:role=filtering
	// +xcaf:type=[]string
	Targets []string `yaml:"targets,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}
