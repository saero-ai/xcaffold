package main

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/prompt"
)

// incrementalImportCtx groups parameters for incremental import operations.
type incrementalImportCtx struct {
	xcafDest      string
	scopeName     string
	config        *ast.XcaffoldConfig
	scannedConfig *ast.XcaffoldConfig
	warnings      *[]string
}

// incrementalImport scans provider resources, diffs against existing xcaf/,
// shows a preview, and imports only new/changed resources after confirmation.
func incrementalImport(platformDir, xcafDest, scopeName, provider string) error {
	scannedConfig := newEmptyConfig()
	var warnings []string
	extractAndPostProcess(platformDir, provider, scannedConfig, &warnings)

	existingConfig, err := parser.ParseDirectory(".")
	if err != nil {
		return fmt.Errorf("parsing existing xcaf/: %w", err)
	}

	// Apply kind filters to scanned config BEFORE diffing
	applyKindFilters(scannedConfig)

	// Make a copy of scannedConfig for use after diffing, since diffResources modifies in-place
	scannedConfigCopy := deepCopyConfig(scannedConfig)

	// Save SourceFiles from existingConfig BEFORE diffing (diffResources clears them)
	savedSourceFiles := extractSourceFiles(existingConfig)

	diff := diffResources(scannedConfig, existingConfig)
	totalNew := diff.TotalNew()
	totalChanged := diff.TotalChanged()

	if totalNew == 0 && totalChanged == 0 {
		fmt.Printf("\n  %s  All provider resources already in xcaf/. Nothing to import.\n", colorGreen(glyphOK()))
		return nil
	}

	renderImportPreview(diff)

	ctx := incrementalImportCtx{
		xcafDest:      xcafDest,
		scopeName:     scopeName,
		config:        existingConfig,
		scannedConfig: scannedConfigCopy,
		warnings:      nil, // not used in confirmAndExecuteImport
	}
	if err := confirmAndExecuteImport(ctx, diff, func() error {
		// Apply saved SourceFiles to scannedConfig so rewrite can find them
		applySourceFilesForChangedResources(scannedConfigCopy, diff, savedSourceFiles)
		return rewriteChangedResourcesInPlace(scannedConfigCopy, diff)
	}); err != nil {
		return err
	}

	if !importDryRun {
		fmt.Printf("\n  %s  Imported %d resources.\n", colorGreen(glyphOK()), totalNew+totalChanged)
	}
	return nil
}

// newEmptyConfig creates a new empty XcaffoldConfig.
func newEmptyConfig() *ast.XcaffoldConfig {
	return &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
}

