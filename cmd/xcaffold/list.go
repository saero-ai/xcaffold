package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
)

var (
	listBlueprintFlag  string
	listResolvedFlag   bool
	listVerboseFlag    bool
	listFilterAgent    string
	listFilterSkill    string
	listFilterRule     string
	listFilterWorkflow string
	listFilterMCP      string
	listFilterHooks    bool
	listFilterSettings bool
	listFilterContext  string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered resources and blueprints",
	Long:  "Scans the current project and displays all discovered resources and blueprints.",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringVar(&listBlueprintFlag, "blueprint", "", "Filter to named blueprint")
	listCmd.Flags().BoolVar(&listResolvedFlag, "resolved", false, "Show transitive deps (use with --blueprint)")
	listCmd.Flags().BoolVarP(&listVerboseFlag, "verbose", "v", false, "show memory entry names per agent")
	f := listCmd.Flags()
	f.StringVar(&listFilterAgent, "agent", "", "List agents (optionally filter by name)")
	f.Lookup("agent").NoOptDefVal = "*"
	f.StringVar(&listFilterSkill, "skill", "", "List skills (optionally filter by name)")
	f.Lookup("skill").NoOptDefVal = "*"
	f.StringVar(&listFilterRule, "rule", "", "List rules (optionally filter by name)")
	f.Lookup("rule").NoOptDefVal = "*"
	f.StringVar(&listFilterWorkflow, "workflow", "", "List workflows (optionally filter by name)")
	f.Lookup("workflow").NoOptDefVal = "*"
	f.StringVar(&listFilterMCP, "mcp", "", "List MCP servers (optionally filter by name)")
	f.Lookup("mcp").NoOptDefVal = "*"
	f.BoolVar(&listFilterHooks, "hooks", false, "List hooks")
	f.BoolVar(&listFilterSettings, "settings", false, "List settings")
	f.StringVar(&listFilterContext, "context", "", "List contexts (optionally filter by name)")
	f.Lookup("context").NoOptDefVal = "*"
	_ = listCmd.Flags().MarkHidden("blueprint")
	_ = listCmd.Flags().MarkHidden("resolved")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	if xcfPath == "" {
		return fmt.Errorf("no project.xcf found; run from a project directory or use --config")
	}

	configDir := projectParseRoot()

	config, err := parser.ParseDirectory(configDir)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	if !globalFlag {
		config.StripInherited()
	}

	if listBlueprintFlag != "" {
		return printBlueprintResources(cmd, config, listBlueprintFlag, listResolvedFlag)
	}

	printAllResources(cmd, config, configDir)
	return nil
}

func listHasFilter() bool {
	return listFilterAgent != "" || listFilterSkill != "" || listFilterRule != "" ||
		listFilterWorkflow != "" || listFilterMCP != "" || listFilterHooks ||
		listFilterSettings || listFilterContext != ""
}

// filterMapByName returns a map with only entries matching the filter string.
// If filter is "*", returns the entire map.
// Otherwise, returns entries where the key contains the filter string.
func filterMapByName[T any](m map[string]T, filter string) map[string]T {
	if filter == "*" {
		return m
	}
	result := make(map[string]T)
	for k, v := range m {
		if strings.Contains(k, filter) {
			result[k] = v
		}
	}
	return result
}

func memorySummary(projectRoot string, cfg *ast.XcaffoldConfig) map[string][]string {
	result := make(map[string][]string)
	memBase := filepath.Join(projectRoot, "xcf", "agents")
	for agentID := range cfg.Agents {
		memDir := filepath.Join(memBase, agentID, "memory")
		entries, err := os.ReadDir(memDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				stem := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
				result[agentID] = append(result[agentID], stem)
			}
		}
		if len(result[agentID]) > 0 {
			sort.Strings(result[agentID])
		}
	}
	return result
}

