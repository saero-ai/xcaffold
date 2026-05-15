package parser

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// parseAgentDocument decodes and stores an agent document.
func parseAgentDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
	// Warn: flat kind:agent files placed directly under xcaf/agents/ are deprecated.
	// Prefer subdirectory layout: xcaf/agents/<id>/agent.xcaf for proper memory discovery.
	if sourceFile != "" {
		parentDir := filepath.Base(filepath.Dir(sourceFile))
		if parentDir == "agents" {
			config.ParseWarnings = append(config.ParseWarnings,
				fmt.Sprintf("agent %q is a flat file under xcaf/agents/; memory discovery requires xcaf/agents/<name>/agent.xcaf layout", filepath.Base(sourceFile)))
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
		if err := validateID("agent", inferredName); err != nil {
			return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
		}
	}
	if err := validateEnvelope(doc.Version, doc.Name, "agent"); err != nil {
		return err
	}
	if strings.TrimSpace(doc.AgentConfig.Description) == "" {
		return fmt.Errorf("agent %q: description is required", doc.Name)
	}
	if config.Agents == nil {
		config.Agents = make(map[string]ast.AgentConfig)
	}
	if _, exists := config.Agents[doc.Name]; exists {
		return fmt.Errorf("duplicate agent ID %q", doc.Name)
	}
	config.Agents[doc.Name] = doc.AgentConfig
	return nil
}

// parseSkillDocument decodes and stores a skill document.
func parseSkillDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
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
		if err := validateID("skill", inferredName); err != nil {
			return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
		}
	}
	if err := validateEnvelope(doc.Version, doc.Name, "skill"); err != nil {
		return err
	}
	if config.Skills == nil {
		config.Skills = make(map[string]ast.SkillConfig)
	}
	if _, exists := config.Skills[doc.Name]; exists {
		return fmt.Errorf("duplicate skill ID %q", doc.Name)
	}
	config.Skills[doc.Name] = doc.SkillConfig
	return nil
}

// parseRuleDocument decodes and stores a rule document.
func parseRuleDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
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
		if err := validateID("rule", inferredName); err != nil {
			return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
		}
	}
	if err := validateEnvelope(doc.Version, doc.Name, "rule"); err != nil {
		return err
	}
	if config.Rules == nil {
		config.Rules = make(map[string]ast.RuleConfig)
	}
	if _, exists := config.Rules[doc.Name]; exists {
		return fmt.Errorf("duplicate rule ID %q", doc.Name)
	}
	config.Rules[doc.Name] = doc.RuleConfig
	return nil
}

// parseWorkflowDocument decodes and stores a workflow document.
func parseWorkflowDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
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
		if err := validateID("workflow", inferredName); err != nil {
			return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
		}
	}
	if err := validateEnvelope(doc.Version, doc.Name, "workflow"); err != nil {
		return err
	}
	if config.Workflows == nil {
		config.Workflows = make(map[string]ast.WorkflowConfig)
	}
	if _, exists := config.Workflows[doc.Name]; exists {
		return fmt.Errorf("duplicate workflow ID %q", doc.Name)
	}
	config.Workflows[doc.Name] = doc.WorkflowConfig
	return nil
}

// parseMCPDocument decodes and stores an MCP document.
func parseMCPDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
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
		if err := validateID("mcp", inferredName); err != nil {
			return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
		}
	}
	if err := validateEnvelope(doc.Version, doc.Name, "mcp"); err != nil {
		return err
	}
	if config.MCP == nil {
		config.MCP = make(map[string]ast.MCPConfig)
	}
	if _, exists := config.MCP[doc.Name]; exists {
		return fmt.Errorf("duplicate mcp ID %q", doc.Name)
	}
	config.MCP[doc.Name] = doc.MCPConfig
	return nil
}

// parseContextDocument decodes and stores a context document.
func parseContextDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
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
		if err := validateID("context", inferredName); err != nil {
			return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
		}
	}
	if err := validateEnvelope(doc.Version, doc.Name, "context"); err != nil {
		return err
	}
	if config.Contexts == nil {
		config.Contexts = make(map[string]ast.ContextConfig)
	}
	if _, exists := config.Contexts[doc.Name]; exists {
		return fmt.Errorf("duplicate context ID %q", doc.Name)
	}
	config.Contexts[doc.Name] = doc.ContextConfig
	return nil
}

// parseMemoryDocument decodes and stores a memory document.
func parseMemoryDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
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
		if err := validateID("memory", inferredName); err != nil {
			return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
		}
	}
	if err := validateEnvelope(doc.Version, doc.Name, "memory"); err != nil {
		return err
	}
	if config.Memory == nil {
		config.Memory = make(map[string]ast.MemoryConfig)
	}
	if _, exists := config.Memory[doc.Name]; exists {
		return fmt.Errorf("duplicate memory ID %q", doc.Name)
	}
	config.Memory[doc.Name] = doc.MemoryConfig
	return nil
}

// parseProjectDocument decodes and stores a project document.
func parseProjectDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
	var doc projectDocFields
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return wrapDecodeError("project", err)
	}
	if err := validateEnvelope(doc.Version, doc.Name, "project"); err != nil {
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
	config.Project.TargetOptions = doc.TargetOptions
	return nil
}

// parseHooksDocument decodes and stores a hooks document.
func parseHooksDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
	var doc hooksDocument
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return wrapDecodeError("hooks", err)
	}
	if err := validateEnvelope(doc.Version, "", "hooks"); err != nil {
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
	return nil
}

// parseSettingsDocument decodes and stores a settings document.
func parseSettingsDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
	var doc settingsDocument
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return wrapDecodeError("settings", err)
	}
	if err := validateEnvelope(doc.Version, "", "settings"); err != nil {
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
	return nil
}

// parseGlobalDocument decodes and stores a global document.
func parseGlobalDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
	var doc globalDocument
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return wrapDecodeError("global", err)
	}
	if err := validateEnvelope(doc.Version, "", "global"); err != nil {
		return err
	}
	config.Extends = doc.Extends

	mergeGlobalSettings(config, doc.Settings)
	mergeGlobalHooks(config, doc.Hooks)

	if err := mergeAgents(config, doc.Agents); err != nil {
		return err
	}
	if err := mergeSkills(config, doc.Skills); err != nil {
		return err
	}
	if err := mergeRules(config, doc.Rules); err != nil {
		return err
	}
	if err := mergeWorkflows(config, doc.Workflows); err != nil {
		return err
	}
	if err := mergeMCPs(config, doc.MCP); err != nil {
		return err
	}

	return nil
}

// parsePolicyDocument decodes and stores a policy document.
func parsePolicyDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
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
		if err := validateID("policy", inferredName); err != nil {
			return fmt.Errorf("filesystem-inferred name %q: %w", inferredName, err)
		}
	}
	if err := validateEnvelope(doc.Version, doc.Name, "policy"); err != nil {
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
	return nil
}

// parseBlueprintDocument decodes bytes into a blueprintDocument with
// KnownFields validation, validates envelope fields and name constraints,
// and inserts the resource into config.Blueprints.
func parseBlueprintDocument(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error {
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
