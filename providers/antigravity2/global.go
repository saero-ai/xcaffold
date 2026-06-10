package antigravity2

import (
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/registry"
)

// scanGlobal discovers Antigravity 2.0 global resources from two surfaces:
//
// CLI surface (~/.gemini/antigravity-cli/):
//
//	skills/<name>/SKILL.md   → skills
//	agents/<name>/agent.json → agents
//	mcp_config.json          → mcp (serverUrl key)
//
// Desktop app surface (~/.gemini/config/):
//
//	skills/<name>/SKILL.md   → skills
//	mcp_config.json          → mcp
//	plugins/                 → not surfaced (no GlobalScanResult field)
//	hooks.json               → not surfaced (no GlobalScanResult field)
//	sidecars/                → not surfaced (no GlobalScanResult field)
//
// Shared:
//
//	~/.gemini/GEMINI.md → rule "gemini-global"
func scanGlobal(userHome string, r *registry.GlobalScanResult) {
	cliDir := filepath.Join(userHome, ".gemini", "antigravity-cli")
	cfgDir := filepath.Join(userHome, ".gemini", "config")

	// CLI surface
	registry.ScanSkillDirs(filepath.Join(cliDir, "skills"), r.Skills)
	scanGlobalAgents(filepath.Join(cliDir, "agents"), r)
	registry.ScanMCPFromJSONFile(
		filepath.Join(cliDir, "mcp_config.json"),
		registry.MCPScanKeys{ServersKey: "mcpServers", CmdKey: "command", URLKey: "serverUrl"},
		r.MCP,
	)

	// Desktop app surface
	registry.ScanSkillDirs(filepath.Join(cfgDir, "skills"), r.Skills)
	registry.ScanMCPFromJSONFile(
		filepath.Join(cfgDir, "mcp_config.json"),
		registry.MCPScanKeys{ServersKey: "mcpServers", CmdKey: "command", URLKey: "serverUrl"},
		r.MCP,
	)

	// Shared global rule
	geminiMD := filepath.Join(userHome, ".gemini", "GEMINI.md")
	if _, err := os.Stat(geminiMD); err == nil {
		if _, exists := r.Rules["gemini-global"]; !exists {
			r.Rules["gemini-global"] = registry.GlobalRuleEntry{
				InstructionsFile: filepath.ToSlash(geminiMD),
			}
		}
	}
}

// scanGlobalAgents registers agent.json entries from a global agents directory.
// Each subdirectory is treated as an agent whose name is the directory name.
// Missing or unreadable entries are silently skipped.
func scanGlobalAgents(agentsDir string, r *registry.GlobalScanResult) {
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		agentFile := filepath.Join(agentsDir, e.Name(), "agent.json")
		if _, err := os.Stat(agentFile); err != nil {
			continue
		}
		if _, exists := r.Agents[e.Name()]; !exists {
			r.Agents[e.Name()] = registry.GlobalAgentEntry{
				InstructionsFile: filepath.ToSlash(agentFile),
			}
		}
	}
}