func printAllResources(cmd *cobra.Command, config *ast.XcaffoldConfig, baseDir string) {
	projectName := filepath.Base(baseDir)

	// Build header with only non-zero kind counts
	sep := "  " + glyphDot() + "  "
	parts := []string{projectName}
	if len(config.Agents) > 0 {
		parts = append(parts, fmt.Sprintf("%d agents", len(config.Agents)))
	}
	if len(config.Skills) > 0 {
		parts = append(parts, fmt.Sprintf("%d skills", len(config.Skills)))
	}
	if len(config.Rules) > 0 {
		parts = append(parts, fmt.Sprintf("%d rules", len(config.Rules)))
	}
	if len(config.Workflows) > 0 {
		parts = append(parts, fmt.Sprintf("%d workflows", len(config.Workflows)))
	}
	if len(config.MCP) > 0 {
		parts = append(parts, fmt.Sprintf("%d mcp", len(config.MCP)))
	}
	if len(config.Contexts) > 0 {
		parts = append(parts, fmt.Sprintf("%d contexts", len(config.Contexts)))
	}
	if len(config.Hooks) > 0 {
		parts = append(parts, fmt.Sprintf("%d hooks", len(config.Hooks)))
	}
	if len(config.Settings) > 0 {
		parts = append(parts, fmt.Sprintf("%d settings", len(config.Settings)))
	}
	cmd.Printf("%s\n\n", strings.Join(parts, sep))

	// If a kind filter is active, show only that kind
	if listHasFilter() {
		if listFilterAgent != "" {
			filtered := filterMapByName(config.Agents, listFilterAgent)
			printSection(cmd, "AGENTS", filtered)
		}
		if listFilterSkill != "" {
			filtered := filterMapByName(config.Skills, listFilterSkill)
			printSection(cmd, "SKILLS", filtered)
		}
		if listFilterRule != "" {
			filtered := filterMapByName(config.Rules, listFilterRule)
			if len(filtered) > 0 {
				cmd.Printf("RULES  (%d)\n\n", len(filtered))
				rules := sortedMapKeys(filtered)
				groups := groupRulesByFolder(rules)
				for _, g := range groups {
					cmd.Printf("  %s  (%d)\n", g.prefix, len(g.names))
					for _, name := range g.names {
						cmd.Printf("    %s\n", name)
					}
					cmd.Println()
				}
			}
		}
		if listFilterWorkflow != "" {
			filtered := filterMapByName(config.Workflows, listFilterWorkflow)
			printSection(cmd, "WORKFLOWS", filtered)
		}
		if listFilterMCP != "" {
			filtered := filterMapByName(config.MCP, listFilterMCP)
			printSection(cmd, "MCP SERVERS", filtered)
		}
		if listFilterContext != "" {
			filtered := filterMapByName(config.Contexts, listFilterContext)
			printSection(cmd, "CONTEXTS", filtered)
		}
		if listFilterHooks {
			printSection(cmd, "HOOKS", config.Hooks)
		}
		if listFilterSettings {
			printSection(cmd, "SETTINGS", config.Settings)
		}
		return
	}

	// Show all sections
	printSection(cmd, "AGENTS", config.Agents)
	printSection(cmd, "SKILLS", config.Skills)

	if len(config.Rules) > 0 {
		cmd.Printf("RULES  (%d)\n\n", len(config.Rules))
		rules := sortedMapKeys(config.Rules)
		groups := groupRulesByFolder(rules)
		for _, g := range groups {
			cmd.Printf("  %s  (%d)\n", g.prefix, len(g.names))
			for _, name := range g.names {
				cmd.Printf("    %s\n", name)
			}
			cmd.Println()
		}
	}

	if len(config.Workflows) > 0 {
		printSection(cmd, "WORKFLOWS", config.Workflows)
	}

	printSection(cmd, "MCP SERVERS", config.MCP)
	printSection(cmd, "CONTEXTS", config.Contexts)
	printSection(cmd, "HOOKS", config.Hooks)
	printSection(cmd, "SETTINGS", config.Settings)

	mem := memorySummary(baseDir, config)
	totalEntries := 0
	agentIdsWithMemory := 0
	var sortedMemAgents []string
	for agentId, entries := range mem {
		if len(entries) > 0 {
			totalEntries += len(entries)
			agentIdsWithMemory++
			sortedMemAgents = append(sortedMemAgents, agentId)
		}
	}

	if totalEntries > 0 {
		cmd.Printf("MEMORY  (%d entries across %d agents)\n", totalEntries, agentIdsWithMemory)
		sort.Strings(sortedMemAgents)

		if listVerboseFlag {
			cmd.Println()
			for _, agentId := range sortedMemAgents {
				entries := mem[agentId]
				cmd.Printf("  %s  (%d)\n", agentId, len(entries))
				for _, entry := range entries {
					cmd.Printf("    %s\n", entry)
				}
				cmd.Println()
			}
		} else {
			var summaries []string
			for _, agentId := range sortedMemAgents {
				summaries = append(summaries, fmt.Sprintf("%s (%d)", agentId, len(mem[agentId])))
			}
			for _, summary := range summaries {
				cmd.Printf("  %s\n", summary)
			}
			cmd.Println()
		}
	}

	cmd.Println("BLUEPRINTS")

	if len(config.Blueprints) == 0 {
		cmd.Println("  (none)")
		return
	}

	names := sortedMapKeys(config.Blueprints)
	for _, name := range names {
		p := config.Blueprints[name]
		var parts []string
		if len(p.Agents) > 0 {
			parts = append(parts, fmt.Sprintf("%d agent(s)", len(p.Agents)))
		}
		if len(p.Skills) > 0 {
			parts = append(parts, fmt.Sprintf("%d skill(s)", len(p.Skills)))
		}
		if len(p.Rules) > 0 {
			parts = append(parts, fmt.Sprintf("%d rule(s)", len(p.Rules)))
		}
		summary := strings.Join(parts, ", ")

		cmd.Printf("  %-20s  %s\n", name, summary)
	}
}

