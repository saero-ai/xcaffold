package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/analyzer"
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/spf13/cobra"
)

var graphFormat string
var graphAgent string
var graphProject string
var graphFull bool
var graphScanOutput bool
var graphAll bool
var graphBlueprintFlag string

const (
	kindAgent  = "agent"
	kindSkill  = "skill"
	kindRule   = "rule"
	kindMCP    = "mcp"
	kindPolicy = "policy"
)

var graphCmd = &cobra.Command{
	Use:   "graph [file]",
	Short: "Visualize the resource dependency graph",
	Long: `xcaffold graph renders a visual map of your agent team and how it connects.

┌───────────────────────────────────────────────────────────────────┐
│                         TOPOLOGY PHASE                            │
└───────────────────────────────────────────────────────────────────┘
 • Shows all agents with their model, effort, tools, skills, rules, and MCP
 • Use --format mermaid for Mermaid markdown (embed in README or docs)
 • Use --format dot    for Graphviz DOT (pipe to dot -Tsvg for images)
 • Use --format json   for machine-readable output (used by the platform)

Formats:
  terminal  Default. ASCII art topology printed to stdout.
  mermaid   Mermaid graph syntax for embedding in markdown.
  dot       Graphviz DOT language for rendering with graphviz.
  json      Machine-readable JSON graph for programmatic use.`,
	Example: `  $ xcaffold graph
  $ xcaffold graph --format mermaid > topology.md
  $ xcaffold graph --format dot | dot -Tsvg > topology.svg
  $ xcaffold graph --format json | jq .`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().StringVar(&graphFormat, "format", "terminal", "Output format: terminal, mermaid, dot, json")
	graphCmd.Flags().StringVarP(&graphAgent, "agent", "a", "", "Target a specific agent (shows only its topology)")
	graphCmd.Flags().StringVarP(&graphProject, "project", "p", "", "Target a specific managed project by registered name or path")
	graphCmd.Flags().BoolVarP(&graphFull, "full", "f", false, "Show the fully expanded topology tree (always true if targeting an agent)")
	graphCmd.Flags().BoolVar(&graphScanOutput, "scan-output", false, "Scan compiled output directories for undeclared artifacts")
	graphCmd.Flags().BoolVar(&graphAll, "all", false, "Show global topology and all registered projects")
	graphCmd.Flags().StringVar(&graphBlueprintFlag, "blueprint", "", "Show graph for the named blueprint only")
	_ = graphCmd.Flags().MarkHidden("blueprint")
	rootCmd.AddCommand(graphCmd)
}

// graphNode represents a node in the topology graph.
type graphNode struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`  // "agent", "skill", "rule", "mcp", "hook", "settings"
	Label        string            `json:"label"` // display label
	Meta         map[string]string `json:"meta,omitempty"`
	Tools        []string          `json:"tools,omitempty"`
	BlockedTools []string          `json:"blocked_tools,omitempty"`
	Paths        []string          `json:"paths,omitempty"`
	Plugins      []string          `json:"plugins,omitempty"`
}

// graphEdge represents a directed edge.
type graphEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"` // "skill", "rule", "mcp"
}

type graphData struct {
	Project     string                   `json:"project"`
	Scope       string                   `json:"scope"`
	ConfigPath  string                   `json:"config_path"`
	DiskEntries []analyzer.ArtifactEntry `json:"disk_entries,omitempty"`
	Nodes       []graphNode              `json:"nodes"`
	Edges       []graphEdge              `json:"edges"`
}

