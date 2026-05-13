package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/parser"
	providerspkg "github.com/saero-ai/xcaffold/providers"
)

// skillExtractionCtx groups the skill extraction parameters.
type skillExtractionCtx struct {
	skillFile string
	id        string
	outDir    string
}

// skillProcessCtx groups parameters for skill file processing functions.
type skillProcessCtx struct {
	id       string
	outDir   string
	manifest *providerspkg.ProviderManifest
	warnings *[]string
}

// extractSkillSubdirs scans the skill directory for known canonical and
// provider-native subdirectories, copies their files to xcaf/skills/<id>/,
// and returns slices of copied paths grouped by canonical category.
//
// manifest provides the provider's SubdirMap; if nil, all subdirectory files
// are routed to passthrough.
//
// The context struct provides skillFile (path to SKILL.md), id (skill identifier),
// and outDir (base directory for output paths xcaf/skills/<id>/...).  When
// outDir is empty, the current working directory is used.
//
// For providers with SkillMDAsReference=true, any .md file alongside SKILL.md
// (not in a subdirectory) is treated as a reference.
//
// Files from subdirectories that have no canonical mapping are copied to
// xcaf/skills/<id>/<subdir>/ alongside canonical subdirectories.
//
// The discoveredDirs slice contains all discovered subdirectory names (both
// canonical and custom) in the skill directory, suitable for populating
// the Artifacts field of SkillConfig.
func extractSkillSubdirs(ctx skillExtractionCtx, manifest *providerspkg.ProviderManifest, warnings *[]string) (discoveredDirs []string, err error) {
	skillDir := filepath.Dir(ctx.skillFile)
	subdirMap := buildSubdirMap(manifest, warnings)

	entries, readErr := os.ReadDir(skillDir)
	if readErr != nil {
		return nil, nil
	}

	skillCtx := skillProcessCtx{
		id:       ctx.id,
		outDir:   ctx.outDir,
		manifest: manifest,
		warnings: warnings,
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			discoveredDirs = append(discoveredDirs, name)
			processSkillSubdir(skillCtx, skillDir, name, subdirMap)
			continue
		}
		processSkillRootFile(skillCtx, skillDir, name)
	}

	return discoveredDirs, nil
}

// buildSubdirMap extracts the subdir mapping from the provider manifest.
func buildSubdirMap(manifest *providerspkg.ProviderManifest, warnings *[]string) map[string]string {
	if manifest == nil {
		*warnings = append(*warnings, "extractSkillSubdirs: unknown provider unknown-provider (nil manifest) — all subdirectory files routed to passthrough")
		return make(map[string]string)
	}
	if len(manifest.SubdirMap) == 0 {
		*warnings = append(*warnings, fmt.Sprintf("extractSkillSubdirs: provider %q has no SubdirMap — all subdirectory files routed to passthrough", manifest.Name))
	}
	return manifest.SubdirMap
}

// processSkillSubdir processes files within a subdirectory of a skill.
func processSkillSubdir(ctx skillProcessCtx, skillDir, subdir string, subdirMap map[string]string) {
	canonicalSubdir := subdirMap[subdir]
	subEntries, _ := os.ReadDir(filepath.Join(skillDir, subdir))
	for _, sub := range subEntries {
		if sub.IsDir() {
			continue
		}
		src := filepath.Join(skillDir, subdir, sub.Name())
		if canonicalSubdir != "" {
			copySkillFile(ctx, src, canonicalSubdir, sub.Name())
		} else {
			copySkillFile(ctx, src, subdir, sub.Name())
		}
	}
}

// processSkillRootFile handles files at the root of a skill directory.
func processSkillRootFile(ctx skillProcessCtx, skillDir, name string) {
	if ctx.manifest != nil && ctx.manifest.SkillMDAsReference && strings.ToLower(name) != "skill.md" && strings.HasSuffix(strings.ToLower(name), ".md") {
		copySkillFile(ctx, filepath.Join(skillDir, name), "references", name)
	}
}

// copySkillFile copies a skill support file to its destination.
func copySkillFile(ctx skillProcessCtx, src, subdir, filename string) {
	var dest string
	if ctx.outDir != "" {
		dest = filepath.Join(ctx.outDir, "xcaf", "skills", ctx.id, subdir, filename)
	} else {
		dest = filepath.Join("xcaf", "skills", ctx.id, subdir, filename)
	}
	if copyErr := copyFile(src, dest); copyErr != nil {
		*ctx.warnings = append(*ctx.warnings, fmt.Sprintf("failed to copy skill file %s: %v", src, copyErr))
	}
}

