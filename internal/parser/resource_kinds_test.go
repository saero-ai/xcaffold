package parser

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// TestParseGlobalDocument_WithPolicies tests parsing a global document with policies.
func TestParseGlobalDocument_WithPolicies(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
policies:
  enforce-naming:
    name: enforce-naming
    description: "Naming convention enforcement."
  require-description:
    name: require-description
    description: "All resources must have descriptions."
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	if err != nil {
		t.Fatalf("parseGlobalDocument failed: %v", err)
	}
	if config.Policies == nil {
		t.Errorf("Policies map is nil, expected initialized map")
	}
	if len(config.Policies) != 2 {
		t.Errorf("expected 2 policies, got %d", len(config.Policies))
	}
	if _, ok := config.Policies["enforce-naming"]; !ok {
		t.Errorf("policy 'enforce-naming' not found")
	}
	if _, ok := config.Policies["require-description"]; !ok {
		t.Errorf("policy 'require-description' not found")
	}
}

// TestParseGlobalDocument_WithContexts tests parsing a global document with contexts.
func TestParseGlobalDocument_WithContexts(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
contexts:
  api-design:
    name: api-design
    description: "REST API design patterns."
  database:
    name: database
    description: "Database architecture guidelines."
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	if err != nil {
		t.Fatalf("parseGlobalDocument failed: %v", err)
	}
	if config.Contexts == nil {
		t.Errorf("Contexts map is nil, expected initialized map")
	}
	if len(config.Contexts) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(config.Contexts))
	}
	if _, ok := config.Contexts["api-design"]; !ok {
		t.Errorf("context 'api-design' not found")
	}
	if _, ok := config.Contexts["database"]; !ok {
		t.Errorf("context 'database' not found")
	}
}

// TestParseGlobalDocument_WithMemory tests parsing a global document with memory entries.
func TestParseGlobalDocument_WithMemory(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
memory:
  project-timeline:
    name: project-timeline
    description: "Historical project milestones."
  team-structure:
    name: team-structure
    description: "Team organization and roles."
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	if err != nil {
		t.Fatalf("parseGlobalDocument failed: %v", err)
	}
	if config.Memory == nil {
		t.Errorf("Memory map is nil, expected initialized map")
	}
	if len(config.Memory) != 2 {
		t.Errorf("expected 2 memory entries, got %d", len(config.Memory))
	}
	if _, ok := config.Memory["project-timeline"]; !ok {
		t.Errorf("memory 'project-timeline' not found")
	}
	if _, ok := config.Memory["team-structure"]; !ok {
		t.Errorf("memory 'team-structure' not found")
	}
}

// TestParseGlobalDocument_WithTemplates tests parsing a global document with templates.
func TestParseGlobalDocument_WithTemplates(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
templates:
  feature-template:
    name: feature-template
    description: "Standard feature development template."
  hotfix-template:
    name: hotfix-template
    description: "Hotfix deployment template."
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	if err != nil {
		t.Fatalf("parseGlobalDocument failed: %v", err)
	}
	if config.Templates == nil {
		t.Errorf("Templates map is nil, expected initialized map")
	}
	if len(config.Templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(config.Templates))
	}
	if _, ok := config.Templates["feature-template"]; !ok {
		t.Errorf("template 'feature-template' not found")
	}
	if _, ok := config.Templates["hotfix-template"]; !ok {
		t.Errorf("template 'hotfix-template' not found")
	}
}

// TestParseGlobalDocument_AllResourceTypes tests parsing a global document with all resource types.
func TestParseGlobalDocument_AllResourceTypes(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
agents:
  reviewer:
    name: reviewer
    description: "Code reviewer."
    model: sonnet
    tools: [Read, Glob, Grep]
skills:
  code-review:
    name: code-review
    description: "Code review workflow."
    allowed-tools: [Read, Glob, Grep]
rules:
  naming:
    name: naming
    description: "Naming conventions."
    always-apply: true
policies:
  compliance:
    name: compliance
    description: "Compliance policy."
contexts:
  backend:
    name: backend
    description: "Backend architecture context."
memory:
  decisions:
    name: decisions
    description: "Architecture decisions."
templates:
  standard:
    name: standard
    description: "Standard project template."
workflows:
  deploy:
    name: deploy
    description: "Deployment workflow."
    steps: []
mcp:
  git:
    name: git
    description: "Git MCP server."
    command: git
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	if err != nil {
		t.Fatalf("parseGlobalDocument failed: %v", err)
	}
	if len(config.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(config.Agents))
	}
	if len(config.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(config.Skills))
	}
	if len(config.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(config.Rules))
	}
	if len(config.Policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(config.Policies))
	}
	if len(config.Contexts) != 1 {
		t.Errorf("expected 1 context, got %d", len(config.Contexts))
	}
	if len(config.Memory) != 1 {
		t.Errorf("expected 1 memory entry, got %d", len(config.Memory))
	}
	if len(config.Templates) != 1 {
		t.Errorf("expected 1 template, got %d", len(config.Templates))
	}
	if len(config.Workflows) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(config.Workflows))
	}
	if len(config.MCP) != 1 {
		t.Errorf("expected 1 MCP, got %d", len(config.MCP))
	}
}

