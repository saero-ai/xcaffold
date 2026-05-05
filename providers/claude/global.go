package claude

import (
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/registry"
)

// scanGlobal discovers Claude Code's global resources.
//
// Layout:
//
//	~/.claude/agents/*.md          → agents
//	~/.claude/skills/<name>/SKILL.md → skills
//	~/.claude/rules/*.md           → rules
//	~/.claude/CLAUDE.md            → rule "claude-memory" + MemoryFile
//	~/.claude.json (mcpServers)    → mcp
func scanGlobal(userHome string, r *registry.GlobalScanResult) {
	dir := filepath.Join(userHome, ".claude")

	registry.ScanMarkdownFilesAsAgents(filepath.Join(dir, "agents"), r.Agents)
	registry.ScanSkillDirs(filepath.Join(dir, "skills"), r.Skills)
	registry.ScanMarkdownFilesAsRules(filepath.Join(dir, "rules"), r.Rules)

	claudeMD := filepath.Join(dir, "CLAUDE.md")
	if _, err := os.Stat(claudeMD); err == nil {
		if _, exists := r.Rules["claude-memory"]; !exists {
			r.Rules["claude-memory"] = registry.GlobalRuleEntry{
				InstructionsFile: filepath.ToSlash(claudeMD),
			}
		}
		if r.MemoryFile == "" {
			r.MemoryFile = filepath.ToSlash(claudeMD)
		}
	}

	registry.ScanMCPFromJSONFile(
		filepath.Join(userHome, ".claude.json"),
		"mcpServers", "command", "url",
		r.MCP,
	)
}
