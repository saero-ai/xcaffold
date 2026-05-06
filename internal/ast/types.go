package ast

import (
	"gopkg.in/yaml.v3"
	"sort"
) // ResourceScope contains all agentic primitives that can appear at both
// global scope (root of XcaffoldConfig) and workspace scope (inside ProjectConfig).
// Embedded with yaml:",inline" so fields appear at the same YAML level as the parent.
type ResourceScope struct {
	Agents    map[string]AgentConfig    `yaml:"agents,omitempty"`
	Skills    map[string]SkillConfig    `yaml:"skills,omitempty"`
	Rules     map[string]RuleConfig     `yaml:"rules,omitempty"`
	MCP       map[string]MCPConfig      `yaml:"mcp,omitempty"`
	Workflows map[string]WorkflowConfig `yaml:"workflows,omitempty"`
	Policies  map[string]PolicyConfig   `yaml:"policies,omitempty"`
	Memory    map[string]MemoryConfig   `yaml:"memory,omitempty"`
	Contexts  map[string]ContextConfig  `yaml:"contexts,omitempty"`
	Templates map[string]TemplateConfig `yaml:"templates,omitempty"`
}

// XcaffoldConfig is the root structure of a parsed .xcf YAML file.
type XcaffoldConfig struct {
	Kind    string `yaml:"-"` // Set by parser routing, not decoded from YAML
	Version string `yaml:"version"`
	Extends string `yaml:"extends,omitempty"`

	Settings map[string]SettingsConfig  `yaml:"settings,omitempty"`
	Hooks    map[string]NamedHookConfig `yaml:"hooks,omitempty"`

	// Blueprints maps named resource subset selectors. Each blueprint selects
	// which agents, skills, rules, workflows, MCP servers, policies, memory
	// entries, settings, and hooks to include during compilation.
	Blueprints map[string]BlueprintConfig `yaml:"blueprints,omitempty"`

	// ProviderExtras holds raw file content keyed by provider name then by path
	// within that provider's output directory. It is populated by the import
	// pipeline and is never serialized to YAML or JSON.
	ProviderExtras map[string]map[string][]byte `yaml:"-" json:"-"`

	// Overrides stores parsed .<provider>.xcf partial configs keyed by [kind][name][provider].
	// Populated by the import pipeline; never serialized to YAML or JSON.
	// The compiler merges these with base resources during compilation.
	Overrides *ResourceOverrides `yaml:"-" json:"-"`

	// ParseWarnings collects non-fatal diagnostic messages produced during parsing,
	// such as name/kind mismatches between YAML declarations and filesystem paths.
	// Never serialized; callers decide how (or whether) to display these.
	ParseWarnings []string `yaml:"-" json:"-"`

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

	// AllowedEnvVars defines which environment variables can be injected via ${env.NAME}.
	// +xcf:optional
	// +xcf:type=[]string
	AllowedEnvVars []string `yaml:"allowed-env-vars,omitempty" json:"allowedEnvVars,omitempty"`
	// Targets lists the compilation targets for this project.
	// Populated by the parser when decoding kind: project documents.
	Targets []string `yaml:"-"`

	Test  TestConfig     `yaml:"test,omitempty"`
	Local SettingsConfig `yaml:"local,omitempty"`

	// Body holds markdown content parsed from the frontmatter+body format.
	// For kind:project, this content is placed into Contexts["root"] by the parser.
	Body string `yaml:"-"`

	// InstructionsImports lists @-import targets preserved verbatim for providers
	// that support them (Claude, Gemini). Emitted as-is into the rendered output.

	// InstructionsScopes defines per-directory nested instruction files.
	// Order in this slice is authoritative (depth ascending, then alphabetical).

	// TargetOptions holds per-provider compile-time options for the project.
	// Keys are registered provider names. Values are TargetOverride
	// instances. Only fields relevant to the named provider are examined.
	TargetOptions map[string]TargetOverride `yaml:"target-options,omitempty"`

	ResourceScope `yaml:",inline"` // Workspace-level resources
}

// FlexStringSlice is a custom type that accepts both YAML scalar strings and list sequences.
// It unmarshals a scalar string into a single-element slice for backward compatibility.
type FlexStringSlice []string

// UnmarshalYAML implements the yaml.Unmarshaler interface for FlexStringSlice.
func (f *FlexStringSlice) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*f = FlexStringSlice{value.Value}
		return nil
	}
	var list []string
	if err := value.Decode(&list); err != nil {
		return err
	}
	*f = FlexStringSlice(list)
	return nil
}

// NewFlexStringSlice constructs a FlexStringSlice from a single string value.
// If the string is empty, it returns nil; otherwise, it returns a slice containing that value.
func NewFlexStringSlice(s string) FlexStringSlice {
	if s == "" {
		return nil
	}
	return FlexStringSlice{s}
}

// ClearableList is a []string wrapper that distinguishes "absent" from "explicitly
// empty" during YAML unmarshaling. Used on list fields that participate in override
// merging where "clear this field" must be expressible.
//
// Semantics:
//   - Field absent in YAML:    nil (inherit base)
//   - Field set to [] or ~:    Cleared=true,  Values=nil  (clear field)
//   - Field set to [a, b]:     Cleared=false, Values=[a,b] (replace)
type ClearableList struct {
	Values  []string
	Cleared bool
}