func runGraph(cmd *cobra.Command, args []string) error {
	// Mutual exclusion checks
	if graphBlueprintFlag != "" && globalFlag {
		return fmt.Errorf("--blueprint cannot be used with --global (blueprints are project-scoped)")
	}
	if graphAll && globalFlag {
		return fmt.Errorf("--all and --global are mutually exclusive")
	}
	if graphAll && graphProject != "" {
		return fmt.Errorf("--all and --project are mutually exclusive")
	}

	// Terminal mode handles its own parsing to avoid duplicate warnings.
	if strings.ToLower(graphFormat) == "terminal" || graphFormat == "" {
		return runGraphTerminalMode()
	}

	var scopes []*graphData

	if graphAll {
		// Global topology
		gGlobal, err := parseGraphData(globalXcfPath, "global")
		if err != nil {
			return err
		}
		scopes = append(scopes, gGlobal)

		// All registered projects
		projects, err := registry.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not list registered projects: %v\n", err)
		} else {
			for _, p := range projects {
				projXcf := filepath.Join(p.Path, "project.xcf")
				if p.ConfigDir != "" && p.ConfigDir != "." {
					projXcf = filepath.Join(p.Path, p.ConfigDir, "project.xcf")
				}
				g, err := parseGraphData(projXcf, fmt.Sprintf("project:%s", p.Name))
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: skipping project %q: %v\n", p.Name, err)
					continue
				}
				scopes = append(scopes, g)
			}
		}
	} else if graphProject != "" {
		p, err := registry.Resolve(graphProject)
		if err != nil {
			return err
		}
		projXcf := filepath.Join(p.Path, "project.xcf")
		g, err := parseGraphData(projXcf, fmt.Sprintf("project:%s", p.Name))
		if err != nil {
			return err
		}
		scopes = append(scopes, g)
	} else if len(args) > 0 {
		g, err := parseGraphData(args[0], "")
		if err != nil {
			return err
		}
		scopes = append(scopes, g)
	} else if globalFlag {
		g, err := parseGraphData(globalXcfPath, "global")
		if err != nil {
			return err
		}
		scopes = append(scopes, g)
	} else {
		g, err := parseGraphData(xcfPath, "project")
		if err != nil {
			return err
		}
		scopes = append(scopes, g)
	}

	if len(scopes) == 0 {
		return fmt.Errorf("no topology generated")
	}

	return printGraphOutput(scopes)
}

//nolint:gocyclo
func printGraphOutput(scopes []*graphData) error {
	switch strings.ToLower(graphFormat) {
	case "mermaid":
		for _, g := range scopes {
			fmt.Print(renderMermaid(g))
		}
	case "dot":
		for _, g := range scopes {
			fmt.Print(renderDOT(g))
		}
	case "json":
		b, err := json.MarshalIndent(scopes, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	default:
		return fmt.Errorf("unknown format %q", graphFormat)
	}
	return nil
}

//nolint:gocyclo
func parseGraphData(configPath, scopeName string) (*graphData, error) {
	config, err := parser.ParseDirectory(projectParseRoot())
	if err != nil {
		if scopeName != "" {
			return nil, fmt.Errorf("[%s] parse error: %w", scopeName, err)
		}
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if graphAgent != "" {
		if _, ok := config.Agents[graphAgent]; !ok {
			return nil, fmt.Errorf("agent %q not found in %s", graphAgent, configPath)
		}

		targetAgent := config.Agents[graphAgent]
		config.Agents = map[string]ast.AgentConfig{graphAgent: targetAgent}

		filteredSkills := make(map[string]ast.SkillConfig)
		for _, s := range targetAgent.Skills {
			if sk, ok := config.Skills[s]; ok {
				filteredSkills[s] = sk
			}
		}
		config.Skills = filteredSkills

		filteredRules := make(map[string]ast.RuleConfig)
		for _, r := range targetAgent.Rules {
			if ru, ok := config.Rules[r]; ok {
				filteredRules[r] = ru
			}
		}
		config.Rules = filteredRules

		filteredMCP := make(map[string]ast.MCPConfig)
		for _, m := range targetAgent.MCP {
			if mcp, ok := config.MCP[m]; ok {
				filteredMCP[m] = mcp
			}
		}
		config.MCP = filteredMCP
	}

	if graphBlueprintFlag != "" {
		filtered, err := blueprint.ApplyBlueprint(config, graphBlueprintFlag)
		if err != nil {
			return nil, fmt.Errorf("blueprint %q: %w", graphBlueprintFlag, err)
		}
		config = filtered
	}

	if scopeName != "global" {
		config.StripInherited()
	}
	g := buildGraph(config)
	g.Scope = scopeName
	g.ConfigPath = configPath

	if graphScanOutput {
		a := analyzer.New()
		declared := make(map[string]bool)
		for _, n := range g.Nodes {
			if n.Kind != "settings" {
				declared[fmt.Sprintf("%s:%s", n.Kind, n.ID)] = true
			}
		}

		baseDir := filepath.Dir(configPath)
		targetDir := filepath.Join(baseDir, compiler.OutputDir(targetFlag))

		entries, err := a.ScanOutputDir(targetDir, declared)
		if err == nil {
			g.DiskEntries = entries
		}
	}

	return g, nil
}

func buildGraph(config *ast.XcaffoldConfig) *graphData {
	var projectName string
	if config.Project != nil {
		projectName = config.Project.Name
	}
	g := &graphData{Project: projectName}
	appendGraphSettings(config, g)
	appendGraphSkills(config, g)
	appendGraphRules(config, g)
	appendGraphMCP(config, g)
	appendGraphPolicies(config, g)
	appendGraphAgents(config, g)
	appendGraphHooks(config, g)
	appendGraphWorkflows(config, g)
	return g
}

func appendGraphSettings(config *ast.XcaffoldConfig, g *graphData) {
	// Get the active settings (first available key after blueprint filtering).
	var es ast.SettingsConfig
	for _, s := range config.Settings {
		es = s
		break
	}
	if len(es.EnabledPlugins) > 0 {
		plugins := make([]string, 0, len(es.EnabledPlugins))
		for p, enabled := range es.EnabledPlugins {
			if enabled {
				plugins = append(plugins, p)
			}
		}
		sort.Strings(plugins)
		g.Nodes = append(g.Nodes, graphNode{
			ID:      "settings:global",
			Kind:    "settings",
			Label:   "Settings",
			Plugins: plugins,
		})
	}
}

func appendGraphSkills(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Skills) {
		skill := config.Skills[id]
		label := id
		if skill.Name != "" {
			label = skill.Name
		}
		var tools []string
		if len(skill.AllowedTools) > 0 {
			tools = skill.AllowedTools
		}
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "skill:" + id,
			Kind:  kindSkill,
			Label: label,
			Tools: tools,
		})
	}
}

