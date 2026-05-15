package parser

import (
	"fmt"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

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

// parseBlueprintDocumentFromNode is a wrapper that converts a yaml.Node to bytes
// and calls parseBlueprintDocument. This is used for the special parsing case in parser.go
// where blueprint documents are parsed directly from nodes.
func parseBlueprintDocumentFromNode(node *yaml.Node, config *ast.XcaffoldConfig) error {
	b, err := nodeToBytes(node)
	if err != nil {
		return fmt.Errorf("failed to marshal blueprint document: %w", err)
	}
	return parseBlueprintDocument(b, config, "", "")
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

// kindParseFunc is the signature for per-kind document parsers.
type kindParseFunc func(b []byte, config *ast.XcaffoldConfig, sourceFile, inferredName string) error

// mergeAgents adds agents from doc to config, checking for duplicates.
func mergeAgents(config *ast.XcaffoldConfig, agents map[string]ast.AgentConfig) error {
	if config.Agents == nil {
		config.Agents = make(map[string]ast.AgentConfig)
	}
	for k, v := range agents {
		if strings.TrimSpace(v.Description) == "" {
			return fmt.Errorf("agent %q: description is required", k)
		}
		if _, exists := config.Agents[k]; exists {
			return fmt.Errorf("duplicate agent ID %q", k)
		}
		config.Agents[k] = v
	}
	return nil
}

// mergeSkills adds skills from doc to config, checking for duplicates.
func mergeSkills(config *ast.XcaffoldConfig, skills map[string]ast.SkillConfig) error {
	if config.Skills == nil {
		config.Skills = make(map[string]ast.SkillConfig)
	}
	for k, v := range skills {
		if _, exists := config.Skills[k]; exists {
			return fmt.Errorf("duplicate skill ID %q", k)
		}
		config.Skills[k] = v
	}
	return nil
}

// mergeRules adds rules from doc to config, checking for duplicates.
func mergeRules(config *ast.XcaffoldConfig, rules map[string]ast.RuleConfig) error {
	if config.Rules == nil {
		config.Rules = make(map[string]ast.RuleConfig)
	}
	for k, v := range rules {
		if _, exists := config.Rules[k]; exists {
			return fmt.Errorf("duplicate rule ID %q", k)
		}
		config.Rules[k] = v
	}
	return nil
}

// mergeWorkflows adds workflows from doc to config, checking for duplicates.
func mergeWorkflows(config *ast.XcaffoldConfig, workflows map[string]ast.WorkflowConfig) error {
	if config.Workflows == nil {
		config.Workflows = make(map[string]ast.WorkflowConfig)
	}
	for k, v := range workflows {
		if _, exists := config.Workflows[k]; exists {
			return fmt.Errorf("duplicate workflow ID %q", k)
		}
		config.Workflows[k] = v
	}
	return nil
}

// mergeMCPs adds MCPs from doc to config, checking for duplicates.
func mergeMCPs(config *ast.XcaffoldConfig, mcp map[string]ast.MCPConfig) error {
	if config.MCP == nil {
		config.MCP = make(map[string]ast.MCPConfig)
	}
	for k, v := range mcp {
		if _, exists := config.MCP[k]; exists {
			return fmt.Errorf("duplicate mcp ID %q", k)
		}
		config.MCP[k] = v
	}
	return nil
}

// mergeGlobalHooks merges hooks from a global document into the config.
func mergeGlobalHooks(config *ast.XcaffoldConfig, hooks ast.HookConfig) {
	if hooks == nil {
		return
	}
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
	for event, groups := range hooks {
		existing.Events[event] = append(existing.Events[event], groups...)
	}
	config.Hooks["default"] = existing
}

// mergeGlobalSettings adds or overwrites settings in the config.
func mergeGlobalSettings(config *ast.XcaffoldConfig, settings ast.SettingsConfig) {
	if config.Settings == nil {
		config.Settings = make(map[string]ast.SettingsConfig)
	}
	sName := settings.Name
	if sName == "" {
		sName = "default"
	}
	config.Settings[sName] = settings
}

// kindParsers is the dispatch map for per-kind document parsing.
var kindParsers = map[string]kindParseFunc{
	"agent":     parseAgentDocument,
	"skill":     parseSkillDocument,
	"rule":      parseRuleDocument,
	"workflow":  parseWorkflowDocument,
	"mcp":       parseMCPDocument,
	"context":   parseContextDocument,
	"memory":    parseMemoryDocument,
	"project":   parseProjectDocument,
	"hooks":     parseHooksDocument,
	"settings":  parseSettingsDocument,
	"global":    parseGlobalDocument,
	"policy":    parsePolicyDocument,
	"blueprint": parseBlueprintDocument,
}

// parseResourceContext groups parameters for parsing a resource document.
type parseResourceContext struct {
	Node         *yaml.Node
	Kind         string
	Config       *ast.XcaffoldConfig
	SourceFile   string
	InferredName string
}

// parseResourceDocument routes a yaml.Node to the appropriate kind parser
// based on the kind string. inferredName is the name inferred from the
// filesystem path when omitted from YAML.
func parseResourceDocument(ctx parseResourceContext) error {
	b, err := nodeToBytes(ctx.Node)
	if err != nil {
		return fmt.Errorf("failed to marshal %s document: %w", ctx.Kind, err)
	}

	parser, ok := kindParsers[ctx.Kind]
	if !ok {
		return fmt.Errorf("unknown resource kind %q", ctx.Kind)
	}

	return parser(b, ctx.Config, ctx.SourceFile, ctx.InferredName)
}
