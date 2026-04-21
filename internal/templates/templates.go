package templates

import (
	"fmt"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// Template defines a topology template for xcaffold init.
type Template struct {
	render      func(projectName, model string) *ast.XcaffoldConfig
	ID          string
	Label       string
	Description string
}

var registry = []Template{
	{
		ID:          "rest-api",
		Label:       "REST API Service",
		Description: "Backend service exposing data via REST endpoints",
		render:      renderRESTAPI,
	},
	{
		ID:          "cli-tool",
		Label:       "CLI Tool",
		Description: "Command-line tool with subcommands",
		render:      renderCLITool,
	},
	{
		ID:          "frontend-app",
		Label:       "Frontend Application",
		Description: "Web application with component architecture",
		render:      renderFrontendApp,
	},
}

// List returns all available templates.
func List() []Template {
	return registry
}

// Render returns a populated *ast.XcaffoldConfig for the given template, project
// name, and model. The caller is responsible for writing the config to disk.
func Render(templateID, projectName, model string) (*ast.XcaffoldConfig, error) {
	for _, tmpl := range registry {
		if tmpl.ID == templateID {
			return tmpl.render(projectName, model), nil
		}
	}
	return nil, fmt.Errorf("unknown template %q; available: %s", templateID, availableIDs())
}

func availableIDs() string {
	ids := make([]string, len(registry))
	for i, t := range registry {
		ids[i] = t.ID
	}
	return strings.Join(ids, ", ")
}

func renderRESTAPI(projectName, model string) *ast.XcaffoldConfig {
	alwaysApply := true
	return &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name:        projectName,
			Description: "REST API service",
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"backend": {
					Name:        "backend",
					Description: "Backend developer for API endpoints, database queries, and business logic.",
					Model:       model,
					Effort:      "high",
					Tools:       []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep"},
					Skills:      []string{"api-testing"},
					Rules:       []string{"api-conventions"},
					Instructions: "You are a backend developer.\n" +
						"Follow RESTful conventions. Write integration tests for every endpoint.\n" +
						"Use parameterized queries. Never use string interpolation for SQL.",
				},
			},
			Skills: map[string]ast.SkillConfig{
				"api-testing": {
					Name:        "api-testing",
					Description: "REST API testing patterns",
					Instructions: "Write integration tests that verify HTTP status codes, response shapes,\n" +
						"and error handling. Test both success and failure paths.",
				},
			},
			Rules: map[string]ast.RuleConfig{
				"api-conventions": {
					Name:        "api-conventions",
					Description: "REST API design conventions",
					AlwaysApply: &alwaysApply,
					Instructions: "Use plural nouns for resource endpoints.\n" +
						"Return appropriate HTTP status codes (201 for creation, 404 for not found).\n" +
						"Version APIs via URL prefix (/v1/).",
				},
			},
		},
	}
}

func renderCLITool(projectName, model string) *ast.XcaffoldConfig {
	alwaysApply := true
	return &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name:        projectName,
			Description: "Command-line tool",
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {
					Name:        "developer",
					Description: "CLI developer for commands, flags, and user-facing output.",
					Model:       model,
					Effort:      "high",
					Tools:       []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep"},
					Rules:       []string{"cli-conventions"},
					Instructions: "You are a CLI developer.\n" +
						"Use stdout for output, stderr for errors. Support --help on every command.\n" +
						"Write unit tests for command logic and integration tests for CLI invocation.",
				},
			},
			Rules: map[string]ast.RuleConfig{
				"cli-conventions": {
					Name:        "cli-conventions",
					Description: "CLI design conventions",
					AlwaysApply: &alwaysApply,
					Instructions: "Use meaningful exit codes (0 success, 1 user error, 2 system error).\n" +
						"Support --json flag for machine-readable output where applicable.\n" +
						"Never prompt for input when stdin is not a TTY.",
				},
			},
		},
	}
}