func appendGraphRules(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Rules) {
		rule := config.Rules[id]
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "rule:" + id,
			Kind:  kindRule,
			Label: id,
			Paths: rule.Paths,
		})
	}
}

func appendGraphMCP(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.MCP) {
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "mcp:" + id,
			Kind:  kindMCP,
			Label: id,
		})
	}
}

func appendGraphAgents(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Agents) {
		agent := config.Agents[id]
		meta := map[string]string{}
		if agent.Model != "" {
			meta["model"] = agent.Model
		}
		if agent.Effort != "" {
			meta["effort"] = agent.Effort
		}
		if len(agent.Memory) > 0 {
			meta["memory"] = strings.Join([]string(agent.Memory), ", ")
		}

		label := id
		if agent.Name != "" {
			label = agent.Name
		}

		g.Nodes = append(g.Nodes, graphNode{
			ID:           "agent:" + id,
			Kind:         kindAgent,
			Label:        label,
			Meta:         meta,
			Tools:        agent.Tools,
			BlockedTools: agent.DisallowedTools,
		})

		for _, skillID := range agent.Skills {
			g.Edges = append(g.Edges, graphEdge{From: "agent:" + id, To: "skill:" + skillID, Label: "skill"})
		}
		for _, ruleID := range agent.Rules {
			g.Edges = append(g.Edges, graphEdge{From: "agent:" + id, To: "rule:" + ruleID, Label: "rule"})
		}
		for _, mcpID := range agent.MCP {
			g.Edges = append(g.Edges, graphEdge{From: "agent:" + id, To: "mcp:" + mcpID, Label: "mcp"})
		}
	}
}

func appendGraphPolicies(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Policies) {
		p := config.Policies[id]
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "policy:" + id,
			Kind:  kindPolicy,
			Label: id,
			Meta:  map[string]string{"severity": p.Severity, "target": p.Target},
		})
	}
}

func appendGraphHooks(config *ast.XcaffoldConfig, g *graphData) {
	var effectiveHooks ast.HookConfig
	if dh, ok := config.Hooks["default"]; ok {
		effectiveHooks = dh.Events
	}
	for _, event := range sortedKeys(effectiveHooks) {
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "hook:" + event,
			Kind:  "hook",
			Label: event,
		})
	}
}

