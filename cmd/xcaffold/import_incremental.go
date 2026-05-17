package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/prompt"
	"github.com/saero-ai/xcaffold/internal/state"
	providerspkg "github.com/saero-ai/xcaffold/providers"
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
		if err := rewriteChangedResourcesInPlace(scannedConfigCopy, diff, provider); err != nil {
			return err
		}
		// Refresh state after rewriting resources (use scannedConfigCopy — has SourceFile fields applied)
		if err := refreshStateAfterImport(scannedConfigCopy, diff, provider); err != nil {
			return err
		}
		return nil
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

// refreshStateAfterImport updates the state file with new hashes for rewritten resources.
func refreshStateAfterImport(cfg *ast.XcaffoldConfig, diff ResourceDiff, target string) error {
	// Read existing state if it exists
	stateFilePath := filepath.Join(".xcaffold", "project.xcaf.state")
	var manifest *state.StateManifest
	if _, err := os.Stat(stateFilePath); err == nil {
		manifest, err = state.ReadState(stateFilePath)
		if err != nil {
			return fmt.Errorf("failed to read state: %w", err)
		}
	} else {
		// Create minimal state if it doesn't exist
		manifest = &state.StateManifest{
			Version:         1,
			XcaffoldVersion: state.XcaffoldVersion,
			Targets:         make(map[string]state.TargetState),
		}
	}

	// Update hashes for changed resources in source files and artifacts
	for kind, entries := range diff.Changed {
		for _, entry := range entries {
			switch kind {
			case "rule":
				if r, ok := cfg.Rules[entry.Name]; ok && r.SourceFile != "" {
					updateResourceHashes(manifest, r.SourceFile, entry.Name, target)
				}
			case "agent":
				if a, ok := cfg.Agents[entry.Name]; ok && a.SourceFile != "" {
					updateResourceHashes(manifest, a.SourceFile, entry.Name, target)
				}
			case "skill":
				if s, ok := cfg.Skills[entry.Name]; ok && s.SourceFile != "" {
					updateResourceHashes(manifest, s.SourceFile, entry.Name, target)
				}
			case "workflow":
				if w, ok := cfg.Workflows[entry.Name]; ok && w.SourceFile != "" {
					updateResourceHashes(manifest, w.SourceFile, entry.Name, target)
				}
			case "mcp":
				if m, ok := cfg.MCP[entry.Name]; ok && m.SourceFile != "" {
					updateResourceHashes(manifest, m.SourceFile, entry.Name, target)
				}
			case "context":
				if c, ok := cfg.Contexts[entry.Name]; ok && c.SourceFile != "" {
					updateResourceHashes(manifest, c.SourceFile, entry.Name, target)
				}
			}
		}
	}

	// Write updated state
	return state.WriteState(manifest, stateFilePath)
}

// updateResourceHashes updates the source file and artifact hashes for a resource.
func updateResourceHashes(manifest *state.StateManifest, sourceFile, name, target string) {
	sourceBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		return
	}
	sum := sha256.Sum256(sourceBytes)
	sourceHashStr := "sha256:" + hex.EncodeToString(sum[:])

	setSourceHash(manifest.SourceFiles, sourceFile, sourceHashStr, &manifest.SourceFiles)

	// Source .xcaf files are shared across targets — refresh the hash everywhere.
	for tn, ts := range manifest.Targets {
		setSourceHash(ts.SourceFiles, sourceFile, sourceHashStr, &ts.SourceFiles)
		manifest.Targets[tn] = ts
	}

	targetState, ok := manifest.Targets[target]
	if !ok {
		return
	}

	outputDir := outputDirForTarget(target)
	for i := range targetState.Artifacts {
		artifact := &targetState.Artifacts[i]
		if !artifactMatchesResource(artifact.Path, name) {
			continue
		}
		diskPath := artifact.Path
		if outputDir != "" && !strings.HasPrefix(diskPath, "root:") {
			diskPath = filepath.Join(outputDir, artifact.Path)
		} else if strings.HasPrefix(diskPath, "root:") {
			diskPath = strings.TrimPrefix(diskPath, "root:")
		}
		artifactBytes, err := os.ReadFile(diskPath)
		if err != nil {
			continue
		}
		hash := sha256.Sum256(artifactBytes)
		artifact.Hash = "sha256:" + hex.EncodeToString(hash[:])
	}
	manifest.Targets[target] = targetState
}

// outputDirForTarget returns the on-disk output directory for a target (e.g. ".claude").
func outputDirForTarget(target string) string {
	m, ok := providerspkg.ManifestFor(target)
	if !ok {
		return ""
	}
	return m.OutputDir
}