func printSection[T any](cmd *cobra.Command, title string, m map[string]T) {
	if len(m) == 0 {
		return
	}
	cmd.Printf("%s  (%d)\n", title, len(m))
	for _, name := range sortedMapKeys(m) {
		cmd.Printf("  %s\n", name)
	}
	cmd.Println()
}

func printBlueprintResources(cmd *cobra.Command, config *ast.XcaffoldConfig, bpName string, doResolve bool) error {
	p, ok := config.Blueprints[bpName]
	if !ok {
		available := sortedMapKeys(config.Blueprints)
		return fmt.Errorf("blueprint %q not found; available: %v", bpName, available)
	}

	if doResolve {
		bpCopy := make(map[string]ast.BlueprintConfig, len(config.Blueprints))
		for k, v := range config.Blueprints {
			bpCopy[k] = v
		}
		if err := blueprint.ResolveBlueprintExtends(bpCopy); err != nil {
			return fmt.Errorf("extends resolution: %w", err)
		}
		scope := &ast.ResourceScope{
			Agents:    config.Agents,
			Skills:    config.Skills,
			Rules:     config.Rules,
			Workflows: config.Workflows,
			MCP:       config.MCP,
			Policies:  config.Policies,
			Memory:    config.Memory,
		}
		resolved := bpCopy[bpName]
		blueprint.ResolveTransitiveDeps(&resolved, scope)
		p = resolved
	}

	cmd.Printf("BLUEPRINT: %s\n\n", bpName)
	if p.Description != "" {
		cmd.Printf("  description: %s\n", p.Description)
	}
	if p.Extends != "" {
		cmd.Printf("  extends: %s\n", p.Extends)
	}
	if len(p.Agents) > 0 {
		cmd.Printf("  AGENTS  (%d)\n", len(p.Agents))
		for _, name := range p.Agents {
			cmd.Printf("    %s\n", name)
		}
		cmd.Println()
	}
	if len(p.Skills) > 0 {
		cmd.Printf("  SKILLS  (%d)\n", len(p.Skills))
		for _, name := range p.Skills {
			cmd.Printf("    %s\n", name)
		}
		cmd.Println()
	}
	if len(p.Rules) > 0 {
		cmd.Printf("  RULES  (%d)\n\n", len(p.Rules))
		rules := p.Rules
		groups := groupRulesByFolder(rules)
		for _, g := range groups {
			cmd.Printf("    %s  (%d)\n", g.prefix, len(g.names))
			for _, name := range g.names {
				cmd.Printf("      %s\n", name)
			}
			cmd.Println()
		}
	}
	if len(p.MCP) > 0 {
		cmd.Printf("  MCP  (%d)\n", len(p.MCP))
		for _, name := range p.MCP {
			cmd.Printf("    %s\n", name)
		}
		cmd.Println()
	}

	return nil
}
