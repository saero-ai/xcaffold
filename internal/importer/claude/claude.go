package claude

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

func init() {
	importer.Register(New())
}

// ClaudeImporter imports resources from a .claude/ directory tree.
// Warnings accumulates non-fatal per-file extraction errors encountered during Import().
// Callers may inspect Warnings after Import() returns to surface skipped files.
type ClaudeImporter struct {
	Warnings     []string
	visitedLinks map[string]bool
}

// New returns a new ClaudeImporter.
func New() *ClaudeImporter {
	return &ClaudeImporter{}
}

// Provider returns the canonical provider name.
func (c *ClaudeImporter) Provider() string { return "claude" }

// GetWarnings returns non-fatal per-file extraction errors collected during the last Import() call.
func (c *ClaudeImporter) GetWarnings() []string { return c.Warnings }

// InputDir returns the root directory relative to the workspace root.
func (c *ClaudeImporter) InputDir() string { return ".claude" }

// claudeMappings maps path patterns to AST kinds. First match wins.
// settings.json appears first for the container-level KindSettings match;
// the embedded mcpServers and hooks keys are handled inside Extract().
var claudeMappings = []importer.KindMapping{
	{Pattern: "agents/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "skills/*/SKILL.md", Kind: importer.KindSkill, Layout: importer.DirectoryPerEntry},
	{Pattern: "rules/**/*.md", Kind: importer.KindRule, Layout: importer.FlatFile},
	{Pattern: "workflows/*.md", Kind: importer.KindWorkflow, Layout: importer.FlatFile},
	{Pattern: "mcp.json", Kind: importer.KindMCP, Layout: importer.StandaloneJSON},
	{Pattern: "settings.json", Kind: importer.KindSettings, Layout: importer.EmbeddedJSONKey},
	{Pattern: "settings.local.json", Kind: importer.KindSettings, Layout: importer.StandaloneJSON},
	{Pattern: "agent-memory/**", Kind: importer.KindMemory, Layout: importer.FlatFile},
}

// Classify returns the Kind and Layout for a given relative path.
// rel is relative to InputDir(). First matching entry in claudeMappings wins.
func (c *ClaudeImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range claudeMappings {
		if matchGlob(m.Pattern, rel) {
			return m.Kind, m.Layout
		}
	}
	return importer.KindUnknown, importer.LayoutUnknown
}

// Extract reads a single file and populates the appropriate section of config.
// rel is relative to InputDir().
func (c *ClaudeImporter) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	rel = filepath.ToSlash(filepath.Clean(rel))
	kind, _ := c.Classify(rel, false)

	switch kind {
	case importer.KindAgent:
		return extractAgent(rel, data, config)
	case importer.KindSkill:
		return extractSkill(rel, data, config)
	case importer.KindRule:
		return extractRule(rel, data, config)
	case importer.KindWorkflow:
		return extractWorkflow(rel, data, config)
	case importer.KindMCP:
		return extractMCPStandalone(rel, data, config)
	case importer.KindSettings:
		return extractSettings(rel, data, config)
	case importer.KindMemory:
		return extractMemory(rel, data, config)
	default:
		return fmt.Errorf("claude: no extractor for kind %q at path %q", kind, rel)
	}
}

// Import walks dir, classifies each entry, extracts classified files, and
// appends unclassified files to config.ProviderExtras["claude"].
//
// Extraction errors for individual files are non-fatal: they are collected in
// c.Warnings and the walk continues. Only I/O errors (unreadable directory or
// file) abort the walk. This ensures that a single malformed file (e.g. a rule
// with unparseable YAML frontmatter) does not prevent all subsequent files from
// being imported.
func (c *ClaudeImporter) Import(dir string, config *ast.XcaffoldConfig) error {
	c.Warnings = c.Warnings[:0]
	c.visitedLinks = make(map[string]bool)
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// I/O error accessing a directory or file — abort the walk.
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Symlinks to directories pass the d.IsDir() check above (IsDir is false
		// for symlinks). Resolve the link target and skip if it's a directory —
		// WalkDir will NOT recurse into it, so we must do it ourselves below.
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				c.Warnings = append(c.Warnings, fmt.Sprintf("resolving symlink %q: %v", path, err))
				return nil
			}
			info, err := os.Stat(target)
			if err != nil {
				c.Warnings = append(c.Warnings, fmt.Sprintf("stat symlink target %q: %v", target, err))
				return nil
			}
			if info.IsDir() {
				return c.importSymlinkedDir(path, dir, config)
			}
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("claude: rel path: %w", err)
		}
		rel = filepath.ToSlash(rel)

		data, err := readFile(path)
		if err != nil {
			c.Warnings = append(c.Warnings, fmt.Sprintf("read %q: %v", rel, err))
			return nil
		}

		kind, _ := c.Classify(rel, false)
		if kind == importer.KindUnknown {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["claude"] == nil {
				config.ProviderExtras["claude"] = make(map[string][]byte)
			}
			config.ProviderExtras["claude"][rel] = data
			return nil
		}

		if extractErr := c.Extract(rel, data, config); extractErr != nil {
			// Extraction failure is non-fatal. Stash the raw bytes so the file
			// is preserved in ProviderExtras (same as unknown files) and record
			// a warning so the caller can surface it.
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["claude"] == nil {
				config.ProviderExtras["claude"] = make(map[string][]byte)
			}
			config.ProviderExtras["claude"][rel] = data
			c.Warnings = append(c.Warnings, fmt.Sprintf("skipped %q: %v", rel, extractErr))
		}
		return nil
	})
}