// setSourceHash updates the entry matching path; appends if not found.
func setSourceHash(entries []state.SourceFile, path, hash string, store *[]state.SourceFile) {
	for i := range entries {
		if entries[i].Path == path {
			entries[i].Hash = hash
			*store = entries
			return
		}
	}
	*store = append(entries, state.SourceFile{Path: path, Hash: hash})
}

// artifactMatchesResource matches paths like "rules/no-secrets.md" against name "no-secrets".
func artifactMatchesResource(path, name string) bool {
	base := filepath.Base(path)
	if base == name {
		return true
	}
	if ext := filepath.Ext(base); ext != "" && strings.TrimSuffix(base, ext) == name {
		return true
	}
	return filepath.Base(filepath.Dir(path)) == name
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
// .xcaf files, respecting the detected layout (flat or nested) for new resources.
// Resources with SourceFile set are written back to their original paths.
// New resources without SourceFile are written using the detected layout.
// target is the optional provider name for override routing (e.g., "claude", "gemini").
func rewriteChangedResourcesInPlace(cfg *ast.XcaffoldConfig, diff ResourceDiff, target string) error {
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

	// Write each resource back to its source file (if available) or use detected layout
	// Skip project resources - they should not be rewritten during incremental import
	for kind, names := range allChanges {
		if kind == "project" {
			continue
		}
		opts := rewriteOpts{
			layout: detectLayout("xcaf", kind),
			target: target,
		}
		for _, name := range names {
			if err := rewriteResourceInPlace(cfg, kind, name, opts); err != nil {
				return err
			}
		}
	}
	return nil
}

type rewriteOpts struct {
	layout layoutMode
	target string
}

func rewriteResourceInPlace(cfg *ast.XcaffoldConfig, kind, name string, opts rewriteOpts) error {
	switch kind {
	case "rule":
		return rewriteRuleInPlace(cfg, name, opts.layout, opts.target)
	case "agent":
		return rewriteAgentInPlace(cfg, name, opts.layout, opts.target)
	case "skill":
		return rewriteSkillInPlace(cfg, name, opts.layout, opts.target)
	case "workflow":
		return rewriteWorkflowInPlace(cfg, name, opts.layout, opts.target)
	case "mcp":
		return rewriteMCPInPlace(cfg, name, opts.layout, opts.target)
	case "context":
		return rewriteContextInPlace(cfg, name, opts.layout, opts.target)
	}
	return nil
}

// isRuleContentUnchanged checks if a rule's content is identical by comparing
// the body text only. This checks if the actual content is the same, regardless
// of metadata or formatting changes.
// Returns true if the rule body has not changed.
func isRuleContentUnchanged(rule ast.RuleConfig) bool {
	if rule.SourceFile == "" {
		// Can't deduplicate new resources without a SourceFile
		return false
	}

	// Read the existing file content
	existingBytes, err := os.ReadFile(rule.SourceFile)
	if err != nil {
		// If we can't read it, assume it's not unchanged (write it)
		return false
	}

	// Extract the body from the existing file by parsing it
	// The file is in frontmatter format, so we need to extract the part after ---
	existingStr := string(existingBytes)
	parts := strings.Split(existingStr, "---")

	var existingBody string
	if len(parts) >= 3 {
		// Format: ---\nmetadata\n---\nbody
		existingBody = strings.TrimSpace(parts[2])
	} else if len(parts) == 2 {
		// Format: ---\nYAML body (no frontmatter)
		existingBody = strings.TrimSpace(parts[1])
	} else {
		// No frontmatter, entire file is body
		existingBody = strings.TrimSpace(existingStr)
	}

	// Compare trimmed bodies
	newBody := strings.TrimSpace(rule.Body)
	return existingBody == newBody
}

// rewriteRuleInPlace writes a rule back to its source file or using the detected layout.
// target is the optional provider name for override routing (e.g., "claude", "gemini").
func rewriteRuleInPlace(cfg *ast.XcaffoldConfig, name string, layout layoutMode, target string) error {
	rule, ok := cfg.Rules[name]
	if !ok {
		return nil
	}

	// If SourceFile is set (from incremental), write to that path
	if rule.SourceFile != "" {
		// Deduplication: skip write if body content is identical to existing file
		if isRuleContentUnchanged(rule) {
			return nil
		}

		body := rule.Body
		doc := ruleDoc{
			Kind:       "rule",
			Version:    "1.0",
			RuleConfig: rule,
		}
		doc.SourceFile = ""
		return writeFrontmatterFile(rule.SourceFile, doc, body)
	}

	// For new resources without SourceFile, use the detected layout
	// TODO: Apply override routing based on target parameter
	body := rule.Body
	doc := ruleDoc{
		Kind:       "rule",
		Version:    "1.0",
		RuleConfig: rule,
	}
	doc.SourceFile = ""

	if layout == layoutFlat {
		// Write to flat location: xcaf/rules/<name>.xcaf
		filePath := filepath.Join("xcaf", kindToPlural("rule"), name+".xcaf")
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return writeFrontmatterFile(filePath, doc, body)
	}

	// Otherwise, write using nested layout (fallback)
	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{name: rule},
		},
	}
	return writeRuleFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteAgentInPlace writes an agent back to its source file or using the detected layout.
