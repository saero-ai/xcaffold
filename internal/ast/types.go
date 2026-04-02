package ast

// XcaffoldConfig is the root structure of a parsed .xcf YAML file.
type XcaffoldConfig struct {
	Version string                 `yaml:"version"`
	Project ProjectConfig          `yaml:"project"`
	Agents  map[string]AgentConfig `yaml:"agents,omitempty"`
	Skills  map[string]SkillConfig `yaml:"skills,omitempty"`
	Rules   map[string]RuleConfig  `yaml:"rules,omitempty"`
	Hooks   map[string]HookConfig  `yaml:"hooks,omitempty"`
	MCP     map[string]MCPConfig   `yaml:"mcp,omitempty"`
}

// ProjectConfig holds project-level metadata.
type ProjectConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// AgentConfig defines a Claude agent persona.
type AgentConfig struct {
	Description  string   `yaml:"description,omitempty"`
	Instructions string   `yaml:"instructions,omitempty"`
	Model        string   `yaml:"model,omitempty"`
	Effort       string   `yaml:"effort,omitempty"`
	Tools        []string `yaml:"tools,omitempty"`
	BlockedTools []string `yaml:"blocked_tools,omitempty"`
	Skills       []string `yaml:"skills,omitempty"`
	Rules        []string `yaml:"rules,omitempty"`
	Mode         string   `yaml:"mode,omitempty"`
	When         string   `yaml:"when,omitempty"`
	MCP          []string `yaml:"mcp,omitempty"`

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
	Description  string   `yaml:"description,omitempty"`
	Instructions string   `yaml:"instructions,omitempty"`
	Paths        []string `yaml:"paths,omitempty"`
	Tools        []string `yaml:"tools,omitempty"`
}

// RuleConfig defines a path-gated formatting guideline.
type RuleConfig struct {
	Paths        []string `yaml:"paths,omitempty"`
	Instructions string   `yaml:"instructions,omitempty"`
}

// HookConfig defines a lifecycle event hook.
type HookConfig struct {
	Event string `yaml:"event,omitempty"`
	Match string `yaml:"match,omitempty"`
	Run   string `yaml:"run,omitempty"`
}

// MCPConfig defines a local MCP server context.
type MCPConfig struct {
	Command string            `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}