func appendGraphWorkflows(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Workflows) {
		wf := config.Workflows[id]
		label := id
		if wf.Description != "" {
			label = wf.Description
		}
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "workflow:" + id,
			Kind:  "workflow",
			Label: label,
		})
	}
}

func renderTerminal(g *graphData) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%s\n", renderTerminalHeader(g))

	if globals := renderTerminalGlobals(g); globals != "" {
		fmt.Fprintf(&sb, "%s\n", globals)
	}

	if agents := renderTerminalAgents(g); agents != "" {
		fmt.Fprintf(&sb, "%s\n", agents)
	}

	if library := renderTerminalLibrary(g); library != "" {
		fmt.Fprintf(&sb, "%s\n", library)
	}

	if orphans := renderTerminalOrphans(g); orphans != "" {
		fmt.Fprintf(&sb, "%s\n", orphans)
	}

	return sb.String()
}

func renderTerminalHeader(g *graphData) string {
	agents, skills, rules, mcps, hooks, policies := 0, 0, 0, 0, 0, 0
	for _, n := range g.Nodes {
		switch n.Kind {
		case kindAgent:
			agents++
		case kindSkill:
			skills++
		case kindRule:
			rules++
		case kindMCP:
			mcps++
		case kindPolicy:
			policies++
		case "hook":
			hooks++
		}
	}

	var parts []string
	if agents > 0 {
		parts = append(parts, fmt.Sprintf("%d agents", agents))
	}
	if skills > 0 {
		parts = append(parts, fmt.Sprintf("%d skills", skills))
	}
	if rules > 0 {
		parts = append(parts, fmt.Sprintf("%d rules", rules))
	}
	if mcps > 0 {
		label := plural(mcps, "mcp server", "mcp servers")
		parts = append(parts, fmt.Sprintf("%d %s", mcps, label))
	}
	if policies > 0 {
		parts = append(parts, fmt.Sprintf("%d policies", policies))
	}
	if hooks > 0 {
		parts = append(parts, fmt.Sprintf("%d hooks", hooks))
	}

	sep := "  " + glyphDot() + "  "
	header := fmt.Sprintf("  %s%s%s  ", g.Project, sep, strings.Join(parts, sep))
	width := len(header) + 2
	border := strings.Repeat("─", width)

	return fmt.Sprintf("\n┌%s┐\n│%s│\n└%s┘\n", border, header, border)
}

func renderTerminalGlobals(g *graphData) string {
	var globals []string
	for _, n := range g.Nodes {
		if n.Kind == "settings" {
			globals = append(globals, "  ● Settings")
			for i, p := range n.Plugins {
				prefix := "      ├─(plugin)─▶ "
				if i == len(n.Plugins)-1 {
					prefix = "      └─(plugin)─▶ "
				}
				globals = append(globals, prefix+p+" [Enabled]")
			}
		}
	}
	if len(globals) > 0 {
		return "  [ GLOBAL ENVIRONMENT ]\n" + strings.Join(globals, "\n") + "\n"
	}
	return ""
}

func renderTerminalLibrary(g *graphData) string {
	var library []string
	for _, n := range g.Nodes {
		var lines []string
		if n.Kind == kindSkill {
			lines = append(lines, fmt.Sprintf("  ● skill: %s", n.Label))
			if len(n.Tools) > 0 {
				lines = append(lines, "      ├─(tools)─▶ "+strings.Join(n.Tools, ", "))
			}
			if len(n.Paths) > 0 {
				lines = append(lines, "      └─(paths)─▶ "+strings.Join(n.Paths, ", "))
			} else if len(n.Tools) > 0 {
				lines[1] = strings.Replace(lines[1], "      ├─", "      └─", 1)
			}
		} else if n.Kind == kindRule {
			lines = append(lines, fmt.Sprintf("  ● rule: %s", n.Label))
			if len(n.Paths) > 0 {
				lines = append(lines, "      └─(paths)─▶ "+strings.Join(n.Paths, ", "))
			}
		}
		if len(lines) > 1 {
			library = append(library, strings.Join(lines, "\n"))
		}
	}

	if len(library) > 0 {
		return "  [ LIBRARY ]\n" + strings.Join(library, "\n\n") + "\n"
	}
	return ""
}

