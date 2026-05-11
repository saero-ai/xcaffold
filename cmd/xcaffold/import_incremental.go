package main

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/prompt"
)

// incrementalImport scans provider resources, diffs against existing xcaf/,
// shows a preview, and imports only new/changed resources after confirmation.
func incrementalImport(platformDir, xcafDest, scopeName, provider string) error {
	// 1. Scan provider resources using existing extractAndPostProcess
	scannedConfig := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
	var warnings []string
	extractAndPostProcess(platformDir, provider, scannedConfig, &warnings)

	// 2. Parse existing xcaf/
	existingConfig, err := parser.ParseDirectory(".")
	if err != nil {
		return fmt.Errorf("parsing existing xcaf/: %w", err)
	}

	// 3. Diff
	diff := diffResources(scannedConfig, existingConfig)
	totalNew := diff.TotalNew()
	totalChanged := diff.TotalChanged()

	if totalNew == 0 && totalChanged == 0 {
		fmt.Printf("\n  %s  All provider resources already in xcaf/. Nothing to import.\n", colorGreen(glyphOK()))
		return nil
	}

	// 4. Render preview
	renderImportPreview(diff)

	// 5. Confirm (unless --dry-run or --yes)
	if !importDryRun && !importYes {
		msg := fmt.Sprintf("Import %d new + %d changed resources?", totalNew, totalChanged)
		ok, err := prompt.Confirm(msg, false)
		if err != nil {
			return fmt.Errorf("prompt error: %w", err)
		}
		if !ok {
			return nil
		}
	}

	if importDryRun {
		fmt.Printf("\n  Run 'xcaffold import' to apply.\n")
		return nil
	}

	// 6. Merge scanned into existing and write
	for _, entries := range diff.New {
		for _, entry := range entries {
			copyResource(existingConfig, scannedConfig, entry.Kind, entry.Name)
		}
	}
	for _, entries := range diff.Changed {
		for _, entry := range entries {
			copyResource(existingConfig, scannedConfig, entry.Kind, entry.Name)
		}
	}

	// Apply filters and write
	applyKindFilters(existingConfig)
	if err := WriteSplitFiles(existingConfig, "."); err != nil {
		return fmt.Errorf("[%s] failed to write split xcaf files: %w", scopeName, err)
	}

	fmt.Printf("\n  %s  Imported %d resources.\n", colorGreen(glyphOK()), totalNew+totalChanged)
	return nil
}

// renderImportPreview displays a diff preview of new, changed, unchanged, and xcaf-only resources.
func renderImportPreview(diff ResourceDiff) {
	fmt.Println()
	for kind, entries := range diff.New {
		for _, e := range entries {
			fmt.Printf("    %s  %-8s  %s\n", colorGreen("+"), kind, e.Name)
		}
	}
	for kind, entries := range diff.Changed {
		for _, e := range entries {
			fmt.Printf("    %s  %-8s  %s\n", colorYellow(glyphSrc()), kind, e.Name)
		}
	}
	total := diff.TotalUnchanged()
	if total > 0 {
		fmt.Printf("    %s  %d resources unchanged (skipped)\n", dim(glyphOK()), total)
	}
	xcafOnlyTotal := diff.TotalXcafOnly()
	if xcafOnlyTotal > 0 {
		fmt.Printf("    %s  %d xcaf-only resources (preserved)\n", colorGreen(glyphOK()), xcafOnlyTotal)
	}
}