// importSymlinkedDir walks a symlinked directory and imports its contents.
// WalkDir does not follow symlinks to directories, so when a skill or rule
// directory is a symlink (e.g. pointing to a shared .agents/ tree), this
// method walks the real target while computing relative paths against the
// original import root so that Classify and Extract see the expected paths.
func (c *ClaudeImporter) importSymlinkedDir(symlinkPath, importRoot string, config *ast.XcaffoldConfig) error {
	target, err := filepath.EvalSymlinks(symlinkPath)
	if err != nil {
		return nil
	}
	if c.visitedLinks[target] {
		return nil
	}
	c.visitedLinks[target] = true

	symlinkRel, err := filepath.Rel(importRoot, symlinkPath)
	if err != nil {
		return nil
	}
	return filepath.WalkDir(target, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		// Handle nested symlinks to directories within the target.
		if d.Type()&fs.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			info, err := os.Stat(resolved)
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return c.importSymlinkedDir(path, importRoot, config)
			}
		}
		relToTarget, err := filepath.Rel(target, path)
		if err != nil {
			return nil
		}
		rel := filepath.ToSlash(filepath.Join(symlinkRel, relToTarget))

		data, err := readFile(path)
		if err != nil {
			c.Warnings = append(c.Warnings, fmt.Sprintf("read %q: %v", rel, err))
			return nil
		}

		kind, _ := c.Classify(rel, false)
		if kind == importer.KindUnknown {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["claude"] == nil {
				config.ProviderExtras["claude"] = make(map[string][]byte)
			}
			config.ProviderExtras["claude"][rel] = data
			return nil
		}

		if extractErr := c.Extract(rel, data, config); extractErr != nil {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["claude"] == nil {
				config.ProviderExtras["claude"] = make(map[string][]byte)
			}
			config.ProviderExtras["claude"][rel] = data
			c.Warnings = append(c.Warnings, fmt.Sprintf("skipped %q: %v", rel, extractErr))
		}
		return nil
	})
}

// --- per-kind extractors ---

func extractAgent(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Name                   string                        `yaml:"name"`
		Description            string                        `yaml:"description"`
		Model                  string                        `yaml:"model"`
		Effort                 string                        `yaml:"effort"`
		MaxTurns               int                           `yaml:"max-turns"`
		Mode                   string                        `yaml:"mode"`
		Tools                  []string                      `yaml:"tools"`
		DisallowedTools        []string                      `yaml:"disallowed-tools"`
		PermissionMode         string                        `yaml:"permission-mode"`
		Background             *bool                         `yaml:"background"`
		Isolation              string                        `yaml:"isolation"`
		When                   string                        `yaml:"when"`
		Memory                 string                        `yaml:"memory"`
		Color                  string                        `yaml:"color"`
		InitialPrompt          string                        `yaml:"initial-prompt"`
		Skills                 []string                      `yaml:"skills"`
		Rules                  []string                      `yaml:"rules"`
		MCP                    []string                      `yaml:"mcp"`
		Assertions             []string                      `yaml:"assertions"`
		Targets                map[string]ast.TargetOverride `yaml:"targets"`
		DisableModelInvocation *bool                         `yaml:"disable-model-invocation"`
		UserInvocable          *bool                         `yaml:"user-invocable"`
		Readonly               *bool                         `yaml:"readonly"`
	}

	body, err := parseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("claude: agent %q: %w", rel, err)
	}

	id := strings.TrimSuffix(filepath.Base(rel), ".md")
	if config.Agents == nil {
		config.Agents = make(map[string]ast.AgentConfig)
	}
	config.Agents[id] = ast.AgentConfig{
		Name:                   front.Name,
		Description:            front.Description,
		Model:                  front.Model,
		Effort:                 front.Effort,
		MaxTurns:               front.MaxTurns,
		Mode:                   front.Mode,
		Tools:                  front.Tools,
		DisallowedTools:        front.DisallowedTools,
		PermissionMode:         front.PermissionMode,
		DisableModelInvocation: front.DisableModelInvocation,
		UserInvocable:          front.UserInvocable,
		Readonly:               front.Readonly,
		Background:             front.Background,
		Isolation:              front.Isolation,
		When:                   front.When,
		Memory:                 front.Memory,
		Color:                  front.Color,
		InitialPrompt:          front.InitialPrompt,
		Skills:                 front.Skills,
		Rules:                  front.Rules,
		MCP:                    front.MCP,
		Assertions:             front.Assertions,
		Targets:                front.Targets,
		Instructions:           body,
		SourceProvider:         "claude",
	}
	return nil
}

