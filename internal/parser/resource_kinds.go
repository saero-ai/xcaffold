package parser

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

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
type contextDocument struct {
	Kind              string `yaml:"kind"`
	Version           string `yaml:"version"`
	ast.ContextConfig `yaml:",inline"`
}

type memoryDocument struct {
	Kind             string `yaml:"kind"`
	Version          string `yaml:"version"`
	ast.MemoryConfig `yaml:",inline"`
}

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
type projectDocFields struct {
	Kind           string                        `yaml:"kind"`
	Version        string                        `yaml:"version"`
	Extends        string                        `yaml:"extends,omitempty"`
	Name           string                        `yaml:"name"`
	Description    string                        `yaml:"description,omitempty"`
	Author         string                        `yaml:"author,omitempty"`
	Homepage       string                        `yaml:"homepage,omitempty"`
	Repository     string                        `yaml:"repository,omitempty"`
	License        string                        `yaml:"license,omitempty"`
	BackupDir      string                        `yaml:"backup-dir,omitempty"`
	AllowedEnvVars []string                      `yaml:"allowed-env-vars,omitempty"`
	Targets        []string                      `yaml:"targets,omitempty"`
	Test           ast.TestConfig                `yaml:"test,omitempty"`
	Local          ast.SettingsConfig            `yaml:"local,omitempty"`
	TargetOptions  map[string]ast.TargetOverride `yaml:"target-options,omitempty"`
}

// hooksDocument wraps HookConfig with envelope fields for kind: hooks.
// HookConfig is a map type, so it cannot be inlined; the "events" field
// wraps it at the YAML level.
type hooksDocument struct {
	Kind        string                        `yaml:"kind"`
	Version     string                        `yaml:"version"`
	Name        string                        `yaml:"name,omitempty"`
	Description string                        `yaml:"description,omitempty"`
	Artifacts   []string                      `yaml:"artifacts,omitempty"`
	Targets     map[string]ast.TargetOverride `yaml:"targets,omitempty"`
	Events      ast.HookConfig                `yaml:"events"`
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
}

