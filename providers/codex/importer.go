package codex

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
)

// CodexImporter imports resources from a .codex/ directory tree.
type CodexImporter struct {
	importer.BaseImporter
}

// NewImporter returns a new CodexImporter.
func NewImporter() *CodexImporter {
	return &CodexImporter{
		BaseImporter: importer.BaseImporter{
			ProviderName: "codex",
			Dir:          ".codex",
		},
	}
}

// codexMappings maps path patterns to AST kinds. First match wins.
var nameRe = regexp.MustCompile("^[a-z0-9-]+$")

var codexMappings = []importer.KindMapping{
	{Pattern: "agents/*.toml", Kind: importer.KindAgent, Layout: importer.FlatFile},
	{Pattern: "hooks.json", Kind: importer.KindHook, Layout: importer.StandaloneJSON},
	{Pattern: "hooks/*.sh", Kind: importer.KindHookScript, Layout: importer.FlatFile},
}

// Classify returns the Kind and Layout for a given relative path.
// rel is relative to InputDir(). First matching entry in codexMappings wins.
func (c *CodexImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, m := range codexMappings {
		if importer.MatchGlob(m.Pattern, rel) {
			return m.Kind, m.Layout
		}
	}
	return importer.KindUnknown, importer.LayoutUnknown
}

// Extract reads a single file and populates the appropriate section of config.
// rel is relative to InputDir().
func (c *CodexImporter) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	rel = filepath.ToSlash(filepath.Clean(rel))
	kind, _ := c.Classify(rel, false)

	switch kind {
	case importer.KindAgent:
		return extractAgent(rel, data, config)
	case importer.KindHook:
		return extractHooks(rel, data, config)
	case importer.KindHookScript:
		return importer.DefaultExtractHookScript(rel, data, config)
	default:
		return fmt.Errorf("codex: no extractor for kind %q at path %q", kind, rel)
	}
}

// Import walks dir, classifies each entry, extracts classified files, and
// appends unclassified files to config.ProviderExtras["codex"].
func (c *CodexImporter) Import(dir string, config *ast.XcaffoldConfig) error {
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
	return nil
}

// agentTOML is the TOML schema for Codex agent files.
type agentTOML struct {
	Name                  string   `toml:"name"`
	Description           string   `toml:"description"`
	Model                 string   `toml:"model"`
	DeveloperInstructions string   `toml:"developer_instructions"`
	SandboxMode           string   `toml:"sandbox_mode"`
	Effort                string   `toml:"effort"`
	InitialPrompt         string   `toml:"initial_prompt"`
	Memory                string   `toml:"memory"`
	Tools                 []string `toml:"tools"`
	DisallowedTools       []string `toml:"disallowed_tools"`
	MaxTurns              int      `toml:"max_turns"`
}

// extractAgent parses a TOML agent file and populates config.Agents.
func extractAgent(rel string, data []byte, config *ast.XcaffoldConfig) error {
	// Size gate: reject files > 1 MiB before parsing
	if len(data) > (1 << 20) {
		return fmt.Errorf("codex: agent %q exceeds 1 MiB size limit", rel)
	}

	var agent agentTOML
	if err := toml.Unmarshal(data, &agent); err != nil {
		return fmt.Errorf("codex: agent %q: %w", rel, err)
	}

	// Validate name against ^[a-z0-9-]+$ to prevent path traversal
	if !nameRe.MatchString(agent.Name) {
		return fmt.Errorf("codex: agent %q: invalid name %q (must match ^[a-z0-9-]+$)", rel, agent.Name)
	}

	// Extract ID from filename (e.g. "agents/my-agent.toml" -> "my-agent")
	id := strings.TrimSuffix(filepath.Base(rel), ".toml")

	if config.Agents == nil {
		config.Agents = make(map[string]ast.AgentConfig)
	}

	config.Agents[id] = ast.AgentConfig{
		Name:            agent.Name,
		Description:     agent.Description,
		Model:           agent.Model,
		Tools:           ast.ClearableList{Values: agent.Tools},
		DisallowedTools: ast.ClearableList{Values: agent.DisallowedTools},
		Effort:          agent.Effort,
		MaxTurns:        intPtrIfNonZero(agent.MaxTurns),
		InitialPrompt:   agent.InitialPrompt,
		Memory:          ast.NewFlexStringSlice(agent.Memory),
		Body:            agent.DeveloperInstructions,
		SourceProvider:  "codex",
	}

	return nil
}

// hooksJSON is the JSON schema for Codex hooks.json.
type hooksJSON struct {
	Hooks ast.HookConfig `json:"hooks"`
}

// extractHooks parses a JSON hooks file and populates config.Hooks.
func extractHooks(rel string, data []byte, config *ast.XcaffoldConfig) error {
	var wrapper hooksJSON
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return fmt.Errorf("codex: hooks parse: %w", err)
	}

	if config.Hooks == nil {
		config.Hooks = make(map[string]ast.NamedHookConfig)
	}
	config.Hooks["default"] = ast.NamedHookConfig{
		Name:   "default",
		Events: wrapper.Hooks,
	}
	return nil
}

// intPtrIfNonZero returns a pointer to n if n is non-zero, otherwise nil.
func intPtrIfNonZero(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}