func extractSkill(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Name                   string                        `yaml:"name"`
		Description            string                        `yaml:"description"`
		WhenToUse              string                        `yaml:"when-to-use"`
		License                string                        `yaml:"license"`
		AllowedTools           []string                      `yaml:"allowed-tools"`
		DisableModelInvocation *bool                         `yaml:"disable-model-invocation"`
		UserInvocable          *bool                         `yaml:"user-invocable"`
		ArgumentHint           string                        `yaml:"argument-hint"`
		References             []string                      `yaml:"references"`
		Scripts                []string                      `yaml:"scripts"`
		Assets                 []string                      `yaml:"assets"`
		Targets                map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := parseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("claude: skill %q: %w", rel, err)
	}

	// For DirectoryPerEntry layout: id is the directory name
	id := filepath.Base(filepath.Dir(rel))
	if config.Skills == nil {
		config.Skills = make(map[string]ast.SkillConfig)
	}
	config.Skills[id] = ast.SkillConfig{
		Name:                   front.Name,
		Description:            front.Description,
		WhenToUse:              front.WhenToUse,
		License:                front.License,
		AllowedTools:           front.AllowedTools,
		DisableModelInvocation: front.DisableModelInvocation,
		UserInvocable:          front.UserInvocable,
		ArgumentHint:           front.ArgumentHint,
		References:             front.References,
		Scripts:                front.Scripts,
		Assets:                 front.Assets,
		Targets:                front.Targets,
		Instructions:           body,
		SourceProvider:         "claude",
	}
	return nil
}

func extractRule(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Name          string                        `yaml:"name"`
		Description   string                        `yaml:"description"`
		AlwaysApply   *bool                         `yaml:"always-apply"`
		Activation    string                        `yaml:"activation"`
		Paths         []string                      `yaml:"paths"`
		ExcludeAgents []string                      `yaml:"exclude-agents"`
		Targets       map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := parseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("claude: rule %q: %w", rel, err)
	}

	// Derive the rule ID as the path relative to "rules/", with extension stripped.
	// For flat rules (rules/security.md) this is "security".
	// For nested rules (rules/cli/testing-framework.md) this is "cli/testing-framework",
	// which preserves uniqueness and mirrors the directory structure.
	rulesPrefix := "rules/"
	relFromRules := strings.TrimPrefix(filepath.ToSlash(rel), rulesPrefix)
	id := strings.TrimSuffix(relFromRules, ".md")
	if config.Rules == nil {
		config.Rules = make(map[string]ast.RuleConfig)
	}
	config.Rules[id] = ast.RuleConfig{
		Name:           front.Name,
		Description:    front.Description,
		AlwaysApply:    front.AlwaysApply,
		Activation:     front.Activation,
		Paths:          front.Paths,
		ExcludeAgents:  front.ExcludeAgents,
		Targets:        front.Targets,
		Instructions:   body,
		SourceProvider: "claude",
	}
	return nil
}

func extractWorkflow(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}

	body, err := parseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("claude: workflow %q: %w", rel, err)
	}

	id := strings.TrimSuffix(filepath.Base(rel), ".md")
	if config.Workflows == nil {
		config.Workflows = make(map[string]ast.WorkflowConfig)
	}
	config.Workflows[id] = ast.WorkflowConfig{
		Name:           front.Name,
		Description:    front.Description,
		Instructions:   body,
		SourceProvider: "claude",
	}
	return nil
}

// mcpFileWrapper is the outer JSON envelope for mcp.json.
// Claude writes {"mcpServers": {...}} as the top-level shape.
type mcpFileWrapper struct {
	MCPServers map[string]ast.MCPConfig `json:"mcpServers"`
}

