package cursor

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

// CursorImporter imports resources from a .cursor/ directory tree.
// Warnings accumulates non-fatal per-file extraction errors encountered during Import().
// Callers may inspect Warnings after Import() returns to surface skipped files.
type CursorImporter struct {
	Warnings     []string
	visitedLinks map[string]bool
}

// New returns a new CursorImporter.
func New() *CursorImporter {
	return &CursorImporter{}
}

// GetWarnings returns non-fatal per-file extraction errors collected during the last Import() call.
func (c *CursorImporter) GetWarnings() []string { return c.Warnings }

// Provider returns the canonical provider name.
func (c *CursorImporter) Provider() string { return "cursor" }

// InputDir returns the root directory relative to the workspace root.
func (c *CursorImporter) InputDir() string { return ".cursor" }

// cursorMappings maps path patterns to AST kinds. First match wins.
var cursorMappings = []importer.KindMapping{
	{Pattern: "agents/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "skills/*/SKILL.md", Kind: importer.KindSkill, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/references/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/scripts/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/assets/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "rules/*.mdc", Kind: importer.KindRule, Layout: importer.FlatFile, Extension: ".mdc"},
	{Pattern: "mcp.json", Kind: importer.KindMCP, Layout: importer.StandaloneJSON},
	{Pattern: "hooks.json", Kind: importer.KindHook, Layout: importer.StandaloneJSON},
}

// Classify returns the Kind and Layout for a given relative path.
// rel is relative to InputDir(). First matching entry in cursorMappings wins.
func (c *CursorImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range cursorMappings {
		if matchGlob(m.Pattern, rel) {
			return m.Kind, m.Layout
		}
	}
	return importer.KindUnknown, importer.LayoutUnknown
}

// Extract reads a single file and populates the appropriate section of config.
// rel is relative to InputDir().
func (c *CursorImporter) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	rel = filepath.ToSlash(filepath.Clean(rel))
	kind, _ := c.Classify(rel, false)

	switch kind {
	case importer.KindAgent:
		return extractAgent(rel, data, config)
	case importer.KindSkill:
		return extractSkill(rel, data, config)
	case importer.KindSkillAsset:
		return extractSkillAsset(rel, data, config)
	case importer.KindRule:
		return extractRule(rel, data, config)
	case importer.KindMCP:
		return extractMCPStandalone(rel, data, config)
	case importer.KindHook:
		return extractHooksStandalone(rel, data, config)
	default:
		return fmt.Errorf("cursor: no extractor for kind %q at path %q", kind, rel)
	}
}

// Import walks dir, classifies each entry, extracts classified files, and
// appends unclassified files to config.ProviderExtras["cursor"].
//
// Extraction errors for individual files are non-fatal: they are collected in
// c.Warnings and the walk continues. Only I/O errors (unreadable directory or
// file) abort the walk.
func (c *CursorImporter) Import(dir string, config *ast.XcaffoldConfig) error {
	c.Warnings = c.Warnings[:0]
	c.visitedLinks = make(map[string]bool)
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Symlinks to directories: resolve and walk if target is a directory.
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
			return fmt.Errorf("cursor: rel path: %w", err)
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
			if config.ProviderExtras["cursor"] == nil {
				config.ProviderExtras["cursor"] = make(map[string][]byte)
			}
			config.ProviderExtras["cursor"][rel] = data
			return nil
		}

		if extractErr := c.Extract(rel, data, config); extractErr != nil {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["cursor"] == nil {
				config.ProviderExtras["cursor"] = make(map[string][]byte)
			}
			config.ProviderExtras["cursor"][rel] = data
			c.Warnings = append(c.Warnings, fmt.Sprintf("skipped %q: %v", rel, extractErr))
		}
		return nil
	})
}

// importSymlinkedDir walks a symlinked directory and imports its contents.
func (c *CursorImporter) importSymlinkedDir(symlinkPath, importRoot string, config *ast.XcaffoldConfig) error {
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
			if config.ProviderExtras["cursor"] == nil {
				config.ProviderExtras["cursor"] = make(map[string][]byte)
			}
			config.ProviderExtras["cursor"][rel] = data
			return nil
		}

		if extractErr := c.Extract(rel, data, config); extractErr != nil {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["cursor"] == nil {
				config.ProviderExtras["cursor"] = make(map[string][]byte)
			}
			config.ProviderExtras["cursor"][rel] = data
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
		return fmt.Errorf("cursor: agent %q: %w", rel, err)
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
		SourceProvider:         "cursor",
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
		return fmt.Errorf("cursor: skill %q: %w", rel, err)
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
		SourceProvider:         "cursor",
	}
	return nil
}

// extractSkillAsset records a skill companion file (references/*, scripts/*, assets/*)
// in the parent skill's corresponding slice. rel is the path relative to InputDir()
// and must match the pattern "skills/<id>/<sub>/<file>".
func extractSkillAsset(rel string, _ []byte, config *ast.XcaffoldConfig) error {
	parts := strings.SplitN(rel, "/", 4)
	if len(parts) < 4 {
		return fmt.Errorf("cursor: skill asset path too short: %q", rel)
	}
	skillID := parts[1]
	subDir := parts[2]
	relWithinSkill := subDir + "/" + parts[3]

	if config.Skills == nil {
		config.Skills = make(map[string]ast.SkillConfig)
	}
	skill := config.Skills[skillID]
	switch subDir {
	case "references":
		skill.References = appendUnique(skill.References, relWithinSkill)
	case "scripts":
		skill.Scripts = appendUnique(skill.Scripts, relWithinSkill)
	case "assets":
		skill.Assets = appendUnique(skill.Assets, relWithinSkill)
	}
	config.Skills[skillID] = skill
	return nil
}

// appendUnique appends s to slice only if it is not already present.
func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

func extractRule(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Name          string                        `yaml:"name"`
		Description   string                        `yaml:"description"`
		AlwaysApply   *bool                         `yaml:"always-apply"`
		Activation    string                        `yaml:"activation"`
		Paths         []string                      `yaml:"paths"`
		Globs         []string                      `yaml:"globs"`
		ExcludeAgents []string                      `yaml:"exclude-agents"`
		Targets       map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := parseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("cursor: rule %q: %w", rel, err)
	}

	// Cursor uses globs: as the path-gating field; map it to the canonical Paths field.
	// If both paths: and globs: are present, merge them (globs: takes precedence by appending).
	paths := front.Paths
	if len(front.Globs) > 0 {
		paths = append(paths, front.Globs...)
	}

	// Strip .mdc extension from rule id
	base := filepath.Base(rel)
	id := strings.TrimSuffix(base, ".mdc")

	if config.Rules == nil {
		config.Rules = make(map[string]ast.RuleConfig)
	}
	config.Rules[id] = ast.RuleConfig{
		Name:           front.Name,
		Description:    front.Description,
		AlwaysApply:    front.AlwaysApply,
		Activation:     front.Activation,
		Paths:          paths,
		ExcludeAgents:  front.ExcludeAgents,
		Targets:        front.Targets,
		Instructions:   body,
		SourceProvider: "cursor",
	}
	return nil
}

// mcpFileWrapper is the outer JSON envelope for mcp.json.
type mcpFileWrapper struct {
	MCPServers map[string]ast.MCPConfig `json:"mcpServers"`
}

func extractMCPStandalone(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var wrapper mcpFileWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("cursor: mcp.json parse: %w", err)
	}
	if config.MCP == nil {
		config.MCP = make(map[string]ast.MCPConfig)
	}
	for k, v := range wrapper.MCPServers {
		v.SourceProvider = "cursor"
		config.MCP[k] = v
	}
	return nil
}

func extractHooksStandalone(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var hooks ast.HookConfig
	if err := json.Unmarshal(data, &hooks); err != nil {
		return fmt.Errorf("cursor: hooks.json parse: %w", err)
	}
	config.Hooks = map[string]ast.NamedHookConfig{"default": {Name: "default", Events: hooks}}
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