func renderTerminalOrphans(g *graphData) string {
	referenced := map[string]bool{}
	for _, e := range g.Edges {
		referenced[e.To] = true
	}
	orphanSkills, orphanRules, orphanMCP, orphanPolicies := []string{}, []string{}, []string{}, []string{}
	for _, n := range g.Nodes {
		if n.Kind == kindSkill && !referenced[n.ID] {
			orphanSkills = append(orphanSkills, n.Label)
		}
		if n.Kind == kindRule && !referenced[n.ID] {
			orphanRules = append(orphanRules, n.Label)
		}
		if n.Kind == kindMCP && !referenced[n.ID] {
			orphanMCP = append(orphanMCP, n.Label)
		}
		if n.Kind == kindPolicy && !referenced[n.ID] {
			orphanPolicies = append(orphanPolicies, n.Label)
		}
	}
	var sb strings.Builder
	if len(orphanSkills) > 0 {
		fmt.Fprintf(&sb, "  Unreferenced skills:    %s\n", strings.Join(orphanSkills, ", "))
	}
	if len(orphanRules) > 0 {
		fmt.Fprintf(&sb, "  Unreferenced rules:     %s\n", strings.Join(orphanRules, ", "))
	}
	if len(orphanMCP) > 0 {
		fmt.Fprintf(&sb, "  Unreferenced mcp:       %s\n", strings.Join(orphanMCP, ", "))
	}
	if len(orphanPolicies) > 0 {
		fmt.Fprintf(&sb, "  Unreferenced policies:  %s\n", strings.Join(orphanPolicies, ", "))
	}
	return sb.String()
}

func renderTerminalAgents(g *graphData) string {
	var agentsBlocks []string
	for _, node := range g.Nodes {
		if node.Kind != kindAgent {
			continue
		}
		if agentStr := renderTerminalAgent(node, g.Edges, g); agentStr != "" {
			agentsBlocks = append(agentsBlocks, agentStr)
		}
	}

	if len(agentsBlocks) > 0 {
		return "  [ AGENTS ]\n" + strings.Join(agentsBlocks, "\n\n") + "\n"
	}
	return ""
}

func renderTerminalAgent(node graphNode, edges []graphEdge, g *graphData) string {
	metaStr := renderAgentMeta(node)
	agentStr := fmt.Sprintf("  ● %s%s\n      │", node.Label, metaStr)
	blocks := []string{}

	if capBlock := renderAgentCapabilities(node); capBlock != "" {
		blocks = append(blocks, capBlock)
	}

	skillsList, rulesList := extractAgentRelations(node.ID, edges)

	if skillsBlock := renderAgentSkills(skillsList); skillsBlock != "" {
		blocks = append(blocks, skillsBlock)
	}
	if rulesBlock := renderAgentRules(rulesList); rulesBlock != "" {
		blocks = append(blocks, rulesBlock)
	}
	if serversBlock := renderAgentServers(node.ID, edges, &blocks); serversBlock != "" {
		blocks = append(blocks, serversBlock)
	}

	if len(blocks) > 0 {
		agentStr += "\n" + strings.Join(blocks, "\n      │\n")
	} else {
		agentStr = strings.TrimSuffix(agentStr, "\n      │")
	}

	return agentStr
}

func renderAgentMeta(node graphNode) string {
	metaParts := []string{}
	if m, ok := node.Meta["model"]; ok {
		short := m
		if idx := strings.LastIndex(m, "-"); idx > 0 && len(m) > 20 {
			short = "..." + m[idx:]
		}
		metaParts = append(metaParts, short)
	}
	if e, ok := node.Meta["effort"]; ok {
		metaParts = append(metaParts, e+" effort")
	}
	if len(metaParts) > 0 {
		return " [" + strings.Join(metaParts, " · ") + "]"
	}
	return ""
}