// UnmarshalYAML implements yaml.Unmarshaler. It detects explicit null and empty
// sequences, setting the Cleared flag accordingly. The zero-value ClearableList{}
// (Cleared=false, Values=nil) represents an absent/inherited field.
// Note: UnmarshalYAML requires pointer receiver per yaml.v3 interface.
func (c *ClearableList) UnmarshalYAML(value *yaml.Node) error {
	// Detect explicit null in YAML (!!null tag)
	if value.Tag == "!!null" {
		c.Cleared = true
		c.Values = nil
		return nil
	}

	// Detect empty sequence (no special tag, empty content)
	if value.Kind == yaml.SequenceNode && len(value.Content) == 0 {
		c.Cleared = true
		c.Values = nil
		return nil
	}

	// Non-empty sequence: decode into Values
	c.Cleared = false
	return value.Decode(&c.Values)
}

// MarshalYAML implements yaml.Marshaler for round-trip serialization.
// Uses value receiver: the zero-value ClearableList means absent/inherit.
func (c ClearableList) MarshalYAML() (interface{}, error) {
	if c.Cleared {
		return []string{}, nil
	}
	if len(c.Values) == 0 {
		return nil, nil
	}
	return c.Values, nil
}

// Len returns the number of values. Convenience for migration from []string.
// Uses value receiver; zero-value returns 0.
func (c ClearableList) Len() int {
	return len(c.Values)
}