// policyDocument wraps PolicyConfig with envelope fields for multi-kind parsing.
// Name is promoted from PolicyConfig.Name.
type policyDocument struct {
	Kind             string `yaml:"kind"`
	Version          string `yaml:"version"`
	ast.PolicyConfig `yaml:",inline"`
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
		return wrapDecodeError("blueprint", err)
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

// wrapDecodeError improves YAML decoding errors with context about
// malformed frontmatter. If the error indicates a string was found where
// YAML structure was expected, it suggests the user may have content
// before the opening --- delimiter.
func wrapDecodeError(kind string, err error) error {
	if strings.Contains(err.Error(), "cannot unmarshal !!str") {
		return fmt.Errorf("invalid %s document: file contains text where YAML was expected — "+
			"check that the file starts with '---' and body text appears after the closing '---': %w", kind, err)
	}
	return fmt.Errorf("invalid %s document: %w", kind, err)
}

// parseResourceDocument decodes a yaml.Node into a kind-specific wrapper
// struct with KnownFields validation, validates envelope fields (name and
// version required), and inserts the resource into the appropriate
// ResourceScope map on config.
// inferredName is the name inferred from the filesystem path when omitted from YAML.
func parseResourceDocument(node *yaml.Node, kind string, config *ast.XcaffoldConfig, sourceFile string, inferredName string) error {
	b, err := nodeToBytes(node)
	if err != nil {
		return fmt.Errorf("failed to marshal %s document: %w", kind, err)
	}

	switch kind {
	case "agent":
		// Guard: reject kind:agent files placed directly under xcf/agents/.
		// They must be in a subdirectory: xcf/agents/<id>/<file>.xcf.
		if sourceFile != "" {
			parentDir := filepath.Base(filepath.Dir(sourceFile))
			if parentDir == "agents" {
				return fmt.Errorf("agent file %q must be in a subdirectory under xcf/agents/<id>/; flat files are not supported", filepath.Base(sourceFile))
			}
		}
		var doc agentDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return wrapDecodeError("agent", err)
		}
		// Warn if YAML name differs from inferred name (when both are present)
		if doc.Name != "" && inferredName != "" && doc.Name != inferredName {
			config.ParseWarnings = append(config.ParseWarnings, fmt.Sprintf("%s declares name: %q but path implies name: %q", sourceFile, doc.Name, inferredName))
		}
		// Filesystem-as-schema inference: use inferred name if YAML name is empty
		wasInferred := false
		if doc.Name == "" && inferredName != "" {
			doc.Name = inferredName
			doc.AgentConfig.Name = inferredName
			wasInferred = true
		}
		// Validate inferred name against ID rules
		if wasInferred {
			if err := validateID(kind, inferredName); err != nil {
				return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
			}
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
			return wrapDecodeError("skill", err)
		}
		// Warn if YAML name differs from inferred name (when both are present)
		if doc.Name != "" && inferredName != "" && doc.Name != inferredName {
			config.ParseWarnings = append(config.ParseWarnings, fmt.Sprintf("%s declares name: %q but path implies name: %q", sourceFile, doc.Name, inferredName))
		}
		// Filesystem-as-schema inference: use inferred name if YAML name is empty
		wasInferred := false
		if doc.Name == "" && inferredName != "" {
			doc.Name = inferredName
			doc.SkillConfig.Name = inferredName
			wasInferred = true
		}
		// Validate inferred name against ID rules
		if wasInferred {
			if err := validateID(kind, inferredName); err != nil {
				return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
			}
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

		// Post-parse: migrate legacy subdirectory fields to artifacts
		skill := doc.SkillConfig
		for _, legacy := range []struct {
			field []string
			name  string
		}{
			{skill.References.Values, "references"},
			{skill.Scripts.Values, "scripts"},
			{skill.Assets.Values, "assets"},
			{skill.Examples.Values, "examples"},
		} {
			if len(legacy.field) > 0 && !slices.Contains(skill.Artifacts, legacy.name) {
				skill.Artifacts = append(skill.Artifacts, legacy.name)
			}
		}

		config.Skills[doc.Name] = skill

	case "rule":
		var doc ruleDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return wrapDecodeError("rule", err)
		}
		// Warn if YAML name differs from inferred name (when both are present)
		if doc.Name != "" && inferredName != "" && doc.Name != inferredName {
			config.ParseWarnings = append(config.ParseWarnings, fmt.Sprintf("%s declares name: %q but path implies name: %q", sourceFile, doc.Name, inferredName))
		}
		// Filesystem-as-schema inference: use inferred name if YAML name is empty
		wasInferred := false
		if doc.Name == "" && inferredName != "" {
			doc.Name = inferredName
			doc.RuleConfig.Name = inferredName
			wasInferred = true
		}
		// Validate inferred name against ID rules
		if wasInferred {
			if err := validateID(kind, inferredName); err != nil {
				return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
			}
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
			return wrapDecodeError("workflow", err)
		}
		// Warn if YAML name differs from inferred name (when both are present)
		if doc.Name != "" && inferredName != "" && doc.Name != inferredName {
			config.ParseWarnings = append(config.ParseWarnings, fmt.Sprintf("%s declares name: %q but path implies name: %q", sourceFile, doc.Name, inferredName))
		}
		// Filesystem-as-schema inference: use inferred name if YAML name is empty
		wasInferred := false
		if doc.Name == "" && inferredName != "" {
			doc.Name = inferredName
			doc.WorkflowConfig.Name = inferredName
			wasInferred = true
		}
		// Validate inferred name against ID rules
		if wasInferred {
			if err := validateID(kind, inferredName); err != nil {
				return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
			}
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
			return wrapDecodeError("mcp", err)
		}
		// Warn if YAML name differs from inferred name (when both are present)
		if doc.Name != "" && inferredName != "" && doc.Name != inferredName {
			config.ParseWarnings = append(config.ParseWarnings, fmt.Sprintf("%s declares name: %q but path implies name: %q", sourceFile, doc.Name, inferredName))
		}
		// Filesystem-as-schema inference: use inferred name if YAML name is empty
		wasInferred := false
		if doc.Name == "" && inferredName != "" {
			doc.Name = inferredName
			doc.MCPConfig.Name = inferredName
			wasInferred = true
		}
		// Validate inferred name against ID rules
		if wasInferred {
			if err := validateID(kind, inferredName); err != nil {
				return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
			}
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

	case "context":
		var doc contextDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return wrapDecodeError("context", err)
		}
		// Warn if YAML name differs from inferred name (when both are present)
		if doc.Name != "" && inferredName != "" && doc.Name != inferredName {
			config.ParseWarnings = append(config.ParseWarnings, fmt.Sprintf("%s declares name: %q but path implies name: %q", sourceFile, doc.Name, inferredName))
		}
		// Filesystem-as-schema inference: use inferred name if YAML name is empty
		wasInferred := false
		if doc.Name == "" && inferredName != "" {
			doc.Name = inferredName
			doc.ContextConfig.Name = inferredName
			wasInferred = true
		}
		// Validate inferred name against ID rules
		if wasInferred {
			if err := validateID(kind, inferredName); err != nil {
				return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
			}
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if config.Contexts == nil {
			config.Contexts = make(map[string]ast.ContextConfig)
		}
		if _, exists := config.Contexts[doc.Name]; exists {
			return fmt.Errorf("duplicate context ID %q", doc.Name)
		}
		config.Contexts[doc.Name] = doc.ContextConfig

	case "memory":
		var doc memoryDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return wrapDecodeError("memory", err)
		}
		// Warn if YAML name differs from inferred name (when both are present)
		if doc.Name != "" && inferredName != "" && doc.Name != inferredName {
			config.ParseWarnings = append(config.ParseWarnings, fmt.Sprintf("%s declares name: %q but path implies name: %q", sourceFile, doc.Name, inferredName))
		}
		// Filesystem-as-schema inference: use inferred name if YAML name is empty
		wasInferred := false
		if doc.Name == "" && inferredName != "" {
			doc.Name = inferredName
			doc.MemoryConfig.Name = inferredName
			wasInferred = true
		}
		// Validate inferred name against ID rules
		if wasInferred {
			if err := validateID(kind, inferredName); err != nil {
				return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
			}
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if config.Memory == nil {
			config.Memory = make(map[string]ast.MemoryConfig)
		}
		if _, exists := config.Memory[doc.Name]; exists {
			return fmt.Errorf("duplicate memory ID %q", doc.Name)
		}
		config.Memory[doc.Name] = doc.MemoryConfig

	case "project":
		var doc projectDocFields
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return wrapDecodeError("project", err)
		}
		if err := validateEnvelope(doc.Version, doc.Name, kind); err != nil {
			return err
		}
		if config.Project == nil {
			config.Project = &ast.ProjectConfig{}
		}
		config.Extends = doc.Extends
		config.Project.Name = doc.Name
		config.Project.Description = doc.Description
		config.Project.Author = doc.Author
		config.Project.Homepage = doc.Homepage
		config.Project.Repository = doc.Repository
		config.Project.License = doc.License
		config.Project.BackupDir = doc.BackupDir
		config.Project.AllowedEnvVars = doc.AllowedEnvVars
		config.Project.Targets = doc.Targets
		config.Project.Test = doc.Test
		config.Project.Local = doc.Local

		config.Project.TargetOptions = doc.TargetOptions

	case "hooks":
		var doc hooksDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return wrapDecodeError("hooks", err)
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
		if doc.Description != "" {
			existing.Description = doc.Description
		}
		if doc.Targets != nil {
			existing.Targets = doc.Targets
		}
		if len(doc.Artifacts) > 0 {
			existing.Artifacts = append(existing.Artifacts, doc.Artifacts...)
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
			return wrapDecodeError("settings", err)
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
			return wrapDecodeError("global", err)
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
	case "policy":
		var doc policyDocument
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&doc); err != nil {
			return wrapDecodeError("policy", err)
		}
		// Filesystem-as-schema inference: use inferred name if YAML name is empty
		wasInferred := false
		if doc.Name == "" && inferredName != "" {
			doc.Name = inferredName
			doc.PolicyConfig.Name = inferredName
			wasInferred = true
		}
		// Validate inferred name against ID rules
		if wasInferred {
			if err := validateID(kind, inferredName); err != nil {
				return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
			}
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
