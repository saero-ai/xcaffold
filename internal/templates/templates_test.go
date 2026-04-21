package templates

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplates(t *testing.T) {
	list := List()
	require.True(t, len(list) >= 3, "must have at least 3 templates")

	ids := make(map[string]bool)
	for _, tmpl := range list {
		ids[tmpl.ID] = true
		assert.NotEmpty(t, tmpl.ID, "template must have an ID")
		assert.NotEmpty(t, tmpl.Label, "template must have a label")
		assert.NotEmpty(t, tmpl.Description, "template must have a description")
	}

	assert.True(t, ids["rest-api"], "rest-api template must exist")
	assert.True(t, ids["cli-tool"], "cli-tool template must exist")
	assert.True(t, ids["frontend-app"], "frontend-app template must exist")
}

func TestRenderTemplate_RESTAPI(t *testing.T) {
	config, err := Render("rest-api", "my-service", "claude-sonnet-4-6")
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.Project)
	assert.Equal(t, "my-service", config.Project.Name)
	require.Contains(t, config.Agents, "backend")
	assert.Equal(t, "claude-sonnet-4-6", config.Agents["backend"].Model)
	assert.NotEmpty(t, config.Agents["backend"].Instructions)
	assert.NotEmpty(t, config.Skills)
	assert.NotEmpty(t, config.Rules)
}

func TestRenderTemplate_CLITool(t *testing.T) {
	config, err := Render("cli-tool", "my-cli", "claude-sonnet-4-6")
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.Project)
	assert.Equal(t, "my-cli", config.Project.Name)
	require.Contains(t, config.Agents, "developer")
	assert.Equal(t, "claude-sonnet-4-6", config.Agents["developer"].Model)
	assert.NotEmpty(t, config.Agents["developer"].Instructions)
}

func TestRenderTemplate_FrontendApp(t *testing.T) {
	config, err := Render("frontend-app", "my-app", "claude-sonnet-4-6")
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.Project)
	assert.Equal(t, "my-app", config.Project.Name)
	require.Contains(t, config.Agents, "frontend")
	assert.Equal(t, "claude-sonnet-4-6", config.Agents["frontend"].Model)
	assert.NotEmpty(t, config.Agents["frontend"].Instructions)
}