// IsEmpty returns true if the list has no values and is not explicitly cleared.
// Uses value receiver; zero-value returns true.
func (c ClearableList) IsEmpty() bool {
	return !c.Cleared && len(c.Values) == 0
}

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
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	// +xcf:role=identity
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this agent.
	// +xcf:optional
	// +xcf:group=Identity
	// +xcf:provider=claude:required,gemini:required,copilot:required,cursor:optional,antigravity:optional
	// +xcf:role=rendering
	Description string `yaml:"description,omitempty"`

	// LLM model identifier or alias resolved at compile time.
	// +xcf:optional
	// +xcf:group=Model & Execution
	// +xcf:example=sonnet
	// +xcf:provider=claude:optional,gemini:optional,copilot:optional,cursor:optional,antigravity:optional
	// +xcf:role=rendering
	Model string `yaml:"model,omitempty"`

	// Reasoning effort level hint for the model provider.
	// +xcf:optional
	// +xcf:group=Model & Execution
	// +xcf:provider=claude:optional,cursor:optional
	// +xcf:role=rendering
	Effort string `yaml:"effort,omitempty"`

	// Maximum conversation turns before the agent exits.
	// +xcf:optional
	// +xcf:group=Model & Execution
	// +xcf:provider=claude:optional,gemini:optional
	// +xcf:role=rendering
	MaxTurns int `yaml:"max-turns,omitempty"`

	// Ordered list of tools this agent may invoke.
	// +xcf:optional
	// +xcf:group=Tool Access
	// +xcf:type=[]string
	// +xcf:provider=claude:optional,gemini:optional,copilot:optional,cursor:unsupported,antigravity:unsupported
	// +xcf:role=rendering
	Tools ClearableList `yaml:"tools,omitempty"`

	// Tools explicitly denied to this agent.
	// +xcf:optional
	// +xcf:group=Tool Access
	// +xcf:type=[]string
	// +xcf:provider=claude:optional
	// +xcf:role=rendering
	DisallowedTools ClearableList `yaml:"disallowed-tools,omitempty"`

	// When true, restricts the agent to read-only tool access.
	// +xcf:optional
	// +xcf:group=Tool Access
	// +xcf:provider=claude:optional,cursor:optional
	// +xcf:role=rendering
	Readonly *bool `yaml:"readonly,omitempty"`

	// Security mode controlling tool authorization behavior.
	// +xcf:optional
	// +xcf:group=Permissions & Invocation
	// +xcf:provider=claude:optional
	// +xcf:role=rendering
	PermissionMode string `yaml:"permission-mode,omitempty"`

	// Prevents the agent from spawning sub-agents.
	// +xcf:optional
	// +xcf:group=Permissions & Invocation
	// +xcf:provider=claude:optional
	// +xcf:role=rendering
	DisableModelInvocation *bool `yaml:"disable-model-invocation,omitempty"`

	// Whether users can invoke this agent directly via slash command.
	// +xcf:optional
	// +xcf:group=Permissions & Invocation
	// +xcf:provider=claude:optional
	// +xcf:role=rendering
	UserInvocable *bool `yaml:"user-invocable,omitempty"`

	// Runs the agent in background mode without interactive prompts.
	// +xcf:optional
	// +xcf:group=Lifecycle
	// +xcf:provider=claude:optional,cursor:optional
	// +xcf:role=rendering
	Background *bool `yaml:"background,omitempty"`

	// Process isolation level for the agent session.
	// +xcf:optional
	// +xcf:group=Lifecycle
	// +xcf:provider=claude:optional
	// +xcf:role=rendering
	Isolation string `yaml:"isolation,omitempty"`

	// Named memory banks attached to this agent.
	// +xcf:optional
	// +xcf:group=Memory & Context
	// +xcf:type=[]string
	// +xcf:provider=claude:optional
	// +xcf:role=rendering
	Memory FlexStringSlice `yaml:"memory,omitempty"`

	// Display color for terminal output differentiation.
	// +xcf:optional
	// +xcf:group=Memory & Context
	// +xcf:role=metadata
	Color string `yaml:"color,omitempty"`

	// System prompt prepended to every conversation.
	// +xcf:optional
	// +xcf:group=Memory & Context
	// +xcf:provider=claude:optional
	// +xcf:role=rendering
	InitialPrompt string `yaml:"initial-prompt,omitempty"`

	// Skill resource IDs attached to this agent.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	// +xcf:provider=claude:optional
	// +xcf:role=composition,rendering
	Skills ClearableList `yaml:"skills,omitempty"`

	// Rule resource IDs governing this agent.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	// +xcf:role=composition
	Rules ClearableList `yaml:"rules,omitempty"`

	// MCP server resource IDs available to this agent.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	// +xcf:role=composition
	MCP ClearableList `yaml:"mcp,omitempty"`

	// Policy assertion IDs evaluated post-compilation.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	// +xcf:role=composition
	Assertions ClearableList `yaml:"assertions,omitempty"`

	// Inline MCP server definitions keyed by server name.
	// +xcf:optional
	// +xcf:group=Inline Composition
	// +xcf:type=map
	// +xcf:provider=claude:optional,gemini:optional,copilot:optional
	// +xcf:role=rendering
	MCPServers map[string]MCPConfig `yaml:"mcp-servers,omitempty"`

	// Inline lifecycle hook definitions for this agent.
	// +xcf:optional
	// +xcf:group=Inline Composition
	// +xcf:provider=claude:optional
	// +xcf:role=rendering
	Hooks HookConfig `yaml:"hooks,omitempty"`

	// Per-provider override configuration keyed by provider name.
	// +xcf:optional
	// +xcf:group=Multi-Target
	// +xcf:type=map
	// +xcf:role=filtering
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
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	// +xcf:role=identity
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this skill.
	// +xcf:optional
	// +xcf:group=Identity
	// +xcf:provider=claude:optional,cursor:optional,gemini:optional,copilot:optional,antigravity:optional
	// +xcf:role=rendering
	Description string `yaml:"description,omitempty"`

	// Guidance for the model on when to invoke this skill.
	// +xcf:optional
	// +xcf:group=Identity
	// +xcf:provider=claude:optional
	// +xcf:role=metadata
	WhenToUse string `yaml:"when-to-use,omitempty"`

	// SPDX license identifier for open-source skills.
	// +xcf:optional
	// +xcf:group=Identity
	// +xcf:provider=claude:optional,copilot:optional
	// +xcf:role=metadata
	License string `yaml:"license,omitempty"`

	// Tools this skill is permitted to use. Skill-specific field.
	// +xcf:optional
	// +xcf:group=Tool Access
	// +xcf:type=[]string
	// +xcf:provider=claude:optional,copilot:optional
	// +xcf:role=rendering
	AllowedTools ClearableList `yaml:"allowed-tools,omitempty"`

	// Prevents the skill from spawning sub-agents.
	// +xcf:optional
	// +xcf:group=Permissions & Invocation
	// +xcf:provider=claude:optional
	// +xcf:role=rendering
	DisableModelInvocation *bool `yaml:"disable-model-invocation,omitempty"`

	// Whether users can invoke this skill directly via slash command.
	// +xcf:optional
	// +xcf:group=Permissions & Invocation
	// +xcf:provider=claude:optional
	// +xcf:role=metadata
	UserInvocable *bool `yaml:"user-invocable,omitempty"`

	// Hint text shown to the user when invoking the skill.
	// +xcf:optional
	// +xcf:group=Permissions & Invocation
	// +xcf:provider=claude:optional
	// +xcf:role=metadata
	ArgumentHint string `yaml:"argument-hint,omitempty"`

	// Named subdirectories to copy from xcf/skills/<id>/ to provider output.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	// +xcf:provider=claude:optional,cursor:optional,gemini:optional,copilot:optional,antigravity:optional
	// +xcf:role=composition
	Artifacts []string `yaml:"artifacts,omitempty"`

	// Docs and data files copied to skills/<id>/references/ at compile time.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	// +xcf:role=composition
	References ClearableList `yaml:"references,omitempty"`

	// Executable helper files copied to skills/<id>/scripts/ at compile time.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	// +xcf:role=composition
	Scripts ClearableList `yaml:"scripts,omitempty"`

	// Output artifact files copied to skills/<id>/assets/ at compile time.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	// +xcf:role=composition
	Assets ClearableList `yaml:"assets,omitempty"`

	// Demonstration files copied to skills/<id>/examples/ at compile time.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	// +xcf:role=composition
	Examples ClearableList `yaml:"examples,omitempty"`

	// Per-provider override configuration keyed by provider name.
	// +xcf:optional
	// +xcf:group=Multi-Target
	// +xcf:type=map
	// +xcf:role=filtering
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
	// +xcf:optional
	// +xcf:group=Activation
	// +xcf:provider=cursor:optional
	// +xcf:role=rendering
	AlwaysApply *bool `yaml:"always-apply,omitempty"`

	// Human-readable purpose of this rule.
	// +xcf:optional
	// +xcf:group=Identity
	// +xcf:provider=claude:optional,cursor:optional,copilot:optional,antigravity:optional
	// +xcf:role=rendering
	Description string `yaml:"description,omitempty"`

	// Cross-provider activation mode for this rule.
	// +xcf:optional
	// +xcf:group=Activation
	// +xcf:enum=always,path-glob,model-decided,manual-mention,explicit-invoke
	// +xcf:provider=cursor:optional
	// +xcf:role=rendering
	Activation string `yaml:"activation,omitempty"`

	// Unique identifier for this rule within the project.
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	// +xcf:role=identity
	Name string `yaml:"name,omitempty"`

	// Populated by the parser from the frontmatter body section.
	Body string `yaml:"-"`

	// Glob patterns for path-based activation.
	// +xcf:optional
	// +xcf:group=Activation
	// +xcf:type=[]string
	// +xcf:provider=claude:optional,cursor:optional,copilot:optional,antigravity:optional
	// +xcf:role=rendering
	Paths ClearableList `yaml:"paths,omitempty"`

	// Agent types excluded from receiving this rule.
	// +xcf:optional
	// +xcf:group=Provider-Specific
	// +xcf:type=[]string
	// +xcf:enum=code-review,cloud-agent
	// +xcf:provider=copilot:optional
	// +xcf:role=rendering
	ExcludeAgents ClearableList `yaml:"exclude-agents,omitempty"`

	// Per-provider override configuration keyed by provider name.
	// +xcf:optional
	// +xcf:group=Multi-Target
	// +xcf:type=map
	// +xcf:role=filtering
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
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this hook block.
	// +xcf:optional
	// +xcf:group=Identity
	Description string `yaml:"description,omitempty"`

	// Named subdirectories to copy from xcf/hooks/<name>/ to provider hook dirs.
	// +xcf:optional
	// +xcf:group=Composition
	// +xcf:type=[]string
	Artifacts []string `yaml:"artifacts,omitempty"`

	// Lifecycle event handlers keyed by event name.
	// +xcf:optional
	// +xcf:group=Events
	// +xcf:provider=claude:optional
	Events HookConfig `yaml:"events,omitempty"`

	// Per-provider behavioral overrides for this hook block.
	// +xcf:optional
	// +xcf:group=Multi-Target
	// +xcf:type=map
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
	// +xcf:optional
	// +xcf:group=Environment & Headers
	// +xcf:type=map
	// +xcf:provider=claude:optional,gemini:optional,copilot:optional
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`

	// HTTP headers for remote server connections.
	// +xcf:optional
	// +xcf:group=Environment & Headers
	// +xcf:type=map
	// +xcf:provider=claude:optional
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// When true, the server is registered but not started.
	// +xcf:optional
	// +xcf:group=Control
	// +xcf:provider=claude:optional
	Disabled *bool `yaml:"disabled,omitempty" json:"disabled,omitempty"`

	// OAuth configuration key-value pairs for authentication.
	// +xcf:optional
	// +xcf:group=Authentication
	// +xcf:type=map
	// +xcf:provider=claude:optional
	OAuth map[string]string `yaml:"oauth,omitempty" json:"oauth,omitempty"`

	// Unique identifier for this MCP server.
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty" json:"-"`

	// Human-readable purpose of this MCP server.
	// +xcf:optional
	// +xcf:group=Identity
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Server type (stdio or sse).
	// +xcf:optional
	// +xcf:group=Connection
	// +xcf:provider=claude:optional,gemini:optional,copilot:optional
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Shell command to start the MCP server process.
	// +xcf:optional
	// +xcf:group=Connection
	// +xcf:provider=claude:optional,gemini:optional,copilot:optional
	Command string `yaml:"command,omitempty" json:"command,omitempty"`

	// URL for remote MCP servers.
	// +xcf:optional
	// +xcf:group=Connection
	// +xcf:provider=claude:optional,gemini:optional,copilot:optional
	URL string `yaml:"url,omitempty" json:"url,omitempty"`

	// Working directory for the MCP server process.
	// +xcf:optional
	// +xcf:group=Connection
	// +xcf:provider=claude:optional
	Cwd string `yaml:"cwd,omitempty" json:"cwd,omitempty"`

	// Authentication provider type for remote servers.
	// +xcf:optional
	// +xcf:group=Authentication
	// +xcf:provider=claude:optional
	AuthProviderType string `yaml:"auth-provider-type,omitempty" json:"authProviderType,omitempty"`

	// Arguments passed to the server command.
	// +xcf:optional
	// +xcf:group=Connection
	// +xcf:type=[]string
	// +xcf:provider=claude:optional,gemini:optional,copilot:optional
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`

	// Tool names to disable on this MCP server.
	// +xcf:optional
	// +xcf:group=Control
	// +xcf:type=[]string
	// +xcf:provider=claude:optional
	DisabledTools []string `yaml:"disabled-tools,omitempty" json:"disabledTools,omitempty"`

	// Per-provider behavioral overrides for this MCP server.
	// +xcf:optional
	// +xcf:group=Multi-Target
	// +xcf:type=map
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-" json:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
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
	// +xcf:required
	// +xcf:group=Identity
	Name string `yaml:"name,omitempty" json:"-"`

	// Human-readable purpose of this settings block.
	// +xcf:optional
	// +xcf:group=Identity
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Agent configuration object passed through to the provider.
	// +xcf:optional
	// +xcf:group=Platform
	Agent any `yaml:"agent,omitempty" json:"agent,omitempty"`

	// Worktree configuration object passed through to the provider.
	// +xcf:optional
	// +xcf:group=Platform
	Worktree any `yaml:"worktree,omitempty" json:"worktree,omitempty"`

	// Auto-mode behavior configuration.
	// +xcf:optional
	// +xcf:group=Platform
	AutoMode any `yaml:"auto-mode,omitempty" json:"autoMode,omitempty"`

	// Days before unused sessions are cleaned up.
	// +xcf:optional
	// +xcf:group=Platform
	CleanupPeriodDays *int `yaml:"cleanup-period-days,omitempty" json:"cleanupPeriodDays,omitempty"`

	// Whether to include git instructions in agent context.
	// +xcf:optional
	// +xcf:group=Platform
	IncludeGitInstructions *bool `yaml:"include-git-instructions,omitempty" json:"includeGitInstructions,omitempty"`

	// Skip the dangerous mode permission confirmation prompt.
	// +xcf:optional
	// +xcf:group=Permissions
	SkipDangerousModePermissionPrompt *bool `yaml:"skip-dangerous-mode-permission-prompt,omitempty" json:"skipDangerousModePermissionPrompt,omitempty"`

	// Tool permission rules (allow, deny, ask lists).
	// +xcf:optional
	// +xcf:group=Permissions
	Permissions *PermissionsConfig `yaml:"permissions,omitempty" json:"permissions,omitempty"`

	// OS-level process isolation configuration.
	// +xcf:optional
	// +xcf:group=Permissions
	Sandbox *SandboxConfig `yaml:"sandbox,omitempty" json:"sandbox,omitempty"`

	// Enable automatic memory persistence.
	// +xcf:optional
	// +xcf:group=Memory
	AutoMemoryEnabled *bool `yaml:"auto-memory-enabled,omitempty" json:"autoMemoryEnabled,omitempty"`

	// Disable all lifecycle hooks globally.
	// +xcf:optional
	// +xcf:group=Hooks
	DisableAllHooks *bool `yaml:"disable-all-hooks,omitempty" json:"disableAllHooks,omitempty"`

	// Enable commit attribution metadata.
	// +xcf:optional
	// +xcf:group=Platform
	Attribution *bool `yaml:"attribution,omitempty" json:"attribution,omitempty"`

	// Inline MCP server definitions.
	// +xcf:optional
	// +xcf:group=MCP
	// +xcf:type=map
	MCPServers map[string]MCPConfig `yaml:"mcp-servers,omitempty" json:"mcpServers,omitempty"`

	// Lifecycle hook definitions.
	// +xcf:optional
	// +xcf:group=Hooks
	Hooks HookConfig `yaml:"hooks,omitempty" json:"hooks,omitempty"`

	// Status line display configuration.
	// +xcf:optional
	// +xcf:group=Platform
	StatusLine *StatusLineConfig `yaml:"status-line,omitempty" json:"statusLine,omitempty"`

	// Whether to respect .gitignore when scanning files.
	// +xcf:optional
	// +xcf:group=Platform
	RespectGitignore *bool `yaml:"respect-gitignore,omitempty" json:"respectGitignore,omitempty"`

	// Environment variables injected into agent sessions.
	// +xcf:optional
	// +xcf:group=Environment
	// +xcf:type=map
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`

	// Plugin enable/disable flags keyed by plugin name.
	// +xcf:optional
	// +xcf:group=Platform
	// +xcf:type=map
	EnabledPlugins map[string]bool `yaml:"enabled-plugins,omitempty" json:"enabledPlugins,omitempty"`

	// Prevent skills from executing shell commands.
	// +xcf:optional
	// +xcf:group=Permissions
	DisableSkillShellExecution *bool `yaml:"disable-skill-shell-execution,omitempty" json:"disableSkillShellExecution,omitempty"`

	// Force extended thinking on every turn.
	// +xcf:optional
	// +xcf:group=Model
	AlwaysThinkingEnabled *bool `yaml:"always-thinking-enabled,omitempty" json:"alwaysThinkingEnabled,omitempty"`

	// Default reasoning effort level.
	// +xcf:optional
	// +xcf:group=Model
	EffortLevel string `yaml:"effort-level,omitempty" json:"effortLevel,omitempty"`

	// Default shell for command execution.
	// +xcf:optional
	// +xcf:group=Platform
	DefaultShell string `yaml:"default-shell,omitempty" json:"defaultShell,omitempty"`

	// UI language preference.
	// +xcf:optional
	// +xcf:group=Platform
	Language string `yaml:"language,omitempty" json:"language,omitempty"`

	// Output verbosity style.
	// +xcf:optional
	// +xcf:group=Platform
	OutputStyle string `yaml:"output-style,omitempty" json:"outputStyle,omitempty"`

	// Directory for storing plan files.
	// +xcf:optional
	// +xcf:group=Platform
	PlansDirectory string `yaml:"plans-directory,omitempty" json:"plansDirectory,omitempty"`

	// Default model for the platform session.
	// +xcf:optional
	// +xcf:group=Model
	Model string `yaml:"model,omitempty" json:"model,omitempty"`

	// Helper command for generating OpenTelemetry headers.
	// +xcf:optional
	// +xcf:group=Platform
	OtelHeadersHelper string `yaml:"otel-headers-helper,omitempty" json:"otelHeadersHelper,omitempty"`

	// Directory for automatic memory file storage.
	// +xcf:optional
	// +xcf:group=Memory
	AutoMemoryDirectory string `yaml:"auto-memory-directory,omitempty" json:"autoMemoryDirectory,omitempty"`

	// List of model IDs available for selection.
	// +xcf:optional
	// +xcf:group=Model
	// +xcf:type=[]string
	AvailableModels []string `yaml:"available-models,omitempty" json:"availableModels,omitempty"`

	// Glob patterns for paths excluded from root context file scanning.
	// +xcf:optional
	// +xcf:group=Platform
	// +xcf:type=[]string
	MdExcludes []string `yaml:"md-excludes,omitempty" json:"mdExcludes,omitempty"`

	// Per-provider behavioral overrides for this settings block.
	// +xcf:optional
	// +xcf:group=Multi-Target
	// +xcf:type=map
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
	MaxTurns int `yaml:"max-turns,omitempty"`
}

