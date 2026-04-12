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

// validateEnvelope checks that mandatory envelope fields are present on a
// resource kind document.
func validateEnvelope(version, name, kind string) error {
	if name == "" {
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

	default:
		return fmt.Errorf("unknown resource kind %q", kind)
	}

	return nil
}
