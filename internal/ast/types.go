package ast

// XcaffoldConfig is the root structure of a parsed .xcf YAML file.
type XcaffoldConfig struct {
	Extends  string                 `yaml:"extends,omitempty"`
	Version  string                 `yaml:"version"`
	Project  ProjectConfig          `yaml:"project"`
	Agents   map[string]AgentConfig `yaml:"agents,omitempty"`
	Skills   map[string]SkillConfig `yaml:"skills,omitempty"`
	Rules    map[string]RuleConfig  `yaml:"rules,omitempty"`
	Hooks    map[string]HookConfig  `yaml:"hooks,omitempty"`
	MCP      map[string]MCPConfig   `yaml:"mcp,omitempty"`
	Settings SettingsConfig         `yaml:"settings,omitempty"`
	Test     TestConfig             `yaml:"test,omitempty"`
}

// ProjectConfig holds project-level metadata.
type ProjectConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// AgentConfig defines a Claude agent persona.
type AgentConfig struct {
	// Name is the display name shown in the Claude Code UI.
	Name string `yaml:"name,omitempty"`

	Description string `yaml:"description,omitempty"`

	// Instructions is the inline system prompt. Mutually exclusive with InstructionsFile.
	Instructions string `yaml:"instructions,omitempty"`

	// InstructionsFile is a path (relative to scaffold.xcf) to an external markdown
	// file whose body (after stripping frontmatter) is used as the system prompt.
	// Mutually exclusive with Instructions.
	InstructionsFile string `yaml:"instructions_file,omitempty"`

	Model  string `yaml:"model,omitempty"`
	Effort string `yaml:"effort,omitempty"`

	// Memory controls the persistent memory scope (e.g. "project", "user").
	Memory string `yaml:"memory,omitempty"`

	// MaxTurns is the maximum number of conversation turns before auto-stop.
	MaxTurns int `yaml:"maxTurns,omitempty"`

	Tools        []string `yaml:"tools,omitempty"`
	BlockedTools []string `yaml:"blocked_tools,omitempty"`
	Skills       []string `yaml:"skills,omitempty"`
	Rules        []string `yaml:"rules,omitempty"`
	Mode         string   `yaml:"mode,omitempty"`
	When         string   `yaml:"when,omitempty"`
	MCP          []string `yaml:"mcp,omitempty"`

	// Assertions are statements the LLM-as-a-Judge evaluates against the
	// execution trace when running `xcaffold test --judge`.
	Assertions []string `yaml:"assertions,omitempty"`

	// Experimental: target-specific configuration overrides.
	Targets map[string]TargetOverride `yaml:"targets,omitempty"`
}

// TargetOverride specifies overrides for multi-provider targets.
type TargetOverride struct {
	Hooks                map[string]string `yaml:"hooks,omitempty"`
	InstructionsOverride string            `yaml:"instructions_override,omitempty"`
}

// SkillConfig defines a reusable prompt package.
type SkillConfig struct {
	// Name is the display name in the skill's SKILL.md frontmatter.
	Name string `yaml:"name,omitempty"`

	// Type is the skill type (e.g. "reference").
	Type string `yaml:"type,omitempty"`

	Description string `yaml:"description,omitempty"`

	// Instructions is the inline skill body. Mutually exclusive with InstructionsFile.
	Instructions string `yaml:"instructions,omitempty"`

	// InstructionsFile is a path (relative to scaffold.xcf) to an external markdown
	// file whose body is used as the skill body. Mutually exclusive with Instructions.
	InstructionsFile string `yaml:"instructions_file,omitempty"`

	Paths []string `yaml:"paths,omitempty"`

	// Tools and AllowedTools are alternative tool-specification formats.
	Tools        []string `yaml:"tools,omitempty"`
	AllowedTools []string `yaml:"allowed-tools,omitempty"`

	// References is a list of supplementary file paths (relative to scaffold.xcf)
	// to copy into skills/<id>/references/. Glob patterns are supported (e.g. "refs/*.md").
	References []string `yaml:"references,omitempty"`
}

// RuleConfig defines a path-gated formatting guideline.
type RuleConfig struct {
	Description string   `yaml:"description,omitempty"`
	Paths       []string `yaml:"paths,omitempty"`

	// Instructions is the inline rule body. Mutually exclusive with InstructionsFile.
	Instructions string `yaml:"instructions,omitempty"`

	// InstructionsFile is a path (relative to scaffold.xcf) to an external markdown
	// file whose body is used as the rule body. Mutually exclusive with Instructions.
	InstructionsFile string `yaml:"instructions_file,omitempty"`
}

// HookConfig defines a lifecycle event hook.
type HookConfig struct {
	Event string `yaml:"event,omitempty" json:"event,omitempty"`
	Match string `yaml:"match,omitempty" json:"match,omitempty"`
	Run   string `yaml:"run,omitempty"   json:"run,omitempty"`
}

// MCPConfig defines a local MCP server context.
type MCPConfig struct {
	Command string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"    json:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"     json:"env,omitempty"`
}

// StatusLineConfig defines the statusLine setting for Claude Code.
// The original format is {"type": "command", "command": "<shell cmd>"}.
type StatusLineConfig struct {
	Type    string `yaml:"type,omitempty"    json:"type,omitempty"`
	Command string `yaml:"command,omitempty" json:"command,omitempty"`
}

// SettingsConfig represents the full Claude Code settings.json structure.
// The mcp: block at the top level of XcaffoldConfig is a convenience shorthand
// that gets merged into the mcpServers key during compilation. Fields defined
// here take precedence over the shorthand for mcpServers if both are set.
type SettingsConfig struct {
	// Env specifies environment variables injected into Claude Code sessions.
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`

	// StatusLine is a custom status bar command shown in the Claude Code UI.
	// Format: {type: "command", command: "bash script.sh"}
	StatusLine *StatusLineConfig `yaml:"statusLine,omitempty" json:"statusLine,omitempty"`

	// EnabledPlugins maps plugin IDs to enabled/disabled state.
	EnabledPlugins map[string]bool `yaml:"enabledPlugins,omitempty" json:"enabledPlugins,omitempty"`

	// AlwaysThinkingEnabled forces extended thinking mode on all requests.
	AlwaysThinkingEnabled bool `yaml:"alwaysThinkingEnabled,omitempty" json:"alwaysThinkingEnabled,omitempty"`

	// EffortLevel is the default effort level (e.g. "high", "medium", "low").
	EffortLevel string `yaml:"effortLevel,omitempty" json:"effortLevel,omitempty"`

	// SkipDangerousModePermissionPrompt suppresses the dangerous-mode consent dialog.
	SkipDangerousModePermissionPrompt bool `yaml:"skipDangerousModePermissionPrompt,omitempty" json:"skipDangerousModePermissionPrompt,omitempty"`

	// Permissions defines the MCP allow/deny permission rules.
	Permissions map[string]interface{} `yaml:"permissions,omitempty" json:"permissions,omitempty"`

	// MCPServers allows direct specification of MCP server configurations
	// within the settings block. Merged with top-level mcp: (settings wins on conflict).
	MCPServers map[string]MCPConfig `yaml:"mcpServers,omitempty" json:"mcpServers,omitempty"`
}

// TestConfig holds project-level configuration for `xcaffold test`.
type TestConfig struct {
	// ClaudePath is the path to the claude binary. Defaults to "claude" on $PATH.
	ClaudePath string `yaml:"claude_path,omitempty"`

	// JudgeModel is the Anthropic model used for LLM-as-a-Judge evaluation.
	JudgeModel string `yaml:"judge_model,omitempty"`
}
