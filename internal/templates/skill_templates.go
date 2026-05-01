package templates

import (
	"strings"
)

// fence is the markdown code fence delimiter.
const fence = "```"

func padField(s string) string {
	const width = 22
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// RenderXcaffoldSkillXCF generates the xcf/skills/xcaffold/xcaffold.xcf content.
// The output uses frontmatter format: YAML metadata between --- delimiters, then plain markdown body.
func RenderXcaffoldSkillXCF(targets []string) string {
	lines := []string{
		"---",
		"# kind: skill - provider field support for your selected targets",
		"#",
		"#  Field                 " + strings.Join(targets, "  "),
	}

	for _, field := range []string{"name / description", "allowed-tools", "instructions"} {
		lines = append(lines, "#  "+padField(field)+strings.Repeat("YES      ", len(targets)))
	}

	lines = append(lines, "",
		"kind: skill",
		`version: "1.0"`,
		"name: xcaffold",
		`description: >-`,
		`  Scaffolds, compiles, and validates AI agent configurations using xcaffold — the`,
		`  deterministic agent configuration compiler. Invoke when the user asks to create agents,`,
		`  skills, rules, or policies; when they want to target Claude Code, Cursor, Gemini CLI,`,
		`  GitHub Copilot, or Antigravity; when setting up a new xcaffold project; or when`,
		`  validating or applying existing xcf configurations.`,
		"allowed-tools: [Bash, Read, Edit, Glob, Grep]",
		"references:",
		"  - xcf/skills/xcaffold/references/operating-guide.md",
		"  - xcf/skills/xcaffold/references/authoring-guide.md",
		"  - .xcaffold/schemas/agent.xcf.reference",
		"  - .xcaffold/schemas/skill.xcf.reference",
		"  - .xcaffold/schemas/rule.xcf.reference",
		"  - .xcaffold/schemas/workflow.xcf.reference",
		"  - .xcaffold/schemas/mcp.xcf.reference",
		"  - .xcaffold/schemas/hooks.xcf.reference",
		"  - .xcaffold/schemas/memory.xcf.reference",
		"  - .xcaffold/schemas/cli-cheatsheet.reference",
		"---",
		"# xcaffold — Agent Configuration Compiler",
		"",
		"xcaffold compiles `.xcf` YAML source files into native AI coding agent configurations",
		"for Claude Code, Cursor, Gemini CLI, GitHub Copilot, and Antigravity. It is the",
		"determinism layer between human (or AI) authoring and provider output — preventing",
		"hallucinated field names, incorrect directory structures, and unsupported frontmatter.",
		"",
		"## When to use this skill",
		"",
		"Use xcaffold when:",
		"- The user asks to create or configure an agent, skill, rule, workflow, or MCP server",
		"- The user wants to target multiple AI providers from a single xcf source",
		`- The user says "scaffold", "initialize", or "set up xcaffold"`,
		"- You need to validate or compile agent configs for a provider",
		"- The user asks to import existing provider configs (.claude/, .cursor/, etc.) into xcaffold",
		`- The user asks "what providers support X field?"`,
		"",
		"## Critical rules",
		"",
		"1. **Never write directly to `.claude/`, `.cursor/`, `.gemini/`, `.github/`, or `.agents/`.**",
		"   Those directories are xcaffold output — owned by `xcaffold apply`. Always edit `xcf/` files.",
		"",
		"2. **Always run `xcaffold validate` before `xcaffold apply`.**",
		"   Policy violations are caught before any files are written.",
		"",
		"3. **Read the companion references before setting fields.**",
		"   See `xcf/skills/xcaffold/references/` for the operating guide, authoring guide, and `.xcaffold/schemas/` for field catalogs.",
		"",
		"4. **Use directory-per-resource layout:**",
		"   Place each resource in `xcf/<kind>/<name>/<name>.xcf`. xcaffold discovers all `.xcf` files under `xcf/` at compile time.",
	)

	return strings.Join(lines, "\n") + "\n"
}