// target is the optional provider name for override routing (e.g., "claude", "gemini").
func rewriteAgentInPlace(cfg *ast.XcaffoldConfig, name string, layout layoutMode, target string) error {
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

	// For new resources without SourceFile, use the detected layout
	// TODO: Apply override routing based on target parameter
	body := agent.Body
	doc := agentDoc{
		Kind:        "agent",
		Version:     "1.0",
		AgentConfig: agent,
	}
	doc.SourceFile = ""

	if layout == layoutFlat {
		// Write to flat location: xcaf/agents/<name>.xcaf
		filePath := filepath.Join("xcaf", kindToPlural("agent"), name+".xcaf")
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return writeFrontmatterFile(filePath, doc, body)
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{name: agent},
		},
	}
	return writeAgentFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteSkillInPlace writes a skill back to its source file or using the detected layout.
// target is the optional provider name for override routing (e.g., "claude", "gemini").
func rewriteSkillInPlace(cfg *ast.XcaffoldConfig, name string, layout layoutMode, target string) error {
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

	// For new resources without SourceFile, use the detected layout
	// TODO: Apply override routing based on target parameter
	body := skill.Body
	doc := skillDoc{
		Kind:        "skill",
		Version:     "1.0",
		SkillConfig: skill,
	}
	doc.SourceFile = ""

	if layout == layoutFlat {
		// Write to flat location: xcaf/skills/<name>.xcaf
		filePath := filepath.Join("xcaf", kindToPlural("skill"), name+".xcaf")
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return writeFrontmatterFile(filePath, doc, body)
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{name: skill},
		},
	}
	return writeSkillFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteWorkflowInPlace writes a workflow back to its source file or using the detected layout.
// target is the optional provider name for override routing (e.g., "claude", "gemini").
func rewriteWorkflowInPlace(cfg *ast.XcaffoldConfig, name string, layout layoutMode, target string) error {
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

	// For new resources without SourceFile, use the detected layout
	// TODO: Apply override routing based on target parameter
	doc := workflowDoc{
		Kind:           "workflow",
		Version:        "1.0",
		WorkflowConfig: workflow,
	}
	doc.SourceFile = ""

	if layout == layoutFlat {
		// Write to flat location: xcaf/workflows/<name>.xcaf
		filePath := filepath.Join("xcaf", kindToPlural("workflow"), name+".xcaf")
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return writeFrontmatterFile(filePath, doc, "")
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: map[string]ast.WorkflowConfig{name: workflow},
		},
	}
	return writeWorkflowFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteMCPInPlace writes an MCP config back to its source file or using the detected layout.
// target is the optional provider name for override routing (e.g., "claude", "gemini").
func rewriteMCPInPlace(cfg *ast.XcaffoldConfig, name string, layout layoutMode, target string) error {
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

	// For new resources without SourceFile, use the detected layout
	// TODO: Apply override routing based on target parameter
	doc := mcpDoc{
		Kind:      "mcp",
		Version:   "1.0",
		MCPConfig: mcp,
	}
	doc.SourceFile = ""

	if layout == layoutFlat {
		// Write to flat location: xcaf/mcp/<name>.xcaf
		filePath := filepath.Join("xcaf", kindToPlural("mcp"), name+".xcaf")
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return writeFrontmatterFile(filePath, doc, "")
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{name: mcp},
		},
	}
	return writeMCPFiles(tmpCfg, "xcaf", "1.0", map[string]bool{name: true})
}

// rewriteContextInPlace writes a context back to its source file or using the detected layout.
// target is the optional provider name for override routing (e.g., "claude", "gemini").
func rewriteContextInPlace(cfg *ast.XcaffoldConfig, name string, layout layoutMode, target string) error {
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

	// For new resources without SourceFile, use the detected layout
	// TODO: Apply override routing based on target parameter
	body := ctx.Body
	doc := contextDoc{
		Kind:          "context",
		Version:       "1.0",
		ContextConfig: ctx,
	}
	doc.SourceFile = ""

	if layout == layoutFlat {
		// Write to flat location: xcaf/contexts/<name>.xcaf
		filePath := filepath.Join("xcaf", kindToPlural("context"), name+".xcaf")
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return writeFrontmatterFile(filePath, doc, body)
	}

	tmpCfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Contexts: map[string]ast.ContextConfig{name: ctx},
		},
	}
	return writeContextFiles(tmpCfg, "xcaf", "1.0") // note: writeContextFiles doesn't take a filter param
}
