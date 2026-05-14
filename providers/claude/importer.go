package claude

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

// ClaudeImporter imports resources from a .claude/ directory tree.
// Warnings accumulates non-fatal per-file extraction errors encountered during Import().
// Callers may inspect Warnings after Import() returns to surface skipped files.
type ClaudeImporter struct {
	importer.BaseImporter
}

// NewImporter returns a new ClaudeImporter.
func NewImporter() *ClaudeImporter {
	return &ClaudeImporter{
		BaseImporter: importer.BaseImporter{
			ProviderName: "claude",
			Dir:          ".claude",
		},
	}
}

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
		return importer.DefaultExtractSkillAsset(rel, data, config)
	case importer.KindHookScript:
		return importer.DefaultExtractHookScript(rel, data, config)
	case importer.KindRule:
		return importer.DefaultExtractRule(rel, data, c.Provider(), config)
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
			// Store unclassified files for later inspection
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras[c.Provider()] == nil {
				config.ProviderExtras[c.Provider()] = make(map[string][]byte)
			}
			config.ProviderExtras[c.Provider()][rel] = data
			return nil
		}
		if extractErr := c.Extract(rel, data, config); extractErr != nil {
			// Store extraction error in ProviderExtras for later review
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras[c.Provider()] == nil {
				config.ProviderExtras[c.Provider()] = make(map[string][]byte)
			}
			config.ProviderExtras[c.Provider()][rel] = data
			c.AppendWarning(fmt.Sprintf("skipped %q: %v", rel, extractErr))
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Cross-reference memory against known agents: warn if the parent agent is absent,
	// but keep the entry so it can still be imported to xcaf/agents/<id>/memory/.
	for memPath := range config.Memory {
		agentID := strings.SplitN(memPath, "/", 2)[0]
		if len(config.Agents) > 0 {
			if _, ok := config.Agents[agentID]; !ok {
				c.AppendWarning(fmt.Sprintf("memory for agent %q has no matching agent definition; will import to xcaf/agents/%s/memory/", agentID, agentID))
			}
		}
	}

	return nil
}

// --- per-kind extractors ---

// agentFrontmatter is the frontmatter schema for Claude agent files.
type agentFrontmatter struct {
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
	Hooks                  ast.HookConfig                `yaml:"hooks"`
	MCPServers             map[string]ast.MCPConfig      `yaml:"mcp-servers"`
	DisableModelInvocation *bool                         `yaml:"disable-model-invocation"`
	UserInvocable          *bool                         `yaml:"user-invocable"`
	Readonly               *bool                         `yaml:"readonly"`
}

// buildAgentConfig constructs an AST AgentConfig from frontmatter and body.
func buildAgentConfig(front *agentFrontmatter, body string) ast.AgentConfig {
	return ast.AgentConfig{
		Name:                   front.Name,
		Description:            front.Description,
		Model:                  front.Model,
		Effort:                 front.Effort,
		MaxTurns:               intPtrIfNonZero(front.MaxTurns),
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
		Hooks:                  front.Hooks,
		MCPServers:             front.MCPServers,
		Body:                   body,
		SourceProvider:         "claude",
	}
}

func extractAgent(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front agentFrontmatter
	body, err := importer.ParseFrontmatterLenient(data, &front)
	if err != nil {
		return fmt.Errorf("claude: agent %q: %w", rel, err)
	}

	id := strings.TrimSuffix(filepath.Base(rel), ".md")
	if config.Agents == nil {
		config.Agents = make(map[string]ast.AgentConfig)
	}
	config.Agents[id] = buildAgentConfig(&front, body)
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
		AllowedTools:           ast.ClearableList{Values: front.AllowedTools},
		DisableModelInvocation: front.DisableModelInvocation,
		UserInvocable:          front.UserInvocable,
		ArgumentHint:           front.ArgumentHint,
		Targets:                front.Targets,
		Body:                   body,
		SourceProvider:         "claude",
	}
	return nil
}

func extractWorkflow(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}

	_, err := importer.ParseFrontmatterLenient(data, &front)
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

	// Skip auto-generated memory index files.
	if strings.HasSuffix(trimmed, "/MEMORY.md") || trimmed == "MEMORY.md" {
		return nil
	}

	id := strings.TrimSuffix(trimmed, filepath.Ext(trimmed))

	// Attempt to parse YAML frontmatter; fall back to raw content as instructions.
	// Type and Lifecycle were removed from MemoryConfig in the agent-scoped
	// memory refactor; only Name and Description survive import.
	var front struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
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
		Description:    front.Description,
		Content:        body,
		SourceProvider: "claude",
	}
	return nil
}

func intPtrIfNonZero(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}
