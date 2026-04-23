package copilot

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

func init() {
	importer.Register(New())
}

// CopilotImporter imports resources from a .github/ directory tree.
// Warnings accumulates non-fatal per-file extraction errors encountered during Import().
// Callers may inspect Warnings after Import() returns to surface skipped files.
type CopilotImporter struct {
	Warnings []string
}

// New returns a new CopilotImporter.
func New() *CopilotImporter {
	return &CopilotImporter{}
}

// GetWarnings returns non-fatal per-file extraction errors collected during the last Import() call.
func (c *CopilotImporter) GetWarnings() []string { return c.Warnings }

// Provider returns the canonical provider name.
func (c *CopilotImporter) Provider() string { return "copilot" }

// InputDir returns the root directory relative to the workspace root.
func (c *CopilotImporter) InputDir() string { return ".github" }

// copilotMappings maps path patterns to AST kinds. First match wins.
// agents/*.agent.md is listed before agents/*.md so the more specific
// renderer-emitted extension takes priority while plain .md still matches
// for backward compatibility.
var copilotMappings = []importer.KindMapping{
	{Pattern: "hooks/*.json", Kind: importer.KindHook, Layout: importer.StandaloneJSON},
	{Pattern: "hooks/scripts/*.sh", Kind: importer.KindHookScript, Layout: importer.FlatFile},
	{Pattern: "agents/*.agent.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "agents/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "skills/*.md", Kind: importer.KindSkill, Layout: importer.FlatFile},
	{Pattern: "instructions/*.instructions.md", Kind: importer.KindRule, Layout: importer.FlatFile, Extension: ".instructions.md"},
	{Pattern: "copilot/mcp-config.json", Kind: importer.KindMCP, Layout: importer.StandaloneJSON},
	{Pattern: "workflows/copilot-setup-steps.yml", Kind: importer.KindWorkflow, Layout: importer.FlatFile},
}

// Classify returns the Kind and Layout for a given relative path.
// rel is relative to InputDir(). First matching entry in copilotMappings wins.
// Note: copilot-instructions.md is NOT handled here — it belongs to the
// orchestrator's project instructions pipeline and must remain KindUnknown.
func (c *CopilotImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range copilotMappings {
		if importer.MatchGlob(m.Pattern, rel) {
			return m.Kind, m.Layout
		}
	}
	return importer.KindUnknown, importer.LayoutUnknown
}

// Extract reads a single file and populates the appropriate section of config.
// rel is relative to InputDir().
func (c *CopilotImporter) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	rel = filepath.ToSlash(filepath.Clean(rel))
	kind, _ := c.Classify(rel, false)

	switch kind {
	case importer.KindAgent:
		return extractAgent(rel, data, config)
	case importer.KindSkill:
		return extractSkill(rel, data, config)
	case importer.KindRule:
		return extractRule(rel, data, config)
	case importer.KindHook:
		return extractHooksStandalone(rel, data, config)
	case importer.KindHookScript:
		return importer.ExtractHookScript(rel, data, config)
	case importer.KindMCP:
		return extractMCP(rel, data, config)
	case importer.KindWorkflow:
		return extractWorkflow(rel, data, config)
	default:
		return fmt.Errorf("copilot: no extractor for kind %q at path %q", kind, rel)
	}
}

// Import walks dir, classifies each entry, extracts classified files, and
// appends unclassified files to config.ProviderExtras["copilot"].
//
// Extraction errors for individual files are non-fatal: they are collected in
// c.Warnings and the walk continues. Only I/O errors (unreadable directory or
// file) abort the walk.
func (c *CopilotImporter) Import(dir string, config *ast.XcaffoldConfig) error {
	c.Warnings = c.Warnings[:0]
	return importer.WalkProviderDir(dir, func(rel string, data []byte) error {
		kind, _ := c.Classify(rel, false)
		if kind == importer.KindUnknown {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["copilot"] == nil {
				config.ProviderExtras["copilot"] = make(map[string][]byte)
			}
			config.ProviderExtras["copilot"][rel] = data
			return nil
		}
		if extractErr := c.Extract(rel, data, config); extractErr != nil {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["copilot"] == nil {
				config.ProviderExtras["copilot"] = make(map[string][]byte)
			}
			config.ProviderExtras["copilot"][rel] = data
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
		return fmt.Errorf("copilot: agent %q: %w", rel, err)
	}

	base := filepath.Base(rel)
	id := strings.TrimSuffix(base, ".agent.md")
	if id == base {
		// Fall back to plain .md for backward compatibility.
		id = strings.TrimSuffix(base, ".md")
	}
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
		SourceProvider:         "copilot",
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
		return fmt.Errorf("copilot: skill %q: %w", rel, err)
	}

	// For FlatFile layout: id is the filename without .md extension.
	id := strings.TrimSuffix(filepath.Base(rel), ".md")
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
		SourceProvider:         "copilot",
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
		return fmt.Errorf("copilot: rule %q: %w", rel, err)
	}

	// Strip the double extension: "security.instructions.md" → "security"
	id := strings.TrimSuffix(filepath.Base(rel), ".instructions.md")
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
		SourceProvider: "copilot",
	}
	return nil
}

// mcpFileWrapper is the outer JSON envelope for mcp-config.json.
type mcpFileWrapper struct {
	MCPServers map[string]ast.MCPConfig `json:"mcpServers"`
}

func extractMCP(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var wrapper mcpFileWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("copilot: mcp-config.json parse: %w", err)
	}
	if config.MCP == nil {
		config.MCP = make(map[string]ast.MCPConfig)
	}
	for k, v := range wrapper.MCPServers {
		v.SourceProvider = "copilot"
		config.MCP[k] = v
	}
	return nil
}

func extractWorkflow(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		ApiVersion  string             `yaml:"api-version"`
		Name        string             `yaml:"name"`
		Description string             `yaml:"description"`
		Steps       []ast.WorkflowStep `yaml:"steps"`
	}

	// Workflows are YAML (not markdown frontmatter), so parse directly.
	if err := yaml.Unmarshal(data, &front); err != nil {
		return fmt.Errorf("copilot: workflow %q: %w", rel, err)
	}

	// id is the filename without .yml extension
	id := strings.TrimSuffix(filepath.Base(rel), ".yml")
	if config.Workflows == nil {
		config.Workflows = make(map[string]ast.WorkflowConfig)
	}
	config.Workflows[id] = ast.WorkflowConfig{
		ApiVersion:     front.ApiVersion,
		Name:           front.Name,
		Description:    front.Description,
		Steps:          front.Steps,
		SourceProvider: "copilot",
	}
	return nil
}

func extractHooksStandalone(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var hooks ast.HookConfig
	if err := json.Unmarshal(data, &hooks); err != nil {
		return fmt.Errorf("copilot: hooks.json parse: %w", err)
	}
	config.Hooks = map[string]ast.NamedHookConfig{"default": {Name: "default", Events: hooks}}
	return nil
}
