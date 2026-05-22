package antigravity2

import (
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/registry"
)

// scanGlobal discovers Antigravity 2.0 global resources.
//
// Layout:
//
//	~/.gemini/antigravity-cli/skills/<name>/SKILL.md → skills
//	~/.gemini/antigravity-cli/agents/<name>/agent.json → agents
//	~/.gemini/antigravity-cli/mcp_config.json  → mcp (serverUrl key)
//	~/.gemini/config/hooks.json                → hooks (global)
//	~/.gemini/GEMINI.md                        → rule "gemini-global"
func scanGlobal(userHome string, r *registry.GlobalScanResult) {
	cliDir := filepath.Join(userHome, ".gemini", "antigravity-cli")
	cfgDir := filepath.Join(userHome, ".gemini", "config")

	registry.ScanSkillDirs(filepath.Join(cliDir, "skills"), r.Skills)
	scanGlobalAgents(filepath.Join(cliDir, "agents"), r)

	geminiMD := filepath.Join(userHome, ".gemini", "GEMINI.md")
	if _, err := os.Stat(geminiMD); err == nil {
		if _, exists := r.Rules["gemini-global"]; !exists {
			r.Rules["gemini-global"] = registry.GlobalRuleEntry{
				InstructionsFile: filepath.ToSlash(geminiMD),
			}
		}
	}

	registry.ScanMCPFromJSONFile(
		filepath.Join(cliDir, "mcp_config.json"),
		registry.MCPScanKeys{ServersKey: "mcpServers", CmdKey: "command", URLKey: "serverUrl"},
		r.MCP,
	)

	_ = cfgDir // hooks.json scanning is not yet surfaced in GlobalScanResult
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
