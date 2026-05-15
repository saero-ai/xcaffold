package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseFile_MultiKind_RuleConfig_NameField(t *testing.T) {
	yaml := `kind: global
version: "1.0"
rules:
  security:
    name: security
    description: "Security conventions"

`
	config, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err)
	require.NotNil(t, config.Rules)
	assert.Equal(t, "security", config.Rules["security"].Name)
}

func TestParseFile_MultiKind_EnvelopeFieldsAccepted(t *testing.T) {
	yamlDoc := `kind: agent
version: "1.0"
name: developer
description: "Dev agent"
model: sonnet
tools: [Bash, Read, Write]
`
	dec := yaml.NewDecoder(strings.NewReader(yamlDoc))
	dec.KnownFields(true)
	var doc agentDocument
	err := dec.Decode(&doc)
	require.NoError(t, err)
	assert.Equal(t, "agent", doc.Kind)
	assert.Equal(t, "1.0", doc.Version)
	assert.Equal(t, "developer", doc.Name)
	assert.Equal(t, "Dev agent", doc.AgentConfig.Description)
	assert.Equal(t, "sonnet", doc.AgentConfig.Model)
}

func TestParseFile_MultiKind_KnownFields_RejectsInvalid(t *testing.T) {
	yamlDoc := `kind: agent
version: "1.0"
name: developer
description: "Dev agent"
always-apply: true
`
	dec := yaml.NewDecoder(strings.NewReader(yamlDoc))
	dec.KnownFields(true)
	var doc agentDocument
	err := dec.Decode(&doc)
	require.Error(t, err, "always-apply is a RuleConfig field, not AgentConfig — KnownFields must reject it")
}

func TestExtractKind_Agent(t *testing.T) {
	yamlStr := "kind: agent\nname: dev\nversion: \"1.0\"\ndescription: \"test agent\"\n"
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &node)
	require.NoError(t, err)
	// yaml.Unmarshal wraps in a DocumentNode
	docNode := &node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		docNode = node.Content[0]
	}
	kind := extractKind(docNode)
	assert.Equal(t, "agent", kind)
}

func TestExtractKind_Empty(t *testing.T) {
	yamlStr := "name: dev\nversion: \"1.0\"\n"
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &node)
	require.NoError(t, err)
	docNode := &node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		docNode = node.Content[0]
	}
	kind := extractKind(docNode)
	assert.Equal(t, "", kind)
}

func TestExtractKind_Config(t *testing.T) {
	yamlStr := "kind: config\nversion: \"1.0\"\n"
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &node)
	require.NoError(t, err)
	docNode := &node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		docNode = node.Content[0]
	}
	kind := extractKind(docNode)
	assert.Equal(t, "config", kind)
}

func TestNodeToBytes_RoundTrip(t *testing.T) {
	yamlStr := "kind: agent\nname: dev\nversion: \"1.0\"\ndescription: \"test agent\"\n"
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &node)
	require.NoError(t, err)
	docNode := &node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		docNode = node.Content[0]
	}
	b, err := nodeToBytes(docNode)
	require.NoError(t, err)
	assert.Contains(t, string(b), "kind: agent")
	assert.Contains(t, string(b), "name: dev")
}

func TestParseFile_MultiKind_MCPConfig_NameField(t *testing.T) {
	yaml := `kind: global
version: "1.0"
mcp:
  filesystem:
    name: filesystem
    type: stdio
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-filesystem"
`
	config, err := Parse(strings.NewReader(yaml))
	require.NoError(t, err)
	require.NotNil(t, config.MCP)
	assert.Equal(t, "filesystem", config.MCP["filesystem"].Name)
}

func makeNodeFromYAML(t *testing.T, yamlStr string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &node)
	require.NoError(t, err)
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return &node
}

func makeEmptyConfig() *ast.XcaffoldConfig {
	return &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			MCP:       make(map[string]ast.MCPConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
		},
	}
}