// confirmAndExecuteImport prompts for confirmation and merges resources if approved.
func confirmAndExecuteImport(ctx incrementalImportCtx, diff ResourceDiff, writeFunc func() error) error {
	if !importDryRun && !importYes {
		msg := fmt.Sprintf("Import %d new + %d changed resources?", diff.TotalNew(), diff.TotalChanged())
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

	mergeResourceDiff(ctx.config, ctx.scannedConfig, diff)
	if err := writeFunc(); err != nil {
		return fmt.Errorf("[%s] failed to write split xcaf files: %w", ctx.scopeName, err)
	}
	return nil
}

// mergeResourceDiff copies new and changed resources from scanned into existing.
func mergeResourceDiff(existingConfig, scannedConfig *ast.XcaffoldConfig, diff ResourceDiff) {
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
}

// extractSourceFiles extracts all SourceFile values from a config before they're
// stripped by diffResources. Returns a map keyed by "kind:name".
func extractSourceFiles(cfg *ast.XcaffoldConfig) map[string]string {
	files := make(map[string]string)
	for name, a := range cfg.Agents {
		if a.SourceFile != "" {
			files["agent:"+name] = a.SourceFile
		}
	}
	for name, s := range cfg.Skills {
		if s.SourceFile != "" {
			files["skill:"+name] = s.SourceFile
		}
	}
	for name, r := range cfg.Rules {
		if r.SourceFile != "" {
			files["rule:"+name] = r.SourceFile
		}
	}
	for name, w := range cfg.Workflows {
		if w.SourceFile != "" {
			files["workflow:"+name] = w.SourceFile
		}
	}
	for name, m := range cfg.MCP {
		if m.SourceFile != "" {
			files["mcp:"+name] = m.SourceFile
		}
	}
	for name, c := range cfg.Contexts {
		if c.SourceFile != "" {
			files["context:"+name] = c.SourceFile
		}
	}
	return files
}

// applySourceFilesForChangedResources restores SourceFile fields for resources
// marked as changed in the diff, using saved source files from before diffing.
func applySourceFilesForChangedResources(scannedConfig *ast.XcaffoldConfig, diff ResourceDiff, savedFiles map[string]string) {
	for _, entries := range diff.Changed {
		for _, entry := range entries {
			key := entry.Kind + ":" + entry.Name
			if srcFile, ok := savedFiles[key]; ok {
				switch entry.Kind {
				case "rule":
					if r, ok := scannedConfig.Rules[entry.Name]; ok {
						r.SourceFile = srcFile
						scannedConfig.Rules[entry.Name] = r
					}
				case "agent":
					if a, ok := scannedConfig.Agents[entry.Name]; ok {
						a.SourceFile = srcFile
						scannedConfig.Agents[entry.Name] = a
					}
				case "skill":
					if s, ok := scannedConfig.Skills[entry.Name]; ok {
						s.SourceFile = srcFile
						scannedConfig.Skills[entry.Name] = s
					}
				case "workflow":
					if w, ok := scannedConfig.Workflows[entry.Name]; ok {
						w.SourceFile = srcFile
						scannedConfig.Workflows[entry.Name] = w
					}
				case "mcp":
					if m, ok := scannedConfig.MCP[entry.Name]; ok {
						m.SourceFile = srcFile
						scannedConfig.MCP[entry.Name] = m
					}
				case "context":
					if c, ok := scannedConfig.Contexts[entry.Name]; ok {
						c.SourceFile = srcFile
						scannedConfig.Contexts[entry.Name] = c
					}
				}
			}
		}
	}
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

// rewriteChangedResourcesInPlace writes changed/new resources back to their source
// .xcaf files in FLAT layout (xcaf/rules/name.xcaf, xcaf/agents/name.xcaf, etc.)
// instead of using nested layout. New resources without SourceFile are written
// via the default nested layout writers as fallback.
func rewriteChangedResourcesInPlace(cfg *ast.XcaffoldConfig, diff ResourceDiff) error {
	// Process both changed and new resources
	allChanges := make(map[string][]string)
	for kind, entries := range diff.Changed {
		for _, e := range entries {
			allChanges[kind] = append(allChanges[kind], e.Name)
		}
	}
	for kind, entries := range diff.New {
		for _, e := range entries {
			allChanges[kind] = append(allChanges[kind], e.Name)
		}
	}

	// Write each resource back to its source file (if available) or use nested layout
	for kind, names := range allChanges {
		for _, name := range names {
			if err := rewriteResourceInPlace(cfg, kind, name); err != nil {
				return err
			}
		}
	}
	return nil
}

// rewriteResourceInPlace writes a single resource back to its SourceFile if set,
// otherwise falls back to writing via the standard nested writer for that kind.
func rewriteResourceInPlace(cfg *ast.XcaffoldConfig, kind, name string) error {
	switch kind {
	case "rule":
		return rewriteRuleInPlace(cfg, name)
	case "agent":
		return rewriteAgentInPlace(cfg, name)
	case "skill":
		return rewriteSkillInPlace(cfg, name)
	case "workflow":
		return rewriteWorkflowInPlace(cfg, name)
	case "mcp":
		return rewriteMCPInPlace(cfg, name)
	case "context":
		return rewriteContextInPlace(cfg, name)
	}
	return nil
}

// rewriteRuleInPlace writes a rule back to its source file or nested layout
func rewriteRuleInPlace(cfg *ast.XcaffoldConfig, name string) error {
	rule, ok := cfg.Rules[name]
	if !ok {
		return nil
	}

	// If SourceFile is set (from incremental), write to that path
	if rule.SourceFile != "" {
		body := rule.Body
		doc := ruleDoc{
			Kind:       "rule",
			Version:    "1.0",
			RuleConfig: rule,
		}
		doc.SourceFile = ""
		return writeFrontmatterFile(rule.SourceFile, doc, body)
	}

	// Otherwise, write using nested layout (fallback for new resources)
	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{name: rule},
		},
	}
	return writeRuleFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteAgentInPlace writes an agent back to its source file or nested layout
func rewriteAgentInPlace(cfg *ast.XcaffoldConfig, name string) error {
	agent, ok := cfg.Agents[name]
	if !ok {
		return nil
	}

	if agent.SourceFile != "" {
		body := agent.Body
		doc := agentDoc{
			Kind:        "agent",
			Version:     "1.0",
			AgentConfig: agent,
		}
		doc.SourceFile = ""
		return writeFrontmatterFile(agent.SourceFile, doc, body)
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{name: agent},
		},
	}
	return writeAgentFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteSkillInPlace writes a skill back to its source file or nested layout
func rewriteSkillInPlace(cfg *ast.XcaffoldConfig, name string) error {
	skill, ok := cfg.Skills[name]
	if !ok {
		return nil
	}

	if skill.SourceFile != "" {
		body := skill.Body
		doc := skillDoc{
			Kind:        "skill",
			Version:     "1.0",
			SkillConfig: skill,
		}
		doc.SourceFile = ""
		return writeFrontmatterFile(skill.SourceFile, doc, body)
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{name: skill},
		},
	}
	return writeSkillFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteWorkflowInPlace writes a workflow back to its source file or nested layout
func rewriteWorkflowInPlace(cfg *ast.XcaffoldConfig, name string) error {
	workflow, ok := cfg.Workflows[name]
	if !ok {
		return nil
	}

	if workflow.SourceFile != "" {
		// Workflows don't have a Body field; write pure YAML
		doc := workflowDoc{
			Kind:           "workflow",
			Version:        "1.0",
			WorkflowConfig: workflow,
		}
		doc.SourceFile = ""
		return writeFrontmatterFile(workflow.SourceFile, doc, "")
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{name: workflow},
		},
	}
	return writeWorkflowFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteMCPInPlace writes an MCP config back to its source file or nested layout
func rewriteMCPInPlace(cfg *ast.XcaffoldConfig, name string) error {
	mcp, ok := cfg.MCP[name]
	if !ok {
		return nil
	}

	if mcp.SourceFile != "" {
		// MCP configs don't have a Body field; write pure YAML
		doc := mcpDoc{
			Kind:      "mcp",
			Version:   "1.0",
			MCPConfig: mcp,
		}
		doc.SourceFile = ""
		return writeFrontmatterFile(mcp.SourceFile, doc, "")
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{name: mcp},
		},
	}
	return writeMCPFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteContextInPlace writes a context back to its source file or nested layout
func rewriteContextInPlace(cfg *ast.XcaffoldConfig, name string) error {
	ctx, ok := cfg.Contexts[name]
	if !ok {
		return nil
	}

	if ctx.SourceFile != "" {
		body := ctx.Body
		doc := contextDoc{
			Kind:          "context",
			Version:       "1.0",
			ContextConfig: ctx,
		}
		doc.SourceFile = ""
		return writeFrontmatterFile(ctx.SourceFile, doc, body)
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{name: ctx},
		},
	}
	return writeContextFiles(tmpCfg, "xcaf", "1.0") // note: writeContextFiles doesn't take a filter param
}