// TestParseGlobalDocument_BackwardCompatible tests that existing global documents without new fields still parse.
func TestParseGlobalDocument_BackwardCompatible(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
agents:
  reviewer:
    name: reviewer
    description: "Code reviewer."
    model: sonnet
    tools: [Read]
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	if err != nil {
		t.Fatalf("parseGlobalDocument failed: %v", err)
	}
	if len(config.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(config.Agents))
	}
	// New fields should be nil or empty when not specified
	if config.Policies != nil && len(config.Policies) != 0 {
		t.Errorf("expected Policies to be nil or empty, got %d", len(config.Policies))
	}
	if config.Contexts != nil && len(config.Contexts) != 0 {
		t.Errorf("expected Contexts to be nil or empty, got %d", len(config.Contexts))
	}
	if config.Memory != nil && len(config.Memory) != 0 {
		t.Errorf("expected Memory to be nil or empty, got %d", len(config.Memory))
	}
	if config.Templates != nil && len(config.Templates) != 0 {
		t.Errorf("expected Templates to be nil or empty, got %d", len(config.Templates))
	}
}

// TestParseGlobalDocument_DuplicatePolicies tests that duplicate policy IDs cause an error.
func TestParseGlobalDocument_DuplicatePolicies(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
policies:
  enforce-naming:
    name: enforce-naming
    description: "Naming convention enforcement."
  enforce-naming:
    name: enforce-naming
    description: "Another one."
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	// YAML parser will reject duplicate keys at decode time before our merge logic
	if err == nil {
		t.Errorf("expected error for duplicate policy, got nil")
	}
}

// TestParseGlobalDocument_DuplicateContexts tests that duplicate context IDs cause an error.
func TestParseGlobalDocument_DuplicateContexts(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
contexts:
  backend:
    name: backend
    description: "Backend context."
contexts:
  backend:
    name: backend
    description: "Another backend."
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	// YAML parser will reject duplicate keys at decode time before our merge logic
	if err == nil {
		t.Errorf("expected error for duplicate context, got nil")
	}
}

// TestParseGlobalDocument_DuplicateMemory tests that duplicate memory IDs cause an error.
func TestParseGlobalDocument_DuplicateMemory(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
memory:
  decisions:
    name: decisions
    description: "First."
memory:
  decisions:
    name: decisions
    description: "Second."
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	// YAML parser will reject duplicate keys at decode time before our merge logic
	if err == nil {
		t.Errorf("expected error for duplicate memory, got nil")
	}
}

// TestParseGlobalDocument_EmptyNewFields tests parsing a global document with empty new field blocks.
func TestParseGlobalDocument_EmptyNewFields(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
policies: {}
contexts: {}
memory: {}
templates: {}
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	if err != nil {
		t.Fatalf("parseGlobalDocument failed: %v", err)
	}
	// Empty maps should be nil or empty in AST, not cause error
	if config.Policies != nil && len(config.Policies) != 0 {
		t.Errorf("expected empty Policies, got %d items", len(config.Policies))
	}
	if config.Contexts != nil && len(config.Contexts) != 0 {
		t.Errorf("expected empty Contexts, got %d items", len(config.Contexts))
	}
	if config.Memory != nil && len(config.Memory) != 0 {
		t.Errorf("expected empty Memory, got %d items", len(config.Memory))
	}
	if config.Templates != nil && len(config.Templates) != 0 {
		t.Errorf("expected empty Templates, got %d items", len(config.Templates))
	}
}

// TestParseGlobalDocument_UnknownField_Error tests that unknown YAML keys in global doc cause error.
func TestParseGlobalDocument_UnknownField_Error(t *testing.T) {
	yaml := []byte(`kind: global
version: "1.0"
foobar: true
`)
	config := &ast.XcaffoldConfig{}
	err := parseGlobalDocument(yaml, config, "test.xcaf", "")
	if err == nil {
		t.Errorf("expected KnownFields error for unknown field 'foobar', got nil")
	}
}
