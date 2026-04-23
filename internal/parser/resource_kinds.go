package parser

import (
	"bytes"
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// kindHeader is a lightweight struct for extracting the kind discriminator
// from a YAML document without decoding the full resource.
type kindHeader struct {
	Kind    string `yaml:"kind"`
	Version string `yaml:"version"`
	Name    string `yaml:"name"`
}

// agentDocument wraps AgentConfig with envelope fields for multi-kind parsing.
// KnownFields(true) validates both envelope and agent-specific fields.
// Name is not redeclared here — it is promoted from AgentConfig.Name to avoid
// a duplicate yaml:"name" tag conflict with yaml.v3.
type agentDocument struct {
	Kind            string `yaml:"kind"`
	Version         string `yaml:"version"`
	ast.AgentConfig `yaml:",inline"`
}

// skillDocument wraps SkillConfig with envelope fields.
// Name is promoted from SkillConfig.Name.
type skillDocument struct {
	Kind            string `yaml:"kind"`
	Version         string `yaml:"version"`
	ast.SkillConfig `yaml:",inline"`
}

// ruleDocument wraps RuleConfig with envelope fields.
// Name is promoted from RuleConfig.Name.
type ruleDocument struct {
	Kind           string `yaml:"kind"`
	Version        string `yaml:"version"`
	ast.RuleConfig `yaml:",inline"`
}

// workflowDocument wraps WorkflowConfig with envelope fields.
// Name is promoted from WorkflowConfig.Name.
type workflowDocument struct {
	Kind               string `yaml:"kind"`
	Version            string `yaml:"version"`
	ast.WorkflowConfig `yaml:",inline"`
}

// mcpDocument wraps MCPConfig with envelope fields.
// Name is promoted from MCPConfig.Name.
type mcpDocument struct {
	Kind          string `yaml:"kind"`
	Version       string `yaml:"version"`
	ast.MCPConfig `yaml:",inline"`
}

// projectDocFields is the deserialization target for kind: project documents.
// It does NOT embed ResourceScope, so "agents" maps to []string (ref list)
// without colliding with ResourceScope's map[string]AgentConfig.
type projectDocFields struct {
	Kind         string                   `yaml:"kind"`
	Version      string                   `yaml:"version"`
	Name         string                   `yaml:"name"`
	Description  string                   `yaml:"description,omitempty"`
	Author       string                   `yaml:"author,omitempty"`
	Homepage     string                   `yaml:"homepage,omitempty"`
	Repository   string                   `yaml:"repository,omitempty"`
	License      string                   `yaml:"license,omitempty"`
	BackupDir    string                   `yaml:"backup-dir,omitempty"`
	Targets      []string                 `yaml:"targets,omitempty"`
	AgentRefs    []ast.AgentManifestEntry `yaml:"agents,omitempty"`
	SkillRefs    []string                 `yaml:"skills,omitempty"`
	RuleRefs     []string                 `yaml:"rules,omitempty"`
	WorkflowRefs []string                 `yaml:"workflows,omitempty"`
	MCPRefs      []string                 `yaml:"mcp,omitempty"`
	PolicyRefs   []string                 `yaml:"policies,omitempty"`
	Test         ast.TestConfig           `yaml:"test,omitempty"`
	Local        ast.SettingsConfig       `yaml:"local,omitempty"`

	// Instructions fields — A-3: KnownFields entries.
	// yaml.KnownFields(true) enforces these recursively through nested types.
	Instructions        string                  `yaml:"instructions,omitempty"`
	InstructionsFile    string                  `yaml:"instructions-file,omitempty"`
	InstructionsImports []string                `yaml:"instructions-imports,omitempty"`
	InstructionsScopes  []ast.InstructionsScope `yaml:"instructions-scopes,omitempty"`
	// TargetOptions holds per-provider compile-time options.
	TargetOptions map[string]ast.TargetOverride `yaml:"target-options,omitempty"`
}

// hooksDocument wraps HookConfig with envelope fields for kind: hooks.
// HookConfig is a map type, so it cannot be inlined; the "events" field
// wraps it at the YAML level.
type hooksDocument struct {
	Kind    string         `yaml:"kind"`
	Version string         `yaml:"version"`
	Name    string         `yaml:"name,omitempty"`
	Events  ast.HookConfig `yaml:"events"`
}

// settingsDocument wraps SettingsConfig with envelope fields for kind: settings.
// SettingsConfig is a struct, so it inlines cleanly.
type settingsDocument struct {
	Kind               string `yaml:"kind"`
	Version            string `yaml:"version"`
	ast.SettingsConfig `yaml:",inline"`
}

// globalDocument wraps global-scope fields for kind: global parsing.
// Global configs have no project metadata -- only resources and settings.
type globalDocument struct {
	Kind      string                        `yaml:"kind"`
	Version   string                        `yaml:"version"`
	Extends   string                        `yaml:"extends,omitempty"`
	Settings  ast.SettingsConfig            `yaml:"settings,omitempty"`
	Agents    map[string]ast.AgentConfig    `yaml:"agents,omitempty"`
	Skills    map[string]ast.SkillConfig    `yaml:"skills,omitempty"`
	Rules     map[string]ast.RuleConfig     `yaml:"rules,omitempty"`
	Hooks     ast.HookConfig                `yaml:"hooks,omitempty"`
	MCP       map[string]ast.MCPConfig      `yaml:"mcp,omitempty"`
	Workflows map[string]ast.WorkflowConfig `yaml:"workflows,omitempty"`
	Memory    map[string]ast.MemoryConfig   `yaml:"memory,omitempty"`
}

// policyDocument wraps PolicyConfig with envelope fields for multi-kind parsing.
// Name is promoted from PolicyConfig.Name.
type policyDocument struct {
	Kind             string `yaml:"kind"`
	Version          string `yaml:"version"`
	ast.PolicyConfig `yaml:",inline"`
}

// referenceDocument wraps ReferenceConfig with envelope fields for kind: reference parsing.
// Name is promoted from ReferenceConfig.Name.
type referenceDocument struct {
	Kind                string `yaml:"kind"`
	Version             string `yaml:"version"`
	ast.ReferenceConfig `yaml:",inline"`
}

// blueprintDocument wraps BlueprintConfig with envelope fields for kind: blueprint parsing.
// Name is promoted from BlueprintConfig.Name.
type blueprintDocument struct {
	Kind                string `yaml:"kind"`
	Version             string `yaml:"version"`
	ast.BlueprintConfig `yaml:",inline"`
}

// isValidResourceName checks that a name contains only lowercase letters,
// digits, and hyphens. Empty names are rejected.
func isValidResourceName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}