func renderFrontendApp(projectName, model string) *ast.XcaffoldConfig {
	alwaysApply := true
	return &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{
			Name:        projectName,
			Description: "Frontend web application",
		},
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"frontend": {
					Name:        "frontend",
					Description: "Frontend developer for components, pages, and styling.",
					Model:       model,
					Effort:      "high",
					Tools:       []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep"},
					Skills:      []string{"component-testing"},
					Rules:       []string{"frontend-conventions"},
					Instructions: "You are a frontend developer.\n" +
						"Write accessible, semantic HTML. Use components for reusable UI.\n" +
						"Write component tests. Never use inline styles for layout.",
				},
			},
			Skills: map[string]ast.SkillConfig{
				"component-testing": {
					Name:        "component-testing",
					Description: "Component testing patterns",
					Instructions: "Test components in isolation. Verify render output, user interactions,\n" +
						"and accessibility attributes. Mock API calls at the network layer.",
				},
			},
			Rules: map[string]ast.RuleConfig{
				"frontend-conventions": {
					Name:        "frontend-conventions",
					Description: "Frontend coding conventions",
					AlwaysApply: &alwaysApply,
					Instructions: "Use semantic HTML elements. Ensure all interactive elements are keyboard accessible.\n" +
						"Keep components focused -- one responsibility per component.",
				},
			},
		},
	}
}

// RenderProjectXCF generates the kind: project project.xcf content.
// targets is the list of provider names selected by the user.
func RenderProjectXCF(projectName string, targets []string) string {
	var sb strings.Builder

	sb.WriteString("# project.xcf - generated by xcaffold init\n")
	sb.WriteString("# Edit agents/rules/settings in xcf/ then run 'xcaffold apply'.\n")
	sb.WriteString("kind: project\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString(fmt.Sprintf("name: %q\n", projectName))
	sb.WriteString("\n")
	sb.WriteString("# Compile targets. Add or remove providers as needed.\n")
	sb.WriteString("# Supported: claude, cursor, gemini, antigravity, copilot\n")
	sb.WriteString("targets:\n")
	for _, t := range targets {
		sb.WriteString(fmt.Sprintf("  - %s\n", t))
	}
	sb.WriteString("\n")
	sb.WriteString("# Resources are split into xcf/ subdirectories for readability.\n")
	sb.WriteString("agents:\n  - developer          # xcf/agents/developer.xcf\n")
	sb.WriteString("skills:\n  - xcaffold           # xcf/skills/xcaffold.xcf\n")
	sb.WriteString("rules:\n  - conventions        # xcf/rules/conventions.xcf\n")
	sb.WriteString("policies:\n  - require-agent-description  # xcf/policies/require-agent-description.xcf\n  - require-agent-instructions  # xcf/policies/require-agent-instructions.xcf\n")
	sb.WriteString("\n")
	sb.WriteString("# Uncomment to configure the 'xcaffold test' simulator.\n")
	sb.WriteString("# test:\n")
	sb.WriteString("#   cli-path: \"claude\"\n")
	sb.WriteString("#   judge-model: \"claude-haiku-4-5-20251001\"\n")

	return sb.String()
}

// RenderAgentXCF generates the xcf/agents/developer.xcf content with a
// provider support matrix comment for the given selectedTargets.
// The output uses frontmatter format: YAML metadata between --- delimiters,
// followed by the instruction body as plain text.
func RenderAgentXCF(agentName, model string, selectedTargets []string) string {
	var sb strings.Builder

	matrix := RenderMatrix("agent", selectedTargets)
	if matrix != "" {
		sb.WriteString(matrix)
	}
	sb.WriteString("\n")
	sb.WriteString("---\n")
	sb.WriteString("kind: agent\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", agentName))
	sb.WriteString(fmt.Sprintf("description: \"General software developer agent.\"\n"))
	sb.WriteString("\n")
	sb.WriteString("# model: used by claude, gemini, antigravity. Ignored by cursor and copilot.\n")
	sb.WriteString(fmt.Sprintf("model: %q\n", model))
	sb.WriteString("\n")
	sb.WriteString("# effort: claude only. Remove if not targeting claude exclusively.\n")
	sb.WriteString("effort: \"high\"\n")
	sb.WriteString("\n")
	sb.WriteString("tools: [Read, Write, Edit, Bash, Glob, Grep]\n")
	sb.WriteString("---\n")
	sb.WriteString("You are a software developer.\n")
	sb.WriteString("Write clean, maintainable code.\n")
	sb.WriteString("\n")
	sb.WriteString("# Optional: per-provider instruction overrides\n")
	sb.WriteString("# targets:\n")
	sb.WriteString("#   cursor:\n")
	sb.WriteString("#     instructions-override: |\n")
	sb.WriteString("#       You are a software developer. Keep rules concise.\n")
	sb.WriteString("\n")
	sb.WriteString("# Optional: assertions for 'xcaffold test --judge'\n")
	sb.WriteString("# assertions:\n")
	sb.WriteString("#   - \"The agent must not write files outside the project directory.\"\n")
	sb.WriteString("#   - \"The agent must run tests before marking a task complete.\"\n")

	return sb.String()
}

