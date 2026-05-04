package gemini

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

// GeminiImporter imports resources from a .gemini/ directory tree.
// Warnings accumulates non-fatal per-file extraction errors encountered during Import().
// Callers may inspect Warnings after Import() returns to surface skipped files.
type GeminiImporter struct {
	Warnings []string
}

// NewImporter returns a new GeminiImporter.
func NewImporter() *GeminiImporter {
	return &GeminiImporter{}
}

// GetWarnings returns non-fatal per-file extraction errors collected during the last Import() call.
func (g *GeminiImporter) GetWarnings() []string { return g.Warnings }

// Provider returns the canonical provider name.
func (g *GeminiImporter) Provider() string { return "gemini" }

// InputDir returns the root directory relative to the workspace root.
func (g *GeminiImporter) InputDir() string { return ".gemini" }

// geminiMappings maps path patterns to AST kinds. First match wins.
// Skills use DirectoryPerEntry layout: skills/<id>/SKILL.md plus assets.
// settings.json appears first for the container-level KindSettings match;
// embedded mcpServers and hooks keys are handled inside Extract().
var geminiMappings = []importer.KindMapping{
	{Pattern: "hooks/*.sh", Kind: importer.KindHookScript, Layout: importer.FlatFile},
	{Pattern: "agents/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "skills/*/SKILL.md", Kind: importer.KindSkill, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/references/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/scripts/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/assets/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "rules/*.md", Kind: importer.KindRule, Layout: importer.FlatFile},
	// settings.json is a container file holding settings, mcpServers, AND hooks.
	// Classify returns KindSettings (first match); extractSettings() handles the
	// two-phase decomposition of mcpServers → config.MCP and hooks → config.Hooks.
	{Pattern: "settings.json", Kind: importer.KindSettings, Layout: importer.EmbeddedJSONKey, JSONKey: ""},
	{Pattern: "hooks/**", Kind: importer.KindHookScript, Layout: importer.FlatFile},
}

// Classify returns the Kind and Layout for a given relative path.
// rel is relative to InputDir(). First matching entry in geminiMappings wins.
func (g *GeminiImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range geminiMappings {
		if importer.MatchGlob(m.Pattern, rel) {
			return m.Kind, m.Layout
		}
	}
	return importer.KindUnknown, importer.LayoutUnknown
}

// Extract reads a single file and populates the appropriate section of config.
// rel is relative to InputDir().
func (g *GeminiImporter) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	rel = filepath.ToSlash(filepath.Clean(rel))
	kind, _ := g.Classify(rel, false)

	switch kind {
	case importer.KindAgent:
		return extractAgent(rel, data, config)
	case importer.KindSkill:
		return extractSkill(rel, data, config)
	case importer.KindHookScript:
		return importer.ExtractHookScript(rel, data, config)
	case importer.KindRule:
		return extractRule(rel, data, config)
	case importer.KindSettings:
		return extractSettings(rel, data, config)
	default:
		return fmt.Errorf("gemini: no extractor for kind %q at path %q", kind, rel)
	}
}

// Import walks dir, classifies each entry, extracts classified files, and
// appends unclassified files to config.ProviderExtras["gemini"].
//
// Extraction errors for individual files are non-fatal: they are collected in
// g.Warnings and the walk continues. Only I/O errors (unreadable directory or
// file) abort the walk.
func (g *GeminiImporter) Import(dir string, config *ast.XcaffoldConfig) error {
	g.Warnings = g.Warnings[:0]
	return importer.WalkProviderDir(dir, func(rel string, data []byte) error {
		kind, _ := g.Classify(rel, false)
		if kind == importer.KindUnknown {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["gemini"] == nil {
				config.ProviderExtras["gemini"] = make(map[string][]byte)
			}
			config.ProviderExtras["gemini"][rel] = data
			return nil
		}
		if extractErr := g.Extract(rel, data, config); extractErr != nil {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["gemini"] == nil {
				config.ProviderExtras["gemini"] = make(map[string][]byte)
			}
			config.ProviderExtras["gemini"][rel] = data
			g.Warnings = append(g.Warnings, fmt.Sprintf("skipped %q: %v", rel, extractErr))
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
		return fmt.Errorf("gemini: agent %q: %w", rel, err)
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
		Tools:                  ast.ClearableList{Values: front.Tools},
		DisallowedTools:        ast.ClearableList{Values: front.DisallowedTools},
		PermissionMode:         front.PermissionMode,
		DisableModelInvocation: front.DisableModelInvocation,
		UserInvocable:          front.UserInvocable,
		Readonly:               front.Readonly,
		Background:             front.Background,
		Isolation:              front.Isolation,
		Memory:                 ast.NewFlexStringSlice(front.Memory),
		Color:                  front.Color,
		InitialPrompt:          front.InitialPrompt,
		Skills:                 ast.ClearableList{Values: front.Skills},
		Rules:                  ast.ClearableList{Values: front.Rules},
		MCP:                    ast.ClearableList{Values: front.MCP},
		Assertions:             ast.ClearableList{Values: front.Assertions},
		Targets:                front.Targets,
		Body:                   body,
		SourceProvider:         "gemini",
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
		return fmt.Errorf("gemini: skill %q: %w", rel, err)
	}

	// DirectoryPerEntry layout: id is the directory name (parent of SKILL.md)
	// rel is "skills/<id>/SKILL.md", so extract the directory name.
	parts := strings.Split(filepath.ToSlash(filepath.Clean(rel)), "/")
	var id string
	if len(parts) >= 2 && parts[0] == "skills" {
		id = parts[1]
	} else {
		id = strings.TrimSuffix(filepath.Base(rel), ".md")
	}

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
		SourceProvider:         "gemini",
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

	body, err := importer.ParseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("gemini: rule %q: %w", rel, err)
	}

	id := strings.TrimSuffix(filepath.Base(rel), ".md")
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
		Body:           body,
		SourceProvider: "gemini",
	}
	return nil
}

func extractSettings(rel string, data []byte, config *ast.XcaffoldConfig) error {
	// Phase 1: parse into raw message map to extract embedded keys separately.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("gemini: %s parse: %w", rel, err)
	}

	// Extract mcpServers → config.MCP
	if mcpRaw, ok := raw["mcpServers"]; ok {
		var servers map[string]ast.MCPConfig
		if err := json.Unmarshal(mcpRaw, &servers); err != nil {
			return fmt.Errorf("gemini: %s mcpServers: %w", rel, err)
		}
		if config.MCP == nil {
			config.MCP = make(map[string]ast.MCPConfig)
		}
		for k, v := range servers {
			v.SourceProvider = "gemini"
			config.MCP[k] = v
		}
	}

	// Extract hooks → config.Hooks
	if hooksRaw, ok := raw["hooks"]; ok {
		var hooks ast.HookConfig
		if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
			return fmt.Errorf("gemini: %s hooks: %w", rel, err)
		}
		config.Hooks = map[string]ast.NamedHookConfig{"default": {Name: "default", Events: hooks}}
	}

	// Extract full settings into config.Settings
	var settings ast.SettingsConfig
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("gemini: %s settings: %w", rel, err)
	}
	settings.SourceProvider = "gemini"
	config.Settings = map[string]ast.SettingsConfig{"default": settings}
	return nil
}