//nolint:gocyclo
func renderAgentCapabilities(node graphNode) string {
	if len(node.Tools) == 0 && len(node.BlockedTools) == 0 {
		return ""
	}

	type groupDef struct {
		Name  string
		Tools []string
	}

	groupAndFormat := func(list []string, kind string, isLastBlock bool) []string {
		var std []string
		mcp := map[string][]string{}

		for _, t := range list {
			if strings.HasPrefix(t, "mcp__") {
				parts := strings.SplitN(t, "__", 3)
				if len(parts) == 3 {
					grp := "mcp__" + parts[1]
					mcp[grp] = append(mcp[grp], parts[2])
					continue
				}
			}
			std = append(std, t)
		}

		var sortedGroups []groupDef
		for k, v := range mcp {
			sort.Strings(v)
			sortedGroups = append(sortedGroups, groupDef{Name: k, Tools: v})
		}
		sort.Slice(sortedGroups, func(i, j int) bool { return sortedGroups[i].Name < sortedGroups[j].Name })

		totalItems := len(std) + len(sortedGroups)
		var out []string
		currentIdx := 0

		spacing := strings.Repeat(" ", 11+len(kind))
		remSpace := spacing[5:] // 5 char offset for "    │"

		for _, t := range std {
			isLast := currentIdx == totalItems-1 && isLastBlock
			prefix := "      │    ├─(" + kind + ")─▶ "
			if isLast {
				prefix = "      │    └─(" + kind + ")─▶ "
			}
			out = append(out, prefix+t)
			currentIdx++
		}

		for _, g := range sortedGroups {
			isLastGroup := currentIdx == totalItems-1 && isLastBlock
			prefix := "      │    ├─(" + kind + ")─▶ "
			if isLastGroup {
				prefix = "      │    └─(" + kind + ")─▶ "
			}
			out = append(out, prefix+g.Name)

			for i, ct := range g.Tools {
				var childPrefix string
				if isLastGroup {
					childPrefix = "      │" + spacing + "├─▶ "
					if i == len(g.Tools)-1 {
						childPrefix = "      │" + spacing + "└─▶ "
					}
				} else {
					childPrefix = "      │    │" + remSpace + "├─▶ "
					if i == len(g.Tools)-1 {
						childPrefix = "      │    │" + remSpace + "└─▶ "
					}
				}
				out = append(out, childPrefix+ct)
			}
			currentIdx++
		}
		return out
	}

	var capLines []string
	if len(node.Tools) > 0 {
		capLines = append(capLines, groupAndFormat(node.Tools, "tool", len(node.BlockedTools) == 0)...)
	}
	if len(node.BlockedTools) > 0 {
		capLines = append(capLines, groupAndFormat(node.BlockedTools, "blocked", true)...)
	}

	return "      ├─▶ [Capabilities]\n" + strings.Join(capLines, "\n")
}

func extractAgentRelations(nodeID string, edges []graphEdge) ([]string, []string) {
	var skillsList []string
	var rulesList []string
	for _, edge := range edges {
		if edge.From == nodeID {
			target := strings.SplitN(edge.To, ":", 2)
			if len(target) == 2 {
				if edge.Label == kindSkill {
					skillsList = append(skillsList, target[1])
				} else if edge.Label == kindRule {
					rulesList = append(rulesList, target[1])
				}
			}
		}
	}
	return skillsList, rulesList
}

const treePrefixMid = "      │    ├─▶ "
const treePrefixLast = "      │    └─▶ "

func renderAgentSkills(skillsList []string) string {
	if len(skillsList) == 0 {
		return ""
	}
	lines := []string{}
	for i, inf := range skillsList {
		prefix := treePrefixMid
		if i == len(skillsList)-1 {
			prefix = treePrefixLast
		}
		lines = append(lines, prefix+inf)
	}
	return "      ├─▶ [Skills]\n" + strings.Join(lines, "\n")
}

func renderAgentRules(rulesList []string) string {
	if len(rulesList) == 0 {
		return ""
	}
	lines := []string{}
	for i, inf := range rulesList {
		prefix := treePrefixMid
		if i == len(rulesList)-1 {
			prefix = treePrefixLast
		}
		lines = append(lines, prefix+inf)
	}
	return "      └─▶ [Rules]\n" + strings.Join(lines, "\n")
}

