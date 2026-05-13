package main

import (
	"fmt"
	"strings"
)

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
		parts = append(parts, fmt.Sprintf("%d %s", agents, plural(agents, "agent", "agents")))
	}
	if skills > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", skills, plural(skills, "skill", "skills")))
	}
	if rules > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", rules, plural(rules, "rule", "rules")))
	}
	if mcps > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", mcps, plural(mcps, "mcp server", "mcp servers")))
	}
	if policies > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", policies, plural(policies, "policy", "policies")))
	}
	if hooks > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", hooks, plural(hooks, "hook", "hooks")))
	}

	sep := "  " + glyphDot() + "  "
	header := fmt.Sprintf("  %s%s%s  ", g.Project, sep, strings.Join(parts, sep))
	width := len(header) + 2
	border := strings.Repeat("─", width)

	return fmt.Sprintf("\n┌%s┐\n│%s│\n└%s┘\n", border, header, border)
}

func renderProjectScoped(g *graphData) string {
	referenced := map[string]bool{}
	for _, e := range g.Edges {
		referenced[e.To] = true
	}
	projectSkills, projectRules, projectMCP, projectPolicies := []string{}, []string{}, []string{}, []string{}
	for _, n := range g.Nodes {
		if n.Kind == kindSkill && !referenced[n.ID] {
			projectSkills = append(projectSkills, n.Label)
		}
		if n.Kind == kindRule && !referenced[n.ID] {
			projectRules = append(projectRules, n.Label)
		}
		if n.Kind == kindMCP && !referenced[n.ID] {
			projectMCP = append(projectMCP, n.Label)
		}
		if n.Kind == kindPolicy && !referenced[n.ID] {
			projectPolicies = append(projectPolicies, n.Label)
		}
	}
	var sb strings.Builder
	if len(projectSkills) > 0 {
		fmt.Fprintf(&sb, "  Project-scoped skills:    %s\n", strings.Join(projectSkills, ", "))
	}
	if len(projectRules) > 0 {
		fmt.Fprintf(&sb, "  Project-scoped rules:     %s\n", strings.Join(projectRules, ", "))
	}
	if len(projectMCP) > 0 {
		fmt.Fprintf(&sb, "  Project-scoped mcp:       %s\n", strings.Join(projectMCP, ", "))
	}
	if len(projectPolicies) > 0 {
		fmt.Fprintf(&sb, "  Project-scoped policies:  %s\n", strings.Join(projectPolicies, ", "))
	}
	return sb.String()
}

const treePrefixMid = "      │    ├─▶ "
const treePrefixLast = "      │    └─▶ "

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

func renderScopeSummary(sb *strings.Builder, g *graphData) {
	counts := countGraphNodesByKind(g)
	label := getScopeLabel(g)
	sb.WriteString(fmt.Sprintf("  ● %s\n    (%s)\n", label, g.ConfigPath))

	agentNodes := extractAgentNodes(g)
	lines := buildSummaryLines(agentNodes, counts)

	printSummaryLines(sb, lines, agentNodes)
}

// countGraphNodesByKind counts nodes by their kind.
func countGraphNodesByKind(g *graphData) map[string]int {
	counts := map[string]int{}
	for _, n := range g.Nodes {
		switch n.Kind {
		case kindAgent:
			counts["agents"]++
		case kindRule:
			counts["rules"]++
		case kindMCP:
			counts["mcp"]++
		case kindPolicy:
			counts["policies"]++
		}
	}
	return counts
}

// getScopeLabel determines the display label for a graph scope.
func getScopeLabel(g *graphData) string {
	label := g.Project
	if label == "" && g.Scope == "global" {
		return "Global Context"
	}
	if label == "" {
		return "Unnamed Context"
	}
	return label
}

// extractAgentNodes filters nodes to return only agents.
func extractAgentNodes(g *graphData) []graphNode {
	var agentNodes []graphNode
	for _, n := range g.Nodes {
		if n.Kind == kindAgent {
			agentNodes = append(agentNodes, n)
		}
	}
	return agentNodes
}

// summaryLine represents a line in the scope summary.
type summaryLine struct {
	Title string
	Kind  string
	Value int
}

// buildSummaryLines constructs summary lines from node counts.
func buildSummaryLines(agentNodes []graphNode, counts map[string]int) []summaryLine {
	var lines []summaryLine
	if len(agentNodes) > 0 {
		lines = append(lines, summaryLine{"Agents", kindAgent, len(agentNodes)})
	}
	if counts["rules"] > 0 {
		lines = append(lines, summaryLine{"Rules", kindRule, counts["rules"]})
	}
	if counts["mcp"] > 0 {
		lines = append(lines, summaryLine{"MCP Servers", kindMCP, counts["mcp"]})
	}
	if counts["policies"] > 0 {
		lines = append(lines, summaryLine{"Policies", kindPolicy, counts["policies"]})
	}
	return lines
}

// printSummaryLines outputs the formatted summary lines.
func printSummaryLines(sb *strings.Builder, lines []summaryLine, agentNodes []graphNode) {
	for idx, ln := range lines {
		prefix := "      ├─▶ "
		if idx == len(lines)-1 {
			prefix = "      └─▶ "
		}
		sb.WriteString(fmt.Sprintf("%s%s: (%d)\n", prefix, ln.Title, ln.Value))

		if ln.Title == "Agents" {
			printAgentList(sb, agentNodes, idx, len(lines))
		}
	}
}

// printAgentList outputs the list of agents under the Agents summary line.
func printAgentList(sb *strings.Builder, agentNodes []graphNode, lineIdx int, totalLines int) {
	for i, n := range agentNodes {
		aprefix := treePrefixMid
		if i == len(agentNodes)-1 {
			aprefix = treePrefixLast
		}
		if lineIdx == totalLines-1 { // Agent line was the last one
			aprefix = "           ├─▶ "
			if i == len(agentNodes)-1 {
				aprefix = "           └─▶ "
			}
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", aprefix, n.Label))
	}
}
