package antigravity2

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

// Antigravity2Importer imports resources from a .agents/ directory tree.
// It extends the v1 importer with hooks, agent.json, plugin, and
// workspace-level mcp_config.json support.
type Antigravity2Importer struct {
	importer.BaseImporter
}

// NewImporter returns a new Antigravity2Importer.
func NewImporter() *Antigravity2Importer {
	return &Antigravity2Importer{
		BaseImporter: importer.BaseImporter{
			ProviderName: "antigravity2",
			Dir:          ".agents",
		},
	}
}

// antigravity2Mappings maps path patterns to AST kinds. First match wins.
var antigravity2Mappings = []importer.KindMapping{
	{Pattern: "agents/*/agent.json", Kind: importer.KindAgent, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/SKILL.md", Kind: importer.KindSkill, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/references/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/scripts/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/examples/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "rules/*.md", Kind: importer.KindRule, Layout: importer.FlatFile},
	{Pattern: "knowledge/*.md", Kind: importer.KindMemory, Layout: importer.FlatFile},
	{Pattern: "hooks.json", Kind: importer.KindHook, Layout: importer.StandaloneJSON},
	{Pattern: "mcp_config.json", Kind: importer.KindMCP, Layout: importer.StandaloneJSON},
	{Pattern: "workflows/*.md", Kind: importer.KindWorkflow, Layout: importer.FlatFile},
}

// Classify returns the Kind and Layout for a given relative path.
func (a *Antigravity2Importer) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range antigravity2Mappings {
		if importer.MatchGlob(m.Pattern, rel) {
			return m.Kind, m.Layout
		}
	}
	return importer.KindUnknown, importer.LayoutUnknown
}

// Extract reads a single file and populates the appropriate section of config.
// rel is relative to InputDir().
func (a *Antigravity2Importer) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	rel = filepath.ToSlash(filepath.Clean(rel))
	kind, _ := a.Classify(rel, false)

	switch kind {
	case importer.KindAgent:
		return extractAgentJSON(rel, data, config)
	case importer.KindSkill:
		return extractSkill2(rel, data, config)
	case importer.KindSkillAsset:
		return importer.DefaultExtractSkillAsset(rel, data, config)
	case importer.KindHook:
		return extractHooksJSON(rel, data, config)
	case importer.KindRule:
		return importer.DefaultExtractRule(rel, data, a.Provider(), config)
	case importer.KindMemory:
		return extractMemory2(rel, data, config)
	case importer.KindMCP:
		return extractMCPConfig(rel, data, config)
	case importer.KindWorkflow:
		return extractWorkflow2(rel, data, config)
	default:
		return fmt.Errorf("antigravity2: no extractor for kind %q at path %q", kind, rel)
	}
}

// Import walks dir, classifies each entry, extracts classified files, and
// appends unclassified files to config.ProviderExtras["antigravity2"].
func (a *Antigravity2Importer) Import(dir string, config *ast.XcaffoldConfig) error {
	a.Warnings = a.Warnings[:0]
	return importer.WalkProviderDir(dir, func(rel string, data []byte) error {
		kind, _ := a.Classify(rel, false)
		if kind == importer.KindUnknown {
			return nil
		}
		if err := a.Extract(rel, data, config); err != nil {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras[a.Provider()] == nil {
				config.ProviderExtras[a.Provider()] = make(map[string][]byte)
			}
			config.ProviderExtras[a.Provider()][rel] = data
			a.AppendWarning(fmt.Sprintf("skipped %q: %v", rel, err))
		}
		return nil
	})
}

// --- per-kind extractors ---

// extractAgentJSON parses an agents/<id>/agent.json file into an AgentConfig.
func extractAgentJSON(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var obj agentJSON
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("antigravity2: agent %q: %w", rel, err)
	}

	// id is the directory name containing agent.json
	parts := strings.Split(filepath.ToSlash(filepath.Clean(rel)), "/")
	var id string
	if len(parts) >= 2 && parts[0] == "agents" {
		id = parts[1]
	} else {
		id = strings.TrimSuffix(filepath.Base(rel), ".json")
	}

	if config.Agents == nil {
		config.Agents = make(map[string]ast.AgentConfig)
	}
	config.Agents[id] = ast.AgentConfig{
		Name:            obj.Name,
		Description:     obj.Description,
		Model:           obj.Model,
		MaxTurns:        obj.MaxTurns,
		Tools:           ast.ClearableList{Values: obj.Tools},
		DisallowedTools: ast.ClearableList{Values: obj.DisabledTools},
		Readonly:        obj.Readonly,
		UserInvocable:   obj.UserInvocable,
		InitialPrompt:   obj.InitialPrompt,
		Skills:          ast.ClearableList{Values: obj.Skills},
		Rules:           ast.ClearableList{Values: obj.Rules},
		Body:            obj.Instructions,
		SourceProvider:  "antigravity2",
	}
	return nil
}

// extractSkill2 parses a skills/<id>/SKILL.md file into a SkillConfig.
// Supports agentskills.io extended fields: when-to-use, argument-hint, allowed-tools.
func extractSkill2(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Name          string                        `yaml:"name"`
		Description   string                        `yaml:"description"`
		WhenToUse     string                        `yaml:"when-to-use"`
		License       string                        `yaml:"license"`
		AllowedTools  []string                      `yaml:"allowed-tools"`
		UserInvocable *bool                         `yaml:"user-invocable"`
		ArgumentHint  string                        `yaml:"argument-hint"`
		Targets       map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := importer.ParseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("antigravity2: skill %q: %w", rel, err)
	}

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
		Name:           front.Name,
		Description:    front.Description,
		WhenToUse:      front.WhenToUse,
		License:        front.License,
		AllowedTools:   ast.ClearableList{Values: front.AllowedTools},
		UserInvocable:  front.UserInvocable,
		ArgumentHint:   front.ArgumentHint,
		Targets:        front.Targets,
		Body:           body,
		SourceProvider: "antigravity2",
	}
	return nil
}

// extractHooksJSON parses the hooks.json file and stores events under a
// "default" named hook block in config.Hooks, consistent with how other
// providers wrap flat event configs into NamedHookConfig.
func extractHooksJSON(_ string, data []byte, config *ast.XcaffoldConfig) error {
	var hooks ast.HookConfig
	if err := json.Unmarshal(data, &hooks); err != nil {
		return fmt.Errorf("antigravity2: hooks.json parse: %w", err)
	}
	if len(hooks) == 0 {
		return nil
	}
	if config.Hooks == nil {
		config.Hooks = make(map[string]ast.NamedHookConfig)
	}
	config.Hooks["default"] = ast.NamedHookConfig{Name: "default", Events: hooks}
	return nil
}

// extractMCPConfig parses the workspace-level mcp_config.json file.
// Antigravity 2.0 uses {"mcpServers": {...}} with serverUrl and disabledTools support.
func extractMCPConfig(_ string, data []byte, config *ast.XcaffoldConfig) error {
	var wrapper struct {
		MCPServers map[string]mcpServerEntry `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("antigravity2: mcp_config.json parse: %w", err)
	}
	if config.MCP == nil {
		config.MCP = make(map[string]ast.MCPConfig)
	}
	for k, v := range wrapper.MCPServers {
		// v1.0.5+ supports both serverUrl and url; prefer serverUrl when both present.
		serverURL := v.ServerURL
		if serverURL == "" {
			serverURL = v.URL
		}
		config.MCP[k] = ast.MCPConfig{
			Command:        v.Command,
			Args:           v.Args,
			Env:            v.Env,
			URL:            serverURL,
			DisabledTools:  v.DisabledTools,
			SourceProvider: "antigravity2",
		}
	}
	return nil
}

// extractMemory2 parses a knowledge/<id>.md file into a MemoryConfig.
// The file is expected to have YAML frontmatter with title and description fields,
// followed by the memory content in the body.
func extractMemory2(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Title       string   `yaml:"title"`
		Description string   `yaml:"description"`
		Tags        []string `yaml:"tags"`
	}

	body, err := importer.ParseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("antigravity2: memory %q: %w", rel, err)
	}

	id := strings.TrimSuffix(filepath.Base(rel), ".md")
	name := front.Title
	if name == "" {
		name = id
	}

	if config.Memory == nil {
		config.Memory = make(map[string]ast.MemoryConfig)
	}
	config.Memory[id] = ast.MemoryConfig{
		Name:        name,
		Description: front.Description,
		Content:     body,
	}
	return nil
}

// extractWorkflow2 parses a workflows/<id>.md file into a WorkflowConfig.
func extractWorkflow2(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Name        string                        `yaml:"name"`
		Description string                        `yaml:"description"`
		Steps       []ast.WorkflowStep            `yaml:"steps"`
		Targets     map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := importer.ParseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("antigravity2: workflow %q: %w", rel, err)
	}

	steps := front.Steps
	trimmedBody := strings.TrimSpace(body)
	if len(steps) == 0 && trimmedBody != "" {
		steps = []ast.WorkflowStep{{
			Name:         "main",
			Instructions: trimmedBody,
		}}
	}

	id := strings.TrimSuffix(filepath.Base(rel), ".md")
	if config.Workflows == nil {
		config.Workflows = make(map[string]ast.WorkflowConfig)
	}
	config.Workflows[id] = ast.WorkflowConfig{
		Name:           front.Name,
		Description:    front.Description,
		Steps:          steps,
		Targets:        front.Targets,
		SourceProvider: "antigravity2",
	}
	return nil
}
