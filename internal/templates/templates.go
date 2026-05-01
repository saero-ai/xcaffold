package templates

import (
	"fmt"
	"strings"
)

// RenderProjectXCF generates the kind: project project.xcf content.
// targets is the list of provider names selected by the user.
func RenderProjectXCF(projectName string, targets []string) string {
	var sb strings.Builder

	sb.WriteString("kind: project\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString(fmt.Sprintf("name: %q\n", projectName))
	sb.WriteString("\n")
	sb.WriteString("targets:\n")
	for _, t := range targets {
		sb.WriteString(fmt.Sprintf("  - %s\n", t))
	}
	sb.WriteString("\n")
	sb.WriteString("agents:\n")
	sb.WriteString("  - xaff\n")
	sb.WriteString("skills:\n")
	sb.WriteString("  - xcaffold\n")
	sb.WriteString("rules:\n")
	sb.WriteString("  - xcf-conventions\n")
	sb.WriteString("policies:\n")
	sb.WriteString("  - require-agent-description\n")
	sb.WriteString("  - require-agent-instructions\n")

	return sb.String()
}

// RenderXaffAgentXCF generates the base xcf/agents/xaff/agent.xcf content.
// This is the universal base agent — provider override files supplement it.
func RenderXaffAgentXCF(model string, selectedTargets []string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("kind: agent\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString("name: xaff\n")
	sb.WriteString("description: \"xcaffold authoring agent. Knows the xcaffold schema, CLI commands, and provider field support.\"\n")
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("model: %q\n", model))
	sb.WriteString("\n")
	sb.WriteString("tools: [Read, Write, Edit, Bash, Glob, Grep]\n")
	sb.WriteString("skills: [xcaffold]\n")
	sb.WriteString("rules: [xcf-conventions]\n")
	sb.WriteString("---\n")
	sb.WriteString("You are Xaff, the xcaffold authoring agent.\n")
	sb.WriteString("\n")
	sb.WriteString("xcaffold is a deterministic agent configuration compiler. It compiles `.xcf` source\n")
	sb.WriteString("files into native AI provider output (.claude/, .cursor/, .gemini/, etc.).\n")
	sb.WriteString("\n")
	sb.WriteString("## Your responsibilities\n")
	sb.WriteString("\n")
	sb.WriteString("- Author and maintain `.xcf` files in the `xcf/` directory\n")
	sb.WriteString("- Run `xcaffold validate` before `xcaffold apply`\n")
	sb.WriteString("- Read `.xcaffold/schemas/*.reference` before setting any field\n")
	sb.WriteString("- Never write directly to `.claude/`, `.cursor/`, or other provider output dirs\n")
	sb.WriteString("- Use `xcaffold status` to check for drift after changes\n")
	sb.WriteString("- Use `xcaffold import` to pull provider changes back into xcf/\n")
	sb.WriteString("\n")
	sb.WriteString("## xcf/ directory conventions\n")
	sb.WriteString("\n")
	sb.WriteString("- One resource per file\n")
	sb.WriteString("- Directory-per-resource layout: `xcf/<kind>/<name>/<name>.xcf`\n")
	sb.WriteString("- Field names use kebab-case (e.g. `allowed-tools`, not `allowedTools`)\n")
	sb.WriteString("- version is always a quoted string: `version: \"1.0\"` (not `version: 1.0`)\n")
	sb.WriteString("- name uses lowercase + hyphens only: `^[a-z0-9-]+$`\n")
	sb.WriteString("\n")
	sb.WriteString("## Provider field support\n")
	sb.WriteString("\n")
	sb.WriteString("Fields vary by provider. The provider matrix comment at the top of each `.xcf`\n")
	sb.WriteString("file shows which fields are supported. Fields marked 'dropped' are silently\n")
	sb.WriteString("removed at compile time — leave them in source, xcaffold manages the drop.\n")
	sb.WriteString("\n")
	sb.WriteString("Always read `.xcaffold/schemas/<kind>.xcf.reference` before authoring a new resource.\n")

	return sb.String()
}

// RenderXaffOverrideXCF generates a per-provider override file for the Xaff agent.
// The override file supplements the base agent.xcf with provider-specific settings.
func RenderXaffOverrideXCF(target string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("kind: agent\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString("name: xaff\n")
	sb.WriteString("\n")

	switch target {
	case "claude":
		sb.WriteString("effort: \"high\"\n")
		sb.WriteString("permission-mode: default\n")
		sb.WriteString("---\n")
	case "cursor":
		sb.WriteString("readonly: false\n")
		sb.WriteString("---\n")
	case "gemini":
		sb.WriteString("---\n")
	case "copilot":
		sb.WriteString("disable-model-invocation: false\n")
		sb.WriteString("---\n")
	case "antigravity":
		sb.WriteString("---\n")
	default:
		sb.WriteString("---\n")
	}

	return sb.String()
}

// RenderXcfConventionsRuleXCF generates the xcf/rules/xcf-conventions/xcf-conventions.xcf content.
// This replaces the old generic "conventions" rule with xcaffold-specific authoring conventions.
func RenderXcfConventionsRuleXCF(selectedTargets []string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("kind: rule\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString("name: xcf-conventions\n")
	sb.WriteString("description: \"xcaffold authoring conventions. Apply when creating or editing .xcf files.\"\n")
	sb.WriteString("\n")
	sb.WriteString("activation: always\n")
	sb.WriteString("---\n")
	sb.WriteString("When authoring or editing xcaffold `.xcf` files, follow these conventions:\n")
	sb.WriteString("\n")
	sb.WriteString("## Field naming\n")
	sb.WriteString("\n")
	sb.WriteString("- Use kebab-case for all field names: `allowed-tools`, `always-apply`, `permission-mode`\n")
	sb.WriteString("- Never use camelCase or snake_case in xcf files\n")
	sb.WriteString("\n")
	sb.WriteString("## Version quoting\n")
	sb.WriteString("\n")
	sb.WriteString("- Always quote version: `version: \"1.0\"` not `version: 1.0`\n")
	sb.WriteString("- Bare numbers are parsed as floats and will fail validation\n")
	sb.WriteString("\n")
	sb.WriteString("## Name pattern\n")
	sb.WriteString("\n")
	sb.WriteString("- name must match `^[a-z0-9-]+$` — lowercase letters, digits, hyphens only\n")
	sb.WriteString("- No uppercase, no underscores, no spaces\n")
	sb.WriteString("\n")
	sb.WriteString("## Directory layout\n")
	sb.WriteString("\n")
	sb.WriteString("- One resource per file\n")
	sb.WriteString("- Use directory-per-resource: `xcf/<kind>/<name>/<name>.xcf`\n")
	sb.WriteString("- Example: `xcf/agents/my-agent/my-agent.xcf`\n")
	sb.WriteString("- Per-provider overrides: `xcf/agents/<name>/agent.<provider>.xcf`\n")
	sb.WriteString("\n")
	sb.WriteString("## Field correctness by kind\n")
	sb.WriteString("\n")
	sb.WriteString("- agent: use `tools:` (not `allowed-tools:`)\n")
	sb.WriteString("- skill: use `allowed-tools:` (not `tools:`)\n")
	sb.WriteString("- hooks: pure YAML format — no `---` frontmatter delimiters\n")
	sb.WriteString("- memory: content comes from file body, not a `content:` field\n")
	sb.WriteString("- mcp: no body and no targets (per-provider overrides not supported)\n")
	sb.WriteString("\n")
	sb.WriteString("## Never do\n")
	sb.WriteString("\n")
	sb.WriteString("- Never write directly to `.claude/`, `.cursor/`, `.gemini/`, `.github/`, or `.agents/`\n")
	sb.WriteString("- Those directories are xcaffold output — owned by `xcaffold apply`\n")
	sb.WriteString("- Always edit `xcf/` files and compile with `xcaffold apply`\n")

	return sb.String()
}

// RenderAgentXCF generates xcf/agents/<name>/<name>.xcf content.
// Kept for backwards compatibility with import pipeline.
func RenderAgentXCF(agentName, model string, selectedTargets []string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("kind: agent\n")
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", agentName))
	sb.WriteString("description: \"General software developer agent.\"\n")
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("model: %q\n", model))
	sb.WriteString("\n")
	sb.WriteString("effort: \"high\"\n")
	sb.WriteString("\n")
	sb.WriteString("tools: [Read, Write, Edit, Bash, Glob, Grep]\n")
	sb.WriteString("---\n")
	sb.WriteString("You are a software developer.\n")
	sb.WriteString("Write clean, maintainable code.\n")

	return sb.String()
}

// RenderSettingsXCF generates the xcf/settings.xcf content.
func RenderSettingsXCF(selectedTargets []string) string {
	var sb strings.Builder

	sb.WriteString("kind: settings\n")
	sb.WriteString("version: \"1.0\"\n")

	return sb.String()
}

// RenderPolicyDescriptionXCF generates the xcf/policies/require-agent-description.xcf content.
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
