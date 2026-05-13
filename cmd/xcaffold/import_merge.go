package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/prompt"
	"github.com/saero-ai/xcaffold/internal/registry"
)

// mergeImportDirs consolidates multiple platform directories into a single project.xcaf.
// Resources present in multiple providers are compared field-by-field: identical content
// produces a universal base tagged with all providers; different content produces a base
// with the first provider's values plus per-provider override files.
func mergeImportDirs(providers []importer.ProviderImporter, xcafDest string) error {
	xcafExists, xcafDirExists := checkExistingXcaf(xcafDest)

	if importForce && (xcafExists || xcafDirExists) {
		if err := performForceRemoveXcaf(xcafDest); err != nil {
			return err
		}
		xcafExists = false
		xcafDirExists = false
	}

	if xcafExists || xcafDirExists {
		return fmt.Errorf("[project] incremental import with multiple providers is not yet supported; use 'xcaffold import --force' to reimport from scratch")
	}

	projectName := inferProjectName()
	config := createMergeImportConfig(projectName)
	var warnings []string

	// Collect and assemble configs
	providerConfigs := scanProviderConfigs(providers, &warnings)
	assembleMultiProviderResources(providerConfigs, config)
	detectAndSetTargets(providers, config)
	applyKindFilters(config)

	// Apply --dry-run guard before writing files
	if importDryRun {
		printMergeImportPlan(config, providers)
		return nil
	}

	// Write files and summarize
	if err := writeMergeImportFiles(config); err != nil {
		return err
	}

	overrideCount := countImportOverrides(config)
	printMergeImportSummary(xcafDest, config, providers, overrideCount)

	cwd, _ := os.Getwd()
	if config.Project != nil {
		_ = registry.Register(cwd, config.Project.Name, nil, ".")
	}

	printImportWarnings(warnings)
	return nil
}

// checkExistingXcaf checks if project.xcaf and xcaf/ directory exist.
func checkExistingXcaf(xcafDest string) (bool, bool) {
	xcafExists := false
	if _, err := os.Stat(xcafDest); err == nil {
		xcafExists = true
	}
	xcafDirExists := false
	if _, err := os.Stat("xcaf"); err == nil {
		xcafDirExists = true
	}
	return xcafExists, xcafDirExists
}

// performForceRemoveXcaf removes existing project.xcaf and xcaf/ if user confirms.
func performForceRemoveXcaf(xcafDest string) error {
	fmt.Fprintf(os.Stderr, "\n  %s  --force will DELETE project.xcaf and xcaf/ directory.\n", colorYellow(glyphSrc()))
	fmt.Fprintf(os.Stderr, "     All manual edits to xcaf files will be lost.\n\n")
	doForce := importYes
	if !importYes {
		var err error
		doForce, err = prompt.Confirm("Proceed with destructive reimport?", false)
		if err != nil {
			return fmt.Errorf("prompt error: %w", err)
		}
	}
	if !doForce {
		return nil
	}
	_ = os.Remove(xcafDest)
	_ = os.RemoveAll("xcaf")
	return nil
}

// createMergeImportConfig creates an empty config for merge import.
func createMergeImportConfig(projectName string) *ast.XcaffoldConfig {
	return &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: projectName},
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
}

// detectAndSetTargets detects compilation targets and sets them on the config.
func detectAndSetTargets(providers []importer.ProviderImporter, config *ast.XcaffoldConfig) {
	var dirNames []string
	for _, imp := range providers {
		dirNames = append(dirNames, imp.InputDir())
	}
	if config.Project != nil {
		config.Project.Targets = detectTargets(dirNames...)
	}
}

// printMergeImportPlan outputs the dry-run preview.
func printMergeImportPlan(config *ast.XcaffoldConfig, providers []importer.ProviderImporter) {
	fmt.Printf("Import plan (dry-run):\n")
	fmt.Printf("  Would create %d agents, %d skills, %d rules, %d workflows, %d MCP servers\n",
		len(config.Agents), len(config.Skills), len(config.Rules), len(config.Workflows), len(config.MCP))
	fmt.Printf("  From %d provider directories\n", len(providers))
}

// writeMergeImportFiles writes all import-related files to disk.
func writeMergeImportFiles(config *ast.XcaffoldConfig) error {
	if memCount, err := writeMemoryFiles(config); err != nil {
		return fmt.Errorf("write memory files: %w", err)
	} else if memCount > 0 {
		fmt.Printf("  Agent memory: %d entry(ies) → xcaf/agents/<id>/memory/\n", memCount)
	}

	discoverRootContextFiles(".", config)

	if err := WriteSplitFiles(config, "."); err != nil {
		return fmt.Errorf("[project] failed to write split xcaf files: %w", err)
	}

	if err := pruneOrphanMemory(config, "."); err != nil {
		return fmt.Errorf("prune memory: %w", err)
	}

	return nil
}

// countImportOverrides counts total provider overrides in the config.
func countImportOverrides(config *ast.XcaffoldConfig) int {
	overrideCount := 0
	if config.Overrides == nil {
		return overrideCount
	}
	for name := range config.Agents {
		overrideCount += len(config.Overrides.AgentProviders(name))
	}
	for name := range config.Skills {
		overrideCount += len(config.Overrides.SkillProviders(name))
	}
	for name := range config.Rules {
		overrideCount += len(config.Overrides.RuleProviders(name))
	}
	for name := range config.Workflows {
		overrideCount += len(config.Overrides.WorkflowProviders(name))
	}
	return overrideCount
}

// printMergeImportSummary outputs the import completion summary.
func printMergeImportSummary(xcafDest string, config *ast.XcaffoldConfig, providers []importer.ProviderImporter, overrideCount int) {
	importCount := len(config.Agents) + len(config.Skills) + len(config.Rules) +
		len(config.Workflows) + len(config.MCP)
	fmt.Printf("\n[project] ✓ Import complete. Created %s with %d resources from %d directories.\n",
		xcafDest, importCount, len(providers))
	fmt.Printf("  Split xcaf/ files written to xcaf/ directory.\n")
	fmt.Printf("  Resources tagged with targets: [%s].\n", strings.Join(sortedProviderNames(providers), ", "))
	if overrideCount > 0 {
		fmt.Printf("  %d conflicts detected — override files created. Run 'xcaffold validate' to review.\n", overrideCount)
	}
	fmt.Println("  Run 'xcaffold apply' when ready to compile to your target platforms.")
}
