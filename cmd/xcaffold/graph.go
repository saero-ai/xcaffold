package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
)

var graphFormat string

var graphCmd = &cobra.Command{
	Use:   "graph [file]",
	Short: "Visualize the agent topology of a scaffold.xcf file",
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
	rootCmd.AddCommand(graphCmd)
}

// graphNode represents a node in the topology graph.
type graphNode struct {
	ID           string
	Kind         string // "agent", "skill", "rule", "mcp", "hook", "settings"
	Label        string // display label
	Meta         map[string]string
	Tools        []string
	BlockedTools []string
	Paths        []string
	Plugins      []string
}

// graphEdge represents a directed edge.
type graphEdge struct {
	From  string
	To    string
	Label string // "skill", "rule", "mcp"
}

// graphData is the full in-memory topology.
type graphData struct {
	Project string
	Nodes   []graphNode
	Edges   []graphEdge
}

func runGraph(cmd *cobra.Command, args []string) error {
	// An explicit file argument bypasses scope resolution entirely.
	if len(args) > 0 {
		return graphScope(args[0], "")
	}

	if scopeFlag == "global" || scopeFlag == "all" {
		if err := graphScope(globalXcfPath, "global"); err != nil {
			return err
		}
	}
	if scopeFlag == "project" || scopeFlag == "all" {
		if err := graphScope(xcfPath, "project"); err != nil {
			return err
		}
	}
	return nil
}

func graphScope(configPath, scopeName string) error {
	config, err := parser.ParseFile(configPath)
	if err != nil {
		if scopeName != "" {
			return fmt.Errorf("[%s] parse error: %w", scopeName, err)
		}
		return fmt.Errorf("parse error: %w", err)
	}

	g := buildGraph(config)

	switch strings.ToLower(graphFormat) {
	case "terminal", "":
		fmt.Print(renderTerminal(g))
	case "mermaid":
		fmt.Print(renderMermaid(g))
	case "dot":
		fmt.Print(renderDOT(g))
	case "json":
		b, err := json.MarshalIndent(g, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	default:
		return fmt.Errorf("unknown format %q — valid values: terminal, mermaid, dot, json", graphFormat)
	}
	return nil
}

func buildGraph(config *ast.XcaffoldConfig) *graphData {
	g := &graphData{Project: config.Project.Name}

	// Global Settings node
	if len(config.Settings.EnabledPlugins) > 0 {
		plugins := make([]string, 0, len(config.Settings.EnabledPlugins))
		for p, enabled := range config.Settings.EnabledPlugins {
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

	// Skill nodes
	skillIDs := sortedKeys(config.Skills)
	for _, id := range skillIDs {
		skill := config.Skills[id]
		label := id
		if skill.Name != "" {
			label = skill.Name
		}
		var tools []string
		if len(skill.Tools) > 0 {
			tools = skill.Tools
		} else if len(skill.AllowedTools) > 0 {
			tools = skill.AllowedTools
		}

		g.Nodes = append(g.Nodes, graphNode{
			ID:    "skill:" + id,
			Kind:  "skill",
			Label: label,
			Tools: tools,
			Paths: skill.Paths,
		})
	}

	// Rule nodes
	ruleIDs := sortedKeys(config.Rules)
	for _, id := range ruleIDs {
		rule := config.Rules[id]
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "rule:" + id,
			Kind:  "rule",
			Label: id,
			Paths: rule.Paths,
		})
	}

	// MCP nodes
	mcpIDs := sortedKeys(config.MCP)
	for _, id := range mcpIDs {
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "mcp:" + id,
			Kind:  "mcp",
			Label: id,
		})
	}

	// Agent nodes + edges
	agentIDs := sortedKeys(config.Agents)
	for _, id := range agentIDs {
		agent := config.Agents[id]
		meta := map[string]string{}
		if agent.Model != "" {
			meta["model"] = agent.Model
		}
		if agent.Effort != "" {
			meta["effort"] = agent.Effort
		}
		if agent.Memory != "" {
			meta["memory"] = agent.Memory
		}

		label := id
		if agent.Name != "" {
			label = agent.Name
		}

		g.Nodes = append(g.Nodes, graphNode{
			ID:           "agent:" + id,
			Kind:         "agent",
			Label:        label,
			Meta:         meta,
			Tools:        agent.Tools,
			BlockedTools: agent.DisallowedTools,
		})

		for _, skillID := range agent.Skills {
			g.Edges = append(g.Edges, graphEdge{
				From:  "agent:" + id,
				To:    "skill:" + skillID,
				Label: "skill",
			})
		}
		for _, ruleID := range agent.Rules {
			g.Edges = append(g.Edges, graphEdge{
				From:  "agent:" + id,
				To:    "rule:" + ruleID,
				Label: "rule",
			})
		}
		for _, mcpID := range agent.MCP {
			g.Edges = append(g.Edges, graphEdge{
				From:  "agent:" + id,
				To:    "mcp:" + mcpID,
				Label: "mcp",
			})
		}
	}

	return g
}

