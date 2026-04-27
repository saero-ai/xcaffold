package cursor

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

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
	Warnings []string
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
	{Pattern: "hooks/*.sh", Kind: importer.KindHookScript, Layout: importer.FlatFile},
	{Pattern: "agents/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "skills/*/SKILL.md", Kind: importer.KindSkill, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/references/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/scripts/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/assets/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "rules/*.mdc", Kind: importer.KindRule, Layout: importer.FlatFile, Extension: ".mdc"},
	{Pattern: "mcp.json", Kind: importer.KindMCP, Layout: importer.StandaloneJSON},
	{Pattern: "hooks.json", Kind: importer.KindHook, Layout: importer.StandaloneJSON},
	{Pattern: "hooks/**", Kind: importer.KindHookScript, Layout: importer.FlatFile},
}

// Classify returns the Kind and Layout for a given relative path.
// rel is relative to InputDir(). First matching entry in cursorMappings wins.
func (c *CursorImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range cursorMappings {
		if importer.MatchGlob(m.Pattern, rel) {
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
	case importer.KindHookScript:
		return importer.ExtractHookScript(rel, data, config)
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
	return importer.WalkProviderDir(dir, func(rel string, data []byte) error {
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

	body, err := importer.ParseFrontmatter(data, &front)
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
		Body:                   body,
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

	body, err := importer.ParseFrontmatter(data, &front)
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
		Body:                   body,
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
		skill.References = importer.AppendUnique(skill.References, relWithinSkill)
	case "scripts":
		skill.Scripts = importer.AppendUnique(skill.Scripts, relWithinSkill)
	case "assets":
		skill.Assets = importer.AppendUnique(skill.Assets, relWithinSkill)
	}
	config.Skills[skillID] = skill
	return nil
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

	body, err := importer.ParseFrontmatter(data, &front)
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
		Body:           body,
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
