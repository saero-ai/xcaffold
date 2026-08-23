package antigravity

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

// AntigravityImporter imports resources from a .agents/ directory tree.
// Warnings accumulates non-fatal per-file extraction errors encountered during Import().
type AntigravityImporter struct {
	importer.BaseImporter
}

// NewImporter returns a new AntigravityImporter.
func NewImporter() *AntigravityImporter {
	return &AntigravityImporter{
		BaseImporter: importer.BaseImporter{
			ProviderName: "antigravity",
			Dir:          ".agents",
		},
	}
}

// antigravityMappings maps path patterns to AST kinds. First match wins.
// Per strict single-pattern decision, agents are imported from agents/*.md.
var antigravityMappings = []importer.KindMapping{
	{Pattern: "agents/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "skills/*/SKILL.md", Kind: importer.KindSkill, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/references/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/scripts/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/examples/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "skills/*/resources/**", Kind: importer.KindSkillAsset, Layout: importer.DirectoryPerEntry},
	{Pattern: "rules/*.md", Kind: importer.KindRule, Layout: importer.FlatFile},
	{Pattern: "hooks.json", Kind: importer.KindHook, Layout: importer.StandaloneJSON},
	{Pattern: "mcp_config.json", Kind: importer.KindMCP, Layout: importer.StandaloneJSON},
	{Pattern: "workflows/*.md", Kind: importer.KindWorkflow, Layout: importer.FlatFile},
}

// Classify returns the Kind and Layout for a given relative path.
// rel is relative to InputDir(). First matching entry in antigravityMappings wins.
func (a *AntigravityImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range antigravityMappings {
		if importer.MatchGlob(m.Pattern, rel) {
			return m.Kind, m.Layout
		}
	}
	return importer.KindUnknown, importer.LayoutUnknown
}

// Extract reads a single file and populates the appropriate section of config.
// rel is relative to InputDir().
func (a *AntigravityImporter) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	rel = filepath.ToSlash(filepath.Clean(rel))
	kind, _ := a.Classify(rel, false)

	switch kind {
	case importer.KindAgent:
		return extractAgent(rel, data, config)
	case importer.KindSkill:
		return extractSkill(rel, data, config)
	case importer.KindSkillAsset:
		return importer.DefaultExtractSkillAsset(rel, data, config)
	case importer.KindHookScript:
		return importer.DefaultExtractHookScript(rel, data, config)
	case importer.KindHook:
		return extractHooksJSON(rel, data, config)
	case importer.KindRule:
		return importer.DefaultExtractRule(rel, data, a.Provider(), config)
	case importer.KindMCP:
		return extractMCPConfig(rel, data, config)
	case importer.KindWorkflow:
		return extractWorkflow(rel, data, config)
	default:
		return fmt.Errorf("antigravity: no extractor for kind %q at path %q", kind, rel)
	}
}

// Import walks dir, classifies each entry, extracts classified files, and
// appends unclassified files to config.ProviderExtras["antigravity"].
func (a *AntigravityImporter) Import(dir string, config *ast.XcaffoldConfig) error {
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

// agentFrontmatter is the frontmatter schema for Antigravity agent files.
type agentFrontmatter struct {
	Name                   string                        `yaml:"name"`
	Description            string                        `yaml:"description"`
	Model                  string                        `yaml:"model"`
	Tools                  []string                      `yaml:"tools"`
	MainAgent              *bool                         `yaml:"mainAgent"`
	Subagent               *bool                         `yaml:"subagent"`
	CommandExecutionPolicy string                        `yaml:"commandExecutionPolicy"`
	Skills                 []string                      `yaml:"skills"`
	Targets                map[string]ast.TargetOverride `yaml:"targets"`
}

func extractAgent(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front agentFrontmatter
	body, err := importer.ParseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("antigravity: agent %q: %w", rel, err)
	}

	id := strings.TrimSuffix(filepath.Base(rel), ".md")
	if config.Agents == nil {
		config.Agents = make(map[string]ast.AgentConfig)
	}

	permMode := ""
	if front.CommandExecutionPolicy == "auto" {
		permMode = "allow"
	}

	config.Agents[id] = ast.AgentConfig{
		Name:           front.Name,
		Description:    front.Description,
		Model:          front.Model,
		Tools:          ast.ClearableList{Values: front.Tools},
		UserInvocable:  front.MainAgent,
		PermissionMode: permMode,
		Skills:         ast.ClearableList{Values: front.Skills},
		Targets:        front.Targets,
		Body:           body,
		SourceProvider: "antigravity",
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
		Targets                map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := importer.ParseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("antigravity: skill %q: %w", rel, err)
	}

	// DirectoryPerEntry layout: id is the directory name (parent of SKILL.md)
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
		AllowedTools:           ast.ClearableList{Values: front.AllowedTools},
		DisableModelInvocation: front.DisableModelInvocation,
		UserInvocable:          front.UserInvocable,
		ArgumentHint:           front.ArgumentHint,
		Targets:                front.Targets,
		Body:                   body,
		SourceProvider:         "antigravity",
	}
	return nil
}

// extractHooksJSON parses the hooks.json file and stores events under a
// "default" named hook block in config.Hooks.
func extractHooksJSON(_ string, data []byte, config *ast.XcaffoldConfig) error {
	var hooks ast.HookConfig
	if err := json.Unmarshal(data, &hooks); err != nil {
		return fmt.Errorf("antigravity: hooks.json parse: %w", err)
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

type mcpServerEntry struct {
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	ServerURL     string            `json:"serverUrl,omitempty"`
	URL           string            `json:"url,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	DisabledTools []string          `json:"disabledTools,omitempty"`
}

func extractMCPConfig(_ string, data []byte, config *ast.XcaffoldConfig) error {
	var wrapper struct {
		MCPServers map[string]mcpServerEntry `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("antigravity: mcp_config.json parse: %w", err)
	}
	if config.MCP == nil {
		config.MCP = make(map[string]ast.MCPConfig)
	}
	for k, v := range wrapper.MCPServers {
		serverURL := v.ServerURL
		if serverURL == "" {
			serverURL = v.URL
		}
		config.MCP[k] = ast.MCPConfig{
			Command:        v.Command,
			Args:           v.Args,
			Env:            v.Env,
			Headers:        v.Headers,
			URL:            serverURL,
			DisabledTools:  v.DisabledTools,
			SourceProvider: "antigravity",
		}
	}
	return nil
}

func extractWorkflow(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		Name        string                        `yaml:"name"`
		Description string                        `yaml:"description"`
		Steps       []ast.WorkflowStep            `yaml:"steps"`
		Targets     map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := importer.ParseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("antigravity: workflow %q: %w", rel, err)
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
		SourceProvider: "antigravity",
	}
	return nil
}