func renderAgentServers(nodeID string, edges []graphEdge, blocks *[]string) string {
	var servers []string
	for _, edge := range edges {
		if edge.From == nodeID && edge.Label == "mcp" {
			target := strings.SplitN(edge.To, ":", 2)
			if len(target) == 2 {
				servers = append(servers, "      │    └─(mcp)─▶ "+target[1])
			}
		}
	}
	if len(servers) > 0 {
		return "      └─▶ [Servers]\n" + strings.Join(servers, "\n")
	} else if len(*blocks) > 0 {
		lastBlock := (*blocks)[len(*blocks)-1]
		lastBlock = strings.Replace(lastBlock, "      ├─▶", "      └─▶", 1)
		lastBlock = strings.ReplaceAll(lastBlock, "      │    ", "           ")
		(*blocks)[len(*blocks)-1] = lastBlock
	}
	return ""
}

func renderMermaid(g *graphData) string {
	var sb strings.Builder
	sb.WriteString("```mermaid\ngraph LR\n")

	// Subgraphs by kind
	agentNodes, skillNodes, ruleNodes, mcpNodes, policyNodes := []graphNode{}, []graphNode{}, []graphNode{}, []graphNode{}, []graphNode{}
	for _, n := range g.Nodes {
		switch n.Kind {
		case kindAgent:
			agentNodes = append(agentNodes, n)
		case kindSkill:
			skillNodes = append(skillNodes, n)
		case kindRule:
			ruleNodes = append(ruleNodes, n)
		case kindMCP:
			mcpNodes = append(mcpNodes, n)
		case kindPolicy:
			policyNodes = append(policyNodes, n)
		}
	}

	appendMermaidAgentSub(&sb, agentNodes)
	appendMermaidSimpleSub(&sb, "Skills", skillNodes)
	appendMermaidSimpleSub(&sb, "Rules", ruleNodes)
	appendMermaidSimpleSub(&sb, "MCP", mcpNodes)
	appendMermaidSimpleSub(&sb, "Policies", policyNodes)

	// Edges
	for _, e := range g.Edges {
		fmt.Fprintf(&sb, "  %s -->|\"%s\"| %s\n", mermaidID(e.From), e.Label, mermaidID(e.To))
	}

	sb.WriteString("```\n")
	return sb.String()
}

func appendMermaidAgentSub(sb *strings.Builder, nodes []graphNode) {
	if len(nodes) == 0 {
		return
	}
	sb.WriteString("  subgraph Agents\n")
	for _, n := range nodes {
		id := mermaidID(n.ID)
		label := n.Label
		if m, ok := n.Meta["model"]; ok {
			label += " / " + m
		}
		fmt.Fprintf(sb, "    %s[\"%s\"]\n", id, label)
	}
	sb.WriteString("  end\n")
}

func appendMermaidSimpleSub(sb *strings.Builder, title string, nodes []graphNode) {
	if len(nodes) == 0 {
		return
	}
	fmt.Fprintf(sb, "  subgraph %s\n", title)
	for _, n := range nodes {
		fmt.Fprintf(sb, "    %s[\"%s\"]\n", mermaidID(n.ID), n.Label)
	}
	sb.WriteString("  end\n")
}

func renderDOT(g *graphData) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "digraph %q {\n", g.Project)
	sb.WriteString("  graph [rankdir=LR fontname=Helvetica]\n")
	sb.WriteString("  node  [fontname=Helvetica shape=box style=rounded]\n\n")

	colors := map[string]string{
		"agent":  "#4A90D9",
		"skill":  "#7ED321",
		"rule":   "#F5A623",
		"mcp":    "#9B59B6",
		"hook":   "#E74C3C",
		"policy": "#E67E22",
	}
	for _, n := range g.Nodes {
		color := colors[n.Kind]
		if color == "" {
			color = "#cccccc"
		}
		dotID := dotSafeID(n.ID)
		label := n.Label
		if n.Kind == kindAgent {
			if m, ok := n.Meta["model"]; ok {
				label += "\\n" + m
			}
		}
		fmt.Fprintf(&sb, "  %s [label=%q fillcolor=%q style=\"rounded,filled\" fontcolor=white]\n",
			dotID, label, color)
	}

	sb.WriteString("\n")
	for _, e := range g.Edges {
		fmt.Fprintf(&sb, "  %s -> %s [label=%q]\n", dotSafeID(e.From), dotSafeID(e.To), e.Label)
	}
	sb.WriteString("}\n")
	return sb.String()
}