// WorkflowConfig defines a named, reusable, multi-step procedure.
// Each workflow maps to an entry under the `workflows:` key in project.xcf.
// api-version: workflow/v1 is the current stable shape; workflow/v2 will add
// parameterized steps and DAG ordering without breaking v1 schemas.
type WorkflowConfig struct {
	// Schema shape discriminator for workflow versioning.
	// +xcf:optional
	// +xcf:group=Identity
	// +xcf:enum=workflow/v1
	// +xcf:default=workflow/v1
	// +xcf:provider=antigravity:optional
	ApiVersion string `yaml:"api-version,omitempty"`

	// Unique identifier for this workflow within the project.
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this workflow.
	// +xcf:optional
	// +xcf:group=Identity
	// +xcf:provider=antigravity:optional
	Description string `yaml:"description,omitempty"`

	// Ordered procedural steps for multi-step workflows.
	// +xcf:optional
	// +xcf:group=Steps
	// +xcf:type=[]WorkflowStep
	// +xcf:provider=antigravity:optional
	Steps []WorkflowStep `yaml:"steps,omitempty"`

	// Per-provider overrides and lowering-strategy directives.
	// +xcf:optional
	// +xcf:group=Multi-Target
	// +xcf:type=map
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`

	// Top-level body for single-step or legacy workflows.
	Body string `yaml:"-"`

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
	Body        string `yaml:"-"`
}

// PolicyConfig defines a declarative constraint evaluated against the AST
// and compiled output during apply and validate.
type PolicyConfig struct {
	// Unique identifier for this policy.
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name"`

	// Human-readable purpose of this policy.
	// +xcf:optional
	// +xcf:group=Identity
	Description string `yaml:"description,omitempty"`

	// Violation severity level when the policy fails.
	// +xcf:required
	// +xcf:group=Evaluation
	// +xcf:enum=error,warning,off
	Severity string `yaml:"severity"`

	// Resource kind this policy evaluates.
	// +xcf:required
	// +xcf:group=Evaluation
	// +xcf:enum=agent,skill,rule,hook,settings,output
	Target string `yaml:"target"`

	// Filter conditions selecting which resources to evaluate.
	// +xcf:optional
	// +xcf:group=Match Filter
	Match *PolicyMatch `yaml:"match,omitempty"`

	// Field constraints applied to matched resources.
	// +xcf:optional
	// +xcf:group=Requirements
	// +xcf:type=[]PolicyRequire
	Require []PolicyRequire `yaml:"require,omitempty"`

	// Forbidden content or path patterns in compiled output.
	// +xcf:optional
	// +xcf:group=Deny Rules
	// +xcf:type=[]PolicyDeny
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

// MemoryConfig defines a named memory entry scoped to an agent.
// Memory is convention-based: the compiler discovers .md files under
// xcf/agents/<agentID>/memory/ and populates Content from the file body.
type MemoryConfig struct {
	// Unique identifier for this memory entry.
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this memory entry.
	// +xcf:optional
	// +xcf:group=Identity
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
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name,omitempty"`

	// Human-readable purpose of this context block.
	// +xcf:optional
	// +xcf:group=Identity
	// +xcf:provider=cursor:optional
	Description string `yaml:"description,omitempty"`

	// Marks this context as tie-breaker when multiple match the same target.
	// +xcf:optional
	// +xcf:group=Behavior
	// +xcf:provider=cursor:optional
	Default bool `yaml:"default,omitempty"`

	// Markdown content of the context block.
	Body string `yaml:"-"`

	// Restricts this context to specific provider targets.
	// +xcf:optional
	// +xcf:group=Targeting
	// +xcf:type=[]string
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
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name"`

	// Human-readable purpose of this template.
	// +xcf:optional
	// +xcf:group=Identity
	Description string `yaml:"description,omitempty"`

	// Default compilation target provider for this template.
	// +xcf:optional
	// +xcf:group=Compilation
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
	// +xcf:required
	// +xcf:group=Identity
	// +xcf:pattern=^[a-z0-9-]+$
	Name string `yaml:"name"`

	// Human-readable purpose of this blueprint.
	// +xcf:optional
	// +xcf:group=Identity
	Description string `yaml:"description,omitempty"`

	// Name of another blueprint to inherit selections from.
	// +xcf:optional
	// +xcf:group=Inheritance
	Extends string `yaml:"extends,omitempty"`

	// Agent resource IDs to include in this blueprint.
	// +xcf:optional
	// +xcf:group=Resource Selectors
	// +xcf:type=[]string
	Agents []string `yaml:"agents,omitempty"`

	// Skill resource IDs to include in this blueprint.
	// +xcf:optional
	// +xcf:group=Resource Selectors
	// +xcf:type=[]string
	Skills []string `yaml:"skills,omitempty"`

	// Rule resource IDs to include in this blueprint.
	// +xcf:optional
	// +xcf:group=Resource Selectors
	// +xcf:type=[]string
	Rules []string `yaml:"rules,omitempty"`

	// Workflow resource IDs to include in this blueprint.
	// +xcf:optional
	// +xcf:group=Resource Selectors
	// +xcf:type=[]string
	Workflows []string `yaml:"workflows,omitempty"`

	// MCP server resource IDs to include in this blueprint.
	// +xcf:optional
	// +xcf:group=Resource Selectors
	// +xcf:type=[]string
	MCP []string `yaml:"mcp,omitempty"`

	// Policy resource IDs to include in this blueprint.
	// +xcf:optional
	// +xcf:group=Resource Selectors
	// +xcf:type=[]string
	Policies []string `yaml:"policies,omitempty"`

	// Memory resource IDs to include in this blueprint.
	// +xcf:optional
	// +xcf:group=Resource Selectors
	// +xcf:type=[]string
	Memory []string `yaml:"memory,omitempty"`

	// Context resource IDs to include in this blueprint.
	// +xcf:optional
	// +xcf:group=Resource Selectors
	// +xcf:type=[]string
	Contexts []string `yaml:"contexts,omitempty"`

	// Name of the settings block to use.
	// +xcf:optional
	// +xcf:group=Singleton Selectors
	Settings string `yaml:"settings,omitempty"`

	// Name of the hooks block to use.
	// +xcf:optional
	// +xcf:group=Singleton Selectors
	Hooks string `yaml:"hooks,omitempty"`

	// Compilation targets for this blueprint.
	// When set, controls which providers this blueprint compiles for.
	// When empty, falls through to project.targets or --target flag.
	// +xcf:optional
	// +xcf:group=Multi-Target
	// +xcf:role=filtering
	// +xcf:type=[]string
	Targets []string `yaml:"targets,omitempty"`

	// Inherited is set by the parser when this resource originates from an
	// extends: global base config. It is never serialized.
	Inherited bool `yaml:"-"`

	// SourceProvider identifies the provider this resource was imported from.
	// Set by the import pipeline; never serialized.
	SourceProvider string `yaml:"-" json:"-"`
}

// ResourceOverrides stores parsed .<provider>.xcf partial configs for all 9 kinds.
// Keyed as [kind][name][provider] → config struct. Populated by the import pipeline
// during filesystem scanning of <config-dir>.<provider>.xcf files.
// Never serialized; used by the compiler for provider-specific config merging.
type ResourceOverrides struct {
	Agent    map[string]map[string]AgentConfig     `json:"-"`
	Skill    map[string]map[string]SkillConfig     `json:"-"`
	Rule     map[string]map[string]RuleConfig      `json:"-"`
	Workflow map[string]map[string]WorkflowConfig  `json:"-"`
	MCP      map[string]map[string]MCPConfig       `json:"-"`
	Hooks    map[string]map[string]NamedHookConfig `json:"-"`
	Settings map[string]map[string]SettingsConfig  `json:"-"`
	Policy   map[string]map[string]PolicyConfig    `json:"-"`
	Template map[string]map[string]TemplateConfig  `json:"-"`
}

// AddAgent stores an AgentConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddAgent(name, provider string, cfg AgentConfig) {
	if r.Agent == nil {
		r.Agent = make(map[string]map[string]AgentConfig)
	}
	if r.Agent[name] == nil {
		r.Agent[name] = make(map[string]AgentConfig)
	}
	r.Agent[name][provider] = cfg
}