func TestRenderTemplate_Unknown(t *testing.T) {
	_, err := Render("nonexistent", "test", "model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

// TestRenderTemplate_RESTAPI_Resources verifies the REST API template includes
// the expected skills and rules in the resource scope.
func TestRenderTemplate_RESTAPI_Resources(t *testing.T) {
	config, err := Render("rest-api", "my-api", "sonnet")
	require.NoError(t, err)
	require.NotNil(t, config)

	// Agent must reference skills and rules
	backend := config.Agents["backend"]
	assert.Contains(t, backend.Skills, "api-testing")
	assert.Contains(t, backend.Rules, "api-conventions")
	assert.Equal(t, "high", backend.Effort)
	assert.Contains(t, backend.Tools, "Bash")

	// Skill must exist at scope level
	require.Contains(t, config.Skills, "api-testing")
	assert.NotEmpty(t, config.Skills["api-testing"].Instructions)

	// Rule must exist at scope level
	require.Contains(t, config.Rules, "api-conventions")
	assert.NotEmpty(t, config.Rules["api-conventions"].Instructions)
}

// TestRenderTemplate_CLITool_Resources verifies the CLI Tool template includes
// the expected rules in the resource scope.
func TestRenderTemplate_CLITool_Resources(t *testing.T) {
	config, err := Render("cli-tool", "my-cli", "sonnet")
	require.NoError(t, err)
	require.NotNil(t, config)

	dev := config.Agents["developer"]
	assert.Contains(t, dev.Rules, "cli-conventions")
	assert.Equal(t, "high", dev.Effort)

	require.Contains(t, config.Rules, "cli-conventions")
	assert.NotEmpty(t, config.Rules["cli-conventions"].Instructions)
}

// TestRenderTemplate_FrontendApp_Resources verifies the frontend-app template
// includes the expected skills and rules in the resource scope.
func TestRenderTemplate_FrontendApp_Resources(t *testing.T) {
	config, err := Render("frontend-app", "my-app", "sonnet")
	require.NoError(t, err)
	require.NotNil(t, config)

	fe := config.Agents["frontend"]
	assert.Contains(t, fe.Skills, "component-testing")
	assert.Contains(t, fe.Rules, "frontend-conventions")
	assert.Equal(t, "high", fe.Effort)

	require.Contains(t, config.Skills, "component-testing")
	assert.NotEmpty(t, config.Skills["component-testing"].Instructions)

	require.Contains(t, config.Rules, "frontend-conventions")
	assert.NotEmpty(t, config.Rules["frontend-conventions"].Instructions)
}

// TestRenderTemplate_Version verifies all topology templates set Version: "1.0".
func TestRenderTemplate_Version(t *testing.T) {
	for _, id := range []string{"rest-api", "cli-tool", "frontend-app"} {
		config, err := Render(id, "proj", "model")
		require.NoError(t, err, "template %q", id)
		assert.Equal(t, "1.0", config.Version, "template %q must have version 1.0", id)
	}
}

// TestRenderTemplate_ProjectDescription verifies all topology templates populate
// a non-empty project description.
func TestRenderTemplate_ProjectDescription(t *testing.T) {
	for _, id := range []string{"rest-api", "cli-tool", "frontend-app"} {
		config, err := Render(id, "proj", "model")
		require.NoError(t, err, "template %q", id)
		require.NotNil(t, config.Project, "template %q must have a Project", id)
		assert.NotEmpty(t, config.Project.Description, "template %q must have a project description", id)
	}
}

// TestRenderTemplate_AgentTools verifies topology template agents declare tools.
func TestRenderTemplate_AgentTools(t *testing.T) {
	for _, id := range []string{"rest-api", "cli-tool", "frontend-app"} {
		config, err := Render(id, "proj", "model")
		require.NoError(t, err, "template %q", id)
		for name, agent := range config.Agents {
			assert.NotEmpty(t, agent.Tools, "agent %q in template %q must declare tools", name, id)
		}
	}
}

// --- Provider-first scaffold render functions ---

func TestRenderProjectXCF_SingleTarget(t *testing.T) {
	out := RenderProjectXCF("my-api", []string{"claude"})

	assert.Contains(t, out, "kind: project")
	assert.Contains(t, out, `name: "my-api"`)
	assert.Contains(t, out, "targets:")
	assert.Contains(t, out, "- claude")
	assert.Contains(t, out, "agents:")
	assert.Contains(t, out, "- developer")
	assert.Contains(t, out, "rules:")
	assert.Contains(t, out, "- conventions")
	assert.Contains(t, out, "skills:")
	assert.Contains(t, out, "- xcaffold")
	assert.Contains(t, out, "policies:")
	assert.Contains(t, out, "- require-agent-description")
	assert.NotContains(t, out, "- cursor")
}

func TestRenderProjectXCF_MultiTarget(t *testing.T) {
	out := RenderProjectXCF("multi", []string{"claude", "cursor", "gemini"})

	assert.Contains(t, out, "- claude")
	assert.Contains(t, out, "- cursor")
	assert.Contains(t, out, "- gemini")
}

func TestRenderAgentXCF_ContainsMatrix(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude", "cursor"})

	assert.Contains(t, out, "kind: agent")
	assert.Contains(t, out, "name: developer")
	assert.Contains(t, out, "model:")
	assert.Contains(t, out, "kind: agent - provider field support")
	assert.Contains(t, out, "claude")
	assert.Contains(t, out, "cursor")
	// effort is claude-only: must show dropped for cursor
	assert.Contains(t, out, "dropped")
}

func TestRenderAgentXCF_FrontmatterFormat(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude"})

	// Must use frontmatter delimiters
	assert.Contains(t, out, "---\nkind: agent")
	// Body must appear after the closing ---
	assert.Contains(t, out, "---\nYou are a software developer.")
	// Must NOT contain the legacy 'instructions: |' field in the YAML block
	assert.NotContains(t, out, "instructions: |")
}

func TestRenderAgentXCF_SingleTarget_NoCursorColumn(t *testing.T) {
	out := RenderAgentXCF("developer", "claude-sonnet-4-6", []string{"claude"})

	// Matrix must exist but cursor column must not appear
	assert.Contains(t, out, "kind: agent - provider field support")

	// Only check the matrix section for the column (before the actual kind declaration)
	matrixBlock := out[:strings.Index(out, "kind: agent\n")]
	assert.NotContains(t, matrixBlock, "cursor")
}

func TestRenderRuleXCF_ContainsMatrix(t *testing.T) {
	out := RenderRuleXCF([]string{"claude", "cursor"})

	assert.Contains(t, out, "kind: rule")
	assert.Contains(t, out, "name: conventions")
	assert.Contains(t, out, "activation: always")
	assert.Contains(t, out, "kind: rule - provider field support")
}

func TestRenderRuleXCF_FrontmatterFormat(t *testing.T) {
	out := RenderRuleXCF([]string{"claude"})

	// Must use frontmatter delimiters
	assert.Contains(t, out, "---\nkind: rule")
	// Body must appear after the closing ---
	assert.Contains(t, out, "---\nFollow standard coding conventions")
	// Must NOT contain the legacy 'instructions: |' field in the YAML block
	assert.NotContains(t, out, "instructions: |")
}

func TestRenderSettingsXCF_ContainsMatrix(t *testing.T) {
	out := RenderSettingsXCF([]string{"claude"})

	assert.Contains(t, out, "kind: settings")
	assert.Contains(t, out, "kind: settings - provider field support")
	assert.Contains(t, out, "mcp-servers")
	assert.Contains(t, out, "permissions")
}

func TestRenderPolicyDescriptionXCF(t *testing.T) {
	out := RenderPolicyDescriptionXCF()

	assert.Contains(t, out, "kind: policy")
	assert.Contains(t, out, "name: require-agent-description")
	assert.Contains(t, out, "severity: warning")
	assert.Contains(t, out, "target: agent")
	assert.Contains(t, out, "is-present: true")
	// Single-document YAML — no multi-doc separator
	assert.NotContains(t, out, "\n---\n")
}

func TestRenderPolicyInstructionsXCF(t *testing.T) {
	out := RenderPolicyInstructionsXCF()

	assert.Contains(t, out, "kind: policy")
	assert.Contains(t, out, "name: require-agent-instructions")
	assert.Contains(t, out, "severity: error")
	assert.Contains(t, out, "target: agent")
	assert.Contains(t, out, "min-length: 10")
	// Single-document YAML — no multi-doc separator
	assert.NotContains(t, out, "\n---\n")
}

// compile-time check: Render returns *ast.XcaffoldConfig.
var _ *ast.XcaffoldConfig = func() *ast.XcaffoldConfig {
	cfg, _ := Render("rest-api", "", "")
	return cfg
}()