func extractMCPStandalone(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var wrapper mcpFileWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("claude: mcp.json parse: %w", err)
	}
	if config.MCP == nil {
		config.MCP = make(map[string]ast.MCPConfig)
	}
	for k, v := range wrapper.MCPServers {
		v.SourceProvider = "claude"
		config.MCP[k] = v
	}
	return nil
}

func extractSettings(rel string, data []byte, config *ast.XcaffoldConfig) error {
	// Phase 1: parse into raw message map to extract embedded keys separately.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("claude: %s parse: %w", rel, err)
	}

	// Extract mcpServers → config.MCP
	if mcpRaw, ok := raw["mcpServers"]; ok {
		var servers map[string]ast.MCPConfig
		if err := json.Unmarshal(mcpRaw, &servers); err != nil {
			return fmt.Errorf("claude: %s mcpServers: %w", rel, err)
		}
		if config.MCP == nil {
			config.MCP = make(map[string]ast.MCPConfig)
		}
		for k, v := range servers {
			v.SourceProvider = "claude"
			config.MCP[k] = v
		}
	}

	// Extract hooks → config.Hooks
	if hooksRaw, ok := raw["hooks"]; ok {
		var hooks ast.HookConfig
		if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
			return fmt.Errorf("claude: %s hooks: %w", rel, err)
		}
		config.Hooks = hooks
	}

	// Extract full settings into config.Settings (overlapping keys are fine —
	// Settings keeps its own copy for provider-specific fields like model, permissions, etc.)
	var settings ast.SettingsConfig
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("claude: %s settings: %w", rel, err)
	}
	settings.SourceProvider = "claude"
	config.Settings = settings
	return nil
}

func extractMemory(rel string, data []byte, config *ast.XcaffoldConfig) error {
	// ID is the relative path from agent-memory/ prefix, with extension stripped.
	// e.g. "agent-memory/go-cli/context.md" → "go-cli/context"
	trimmed := strings.TrimPrefix(rel, "agent-memory/")
	id := strings.TrimSuffix(trimmed, filepath.Ext(trimmed))

	// Attempt to parse YAML frontmatter; fall back to raw content as instructions.
	var front struct {
		Name        string `yaml:"name"`
		Type        string `yaml:"type"`
		Description string `yaml:"description"`
		Lifecycle   string `yaml:"lifecycle"`
	}
	body, err := parseFrontmatter(data, &front)
	if err != nil {
		// Not valid frontmatter — treat entire content as instructions.
		body = string(data)
	}

	if config.Memory == nil {
		config.Memory = make(map[string]ast.MemoryConfig)
	}
	config.Memory[id] = ast.MemoryConfig{
		Name:           front.Name,
		Type:           front.Type,
		Description:    front.Description,
		Lifecycle:      front.Lifecycle,
		Instructions:   body,
		SourceProvider: "claude",
	}
	return nil
}

// --- helpers ---

// parseFrontmatter parses YAML frontmatter from markdown content.
// The body after the closing "---" delimiter is returned as a trimmed string.
// If the content has no frontmatter, the full content is returned as the body
// and v is left unmodified.
func parseFrontmatter(data []byte, v interface{}) (body string, err error) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return strings.TrimSpace(content), nil
	}
	// content[4:] skips the leading "---\n"
	parts := strings.SplitN(content[4:], "\n---", 2)
	if len(parts) < 2 {
		return strings.TrimSpace(content), nil
	}
	if err := yaml.Unmarshal([]byte(parts[0]), v); err != nil {
		return "", fmt.Errorf("frontmatter: %w", err)
	}
	// parts[1] starts with "\n" after the "---"; trim leading newline.
	return strings.TrimSpace(strings.TrimPrefix(parts[1], "\n")), nil
}

// matchGlob matches a relative path against a glob pattern.
// Supports "*" (any single segment) and "**" (any number of segments).
func matchGlob(pattern, rel string) bool {
	patParts := strings.Split(pattern, "/")
	relParts := strings.Split(rel, "/")
	return matchSegments(patParts, relParts)
}

func matchSegments(pat, rel []string) bool {
	for len(pat) > 0 && len(rel) > 0 {
		switch pat[0] {
		case "**":
			// Try consuming zero or more rel segments.
			for i := 0; i <= len(rel); i++ {
				if matchSegments(pat[1:], rel[i:]) {
					return true
				}
			}
			return false
		default:
			ok, err := filepath.Match(pat[0], rel[0])
			if err != nil || !ok {
				return false
			}
			pat, rel = pat[1:], rel[1:]
		}
	}
	return len(pat) == 0 && len(rel) == 0
}

// readFile reads a file from disk.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
