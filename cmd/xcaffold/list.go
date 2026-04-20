package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/spf13/cobra"
)

var (
	listBlueprintFlag string
	listResolvedFlag  bool
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
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	if xcfPath == "" {
		return fmt.Errorf("no project.xcf found; run from a project directory or use --config")
	}

	configDir := xcfPath
	if info, err := os.Stat(xcfPath); err != nil || !info.IsDir() {
		configDir = filepath.Dir(xcfPath)
	}

	config, err := parser.ParseDirectory(configDir)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	if listBlueprintFlag != "" {
		return printBlueprintResources(cmd, config, listBlueprintFlag, listResolvedFlag)
	}

	printAllResources(cmd, config)
	return nil
}

// printAllResources prints a summary of every resource type and every blueprint.
func printAllResources(cmd *cobra.Command, config *ast.XcaffoldConfig) {
	cmd.Println("Resources:")
	printResourceLine(cmd, "agents", sortedMapKeys(config.Agents))
	printResourceLine(cmd, "skills", sortedMapKeys(config.Skills))
	printResourceLine(cmd, "rules", sortedMapKeys(config.Rules))
	printResourceLine(cmd, "workflows", sortedMapKeys(config.Workflows))
	printResourceLine(cmd, "mcp", sortedMapKeys(config.MCP))
	printResourceLine(cmd, "policies", sortedMapKeys(config.Policies))
	printResourceLine(cmd, "memory", sortedMapKeys(config.Memory))

	cmd.Println()
	cmd.Println("Blueprints:")

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

// printBlueprintResources prints resources for a single named blueprint.
// When resolved is true, transitive dependencies are expanded first.
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
		// Build a ResourceScope from config for transitive dep resolution.
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

	cmd.Printf("Blueprint: %s\n", bpName)
	if p.Description != "" {
		cmd.Printf("  description: %s\n", p.Description)
	}
	if p.Extends != "" {
		cmd.Printf("  extends: %s\n", p.Extends)
	}
	if p.Active {
		cmd.Println("  status: active")
	}
	printBlueprintSection(cmd, "agents", p.Agents)
	printBlueprintSection(cmd, "skills", p.Skills)
	printBlueprintSection(cmd, "rules", p.Rules)
	printBlueprintSection(cmd, "workflows", p.Workflows)
	printBlueprintSection(cmd, "mcp", p.MCP)
	printBlueprintSection(cmd, "policies", p.Policies)
	printBlueprintSection(cmd, "memory", p.Memory)
	return nil
}

func printResourceLine(cmd *cobra.Command, label string, names []string) {
	if len(names) == 0 {
		return
	}
	cmd.Printf("  %-12s %s (%d)\n", label+":", strings.Join(names, ", "), len(names))
}

func printBlueprintSection(cmd *cobra.Command, label string, items []string) {
	if len(items) == 0 {
		return
	}
	cmd.Printf("  %-12s %s\n", label+":", strings.Join(items, ", "))
}