// parseBlueprintDocument decodes a yaml.Node into a blueprintDocument with
// KnownFields validation, validates envelope fields and name constraints,
// and inserts the resource into config.Blueprints.
func parseBlueprintDocument(node *yaml.Node, config *ast.XcaffoldConfig) error {
	b, err := nodeToBytes(node)
	if err != nil {
		return fmt.Errorf("failed to marshal blueprint document: %w", err)
	}
	var doc blueprintDocument
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("invalid blueprint document: %w", err)
	}
	if err := validateEnvelope(doc.Version, doc.Name, "blueprint"); err != nil {
		return err
	}
	if !isValidResourceName(doc.Name) {
		return fmt.Errorf("blueprint name %q is invalid: must contain only lowercase letters, digits, and hyphens", doc.Name)
	}
	if config.Blueprints == nil {
		config.Blueprints = make(map[string]ast.BlueprintConfig)
	}
	if _, exists := config.Blueprints[doc.Name]; exists {
		return fmt.Errorf("duplicate blueprint name %q", doc.Name)
	}
	config.Blueprints[doc.Name] = doc.BlueprintConfig
	return nil
}

// parseReferenceDocument decodes a yaml.Node into a referenceDocument with
// KnownFields validation, validates envelope fields, and inserts the resource
// into config.References.
func parseReferenceDocument(node *yaml.Node, config *ast.XcaffoldConfig) error {
	b, err := nodeToBytes(node)
	if err != nil {
		return fmt.Errorf("failed to marshal reference document: %w", err)
	}
	var doc referenceDocument
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("invalid reference document: %w", err)
	}
	if err := validateEnvelope(doc.Version, doc.Name, "reference"); err != nil {
		return err
	}
	if config.References == nil {
		config.References = make(map[string]ast.ReferenceConfig)
	}
	if _, exists := config.References[doc.Name]; exists {
		return fmt.Errorf("duplicate reference ID %q", doc.Name)
	}
	config.References[doc.Name] = doc.ReferenceConfig
	return nil
}

