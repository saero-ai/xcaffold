package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/analyzer"
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/registry"
)

func runGraphTerminalMode() error {
	switch {
	case graphAll:
		return runGraphAll()
	case graphBlueprintFlag != "":
		return runGraphBlueprint(graphBlueprintFlag)
	case graphFull:
		return runGraphFull()
	case globalFlag:
		return runGraphGlobal()
	default:
		return runGraphProject()
	}
}

func runGraphProject() error {
	cfg, err := parser.ParseDirectory(projectParseRoot())
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	cfg.StripInherited()

	projectName := "project"
	if cfg.Project != nil {
		projectName = cfg.Project.Name
	}

	fmt.Printf("%s  ·  %d agents  ·  %d skills  ·  %d rules  ·  %d mcp server\n",
		projectName, len(cfg.Agents), len(cfg.Skills), len(cfg.Rules), len(cfg.MCP))

	renderAgentTree(cfg, projectParseRoot())
	printMCPFooter(cfg)
	printRulesFooter(cfg)
	return nil
}

func runGraphGlobal() error {
	cfg, err := parser.ParseDirectory(globalXcfHome)
	if err != nil {
		return fmt.Errorf("global parse error: %w", err)
	}

	fmt.Printf("global  ·  %d agents  ·  %d skills  ·  %d rules  ·  %d mcp server\n",
		len(cfg.Agents), len(cfg.Skills), len(cfg.Rules), len(cfg.MCP))

	renderAgentTree(cfg, globalXcfHome)
	printMCPFooter(cfg)
	printRulesFooter(cfg)
	return nil
}

func runGraphFull() error {
	globalCfg, err := parser.ParseDirectory(globalXcfHome)
	if err != nil {
		// It's ok if global doesn't exist
		globalCfg = &ast.XcaffoldConfig{}
	}

	projectCfg, err := parser.ParseDirectory(projectParseRoot())
	if err != nil {
		return fmt.Errorf("project parse error: %w", err)
	}
	projectCfg.StripInherited()

	projectName := "project"
	if projectCfg.Project != nil {
		projectName = projectCfg.Project.Name
	}

	fmt.Printf("%s  ·  %d agents (global)  ·  %d agents (project)  ·  %d rules  ·  %d mcp server\n\n",
		projectName, len(globalCfg.Agents), len(projectCfg.Agents), len(projectCfg.Rules), len(projectCfg.MCP))

	fmt.Printf("══════════════════════════════════════════\n  GLOBAL\n══════════════════════════════════════════\n")
	renderAgentTree(globalCfg, globalXcfHome)
	printRULESFooterIfAny(globalCfg)

	fmt.Printf("\n══════════════════════════════════════════\n  PROJECT: %s\n══════════════════════════════════════════\n", projectName)
	renderAgentTree(projectCfg, projectParseRoot())
	printMCPFooter(projectCfg)
	printRulesFooter(projectCfg)

	return nil
}

func runGraphBlueprint(bpName string) error {
	cfg, err := parser.ParseDirectory(projectParseRoot())
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}
	cfg.StripInherited()

	filtered, err := blueprint.ApplyBlueprint(cfg, bpName)
	if err != nil {
		return fmt.Errorf("blueprint error: %w", err)
	}

	fmt.Printf("blueprint: %s  ·  %d agents  ·  %d skills  ·  %d rules\n",
		bpName, len(filtered.Agents), len(filtered.Skills), len(filtered.Rules))

	renderAgentTree(filtered, projectParseRoot())
	printMCPFooter(filtered)
	printRulesFooter(filtered)
	return nil
}

func runGraphAll() error {
	globalCfg, _ := parser.ParseDirectory(globalXcfHome)
	if globalCfg == nil {
		globalCfg = &ast.XcaffoldConfig{}
	}

	fmt.Printf("══════════════════════════════════════════\n  GLOBAL\n══════════════════════════════════════════\n")
	renderAgentTree(globalCfg, globalXcfHome)

	projects, err := registry.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not list registered projects: %v\n", err)
	} else {
		for _, p := range projects {
			var xcfProjectPath string
			candidates := []string{
				filepath.Join(p.Path, ".xcaffold", "project.xcf"),
				filepath.Join(p.Path, "project.xcf"),
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					xcfProjectPath = c
					break
				}
			}
			if xcfProjectPath != "" {
				cfg, err := parser.ParseDirectory(getParseRoot(xcfProjectPath))
				if err == nil {
					cfg.StripInherited()
					fmt.Printf("\n══════════════════════════════════════════\n  PROJECT: %s\n══════════════════════════════════════════\n", p.Name)
					renderAgentTree(cfg, getParseRoot(xcfProjectPath))
				}
			}
		}
	}
	return nil
}