// extractBodyAfterFrontmatter returns the markdown body that follows the YAML frontmatter block.
// If the data has no frontmatter (does not start with "---\n"), the entire content is returned.
// Leading and trailing whitespace is trimmed from the returned body.
func extractBodyAfterFrontmatter(data []byte) string {
	if !bytes.HasPrefix(data, []byte("---\n")) && !bytes.HasPrefix(data, []byte("---\r\n")) {
		return strings.TrimSpace(string(data))
	}
	// Find the closing "---" marker after the opening one.
	idx := bytes.Index(data[4:], []byte("\n---"))
	if idx == -1 {
		return strings.TrimSpace(string(data))
	}
	// Skip past the closing "---\n" line to get to the body.
	bodyStart := 4 + idx + len("\n---")
	if bodyStart >= len(data) {
		return ""
	}
	// Consume a trailing newline after the closing marker.
	if data[bodyStart] == '\n' {
		bodyStart++
	}
	return strings.TrimSpace(string(data[bodyStart:]))
}

// pruneOrphanMemory removes xcaf/agents/<id>/memory/ directories for agents
// that are not present in the current import scope. Agents referenced only via
// config.Memory (e.g. global agents whose project-scoped memory was imported)
// are preserved even when they have no entry in config.Agents.
// After pruning, any now-empty agent directory (no .xcaf file, no memory/) is
// also removed.
func pruneOrphanMemory(config *ast.XcaffoldConfig, rootDir string) error {
	agentsDir := filepath.Join(rootDir, "xcaf", "agents")
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		return nil
	}

	validAgents := buildValidAgentSet(config)
	memoryAgents := buildMemoryAgentSet(config)

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if err := pruneMemoryForAgent(agentsDir, entry.Name(), validAgents, memoryAgents); err != nil {
			return err
		}
	}

	return removeEmptyAgentDirs(agentsDir)
}

// buildValidAgentSet creates a set of all declared agents.
func buildValidAgentSet(config *ast.XcaffoldConfig) map[string]bool {
	validAgents := make(map[string]bool)
	for id := range config.Agents {
		validAgents[id] = true
	}
	return validAgents
}

// buildMemoryAgentSet creates a set of agents with explicitly imported memory entries.
func buildMemoryAgentSet(config *ast.XcaffoldConfig) map[string]bool {
	memoryAgents := make(map[string]bool)
	for memPath := range config.Memory {
		agentID := strings.SplitN(filepath.ToSlash(memPath), "/", 2)[0]
		memoryAgents[agentID] = true
	}
	return memoryAgents
}

// pruneMemoryForAgent removes memory directory if agent is not valid or declared.
func pruneMemoryForAgent(agentsDir, agentID string, validAgents, memoryAgents map[string]bool) error {
	memDir := filepath.Join(agentsDir, agentID, "memory")
	if _, err := os.Stat(memDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !validAgents[agentID] && !memoryAgents[agentID] {
		return os.RemoveAll(memDir)
	}
	return nil
}

// removeEmptyAgentDirs deletes agent directories with no content.
func removeEmptyAgentDirs(agentsDir string) error {
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentDir := filepath.Join(agentsDir, entry.Name())
		dirEntries, err := os.ReadDir(agentDir)
		if err == nil && len(dirEntries) == 0 {
			_ = os.Remove(agentDir)
		}
	}
	return nil
}

// writeMemoryFiles writes each memory entry in config to a plain .md file under
// xcaf/agents/<agentID>/memory/<name>.md, mirroring the convention the compiler
// uses to discover memory at build time. Returns the number of files written.
func writeMemoryFiles(config *ast.XcaffoldConfig) (int, error) {
	if len(config.Memory) == 0 {
		return 0, nil
	}
	count := 0
	for _, k := range sortedMapKeys(config.Memory) {
		mem := config.Memory[k]
		parts := strings.SplitN(filepath.ToSlash(k), "/", 2)
		var agentID, memName string
		if len(parts) == 2 {
			agentID = parts[0]
			memName = parts[1]
		} else {
			agentID = mem.AgentRef
			if agentID == "" {
				agentID = k
			}
			memName = k
		}
		outPath := filepath.Join("xcaf", "agents", agentID, "memory", filepath.FromSlash(memName)+".md")
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return count, fmt.Errorf("create memory dir: %w", err)
		}
		if err := os.WriteFile(filepath.Clean(outPath), []byte(mem.Content), 0644); err != nil {
			return count, fmt.Errorf("write memory file: %w", err)
		}
		count++
	}
	return count, nil
}

// runPostImportSteps executes the shared post-import steps: memory writing,
// root context discovery, and memory pruning. The injectToolkit parameter
// controls whether toolkit reference templates and xaff toolkit are injected
// (typically false for import, true for init).
func runPostImportSteps(config *ast.XcaffoldConfig, projectDir string, injectToolkit bool) error {
	if memCount, err := writeMemoryFiles(config); err != nil {
		return fmt.Errorf("write memory: %w", err)
	} else if memCount > 0 {
		fmt.Printf("  Agent memory: %d entry(ies) → xcaf/agents/<id>/memory/\n", memCount)
	}

	discoverRootContextFiles(projectDir, config)

	if err := pruneOrphanMemory(config, projectDir); err != nil {
		return fmt.Errorf("prune memory: %w", err)
	}

	if injectToolkit {
		if err := injectXaffToolkitAfterImport(projectDir); err != nil {
			return err
		}
	}
	return nil
}

