package claude

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

// ClaudeImporter imports resources from a .claude/ directory tree.
// Warnings accumulates non-fatal per-file extraction errors encountered during Import().
// Callers may inspect Warnings after Import() returns to surface skipped files.
type ClaudeImporter struct {
	Warnings []string
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
// Skill companion patterns (references/**, scripts/**, assets/**) must appear
// before the catch-all agent-memory/** so they are matched first.
var claudeMappings = []importer.KindMapping{
	{Pattern: "hooks/*.sh", Kind: importer.KindHookScript, Layout: importer.FlatFile},
	{Pattern: "agents/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "skills/*/SKILL.md", Kind: importer.KindSkill, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/references/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/scripts/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/assets/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "rules/**/*.md", Kind: importer.KindRule, Layout: importer.FlatFile},
	{Pattern: "workflows/*.md", Kind: importer.KindWorkflow, Layout: importer.FlatFile},
	{Pattern: "mcp.json", Kind: importer.KindMCP, Layout: importer.StandaloneJSON},
	{Pattern: "settings.json", Kind: importer.KindSettings, Layout: importer.EmbeddedJSONKey},
	{Pattern: "settings.local.json", Kind: importer.KindSettings, Layout: importer.StandaloneJSON},
	{Pattern: "hooks/**", Kind: importer.KindHookScript, Layout: importer.FlatFile},
	{Pattern: "agent-memory/**", Kind: importer.KindMemory, Layout: importer.FlatFile},
}

// Classify returns the Kind and Layout for a given relative path.
// rel is relative to InputDir(). First matching entry in claudeMappings wins.
func (c *ClaudeImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range claudeMappings {
		if importer.MatchGlob(m.Pattern, rel) {
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
	case importer.KindSkillAsset:
		return extractSkillAsset(rel, data, config)
	case importer.KindHookScript:
		return importer.ExtractHookScript(rel, data, config)
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
	err := importer.WalkProviderDir(dir, func(rel string, data []byte) error {
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
	if err != nil {
		return err
	}

	// Cross-reference memory against known agents (B-12).
	for memPath := range config.Memory {
		agentID := strings.SplitN(memPath, "/", 2)[0]
		if len(config.Agents) > 0 {
			if _, ok := config.Agents[agentID]; !ok {
				delete(config.Memory, memPath)
				c.Warnings = append(c.Warnings, fmt.Sprintf("skipped %q: agent %q not found in xcf/agents", "agent-memory/"+memPath, agentID))
			}
		}
	}

	return nil
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

	body, err := importer.ParseFrontmatterLenient(data, &front)
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

	body, err := importer.ParseFrontmatterLenient(data, &front)
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

// extractSkillAsset records a skill companion file (references/*, scripts/*, assets/*)
// in the parent skill's corresponding slice. rel is the path relative to InputDir()
// and must match the pattern "skills/<id>/<sub>/<file>".
// If the parent skill does not yet exist in config, it is created with a minimal entry
// so that the path reference is preserved even when SKILL.md is walked after the asset.
func extractSkillAsset(rel string, _ []byte, config *ast.XcaffoldConfig) error {
	// rel looks like: skills/tdd/references/schema.sql
	parts := strings.SplitN(rel, "/", 4)
	if len(parts) < 4 {
		return fmt.Errorf("claude: skill asset path too short: %q", rel)
	}
	skillID := parts[1]
	subDir := parts[2]                        // "references", "scripts", or "assets"
	relWithinSkill := subDir + "/" + parts[3] // e.g. "references/schema.sql"

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
		ExcludeAgents []string                      `yaml:"exclude-agents"`
		Targets       map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := importer.ParseFrontmatterLenient(data, &front)
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

	body, err := importer.ParseFrontmatterLenient(data, &front)
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
		config.Hooks = map[string]ast.NamedHookConfig{"default": {Name: "default", Events: hooks}}
	}

	// Extract full settings into config.Settings (overlapping keys are fine —
	// Settings keeps its own copy for provider-specific fields like model, permissions, etc.)
	var settings ast.SettingsConfig
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("claude: %s settings: %w", rel, err)
	}
	settings.SourceProvider = "claude"
	config.Settings = map[string]ast.SettingsConfig{"default": settings}
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
	body, err := importer.ParseFrontmatterLenient(data, &front)
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