// mermaidID converts a node ID to a valid Mermaid node identifier.
func mermaidID(id string) string {
	return strings.NewReplacer(":", "_", "-", "_", ".", "_", " ", "_").Replace(id)
}

// dotSafeID converts a node ID to a valid Graphviz DOT identifier.
func dotSafeID(id string) string {
	return strings.NewReplacer(":", "_", "-", "_", ".", "_", " ", "_").Replace(id)
}

// sortedKeys returns sorted keys of any map[string]V.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func renderTerminalSummary(scopes []*graphData) string {
	var sb strings.Builder

	for _, g := range scopes {
		if g.Scope == "global" {
			sb.WriteString("\n[ GLOBAL ]\n")
		} else if g.Scope == "project" {
			sb.WriteString("\n[ PROJECTS ]\n")
		} else {
			sb.WriteString(fmt.Sprintf("\n[ TARGET: %s ]\n", g.ConfigPath))
		}

		renderScopeSummary(&sb, g)
	}

	return sb.String() + "\n"
}

//nolint:gocyclo
func renderScopeSummary(sb *strings.Builder, g *graphData) {
	var agents, skills, rules, mcp, policies, orphans int
	for _, n := range g.Nodes {
		switch n.Kind {
		case kindAgent:
			agents++
		case kindSkill:
			skills++
		case kindRule:
			rules++
		case kindMCP:
			mcp++
		case kindPolicy:
			policies++
		}
	}

	referenced := make(map[string]bool)
	for _, e := range g.Edges {
		referenced[e.To] = true
	}
	for _, n := range g.Nodes {
		if n.Kind == kindSkill && !referenced[n.ID] {
			orphans++
		}
	}

	label := g.Project
	if label == "" && g.Scope == "global" {
		label = "Global Context"
	} else if label == "" {
		label = "Unnamed Context"
	}
	sb.WriteString(fmt.Sprintf("  ● %s\n    (%s)\n", label, g.ConfigPath))

	agentNodes := []graphNode{}
	for _, n := range g.Nodes {
		if n.Kind == kindAgent {
			agentNodes = append(agentNodes, n)
		}
	}

	type summaryLine struct {
		Title string
		Kind  string
		Value int
	}
	var lines []summaryLine
	if len(agentNodes) > 0 {
		lines = append(lines, summaryLine{"Agents", kindAgent, len(agentNodes)})
	}
	if rules > 0 {
		lines = append(lines, summaryLine{"Rules", kindRule, rules})
	}
	if mcp > 0 {
		lines = append(lines, summaryLine{"MCP Servers", kindMCP, mcp})
	}
	if policies > 0 {
		lines = append(lines, summaryLine{"Policies", kindPolicy, policies})
	}
	if orphans > 0 {
		lines = append(lines, summaryLine{"Unreferenced Skills", kindSkill, orphans})
	}

	for idx, ln := range lines {
		prefix := "      ├─▶ "
		if idx == len(lines)-1 {
			prefix = "      └─▶ "
		}
		sb.WriteString(fmt.Sprintf("%s%s: (%d)\n", prefix, ln.Title, ln.Value))

		if ln.Title == "Agents" {
			for i, n := range agentNodes {
				aprefix := treePrefixMid
				if i == len(agentNodes)-1 {
					aprefix = treePrefixLast
				}
				if idx == len(lines)-1 { // Agent line was the last one
					aprefix = "           ├─▶ "
					if i == len(agentNodes)-1 {
						aprefix = "           └─▶ "
					}
				}
				sb.WriteString(fmt.Sprintf("%s%s\n", aprefix, n.Label))
			}
		}
	}
}

func renderTerminalDiskEntries(g *graphData) string {
	var sb strings.Builder
	sb.WriteString("\n  [ UNDECLARED FILES ]  (!)\n")
	for _, e := range g.DiskEntries {
		sb.WriteString(fmt.Sprintf("      - [%s] %s\n", e.Kind, e.ID))
	}
	return sb.String()
}