func renderAgentTree(cfg *ast.XcaffoldConfig, parseRoot string) {
	agentIDs := sortedAgentIDs(cfg)
	for _, id := range agentIDs {
		agent := cfg.Agents[id]
		fmt.Printf("\n  ● %s\n", id)
		hasAssociations := false

		// Print capabilities if any
		if len(agent.Tools) > 0 || len(agent.DisallowedTools) > 0 {
			hasAssociations = true
			fmt.Printf("  │   tools    ")
			for _, t := range agent.Tools {
				fmt.Printf("%s  ", t)
			}
			fmt.Printf("\n")
		}

		memEntries := agentMemoryEntries(parseRoot, id)

		var blocks []string
		if len(agent.Skills) > 0 {
			blocks = append(blocks, "skills")
		}
		if len(agent.Rules) > 0 {
			blocks = append(blocks, "rules")
		}
		if len(agent.MCP) > 0 {
			blocks = append(blocks, "mcp")
		}
		if len(memEntries) > 0 {
			blocks = append(blocks, "memory")
		}

		for bIdx, block := range blocks {
			hasAssociations = true
			isLastBlock := bIdx == len(blocks)-1
			blockConnector := "├──"
			childPrefix := "  │    "
			if isLastBlock {
				blockConnector = "└──"
				childPrefix = "       "
			}

			fmt.Printf("  │\n  %s %s", blockConnector, block)
			if block == "memory" {
				fmt.Printf("  (%d %s)\n", len(memEntries), pluralize("entry", "entries", len(memEntries)))
			} else {
				fmt.Printf("\n")
			}

			switch block {
			case "skills":
				for i, s := range agent.Skills {
					connector := "├──"
					if i == len(agent.Skills)-1 {
						connector = "└──"
					}
					fmt.Printf("  %s %s %s\n", childPrefix, connector, s)
				}
			case "rules":
				for i, r := range agent.Rules {
					connector := "├──"
					if i == len(agent.Rules)-1 {
						connector = "└──"
					}
					fmt.Printf("  %s %s %s\n", childPrefix, connector, r)
				}
			case "mcp":
				for i, m := range agent.MCP {
					connector := "├──"
					if i == len(agent.MCP)-1 {
						connector = "└──"
					}
					fmt.Printf("  %s %s %s\n", childPrefix, connector, m)
				}
			case "memory":
				for i, e := range memEntries {
					connector := "├──"
					if i == len(memEntries)-1 {
						connector = "└──"
					}
					fmt.Printf("  %s %s %s\n", childPrefix, connector, e)
				}
			}
		}

		if !hasAssociations {
			fmt.Printf("      (no skill, rule, mcp, or memory associations)\n")
		}
	}
}

func printRULESFooterIfAny(cfg *ast.XcaffoldConfig) {
	if len(cfg.Rules) > 0 {
		printRulesFooter(cfg)
	}
}

func printMCPFooter(cfg *ast.XcaffoldConfig) {
	if len(cfg.MCP) == 0 {
		return
	}
	fmt.Printf("\n  %s\n  MCP SERVERS  (%d)\n", strings.Repeat("─", 42), len(cfg.MCP))
	for _, id := range sortedRuleIDs(cfg.MCP) {
		fmt.Printf("    ● %s\n", id)
	}
}

func printRulesFooter(cfg *ast.XcaffoldConfig) {
	if len(cfg.Rules) == 0 {
		return
	}
	fmt.Printf("\n  %s\n", strings.Repeat("─", 42))
	fmt.Printf("  RULES  (%d)\n", len(cfg.Rules))
	groups := groupRulesByFolder(sortedRuleIDs(cfg.Rules))
	for _, g := range groups {
		names := strings.Join(g.names, "  ")
		fmt.Printf("    %-8s %s\n", g.prefix, names)
	}
}

func agentMemoryEntries(projectRoot, agentID string) []string {
	dir := filepath.Join(projectRoot, "xcf", "agents", agentID, "memory")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var stems []string
	for _, e := range entries {
		if !e.IsDir() {
			stems = append(stems, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		}
	}
	sort.Strings(stems)
	return stems
}

func sortedAgentIDs(cfg *ast.XcaffoldConfig) []string {
	var ids []string
	for id := range cfg.Agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedRuleIDs[T any](m map[string]T) []string {
	var ids []string
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

type ruleGroup struct {
	prefix string
	names  []string
}

func groupRulesByFolder(ruleIDs []string) []ruleGroup {
	gm := make(map[string][]string)
	for _, id := range ruleIDs {
		parts := strings.Split(id, "/")
		if len(parts) == 1 {
			gm["(root)"] = append(gm["(root)"], id)
		} else {
			prefix := parts[0] + "/"
			leaf := parts[len(parts)-1]
			gm[prefix] = append(gm[prefix], leaf)
		}
	}
	var groups []ruleGroup
	for p, names := range gm {
		groups = append(groups, ruleGroup{prefix: p, names: names})
	}
	sort.Slice(groups, func(i, j int) bool {
		// Ensure (root) is placed last, though normally smaller alphabetically. Wait, (root) has '(' which is smaller than 'a'.
		// Actually the spec said to sort "(root)" last.
		if groups[i].prefix == "(root)" {
			return false
		}
		if groups[j].prefix == "(root)" {
			return true
		}
		return groups[i].prefix < groups[j].prefix
	})
	return groups
}

func printDiskEntriesIfAny(cfg *ast.XcaffoldConfig, parseRoot string) {
	if !graphScanOutput {
		return
	}
	a := analyzer.New()
	declared := make(map[string]bool)
	for id := range cfg.Agents {
		declared["agent:"+id] = true
	}
	for id := range cfg.Skills {
		declared["skill:"+id] = true
	}
	for id := range cfg.Rules {
		declared["rule:"+id] = true
	}
	for id := range cfg.MCP {
		declared["mcp:"+id] = true
	}
	for id := range cfg.Policies {
		declared["policy:"+id] = true
	}

	targetDir := filepath.Join(parseRoot, ".claude") // Default target, should ideally use compiler.OutputDir(targetFlag) but we don't have access easily here.
	entries, err := a.ScanOutputDir(targetDir, declared)
	if err == nil && len(entries) > 0 {
		fmt.Printf("\n  [ UNDECLARED FILES ]  (!)\n")
		for _, e := range entries {
			fmt.Printf("      - [%s] %s\n", e.Kind, e.ID)
		}
	}
}

func pluralize(singular, plural string, count int) string {
	if count == 1 {
		return singular
	}
	return plural
}
