package ast

import (
	"gopkg.in/yaml.v3"
)

// ResourceScope contains all agentic primitives that can appear at both
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

// XcaffoldConfig is the root structure of a parsed .xcaf YAML file.
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

	// Overrides stores parsed .<provider>.xcaf partial configs keyed by [kind][name][provider].
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
	// +xcaf:optional
	// +xcaf:type=[]string
	AllowedEnvVars []string `yaml:"allowed-env-vars,omitempty" json:"allowedEnvVars,omitempty"`
	// Targets lists the compilation targets for this project.
	// Populated by the parser when decoding kind: project documents.
	Targets []string `yaml:"-"`

	Test TestConfig `yaml:"test,omitempty"`

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