// RenderRuleXCF generates the xcf/rules/conventions.xcf content.
// The output uses frontmatter format: YAML metadata between --- delimiters,
// followed by the rule body as plain text.
func RenderRuleXCF(selectedTargets []string) string {
	var sb strings.Builder

	matrix := RenderMatrix("rule", selectedTargets)
	if matrix != "" {
		sb.WriteString(matrix)
	}
	sb.WriteString("\n")
	sb.WriteString("---\n")
	sb.WriteString("kind: rule\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString("name: conventions\n")
	sb.WriteString("description: \"Core coding conventions for this project.\"\n")
	sb.WriteString("\n")
	sb.WriteString("# activation: always | path-glob | model-decided | manual-mention | explicit-invoke\n")
	sb.WriteString("activation: always\n")
	sb.WriteString("---\n")
	sb.WriteString("Follow standard coding conventions for this project.\n")
	sb.WriteString("Write clean, readable, well-documented code.\n")
	sb.WriteString("Prefer explicit over implicit.\n")
	sb.WriteString("\n")
	sb.WriteString("# Optional: path-scoped activation\n")
	sb.WriteString("# paths:\n")
	sb.WriteString("#   - \"src/**\"\n")
	sb.WriteString("#   - \"lib/**\"\n")

	return sb.String()
}

// RenderSettingsXCF generates the xcf/settings.xcf content.
func RenderSettingsXCF(selectedTargets []string) string {
	var sb strings.Builder

	matrix := RenderMatrix("settings", selectedTargets)
	if matrix != "" {
		sb.WriteString(matrix)
	}
	sb.WriteString("\n")
	sb.WriteString("kind: settings\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString("\n")
	sb.WriteString("# MCP Servers (claude, cursor, gemini, antigravity)\n")
	sb.WriteString("# Uncomment and configure to register MCP tools.\n")
	sb.WriteString("# mcp-servers:\n")
	sb.WriteString("#   filesystem:\n")
	sb.WriteString("#     type: stdio\n")
	sb.WriteString("#     command: npx\n")
	sb.WriteString("#     args: [\"-y\", \"@modelcontextprotocol/server-filesystem\", \".\"]\n")
	sb.WriteString("\n")
	sb.WriteString("# Permissions (claude only)\n")
	sb.WriteString("# permissions:\n")
	sb.WriteString("#   allow:\n")
	sb.WriteString("#     - \"Bash(npm test *)\"\n")
	sb.WriteString("#   deny:\n")
	sb.WriteString("#     - \"Bash(rm -rf *)\"\n")
	sb.WriteString("\n")
	sb.WriteString("# Hooks (claude only)\n")
	sb.WriteString("# hooks:\n")
	sb.WriteString("#   PreToolUse:\n")
	sb.WriteString("#     - matcher: \"Bash\"\n")
	sb.WriteString("#       hooks:\n")
	sb.WriteString("#         - type: command\n")
	sb.WriteString("#           command: \"echo 'running bash'\"\n")

	return sb.String()
}

// RenderPolicyDescriptionXCF generates the xcf/policies/require-agent-description.xcf content.
// Returns a single-document YAML policy that enforces agent descriptions.
func RenderPolicyDescriptionXCF() string {
	return `# kind: policy - guardrails enforced at 'xcaffold apply' time.
# Violations are caught before any files are written.
kind: policy
version: "1.0"
name: require-agent-description
description: "Every agent must have a description."
severity: warning
target: agent
require:
  - field: description
    is-present: true
`
}

// RenderPolicyInstructionsXCF generates the xcf/policies/require-agent-instructions.xcf content.
// Returns a single-document YAML policy that enforces agent instructions.
func RenderPolicyInstructionsXCF() string {
	return `kind: policy
version: "1.0"
name: require-agent-instructions
description: "Every agent must have instructions."
severity: error
target: agent
require:
  - field: instructions
    min-length: 10
`
}