// extractKind reads the "kind" value from a yaml.Node MappingNode
// without decoding the full document.
func extractKind(node *yaml.Node) string {
	return extractScalarField(node, "kind")
}

// extractVersion reads the "version" value from a yaml.Node MappingNode
// without decoding the full document.
func extractVersion(node *yaml.Node) string {
	return extractScalarField(node, "version")
}

// extractScalarField returns the string value of a named scalar key in a
// MappingNode. Returns "" when the node is not a MappingNode or the key is
// absent.
func extractScalarField(node *yaml.Node, key string) string {
	if node.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1].Value
		}
	}
	return ""
}

// nodeToBytes marshals a yaml.Node back to YAML bytes for re-decoding
// with kind-specific KnownFields validation.
func nodeToBytes(node *yaml.Node) ([]byte, error) {
	return yaml.Marshal(node)
}

// singletonKinds are resource kinds that do not require a name field.
var singletonKinds = map[string]bool{
	"hooks":    true,
	"settings": true,
	"global":   true,
}

// validatePolicyFields checks semantic constraints beyond KnownFields.
func validatePolicyFields(p ast.PolicyConfig) error {
	validSeverities := map[string]bool{
		"error":   true,
		"warning": true,
		"off":     true,
	}
	if !validSeverities[p.Severity] {
		return fmt.Errorf("policy %q: severity must be \"error\", \"warning\", or \"off\", got %q", p.Name, p.Severity)
	}

	validTargets := map[string]bool{
		"agent":    true,
		"skill":    true,
		"rule":     true,
		"hook":     true,
		"settings": true,
		"output":   true,
	}
	if !validTargets[p.Target] {
		return fmt.Errorf("policy %q: target must be one of agent, skill, rule, hook, settings, output; got %q", p.Name, p.Target)
	}

	for i, r := range p.Require {
		if r.Field == "" {
			return fmt.Errorf("policy %q: require[%d].field is required", p.Name, i)
		}
	}

	for i, d := range p.Deny {
		if len(d.ContentContains) == 0 && d.ContentMatches == "" && d.PathContains == "" {
			return fmt.Errorf("policy %q: deny[%d] must specify at least one of content-contains, content-matches, or path-contains", p.Name, i)
		}
	}

	return nil
}

// validateEnvelope checks that mandatory envelope fields are present on a
// resource kind document. Singleton kinds (hooks, settings) skip the name check.
func validateEnvelope(version, name, kind string) error {
	if !singletonKinds[kind] && name == "" {
		return fmt.Errorf("%s document: name is required", kind)
	}
	if version == "" {
		return fmt.Errorf("%s document: version is required", kind)
	}
	return nil
}

