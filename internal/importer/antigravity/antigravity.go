package antigravity

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

// AntigravityImporter imports resources from a .agents/ directory tree.
// Warnings accumulates non-fatal per-file extraction errors encountered during Import().
// Callers may inspect Warnings after Import() returns to surface skipped files.
type AntigravityImporter struct {
	Warnings []string
}

// New returns a new AntigravityImporter.
func New() *AntigravityImporter {
	return &AntigravityImporter{}
}

// GetWarnings returns non-fatal per-file extraction errors collected during the last Import() call.
func (a *AntigravityImporter) GetWarnings() []string { return a.Warnings }

// Provider returns the canonical provider name.
func (a *AntigravityImporter) Provider() string { return "antigravity" }

// InputDir returns the root directory relative to the workspace root.
func (a *AntigravityImporter) InputDir() string { return ".agents" }

// antigravityMappings maps path patterns to AST kinds. First match wins.
// Agents are stored in prompts/ (not agents/) — this is the key Antigravity difference.
var antigravityMappings = []importer.KindMapping{
	{Pattern: "prompts/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "skills/*.md", Kind: importer.KindSkill, Layout: importer.FlatFile},
	{Pattern: "rules/*.md", Kind: importer.KindRule, Layout: importer.FlatFile},
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
	case importer.KindRule:
		return extractRule(rel, data, config)
	case importer.KindMCP:
		return extractMCPStandalone(rel, data, config)
	case importer.KindWorkflow:
		return extractWorkflow(rel, data, config)
	default:
		return fmt.Errorf("antigravity: no extractor for kind %q at path %q", kind, rel)
	}
}

// Import walks dir, classifies each entry, extracts classified files, and
// appends unclassified files to config.ProviderExtras["antigravity"].
//
// Extraction errors for individual files are non-fatal: they are collected in
// a.Warnings and the walk continues. Only I/O errors (unreadable directory or
// file) abort the walk.
func (a *AntigravityImporter) Import(dir string, config *ast.XcaffoldConfig) error {
	a.Warnings = a.Warnings[:0]
	return importer.WalkProviderDir(dir, func(rel string, data []byte) error {
		kind, _ := a.Classify(rel, false)
		if kind == importer.KindUnknown {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["antigravity"] == nil {
				config.ProviderExtras["antigravity"] = make(map[string][]byte)
			}
			config.ProviderExtras["antigravity"][rel] = data
			return nil
		}
		if extractErr := a.Extract(rel, data, config); extractErr != nil {
			if config.ProviderExtras == nil {
				config.ProviderExtras = make(map[string]map[string][]byte)
			}
			if config.ProviderExtras["antigravity"] == nil {
				config.ProviderExtras["antigravity"] = make(map[string][]byte)
			}
			config.ProviderExtras["antigravity"][rel] = data
			a.Warnings = append(a.Warnings, fmt.Sprintf("skipped %q: %v", rel, extractErr))
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
		return fmt.Errorf("antigravity: agent %q: %w", rel, err)
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
		Memory:                 ast.NewFlexStringSlice(front.Memory),
		Color:                  front.Color,
		InitialPrompt:          front.InitialPrompt,
		Skills:                 front.Skills,
		Rules:                  front.Rules,
		MCP:                    front.MCP,
		Assertions:             front.Assertions,
		Targets:                front.Targets,
		Body:                   body,
		SourceProvider:         "antigravity",
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
		return fmt.Errorf("antigravity: skill %q: %w", rel, err)
	}

	// FlatFile layout: id is the filename stem.
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
		Body:                   body,
		SourceProvider:         "antigravity",
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
		return fmt.Errorf("antigravity: rule %q: %w", rel, err)
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
		SourceProvider: "antigravity",
	}
	return nil
}

// mcpFileWrapper is the outer JSON envelope for mcp_config.json.
// Antigravity writes {"mcpServers": {...}} as the top-level shape.
type mcpFileWrapper struct {
	MCPServers map[string]ast.MCPConfig `json:"mcpServers"`
}

func extractMCPStandalone(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var wrapper mcpFileWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("antigravity: mcp_config.json parse: %w", err)
	}
	if config.MCP == nil {
		config.MCP = make(map[string]ast.MCPConfig)
	}
	for k, v := range wrapper.MCPServers {
		v.SourceProvider = "antigravity"
		config.MCP[k] = v
	}
	return nil
}

func extractWorkflow(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var front struct {
		ApiVersion  string                        `yaml:"api-version"`
		Name        string                        `yaml:"name"`
		Description string                        `yaml:"description"`
		Steps       []ast.WorkflowStep            `yaml:"steps"`
		Targets     map[string]ast.TargetOverride `yaml:"targets"`
	}

	body, err := importer.ParseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("antigravity: workflow %q: %w", rel, err)
	}

	id := strings.TrimSuffix(filepath.Base(rel), ".md")
	if config.Workflows == nil {
		config.Workflows = make(map[string]ast.WorkflowConfig)
	}
	config.Workflows[id] = ast.WorkflowConfig{
		ApiVersion:     front.ApiVersion,
		Name:           front.Name,
		Description:    front.Description,
		Steps:          front.Steps,
		Targets:        front.Targets,
		Body:           body,
		SourceProvider: "antigravity",
	}
	return nil
}