func TestParseResourceDocument_SingleAgent(t *testing.T) {
	yamlDoc := `kind: agent
version: "1.0"
name: developer
description: "Dev"
model: sonnet
tools: [Bash, Read]
`
	node := makeNodeFromYAML(t, yamlDoc)
	config := makeEmptyConfig()
	err := parseResourceDocument(parseResourceContext{
		Node:         node,
		Kind:         "agent",
		Config:       config,
		SourceFile:   "",
		InferredName: "",
	})
	require.NoError(t, err)
	agent, ok := config.Agents["developer"]
	require.True(t, ok, "expected agent 'developer' in config.Agents")
	assert.Equal(t, "Dev", agent.Description)
	assert.Equal(t, "sonnet", agent.Model)
}

func TestParseResourceDocument_MissingName_Error(t *testing.T) {
	yamlDoc := `kind: agent
version: "1.0"
description: "Dev"
`
	node := makeNodeFromYAML(t, yamlDoc)
	config := makeEmptyConfig()
	err := parseResourceDocument(parseResourceContext{
		Node:         node,
		Kind:         "agent",
		Config:       config,
		SourceFile:   "",
		InferredName: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestParseResourceDocument_MissingVersion_Error(t *testing.T) {
	yamlDoc := `kind: agent
name: dev
description: "Dev"
`
	node := makeNodeFromYAML(t, yamlDoc)
	config := makeEmptyConfig()
	err := parseResourceDocument(parseResourceContext{
		Node:         node,
		Kind:         "agent",
		Config:       config,
		SourceFile:   "",
		InferredName: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestParseResourceDocument_EmptyName_Error(t *testing.T) {
	yamlDoc := `kind: agent
version: "1.0"
name: ""
description: "Dev"
`
	node := makeNodeFromYAML(t, yamlDoc)
	config := makeEmptyConfig()
	err := parseResourceDocument(parseResourceContext{
		Node:         node,
		Kind:         "agent",
		Config:       config,
		SourceFile:   "",
		InferredName: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestParseResourceDocument_DuplicateName_Error(t *testing.T) {
	config := makeEmptyConfig()
	config.Agents["developer"] = ast.AgentConfig{Name: "developer"}

	yamlDoc := `kind: agent
version: "1.0"
name: developer
description: "Another dev"
`
	node := makeNodeFromYAML(t, yamlDoc)
	err := parseResourceDocument(parseResourceContext{
		Node:         node,
		Kind:         "agent",
		Config:       config,
		SourceFile:   "",
		InferredName: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

// Multi-document parsing tests (Phase 5)

func TestParseFile_MultiKind_MultipleDocuments(t *testing.T) {
	// Multi-document .xcaf files are no longer supported. Parse must return an error.
	input := `---
kind: agent
version: "1.0"
name: developer
description: "Dev agent"
model: sonnet
---
kind: skill
version: "1.0"
name: tdd
description: "TDD workflow"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "multi-document .xcaf files must be rejected")
	assert.Contains(t, err.Error(), "multi-document .xcaf files are no longer supported")
}

func TestParseFile_MultiKind_UnknownKind_Error(t *testing.T) {
	input := `kind: invalid
version: "1.0"
name: test
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource kind")
}

func TestParseFile_MultiKind_DuplicateName_Error(t *testing.T) {
	input := `---
kind: agent
version: "1.0"
name: developer
description: "First"
model: sonnet
---
kind: agent
version: "1.0"
name: developer
description: "Second"
model: haiku
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestParseFile_MultiKind_WorkflowDocument(t *testing.T) {
	input := `kind: workflow
version: "1.0"
name: deploy
description: "Deploy workflow"
steps:
  - name: build
    instructions: "Build the project"
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	assert.Contains(t, config.Workflows, "deploy")
}

// Phase 7: cross-reference, collision, extends rejection, project-scope, and
// mutual-exclusion validation tests.

func TestParseFile_MultiKind_CrossRefValidation(t *testing.T) {
	// A skill defined in one file and referenced by an agent in another must
	// resolve successfully after all files are merged via ParseDirectory.
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill.xcaf"), []byte(`kind: skill
version: "1.0"
name: tdd
description: "TDD"
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte(`kind: agent
version: "1.0"
name: developer
description: "Dev"
model: sonnet
skills: [tdd]
`), 0600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Contains(t, config.Agents, "developer")
	assert.Contains(t, config.Skills, "tdd")
}

func TestParseFile_MultiKind_CrossRefValidation_Missing(t *testing.T) {
	// An agent that references a skill not present anywhere in the manifest
	// must fail with an error that names the missing skill.
	input := `---
kind: agent
version: "1.0"
name: developer
description: "Dev"
model: sonnet
skills: [nonexistent]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestParseFile_MultiKind_ExtendsOnResourceKind_Error(t *testing.T) {
	// "extends:" is not a declared field on agentDocument. KnownFields(true)
	// must reject it immediately during document parsing.
	input := `kind: agent
version: "1.0"
name: developer
description: "Dev"
extends: base.xcaf
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err, "KnownFields must reject 'extends' as an unknown field on agentDocument")
}

func TestParseFile_MultiKind_ProjectScopedMerge(t *testing.T) {
	// A kind:project file and a kind:agent file in the same directory: both are
	// valid together via ParseDirectory. Resource kind documents merge into the
	// root ResourceScope (config.Agents), not a project-scoped map.
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte(`kind: project
version: "1.0"
name: test-project
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte(`kind: agent
version: "1.0"
name: developer
description: "Dev"
model: sonnet
`), 0600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	require.NotNil(t, config.Project)
	assert.Equal(t, "test-project", config.Project.Name)
	// Resource kind documents merge into root-level config.Agents.
	assert.Contains(t, config.Agents, "developer")
}

func TestParseFile_MultiKind_MutuallyExclusiveInstructions_Error(t *testing.T) {
	// A kind:agent document that sets both instructions-file and has a markdown body
	// must fail: they are mutually exclusive.
	input := `---
kind: agent
version: "1.0"
name: developer
instructions-file: "dev.md"
---
Inline instructions.
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instructions")
}

// Phase 6: isConfigFile broadened to accept all resource kind values.

func TestParseDirectory_MultiKind_AcrossFiles(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte(
		"kind: project\nversion: \"1.0\"\nname: test-project\n",
	), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte(
		"kind: agent\nversion: \"1.0\"\nname: developer\ndescription: \"Dev\"\nmodel: sonnet\n",
	), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill.xcaf"), []byte(
		"kind: skill\nversion: \"1.0\"\nname: tdd\ndescription: \"TDD\"\n",
	), 0600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	require.NotNil(t, config.Project)
	assert.Equal(t, "test-project", config.Project.Name)
	assert.Contains(t, config.Agents, "developer")
	assert.Contains(t, config.Skills, "tdd")
}

func TestParseFile_MultiKind_MCPDocument(t *testing.T) {
	input := `kind: mcp
version: "1.0"
name: playwright
command: npx
args: ["@anthropic/mcp-playwright"]
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	assert.Contains(t, config.MCP, "playwright")
}

// Phase 9: round-trip equivalence tests.

func TestRoundTrip_MultiKind_EquivalentToMonolithic(t *testing.T) {
	// Multi-document .xcaf files (kind:project + kind:global in one file) are no
	// longer supported. This test verifies that both formats are now rejected and
	// that ParseDirectory with separate single-kind files produces equivalent results.
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")

	// Verify multi-doc monolithic format is rejected.
	monolithic := `---
kind: project
version: "1.0"
name: test-project
---
kind: global
version: "1.0"
agents:
  developer:
    description: "Dev agent"
    model: sonnet
`
	_, err := Parse(strings.NewReader(monolithic))
	require.Error(t, err, "multi-doc monolithic format must be rejected")

	// Verify multi-doc multi-kind format is rejected.
	multiKind := `---
kind: project
version: "1.0"
name: test-project
---
kind: agent
version: "1.0"
name: developer
description: "Dev agent"
model: sonnet
`
	_, err = Parse(strings.NewReader(multiKind))
	require.Error(t, err, "multi-doc multi-kind format must be rejected")

	// ParseDirectory with separate single-kind files produces the correct result.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte(`kind: project
version: "1.0"
name: test-project
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte(`kind: agent
version: "1.0"
name: developer
description: "Dev agent"
model: sonnet
tools: [Bash, Read, Write]
skills: [tdd]
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill.xcaf"), []byte(`kind: skill
version: "1.0"
name: tdd
description: "TDD workflow"

`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rule.xcaf"), []byte(`kind: rule
version: "1.0"
name: security
description: "Security rules"

`), 0600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, "test-project", config.Project.Name)
	require.Contains(t, config.Agents, "developer")
	assert.Equal(t, "Dev agent", config.Agents["developer"].Description)
	require.Contains(t, config.Skills, "tdd")
	require.Contains(t, config.Rules, "security")
}

func TestRoundTrip_MultiKind_Validate(t *testing.T) {
	// A complete multi-resource config with cross-references via ParseDirectory.
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte(`kind: project
version: "1.0"
name: validation-test
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill-tdd.xcaf"), []byte(`kind: skill
version: "1.0"
name: tdd
description: "TDD"

`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill-cr.xcaf"), []byte(`kind: skill
version: "1.0"
name: code-review
description: "Code review"

`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rule.xcaf"), []byte(`kind: rule
version: "1.0"
name: security
description: "Security"

`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte(`kind: agent
version: "1.0"
name: developer
description: "Dev"
model: sonnet
skills: [tdd, code-review]
rules: [security]

`), 0600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, "validation-test", config.Project.Name)
	assert.Len(t, config.Agents, 1)
	assert.Len(t, config.Skills, 2)
	assert.Len(t, config.Rules, 1)
	assert.Equal(t, ast.ClearableList{Values: []string{"tdd", "code-review"}}, config.Agents["developer"].Skills)
	assert.Equal(t, ast.ClearableList{Values: []string{"security"}}, config.Agents["developer"].Rules)
}

// Phase N: kind:project, kind:hooks, kind:settings (RED — all tests below will
// fail until the AST fields and parser switch cases are added).

func TestParsePartial_KindProject_Basic(t *testing.T) {
	input := `kind: project
version: "1.0"
name: my-project
description: "Test project"
targets:
  - claude
  - antigravity
`
	config, err := parsePartial(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, config.Project, "config.Project must not be nil for kind:project")
	assert.Equal(t, "my-project", config.Project.Name)
	assert.Equal(t, []string{"claude", "antigravity"}, config.Project.Targets)
	// Ref lists are no longer populated by the parser; resources come from xcaf/ directory scanning
	assert.Len(t, config.Agents, 0, "ref lists no longer used; agents come from filesystem")
}

func TestParsePartial_KindHooks_Basic(t *testing.T) {
	input := `kind: hooks
version: "1.0"
events:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "echo pre-hook"
`
	config, err := parsePartial(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, config.Hooks, "config.Hooks must not be nil for kind:hooks")
	var effectiveHooks ast.HookConfig
	if dh, ok := config.Hooks["default"]; ok {
		effectiveHooks = dh.Events
	}
	require.Contains(t, effectiveHooks, "PreToolUse")
	assert.Len(t, effectiveHooks["PreToolUse"], 1)
}

func TestParsePartial_KindSettings_Basic(t *testing.T) {
	input := `kind: settings
version: "1.0"
model: sonnet
effort-level: high
`
	config, err := parsePartial(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "sonnet", config.Settings["default"].Model)
	assert.Equal(t, "high", config.Settings["default"].EffortLevel)
}

func TestParsePartial_MultiDoc_ProjectAgentHooks(t *testing.T) {
	// Multi-document .xcaf files are no longer supported; parsePartial rejects them.
	// Use ParseDirectory with separate files to achieve the same composition.
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")

	// Verify multi-doc input is rejected by parsePartial.
	input := `---
kind: project
version: "1.0"
name: multi-doc-project
agents:
  - backend-engineer
---
kind: agent
version: "1.0"
name: backend-engineer
description: "Backend dev"
model: sonnet
`
	_, err := parsePartial(strings.NewReader(input))
	require.Error(t, err, "multi-document input must be rejected by parsePartial")

	// ParseDirectory with separate files achieves the same result.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte(`kind: project
version: "1.0"
name: multi-doc-project
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte(`kind: agent
version: "1.0"
name: backend-engineer
description: "Backend dev"
model: sonnet
skills: [tdd]
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skill.xcaf"), []byte(`kind: skill
version: "1.0"
name: tdd
description: "TDD"
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hooks.xcaf"), []byte(`kind: hooks
version: "1.0"
events:
  PreToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "echo hook"
`), 0600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	require.NotNil(t, config.Project, "config.Project must not be nil")
	assert.Equal(t, "multi-doc-project", config.Project.Name)
	// Ref lists are no longer populated by the parser
	assert.Contains(t, config.Agents, "backend-engineer")
	require.NotNil(t, config.Hooks)
	assert.Contains(t, config.Hooks["default"].Events, "PreToolUse")
}

func TestParsePartial_KindProject_KnownFields_RejectsInvalid(t *testing.T) {
	input := `kind: project
version: "1.0"
name: my-project
badField: true
`
	_, err := parsePartial(strings.NewReader(input))
	require.Error(t, err, "KnownFields must reject unknown field 'badField' on kind:project")
}

func TestParsePartial_KindProject_RejectsWithoutName(t *testing.T) {
	input := `kind: project
version: "1.0"
description: "No name here"
`
	_, err := parsePartial(strings.NewReader(input))
	require.Error(t, err, "kind:project without a name field must return an error")
}

func TestParsePartial_KindHooks_NoNameRequired(t *testing.T) {
	// hooks are a singleton; name is not required.
	input := `kind: hooks
version: "1.0"
events:
  Stop:
    - hooks:
        - type: command
          command: "echo stop"
`
	config, err := parsePartial(strings.NewReader(input))
	require.NoError(t, err, "kind:hooks must succeed without a name field")
	require.NotNil(t, config.Hooks)
	assert.Contains(t, config.Hooks["default"].Events, "Stop")
}

func TestParsePartial_KindSettings_NoNameRequired(t *testing.T) {
	// settings are a singleton; name is not required.
	input := `kind: settings
version: "1.0"
model: haiku
`
	config, err := parsePartial(strings.NewReader(input))
	require.NoError(t, err, "kind:settings must succeed without a name field")
	assert.Equal(t, "haiku", config.Settings["default"].Model)
}

func TestParseableKinds_IncludesNewKinds(t *testing.T) {
	assert.True(t, parseableKinds["project"], "parseableKinds must contain 'project'")
	assert.True(t, parseableKinds["hooks"], "parseableKinds must contain 'hooks'")
	assert.True(t, parseableKinds["settings"], "parseableKinds must contain 'settings'")
	assert.True(t, parseableKinds["global"], "parseableKinds must contain 'global'")
	assert.True(t, parseableKinds["policy"], "parseableKinds must contain 'policy'")
}

func TestParseFile_EmptyKind_Error(t *testing.T) {
	input := `version: "1.0"
agents:
  dev:

`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kind field is required")
}

func TestParseFile_ConfigKind_Error(t *testing.T) {
	input := `kind: config
version: "1.0"
agents:
  dev:

`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kind \"config\" has been removed")
}

func TestIsParseableFile_ConfigKind_ReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.xcaf")
	os.WriteFile(path, []byte("kind: config\nversion: \"1.0\"\nagents: {}\n"), 0644)
	assert.False(t, isParseableFile(path))
}

func TestIsParseableFile_EmptyKind_ReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.xcaf")
	os.WriteFile(path, []byte("version: \"1.0\"\nagents: {}\n"), 0644)
	assert.False(t, isParseableFile(path))
}

func TestParseFile_GlobalKind_Basic(t *testing.T) {
	input := `kind: global
version: "1.0"
agents:
  shared-dev:
    description: "Global developer"

settings:
  model: sonnet
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, config.Agents)
	assert.Equal(t, "Global developer", config.Agents["shared-dev"].Description)
	assert.Equal(t, "sonnet", config.Settings["default"].Model)
}

func TestParseFile_GlobalKind_DuplicateAgent_Error(t *testing.T) {
	input := `kind: global
version: "1.0"
agents:
  dev:
    description: "Developer"
    model: sonnet
    tools: [Read]

---
kind: agent
version: "1.0"
name: dev
description: "Developer agent"

`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate agent ID")
}

func TestParseFile_GlobalKind_Singleton(t *testing.T) {
	input := `kind: global
version: "1.0"
settings:
  model: sonnet
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "sonnet", config.Settings["default"].Model)
}

func TestParseFile_GlobalKind_NoProject_Error(t *testing.T) {
	input := `kind: global
version: "1.0"
project:
  name: "should-not-exist"
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field project not found")
}

func TestIsParseableFile_GlobalKind_ReturnsTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "global.xcaf")
	os.WriteFile(path, []byte("kind: global\nversion: \"1.0\"\nagents: {}\n"), 0644)
	assert.True(t, isParseableFile(path))
}

func TestParseResourceDocument_SinglePolicy(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: require-approved-model
description: Agents must use an approved model
severity: error
target: agent
require:
  - field: model
    one-of:
      - claude-opus-4-5-20250514
      - claude-sonnet-4-5-20250514
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	require.NotNil(t, config.Policies)
	p, ok := config.Policies["require-approved-model"]
	require.True(t, ok)
	assert.Equal(t, "require-approved-model", p.Name)
	assert.Equal(t, "error", p.Severity)
	assert.Equal(t, "agent", p.Target)
	require.Len(t, p.Require, 1)
	assert.Equal(t, "model", p.Require[0].Field)
	assert.Equal(t, []string{"claude-opus-4-5-20250514", "claude-sonnet-4-5-20250514"}, p.Require[0].OneOf)
}

func TestParseFile_MultiKind_ProjectWithPolicies(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte(`kind: project
version: "1.0"
name: my-api
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent.xcaf"), []byte(`kind: agent
version: "1.0"
name: developer
description: "Developer agent"

`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "policy.xcaf"), []byte(`kind: policy
version: "1.0"
name: require-approved-model
severity: error
target: agent
require:
  - field: model
    one-of:
      - sonnet
`), 0600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	require.NotNil(t, config.Project)
	// Ref lists are no longer populated by the parser
	require.NotNil(t, config.Policies)
	_, ok := config.Policies["require-approved-model"]
	assert.True(t, ok)
}

func TestParseFile_MultiKind_PolicyCrossRef_Missing_Error(t *testing.T) {
	input := `kind: project
version: "1.0"
name: my-api
policies:
  - nonexistent-policy
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	// Ref lists are no longer supported; policies field should not be in projectDocFields
	assert.Contains(t, err.Error(), "field policies not found")
}

func TestParseFile_MultiKind_PolicyCrossRef_Valid(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "project.xcaf"), []byte(`kind: project
version: "1.0"
name: my-api
`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "policy.xcaf"), []byte(`kind: policy
version: "1.0"
name: my-policy
severity: warning
target: agent
require:
  - field: description
    is-present: true
`), 0600))

	_, err := ParseDirectory(dir)
	require.NoError(t, err)
	// Ref lists are no longer populated by the parser
}

func TestParseDirectory_PolicyInSubdir(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()

	// main.xcaf: project
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.xcaf"), []byte(`kind: project
version: "1.0"
name: test-project
`), 0644))

	// xcaf/policies/approved-model.xcaf: policy in subdirectory
	policiesDir := filepath.Join(dir, "xcaf", "policies")
	require.NoError(t, os.MkdirAll(policiesDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(policiesDir, "approved-model.xcaf"), []byte(`kind: policy
version: "1.0"
name: approved-model
severity: error
target: agent
require:
  - field: model
    one-of:
      - claude-sonnet-4-5-20250514
`), 0644))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	require.NotNil(t, config.Policies)
	p, ok := config.Policies["approved-model"]
	require.True(t, ok, "policy should be discovered from xcaf/policies/ subdirectory")
	assert.Equal(t, "error", p.Severity)
	assert.Equal(t, "agent", p.Target)
	// Ref lists are no longer populated by the parser
}

func TestParseDirectory_MultiKind_MixedFormats(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	dir := t.TempDir()

	// main.xcaf: kind global containing an inline agent
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.xcaf"), []byte(`kind: global
version: "1.0"
agents:
  reviewer:
    name: reviewer
    description: "Code reviewer"
    model: sonnet
`), 0600))

	// dev.xcaf: kind agent — separate file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dev.xcaf"), []byte(
		"kind: agent\nversion: \"1.0\"\nname: developer\ndescription: \"Dev\"\nmodel: haiku\n",
	), 0600))

	config, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Contains(t, config.Agents, "reviewer")
	assert.Contains(t, config.Agents, "developer")
}

func TestParseResourceDocument_Policy_MissingName_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
severity: error
target: agent
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestParseResourceDocument_Policy_MissingVersion_Error(t *testing.T) {
	input := `kind: policy
name: test-policy
severity: error
target: agent
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestParseResourceDocument_Policy_InvalidSeverity_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: bad-severity
severity: err
target: agent
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "severity must be")
}

func TestParseResourceDocument_Policy_InvalidTarget_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: bad-target
severity: error
target: agents
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target must be one of")
}

func TestParseResourceDocument_Policy_DuplicateName_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: dup-policy
severity: error
target: agent
---
kind: policy
version: "1.0"
name: dup-policy
severity: warning
target: skill
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate policy ID")
}

func TestParseResourceDocument_Policy_UnknownField_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: bad-fields
severity: error
target: agent
unknown_field: true
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid policy document")
}

func TestParseResourceDocument_Policy_RequireEmptyField_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: empty-field
severity: error
target: agent
require:
  - one-of: ["sonnet"]
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "require[0].field is required")
}

func TestParseResourceDocument_Policy_DenyEmpty_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: empty-deny
severity: error
target: output
deny:
  - {}
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deny[0] must specify at least one of")
}

func TestParseResourceDocument_Policy_MinimalValid(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: minimal
severity: off
target: agent
`
	config, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	p, ok := config.Policies["minimal"]
	require.True(t, ok)
	assert.Equal(t, "off", p.Severity)
	assert.Nil(t, p.Match)
	assert.Empty(t, p.Require)
	assert.Empty(t, p.Deny)
}

func TestParseResourceDocument_Policy_SeverityTypo_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: typo
severity: Error
target: agent
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "severity must be")
}

func TestParseResourceDocument_Policy_TargetTypo_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: typo
severity: error
target: Agent
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target must be one of")
}

func TestParseResourceDocument_Policy_MatchAndNoTarget_Error(t *testing.T) {
	input := `kind: policy
version: "1.0"
name: match-no-target
severity: error
target: ""
match:
  has-tool: Bash
`
	_, err := Parse(strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target must be one of")
}
