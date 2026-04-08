package templates

import (
	"fmt"
	"strings"
)

// Template defines a topology template for xcaffold init.
type Template struct {
	render      func(projectName, model string) string
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

// Render generates .xcf content for the given template, project name, and model.
func Render(templateID, projectName, model string) (string, error) {
	for _, tmpl := range registry {
		if tmpl.ID == templateID {
			return tmpl.render(projectName, model), nil
		}
	}
	return "", fmt.Errorf("unknown template %q; available: %s", templateID, availableIDs())
}

func availableIDs() string {
	ids := make([]string, len(registry))
	for i, t := range registry {
		ids[i] = t.ID
	}
	return strings.Join(ids, ", ")
}

func renderRESTAPI(projectName, model string) string {
	return fmt.Sprintf(`version: "1.0"
project:
  name: %q
  description: "REST API service"

agents:
  backend:
    description: "Backend developer for API endpoints, database queries, and business logic."
    instructions: |
      You are a backend developer.
      Follow RESTful conventions. Write integration tests for every endpoint.
      Use parameterized queries. Never use string interpolation for SQL.
    model: %q
    effort: "high"
    tools: [Bash, Read, Write, Edit, Glob, Grep]
    skills: [api-testing]
    rules: [api-conventions]

skills:
  api-testing:
    description: "REST API testing patterns"
    instructions: |
      Write integration tests that verify HTTP status codes, response shapes,
      and error handling. Test both success and failure paths.

rules:
  api-conventions:
    description: "REST API design conventions"
    instructions: |
      Use plural nouns for resource endpoints.
      Return appropriate HTTP status codes (201 for creation, 404 for not found).
      Version APIs via URL prefix (/v1/).
    alwaysApply: true
`, projectName, model)
}

func renderCLITool(projectName, model string) string {
	return fmt.Sprintf(`version: "1.0"
project:
  name: %q
  description: "Command-line tool"

agents:
  developer:
    description: "CLI developer for commands, flags, and user-facing output."
    instructions: |
      You are a CLI developer.
      Use stdout for output, stderr for errors. Support --help on every command.
      Write unit tests for command logic and integration tests for CLI invocation.
    model: %q
    effort: "high"
    tools: [Bash, Read, Write, Edit, Glob, Grep]
    rules: [cli-conventions]

rules:
  cli-conventions:
    description: "CLI design conventions"
    instructions: |
      Use meaningful exit codes (0 success, 1 user error, 2 system error).
      Support --json flag for machine-readable output where applicable.
      Never prompt for input when stdin is not a TTY.
    alwaysApply: true
`, projectName, model)
}

func renderFrontendApp(projectName, model string) string {
	return fmt.Sprintf(`version: "1.0"
project:
  name: %q
  description: "Frontend web application"

agents:
  frontend:
    description: "Frontend developer for components, pages, and styling."
    instructions: |
      You are a frontend developer.
      Write accessible, semantic HTML. Use components for reusable UI.
      Write component tests. Never use inline styles for layout.
    model: %q
    effort: "high"
    tools: [Bash, Read, Write, Edit, Glob, Grep]
    skills: [component-testing]
    rules: [frontend-conventions]

skills:
  component-testing:
    description: "Component testing patterns"
    instructions: |
      Test components in isolation. Verify render output, user interactions,
      and accessibility attributes. Mock API calls at the network layer.

rules:
  frontend-conventions:
    description: "Frontend coding conventions"
    instructions: |
      Use semantic HTML elements. Ensure all interactive elements are keyboard accessible.
      Keep components focused -- one responsibility per component.
    alwaysApply: true
`, projectName, model)
}