// GetAgent retrieves an AgentConfig override by [name][provider].
func (r *ResourceOverrides) GetAgent(name, provider string) (AgentConfig, bool) {
	if r == nil || r.Agent == nil {
		return AgentConfig{}, false
	}
	if r.Agent[name] == nil {
		return AgentConfig{}, false
	}
	cfg, ok := r.Agent[name][provider]
	return cfg, ok
}

// AgentProviders returns a sorted list of provider names for a given agent.
func (r *ResourceOverrides) AgentProviders(name string) []string {
	if r == nil || r.Agent == nil || r.Agent[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Agent[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddSkill stores a SkillConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddSkill(name, provider string, cfg SkillConfig) {
	if r.Skill == nil {
		r.Skill = make(map[string]map[string]SkillConfig)
	}
	if r.Skill[name] == nil {
		r.Skill[name] = make(map[string]SkillConfig)
	}
	r.Skill[name][provider] = cfg
}

// GetSkill retrieves a SkillConfig override by [name][provider].
func (r *ResourceOverrides) GetSkill(name, provider string) (SkillConfig, bool) {
	if r == nil || r.Skill == nil {
		return SkillConfig{}, false
	}
	if r.Skill[name] == nil {
		return SkillConfig{}, false
	}
	cfg, ok := r.Skill[name][provider]
	return cfg, ok
}

// SkillProviders returns a sorted list of provider names for a given skill.
func (r *ResourceOverrides) SkillProviders(name string) []string {
	if r == nil || r.Skill == nil || r.Skill[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Skill[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddRule stores a RuleConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddRule(name, provider string, cfg RuleConfig) {
	if r.Rule == nil {
		r.Rule = make(map[string]map[string]RuleConfig)
	}
	if r.Rule[name] == nil {
		r.Rule[name] = make(map[string]RuleConfig)
	}
	r.Rule[name][provider] = cfg
}

// GetRule retrieves a RuleConfig override by [name][provider].
func (r *ResourceOverrides) GetRule(name, provider string) (RuleConfig, bool) {
	if r == nil || r.Rule == nil {
		return RuleConfig{}, false
	}
	if r.Rule[name] == nil {
		return RuleConfig{}, false
	}
	cfg, ok := r.Rule[name][provider]
	return cfg, ok
}

// RuleProviders returns a sorted list of provider names for a given rule.
func (r *ResourceOverrides) RuleProviders(name string) []string {
	if r == nil || r.Rule == nil || r.Rule[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Rule[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddWorkflow stores a WorkflowConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddWorkflow(name, provider string, cfg WorkflowConfig) {
	if r.Workflow == nil {
		r.Workflow = make(map[string]map[string]WorkflowConfig)
	}
	if r.Workflow[name] == nil {
		r.Workflow[name] = make(map[string]WorkflowConfig)
	}
	r.Workflow[name][provider] = cfg
}

// GetWorkflow retrieves a WorkflowConfig override by [name][provider].
func (r *ResourceOverrides) GetWorkflow(name, provider string) (WorkflowConfig, bool) {
	if r == nil || r.Workflow == nil {
		return WorkflowConfig{}, false
	}
	if r.Workflow[name] == nil {
		return WorkflowConfig{}, false
	}
	cfg, ok := r.Workflow[name][provider]
	return cfg, ok
}

// WorkflowProviders returns a sorted list of provider names for a given workflow.
func (r *ResourceOverrides) WorkflowProviders(name string) []string {
	if r == nil || r.Workflow == nil || r.Workflow[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Workflow[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddMCP stores an MCPConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddMCP(name, provider string, cfg MCPConfig) {
	if r.MCP == nil {
		r.MCP = make(map[string]map[string]MCPConfig)
	}
	if r.MCP[name] == nil {
		r.MCP[name] = make(map[string]MCPConfig)
	}
	r.MCP[name][provider] = cfg
}

// GetMCP retrieves an MCPConfig override by [name][provider].
func (r *ResourceOverrides) GetMCP(name, provider string) (MCPConfig, bool) {
	if r == nil || r.MCP == nil {
		return MCPConfig{}, false
	}
	if r.MCP[name] == nil {
		return MCPConfig{}, false
	}
	cfg, ok := r.MCP[name][provider]
	return cfg, ok
}

// MCPProviders returns a sorted list of provider names for a given MCP server.
func (r *ResourceOverrides) MCPProviders(name string) []string {
	if r == nil || r.MCP == nil || r.MCP[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.MCP[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddHooks stores a NamedHookConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddHooks(name, provider string, cfg NamedHookConfig) {
	if r.Hooks == nil {
		r.Hooks = make(map[string]map[string]NamedHookConfig)
	}
	if r.Hooks[name] == nil {
		r.Hooks[name] = make(map[string]NamedHookConfig)
	}
	r.Hooks[name][provider] = cfg
}

// GetHooks retrieves a NamedHookConfig override by [name][provider].
func (r *ResourceOverrides) GetHooks(name, provider string) (NamedHookConfig, bool) {
	if r == nil || r.Hooks == nil {
		return NamedHookConfig{}, false
	}
	if r.Hooks[name] == nil {
		return NamedHookConfig{}, false
	}
	cfg, ok := r.Hooks[name][provider]
	return cfg, ok
}

// HooksProviders returns a sorted list of provider names for a given hooks block.
func (r *ResourceOverrides) HooksProviders(name string) []string {
	if r == nil || r.Hooks == nil || r.Hooks[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Hooks[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddSettings stores a SettingsConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddSettings(name, provider string, cfg SettingsConfig) {
	if r.Settings == nil {
		r.Settings = make(map[string]map[string]SettingsConfig)
	}
	if r.Settings[name] == nil {
		r.Settings[name] = make(map[string]SettingsConfig)
	}
	r.Settings[name][provider] = cfg
}

// GetSettings retrieves a SettingsConfig override by [name][provider].
func (r *ResourceOverrides) GetSettings(name, provider string) (SettingsConfig, bool) {
	if r == nil || r.Settings == nil {
		return SettingsConfig{}, false
	}
	if r.Settings[name] == nil {
		return SettingsConfig{}, false
	}
	cfg, ok := r.Settings[name][provider]
	return cfg, ok
}

// SettingsProviders returns a sorted list of provider names for a given settings block.
func (r *ResourceOverrides) SettingsProviders(name string) []string {
	if r == nil || r.Settings == nil || r.Settings[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Settings[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddPolicy stores a PolicyConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddPolicy(name, provider string, cfg PolicyConfig) {
	if r.Policy == nil {
		r.Policy = make(map[string]map[string]PolicyConfig)
	}
	if r.Policy[name] == nil {
		r.Policy[name] = make(map[string]PolicyConfig)
	}
	r.Policy[name][provider] = cfg
}

// GetPolicy retrieves a PolicyConfig override by [name][provider].
func (r *ResourceOverrides) GetPolicy(name, provider string) (PolicyConfig, bool) {
	if r == nil || r.Policy == nil {
		return PolicyConfig{}, false
	}
	if r.Policy[name] == nil {
		return PolicyConfig{}, false
	}
	cfg, ok := r.Policy[name][provider]
	return cfg, ok
}

// PolicyProviders returns a sorted list of provider names for a given policy.
func (r *ResourceOverrides) PolicyProviders(name string) []string {
	if r == nil || r.Policy == nil || r.Policy[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Policy[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// AddTemplate stores a TemplateConfig override keyed by [name][provider].
func (r *ResourceOverrides) AddTemplate(name, provider string, cfg TemplateConfig) {
	if r.Template == nil {
		r.Template = make(map[string]map[string]TemplateConfig)
	}
	if r.Template[name] == nil {
		r.Template[name] = make(map[string]TemplateConfig)
	}
	r.Template[name][provider] = cfg
}

// GetTemplate retrieves a TemplateConfig override by [name][provider].
func (r *ResourceOverrides) GetTemplate(name, provider string) (TemplateConfig, bool) {
	if r == nil || r.Template == nil {
		return TemplateConfig{}, false
	}
	if r.Template[name] == nil {
		return TemplateConfig{}, false
	}
	cfg, ok := r.Template[name][provider]
	return cfg, ok
}

// TemplateProviders returns a sorted list of provider names for a given template.
func (r *ResourceOverrides) TemplateProviders(name string) []string {
	if r == nil || r.Template == nil || r.Template[name] == nil {
		return nil
	}
	var providers []string
	for p := range r.Template[name] {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
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
	for k, v := range c.Hooks {
		if v.Inherited {
			delete(c.Hooks, k)
		}
	}
	for k, v := range c.Settings {
		if v.Inherited {
			delete(c.Settings, k)
		}
	}
	// Hooks and Settings use additive/override merge semantics — the parser
	// does not currently mark them as Inherited during extends resolution.
	// The loops above are forward-compatible for provider import pipelines
	// that will set Inherited on provider-sourced entries.
	//
	// Memory and Policies are convention-based or import-only; their Inherited
	// fields are reserved for the import pipeline, not extends resolution.
}
