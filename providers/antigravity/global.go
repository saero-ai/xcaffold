package antigravity

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/registry"
)

// scanGlobal discovers Antigravity global resources across surfaces:
//
// CLI surface (~/.gemini/antigravity-cli/):
//
//	skills/<name>/SKILL.md   → skills
//	agents/<name>.md         → agents
//	agents/<name>/agent.md   → agents
//	mcp_config.json          → mcp (serverUrl key)
//
// Desktop app surface (~/.gemini/config/):
//
//	skills/<name>/SKILL.md   → skills
//	agents/<name>.md         → agents
//	mcp_config.json          → mcp
//
// Legacy surface (~/.gemini/antigravity/):
//
//	skills/<name>/SKILL.md   → skills
//	mcp_config.json          → mcp
//
// Shared:
//
//	~/.gemini/GEMINI.md → rule "gemini-global"
func scanGlobal(userHome string, r *registry.GlobalScanResult) {
	cliDir := filepath.Join(userHome, ".gemini", "antigravity-cli")
	cfgDir := filepath.Join(userHome, ".gemini", "config")
	legacyDir := filepath.Join(userHome, ".gemini", "antigravity")

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
	scanGlobalAgents(filepath.Join(cfgDir, "agents"), r)
	registry.ScanMCPFromJSONFile(
		filepath.Join(cfgDir, "mcp_config.json"),
		registry.MCPScanKeys{ServersKey: "mcpServers", CmdKey: "command", URLKey: "serverUrl"},
		r.MCP,
	)

	// Legacy surface
	registry.ScanSkillDirs(filepath.Join(legacyDir, "skills"), r.Skills)
	registry.ScanMCPFromJSONFile(
		filepath.Join(legacyDir, "mcp_config.json"),
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

// scanGlobalAgents registers agent definitions from a global agents directory.
// Supports both flat <name>.md and nested <name>/agent.md or <name>/agent.json.
func scanGlobalAgents(agentsDir string, r *registry.GlobalScanResult) {
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			if strings.HasSuffix(e.Name(), ".md") {
				name := strings.TrimSuffix(e.Name(), ".md")
				agentFile := filepath.Join(agentsDir, e.Name())
				if _, exists := r.Agents[name]; !exists {
					r.Agents[name] = registry.GlobalAgentEntry{
						InstructionsFile: filepath.ToSlash(agentFile),
					}
				}
			}
			continue
		}

		// Check directory-based layouts
		name := e.Name()
		agentMD := filepath.Join(agentsDir, name, "agent.md")
		if _, err := os.Stat(agentMD); err == nil {
			if _, exists := r.Agents[name]; !exists {
				r.Agents[name] = registry.GlobalAgentEntry{
					InstructionsFile: filepath.ToSlash(agentMD),
				}
			}
			continue
		}

		agentJSON := filepath.Join(agentsDir, name, "agent.json")
		if _, err := os.Stat(agentJSON); err == nil {
			if _, exists := r.Agents[name]; !exists {
				r.Agents[name] = registry.GlobalAgentEntry{
					InstructionsFile: filepath.ToSlash(agentJSON),
				}
			}
		}
	}
}