// parseResourceDocument decodes a yaml.Node into a kind-specific wrapper
// struct with KnownFields validation, validates envelope fields (name and
// version required), and inserts the resource into the appropriate
// ResourceScope map on config.
func parseResourceDocument(node *yaml.Node, kind string, config *ast.XcaffoldConfig, sourceFile string) error {
	b, err := nodeToBytes(node)
	if err != nil {
		return fmt.Errorf("failed to marshal %s document: %w", kind, err)
	}

	switch kind {
	case "agent":
		var doc agentDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid agent document: %w", err)
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if config.Agents == nil {
			config.Agents = make(map[string]ast.AgentConfig)
		}
		if _, exists := config.Agents[doc.Name]; exists {
			return fmt.Errorf("duplicate agent ID %q", doc.Name)
		}
		config.Agents[doc.Name] = doc.AgentConfig

	case "skill":
		var doc skillDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid skill document: %w", err)
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if config.Skills == nil {
			config.Skills = make(map[string]ast.SkillConfig)
		}
		if _, exists := config.Skills[doc.Name]; exists {
			return fmt.Errorf("duplicate skill ID %q", doc.Name)
		}
		config.Skills[doc.Name] = doc.SkillConfig

	case "rule":
		var doc ruleDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid rule document: %w", err)
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if config.Rules == nil {
			config.Rules = make(map[string]ast.RuleConfig)
		}
		if _, exists := config.Rules[doc.Name]; exists {
			return fmt.Errorf("duplicate rule ID %q", doc.Name)
		}
		config.Rules[doc.Name] = doc.RuleConfig

	case "workflow":
		var doc workflowDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid workflow document: %w", err)
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if config.Workflows == nil {
			config.Workflows = make(map[string]ast.WorkflowConfig)
		}
		if _, exists := config.Workflows[doc.Name]; exists {
			return fmt.Errorf("duplicate workflow ID %q", doc.Name)
		}
		config.Workflows[doc.Name] = doc.WorkflowConfig

	case "mcp":
		var doc mcpDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid mcp document: %w", err)
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if config.MCP == nil {
			config.MCP = make(map[string]ast.MCPConfig)
		}
		if _, exists := config.MCP[doc.Name]; exists {
			return fmt.Errorf("duplicate mcp ID %q", doc.Name)
		}
		config.MCP[doc.Name] = doc.MCPConfig

	case "project":
		var doc projectDocFields
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid project document: %w", err)
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if config.Project == nil {
			config.Project = &ast.ProjectConfig{}
		}
		config.Project.Name = doc.Name
		config.Project.Description = doc.Description
		config.Project.Author = doc.Author
		config.Project.Homepage = doc.Homepage
		config.Project.Repository = doc.Repository
		config.Project.License = doc.License
		config.Project.BackupDir = doc.BackupDir
		config.Project.Targets = doc.Targets
		config.Project.AgentRefs = doc.AgentRefs
		config.Project.SkillRefs = doc.SkillRefs
		config.Project.RuleRefs = doc.RuleRefs
		config.Project.WorkflowRefs = doc.WorkflowRefs
		config.Project.MCPRefs = doc.MCPRefs
		config.Project.PolicyRefs = doc.PolicyRefs
		config.Project.Test = doc.Test
		config.Project.Local = doc.Local
		config.Project.Instructions = doc.Instructions
		config.Project.InstructionsFile = doc.InstructionsFile
		config.Project.InstructionsImports = doc.InstructionsImports
		config.Project.InstructionsScopes = doc.InstructionsScopes
		config.Project.TargetOptions = doc.TargetOptions

	case "hooks":
		var doc hooksDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid hooks document: %w", err)
		}
		if err := validateEnvelope(doc.Version, "", kind); err != nil {
			return err
		}
		name := doc.Name
		if name == "" {
			name = "default"
		}
		if config.Hooks == nil {
			config.Hooks = make(map[string]ast.NamedHookConfig)
		}
		existing, ok := config.Hooks[name]
		if !ok {
			existing = ast.NamedHookConfig{Name: name}
		}
		if existing.Events == nil {
			existing.Events = make(ast.HookConfig)
		}
		for event, groups := range doc.Events {
			existing.Events[event] = append(existing.Events[event], groups...)
		}
		config.Hooks[name] = existing

	case "settings":
		var doc settingsDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid settings document: %w", err)
		}
		if err := validateEnvelope(doc.Version, "", kind); err != nil {
			return err
		}
		name := doc.SettingsConfig.Name
		if name == "" {
			name = "default"
			doc.SettingsConfig.Name = name
		}
		if config.Settings == nil {
			config.Settings = make(map[string]ast.SettingsConfig)
		}
		config.Settings[name] = doc.SettingsConfig

	case "global":
		var doc globalDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid global document: %w", err)
		}
		if err := validateEnvelope(doc.Version, "", kind); err != nil {
			return err
		}
		config.Extends = doc.Extends
		if config.Settings == nil {
			config.Settings = make(map[string]ast.SettingsConfig)
		}
		sName := doc.Settings.Name
		if sName == "" {
			sName = "default"
		}
		config.Settings[sName] = doc.Settings
		if doc.Hooks != nil {
			if config.Hooks == nil {
				config.Hooks = make(map[string]ast.NamedHookConfig)
			}
			existing, ok := config.Hooks["default"]
			if !ok {
				existing = ast.NamedHookConfig{Name: "default"}
			}
			if existing.Events == nil {
				existing.Events = make(ast.HookConfig)
			}
			for event, groups := range doc.Hooks {
				existing.Events[event] = append(existing.Events[event], groups...)
			}
			config.Hooks["default"] = existing
		}
		for k, v := range doc.Agents {
			if config.Agents == nil {
				config.Agents = make(map[string]ast.AgentConfig)
			}
			if _, exists := config.Agents[k]; exists {
				return fmt.Errorf("duplicate agent ID %q", k)
			}
			config.Agents[k] = v
		}
		for k, v := range doc.Skills {
			if config.Skills == nil {
				config.Skills = make(map[string]ast.SkillConfig)
			}
			if _, exists := config.Skills[k]; exists {
				return fmt.Errorf("duplicate skill ID %q", k)
			}
			config.Skills[k] = v
		}
		for k, v := range doc.Rules {
			if config.Rules == nil {
				config.Rules = make(map[string]ast.RuleConfig)
			}
			if _, exists := config.Rules[k]; exists {
				return fmt.Errorf("duplicate rule ID %q", k)
			}
			config.Rules[k] = v
		}
		for k, v := range doc.Workflows {
			if config.Workflows == nil {
				config.Workflows = make(map[string]ast.WorkflowConfig)
			}
			if _, exists := config.Workflows[k]; exists {
				return fmt.Errorf("duplicate workflow ID %q", k)
			}
			config.Workflows[k] = v
		}
		for k, v := range doc.MCP {
			if config.MCP == nil {
				config.MCP = make(map[string]ast.MCPConfig)
			}
			if _, exists := config.MCP[k]; exists {
				return fmt.Errorf("duplicate mcp ID %q", k)
			}
			config.MCP[k] = v
		}
		for k, v := range doc.Memory {
			if config.Memory == nil {
				config.Memory = make(map[string]ast.MemoryConfig)
			}
			if _, exists := config.Memory[k]; exists {
				return fmt.Errorf("duplicate memory ID %q", k)
			}
			config.Memory[k] = v
		}

	case "policy":
		var doc policyDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return fmt.Errorf("invalid policy document: %w", err)
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if err := validatePolicyFields(doc.PolicyConfig); err != nil {
			return err
		}
		if config.Policies == nil {
			config.Policies = make(map[string]ast.PolicyConfig)
		}
		if _, exists := config.Policies[doc.Name]; exists {
			return fmt.Errorf("duplicate policy ID %q", doc.Name)
		}
		config.Policies[doc.Name] = doc.PolicyConfig

	default:
		return fmt.Errorf("unknown resource kind %q", kind)
	}

	return nil
}
