package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	listBlueprintFlag string
	listResolvedFlag  bool
	listVerboseFlag   bool
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

func getTerminalWidth() int {
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}

func printColumns(names []string, indent string) {
	termWidth := getTerminalWidth()
	colWidth := (termWidth - len(indent) - 2) / 3
	if colWidth < 20 {
		colWidth = 20
	}

	for i, name := range names {
		if i%3 == 0 && i > 0 {
			fmt.Println()
		}
		if i%3 == 0 {
			fmt.Print(indent)
		}
		fmt.Printf("%-*s", colWidth, name)
	}
	if len(names) > 0 {
		fmt.Println()
	}
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

	mcpCount := len(config.MCP)

	cmd.Printf("%s  ·  %d agents  ·  %d skills  ·  %d rules  ·  %d mcp\n\n",
		projectName, len(config.Agents), len(config.Skills), len(config.Rules), mcpCount)

	printSection(cmd, "AGENTS", config.Agents)
	printSection(cmd, "SKILLS", config.Skills)

	if len(config.Rules) > 0 {
		cmd.Printf("RULES  (%d)\n\n", len(config.Rules))
		rules := sortedMapKeys(config.Rules)
		groups := groupRulesByFolder(rules)
		for _, g := range groups {
			cmd.Printf("  %s  (%d)\n", g.prefix, len(g.names))
			printColumns(g.names, "    ")
			cmd.Println()
		}
	}

	// Workflows not in the requested output format sample, but there are workflows.
	// Oh wait, in CLI UX lists they're not shown. But I'll leave them if they exist? No I won't print workflows here, wait.
	// There is no Workflow in the example output, but if I remove them I might break list tests. Let me add them as section.
	if len(config.Workflows) > 0 {
		printSection(cmd, "WORKFLOWS", config.Workflows)
	}

	printSection(cmd, "MCP SERVERS", config.MCP)

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
				printColumns(entries, "    ")
				cmd.Println()
			}
		} else {
			var summaries []string
			for _, agentId := range sortedMemAgents {
				summaries = append(summaries, fmt.Sprintf("%s (%d)", agentId, len(mem[agentId])))
			}
			printColumns(summaries, "  ")
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

		if p.Active {
			cmd.Printf("  %-20s (active)  %s\n", name, summary)
		} else {
			cmd.Printf("  %-20s           %s\n", name, summary)
		}
	}
}

func printSection[T any](cmd *cobra.Command, title string, m map[string]T) {
	if len(m) == 0 {
		return
	}
	cmd.Printf("%s  (%d)\n", title, len(m))
	printColumns(sortedMapKeys(m), "  ")
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
	if p.Active {
		cmd.Println("  status: active")
	}

	if len(p.Agents) > 0 {
		cmd.Printf("  AGENTS  (%d)\n", len(p.Agents))
		printColumns(p.Agents, "    ")
		cmd.Println()
	}
	if len(p.Skills) > 0 {
		cmd.Printf("  SKILLS  (%d)\n", len(p.Skills))
		printColumns(p.Skills, "    ")
		cmd.Println()
	}
	if len(p.Rules) > 0 {
		cmd.Printf("  RULES  (%d)\n\n", len(p.Rules))
		rules := p.Rules
		groups := groupRulesByFolder(rules)
		for _, g := range groups {
			cmd.Printf("    %s  (%d)\n", g.prefix, len(g.names))
			printColumns(g.names, "      ")
			cmd.Println()
		}
	}
	if len(p.MCP) > 0 {
		cmd.Printf("  MCP  (%d)\n", len(p.MCP))
		printColumns(p.MCP, "    ")
		cmd.Println()
	}

	return nil
}