func renderTerminal(g *graphData) string {
	var sb strings.Builder

	// Count by kind
	agents, skills, rules, mcps, hooks := 0, 0, 0, 0, 0
	for _, n := range g.Nodes {
		switch n.Kind {
		case "agent":
			agents++
		case "skill":
			skills++
		case "rule":
			rules++
		case "mcp":
			mcps++
		case "hook":
			hooks++
		}
	}

	parts := []string{fmt.Sprintf("%d agents", agents)}
	if skills > 0 {
		parts = append(parts, fmt.Sprintf("%d skills", skills))
	}
	if rules > 0 {
		parts = append(parts, fmt.Sprintf("%d rules", rules))
	}
	if mcps > 0 {
		parts = append(parts, fmt.Sprintf("%d mcp servers", mcps))
	}
	if hooks > 0 {
		parts = append(parts, fmt.Sprintf("%d hooks", hooks))
	}

	header := fmt.Sprintf("  %s  •  %s  ", g.Project, strings.Join(parts, "  •  "))
	width := len(header) + 2
	border := strings.Repeat("─", width)

	fmt.Fprintf(&sb, "\n┌%s┐\n", border)
	fmt.Fprintf(&sb, "│%s│\n", header)
	fmt.Fprintf(&sb, "└%s┘\n\n", border)

	// [ GLOBAL ENVIRONMENT ]
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
		fmt.Fprintf(&sb, "  [ GLOBAL ENVIRONMENT ]\n%s\n\n", strings.Join(globals, "\n"))
	}

	// [ AGENTS ]
	var agentsBlocks []string
	for _, node := range g.Nodes {
		if node.Kind != "agent" {
			continue
		}

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
		metaStr := ""
		if len(metaParts) > 0 {
			metaStr = " [" + strings.Join(metaParts, " · ") + "]"
		}

		agentStr := fmt.Sprintf("  ● %s%s\n      │", node.Label, metaStr)

		blocks := []string{}

		// Capabilities
		if len(node.Tools) > 0 || len(node.BlockedTools) > 0 {
			var capLines []string
			if len(node.Tools) > 0 {
				prefix := "      │    ├─(tool)─▶ "
				if len(node.BlockedTools) == 0 {
					prefix = "      │    └─(tool)─▶ "
				}
				capLines = append(capLines, prefix+strings.Join(node.Tools, ", "))
			}
			if len(node.BlockedTools) > 0 {
				capLines = append(capLines, "      │    └─(blocked)─▶ "+strings.Join(node.BlockedTools, ", "))
			}
			blocks = append(blocks, "      ├─▶ [Capabilities]\n"+strings.Join(capLines, "\n"))
		}

		// Skills and Rules
		var skillsList []string
		var rulesList []string
		for _, edge := range g.Edges {
			if edge.From == node.ID {
				target := strings.SplitN(edge.To, ":", 2)
				if len(target) == 2 {
					if edge.Label == "skill" {
						skillsList = append(skillsList, target[1])
					} else if edge.Label == "rule" {
						rulesList = append(rulesList, target[1])
					}
				}
			}
		}

		if len(skillsList) > 0 {
			lines := []string{}
			for i, inf := range skillsList {
				prefix := "      │    ├─▶ "
				if i == len(skillsList)-1 {
					prefix = "      │    └─▶ "
				}
				lines = append(lines, prefix+inf)
			}
			blocks = append(blocks, "      ├─▶ [Skills]\n"+strings.Join(lines, "\n"))
		}

		if len(rulesList) > 0 {
			lines := []string{}
			for i, inf := range rulesList {
				prefix := "      │    ├─▶ "
				if i == len(rulesList)-1 {
					prefix = "      │    └─▶ "
				}
				lines = append(lines, prefix+inf)
			}
			blocks = append(blocks, "      ├─▶ [Rules]\n"+strings.Join(lines, "\n"))
		}

		// Servers
		var servers []string
		for _, edge := range g.Edges {
			if edge.From == node.ID && edge.Label == "mcp" {
				target := strings.SplitN(edge.To, ":", 2)
				if len(target) == 2 {
					servers = append(servers, "      │    └─(mcp)─▶ "+target[1])
				}
			}
		}
		if len(servers) > 0 {
			blocks = append(blocks, "      └─▶ [Servers]\n"+strings.Join(servers, "\n"))
		} else if len(blocks) > 0 {
			lastBlock := blocks[len(blocks)-1]
			lastBlock = strings.Replace(lastBlock, "      ├─▶", "      └─▶", 1)
			lastBlock = strings.ReplaceAll(lastBlock, "      │    ", "           ")
			blocks[len(blocks)-1] = lastBlock
		}

		if len(blocks) > 0 {
			agentStr += "\n" + strings.Join(blocks, "\n      │\n")
		} else {
			agentStr = strings.TrimSuffix(agentStr, "\n      │")
		}

		agentsBlocks = append(agentsBlocks, agentStr)
	}

	if len(agentsBlocks) > 0 {
		fmt.Fprintf(&sb, "  [ AGENTS ]\n%s\n\n", strings.Join(agentsBlocks, "\n\n"))
	}

	// [ LIBRARY ]
	var library []string
	for _, n := range g.Nodes {
		var lines []string
		if n.Kind == "skill" {
			lines = append(lines, fmt.Sprintf("  ● skill: %s", n.Label))
			if len(n.Tools) > 0 {
				lines = append(lines, "      ├─(tools)─▶ "+strings.Join(n.Tools, ", "))
			}
			if len(n.Paths) > 0 {
				lines = append(lines, "      └─(paths)─▶ "+strings.Join(n.Paths, ", "))
			} else if len(n.Tools) > 0 {
				lines[1] = strings.Replace(lines[1], "      ├─", "      └─", 1)
			}
		} else if n.Kind == "rule" {
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
		fmt.Fprintf(&sb, "  [ LIBRARY ]\n%s\n\n", strings.Join(library, "\n\n"))
	}

	// Unreferenced blocks
	referenced := map[string]bool{}
	for _, e := range g.Edges {
		referenced[e.To] = true
	}
	orphanSkills, orphanRules, orphanMCP := []string{}, []string{}, []string{}
	for _, n := range g.Nodes {
		if n.Kind == "skill" && !referenced[n.ID] {
			orphanSkills = append(orphanSkills, n.Label)
		}
		if n.Kind == "rule" && !referenced[n.ID] {
			orphanRules = append(orphanRules, n.Label)
		}
		if n.Kind == "mcp" && !referenced[n.ID] {
			orphanMCP = append(orphanMCP, n.Label)
		}
	}

	if len(orphanSkills) > 0 {
		fmt.Fprintf(&sb, "  Unreferenced skills:  %s\n", strings.Join(orphanSkills, ", "))
	}
	if len(orphanRules) > 0 {
		fmt.Fprintf(&sb, "  Unreferenced rules:   %s\n", strings.Join(orphanRules, ", "))
	}
	if len(orphanMCP) > 0 {
		fmt.Fprintf(&sb, "  Unreferenced mcp:     %s\n", strings.Join(orphanMCP, ", "))
	}

	return sb.String()
}

