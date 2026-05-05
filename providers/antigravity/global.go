package antigravity

import (
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/registry"
)

// scanGlobal discovers Antigravity's global resources.
//
// Layout:
//
//	~/.gemini/antigravity/skills/<name>/SKILL.md → skills
//	~/.gemini/GEMINI.md                          → rule "gemini-global"
//	~/.gemini/antigravity/mcp_config.json        → mcp (serverUrl key)
func scanGlobal(userHome string, r *registry.GlobalScanResult) {
	dir := filepath.Join(userHome, ".gemini", "antigravity")

	registry.ScanSkillDirs(filepath.Join(dir, "skills"), r.Skills)

	geminiMD := filepath.Join(userHome, ".gemini", "GEMINI.md")
	if _, err := os.Stat(geminiMD); err == nil {
		if _, exists := r.Rules["gemini-global"]; !exists {
			r.Rules["gemini-global"] = registry.GlobalRuleEntry{
				InstructionsFile: filepath.ToSlash(geminiMD),
			}
		}
	}

	registry.ScanMCPFromJSONFile(
		filepath.Join(dir, "mcp_config.json"),
		"mcpServers", "command", "serverUrl",
		r.MCP,
	)
}
