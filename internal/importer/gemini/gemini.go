package gemini

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

// GeminiImporter imports resources from a .gemini/ directory tree.
type GeminiImporter struct{}

// New returns a new GeminiImporter.
func New() importer.ProviderImporter {
	return &GeminiImporter{}
}

// Provider returns the canonical provider name.
func (g *GeminiImporter) Provider() string { return "gemini" }

// InputDir returns the root directory relative to the workspace root.
func (g *GeminiImporter) InputDir() string { return ".gemini" }

// geminiMappings maps path patterns to AST kinds. First match wins.
// Skills are FlatFile layout — one .md per skill, no directory-per-entry.
// settings.json appears first for the container-level KindSettings match;
// embedded mcpServers and hooks keys are handled inside Extract().
var geminiMappings = []importer.KindMapping{
	{Pattern: "agents/*.md", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "skills/*.md", Kind: importer.KindSkill, Layout: importer.FlatFile},
	{Pattern: "rules/*.md", Kind: importer.KindRule, Layout: importer.FlatFile},
	// settings.json is a container file holding settings, mcpServers, AND hooks.
	// Classify returns KindSettings (first match); extractSettings() handles the
	// two-phase decomposition of mcpServers → config.MCP and hooks → config.Hooks.
	{Pattern: "settings.json", Kind: importer.KindSettings, Layout: importer.EmbeddedJSONKey, JSONKey: ""},
}

// Classify returns the Kind and Layout for a given relative path.
// rel is relative to InputDir(). First matching entry in geminiMappings wins.
func (g *GeminiImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range geminiMappings {
		if matchGlob(m.Pattern, rel) {
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
func (g *GeminiImporter) Import(dir string, config *ast.XcaffoldConfig) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("gemini: rel path: %w", err)
		}
		rel = filepath.ToSlash(rel)

		data, err := readFile(path)
		if err != nil {
			return fmt.Errorf("gemini: read %q: %w", rel, err)
		}

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

		return g.Extract(rel, data, config)
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

	body, err := parseFrontmatter(data, &front)
	if err != nil {
		return fmt.Errorf("gemini: skill %q: %w", rel, err)
	}

	// FlatFile layout: id is the filename stem (not directory name)
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

	body, err := parseFrontmatter(data, &front)
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
		Instructions:   body,
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
		config.Hooks = hooks
	}

	// Extract full settings into config.Settings
	var settings ast.SettingsConfig
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("gemini: %s settings: %w", rel, err)
	}
	settings.SourceProvider = "gemini"
	config.Settings = settings
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