func renderMermaid(g *graphData) string {
	var sb strings.Builder
	sb.WriteString("```mermaid\ngraph LR\n")

	// Subgraphs by kind
	agentNodes, skillNodes, ruleNodes, mcpNodes := []graphNode{}, []graphNode{}, []graphNode{}, []graphNode{}
	for _, n := range g.Nodes {
		switch n.Kind {
		case "agent":
			agentNodes = append(agentNodes, n)
		case "skill":
			skillNodes = append(skillNodes, n)
		case "rule":
			ruleNodes = append(ruleNodes, n)
		case "mcp":
			mcpNodes = append(mcpNodes, n)
		}
	}

	if len(agentNodes) > 0 {
		sb.WriteString("  subgraph Agents\n")
		for _, n := range agentNodes {
			id := mermaidID(n.ID)
			label := n.Label
			if m, ok := n.Meta["model"]; ok {
				label += " / " + m
			}
			fmt.Fprintf(&sb, "    %s[\"%s\"]\n", id, label)
		}
		sb.WriteString("  end\n")
	}
	if len(skillNodes) > 0 {
		sb.WriteString("  subgraph Skills\n")
		for _, n := range skillNodes {
			fmt.Fprintf(&sb, "    %s[\"%s\"]\n", mermaidID(n.ID), n.Label)
		}
		sb.WriteString("  end\n")
	}
	if len(ruleNodes) > 0 {
		sb.WriteString("  subgraph Rules\n")
		for _, n := range ruleNodes {
			fmt.Fprintf(&sb, "    %s[\"%s\"]\n", mermaidID(n.ID), n.Label)
		}
		sb.WriteString("  end\n")
	}
	if len(mcpNodes) > 0 {
		sb.WriteString("  subgraph MCP\n")
		for _, n := range mcpNodes {
			fmt.Fprintf(&sb, "    %s[\"%s\"]\n", mermaidID(n.ID), n.Label)
		}
		sb.WriteString("  end\n")
	}

	// Edges
	for _, e := range g.Edges {
		fmt.Fprintf(&sb, "  %s -->|\"%s\"| %s\n", mermaidID(e.From), e.Label, mermaidID(e.To))
	}

	sb.WriteString("```\n")
	return sb.String()
}

func renderDOT(g *graphData) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "digraph %q {\n", g.Project)
	sb.WriteString("  graph [rankdir=LR fontname=Helvetica]\n")
	sb.WriteString("  node  [fontname=Helvetica shape=box style=rounded]\n\n")

	colors := map[string]string{
		"agent": "#4A90D9",
		"skill": "#7ED321",
		"rule":  "#F5A623",
		"mcp":   "#9B59B6",
		"hook":  "#E74C3C",
	}
	for _, n := range g.Nodes {
		color := colors[n.Kind]
		if color == "" {
			color = "#cccccc"
		}
		dotID := dotSafeID(n.ID)
		label := n.Label
		if n.Kind == "agent" {
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