// runProviderPostImport executes provider-specific post-import steps that fall
// outside the scope of the ProviderImporter interface (cross-boundary files,
// out-of-tree memory sources, unsupported-provider warnings).
func runProviderPostImport(provider, projectDir string, config *ast.XcaffoldConfig, warnings *[]string) error {
	m, ok := providerspkg.ManifestFor(provider)
	if !ok {
		return nil
	}
	for _, mcpPath := range m.RootMCPPaths {
		fullPath := filepath.Join(projectDir, mcpPath)
		if data, err := os.ReadFile(fullPath); err == nil {
			count := 0
			if err := importSettings(data, config, &count, warnings); err != nil {
				*warnings = append(*warnings, fmt.Sprintf("%s partially imported: %v", mcpPath, err))
			}
		}
	}
	if m.PostImportWarning != "" {
		*warnings = append(*warnings, m.PostImportWarning)
	}
	return nil
}

// discoverRootContextFiles scans the project root for known root context files
// using the provider manifest registry and populates config.Contexts.
func discoverRootContextFiles(projectDir string, config *ast.XcaffoldConfig) {
	if config.Contexts == nil {
		config.Contexts = make(map[string]ast.ContextConfig)
	}

	for _, m := range providerspkg.Manifests() {
		if m.RootContextFile == "" {
			continue
		}
		fullPath := filepath.Join(projectDir, filepath.FromSlash(m.RootContextFile))
		if data, err := os.ReadFile(fullPath); err == nil {
			config.Contexts[m.Name] = ast.ContextConfig{
				Name:    m.Name,
				Targets: []string{m.Name},
				Body:    string(data),
			}
		}
	}
}

// copyFile copies the file at src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	return os.WriteFile(dst, data, 0600)
}

// findImporterByProvider returns the registered ProviderImporter for the given
// provider name, or nil when no importer is registered for that provider.
func findImporterByProvider(provider string) importer.ProviderImporter {
	for _, imp := range importer.DefaultImporters() {
		if imp.Provider() == provider {
			return imp
		}
	}
	return nil
}

// inferProjectName derives a project name from the current working directory.
func inferProjectName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "imported-project"
	}
	base := filepath.Base(wd)
	if base == "." || base == "/" || base == "" {
		return "imported-project"
	}
	return base
}

// printImportWarnings outputs any warnings collected during import.
func printImportWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	fmt.Println("\nWarnings:")
	for _, w := range warnings {
		fmt.Println(" ⚠", w)
	}
}

// scanProviderConfigs scans each provider via ProviderImporter and populates a map of provider -> XcaffoldConfig.
// It handles importer invocation, skill subdirectory extraction, extras reclassification, and post-import steps.
func scanProviderConfigs(providers []importer.ProviderImporter, warnings *[]string) map[string]*ast.XcaffoldConfig {
	providerConfigs := make(map[string]*ast.XcaffoldConfig)

	for _, imp := range providers {
		dir := imp.InputDir()
		provider := imp.Provider()
		fmt.Printf("  Scanning %s ...\n", dir)

		tmpConfig := &ast.XcaffoldConfig{
			ResourceScope: ast.ResourceScope{
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
		}

		if err := imp.Import(dir, tmpConfig); err != nil {
			*warnings = append(*warnings, fmt.Sprintf("%s import: %v", provider, err))
		}

		manifest, _ := providerspkg.ManifestFor(provider)
		for id := range tmpConfig.Skills {
			skillFile := filepath.Join(dir, "skills", id, "SKILL.md")
			if _, err := os.Stat(skillFile); err == nil {
				discoveredDirs, subdirsErr := extractSkillSubdirs(skillExtractionCtx{skillFile, id, ""}, &manifest, warnings)
				if subdirsErr != nil {
					*warnings = append(*warnings, fmt.Sprintf("extractSkillSubdirs %s: %v", id, subdirsErr))
				}
				sc := tmpConfig.Skills[id]
				if len(discoveredDirs) > 0 {
					sc.Artifacts = discoveredDirs
				}
				tmpConfig.Skills[id] = sc
			}
		}

		if err := parser.ReclassifyExtras(tmpConfig, importer.DefaultImporters()); err != nil {
			*warnings = append(*warnings, fmt.Sprintf("reclassify extras (%s): %v", provider, err))
		}

		dirAbs, _ := filepath.Abs(dir)
		projectDir := filepath.Dir(dirAbs)
		if err := runProviderPostImport(provider, projectDir, tmpConfig, warnings); err != nil {
			// Note: caller must handle this error
			*warnings = append(*warnings, fmt.Sprintf("post-import error: %v", err))
			continue
		}

		providerConfigs[provider] = tmpConfig
	}

	return providerConfigs
}
