package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// overrideFileEntry represents a detected override file with its parsed metadata.
type overrideFileEntry struct {
	Path     string
	Kind     string
	Provider string
}

// classifyOverrideFile parses a filename to detect <kind>.<provider>.xcf pattern.
// Returns (kind, provider, isOverride). If not an override file, isOverride is false.
func classifyOverrideFile(filename string) (kind, provider string, isOverride bool) {
	name := strings.TrimSuffix(filename, ".xcf")
	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	if !canonicalKindFilenames[parts[0]] {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// parseOverrideFile parses a single override file (.provider.xcf) and stores
// the partial resource config in XcaffoldConfig.Overrides.
func parseOverrideFile(entry overrideFileEntry, config *ast.XcaffoldConfig) error {
	data, err := os.ReadFile(entry.Path)
	if err != nil {
		return fmt.Errorf("read override %s: %w", entry.Path, err)
	}

	frontmatter, body, err := extractFrontmatterAndBody(data)
	if err != nil {
		return fmt.Errorf("parse override %s: %w", entry.Path, err)
	}

	// Memory does not participate in the override system.
	if entry.Kind == "memory" {
		return nil
	}

	// Infer resource name from directory: xcf/agents/<name>/agent.claude.xcf -> name
	resourceName := filepath.Base(filepath.Dir(entry.Path))
	trimmedBody := strings.TrimSpace(string(body))

	// Initialize Overrides if nil
	if config.Overrides == nil {
		config.Overrides = &ast.ResourceOverrides{}
	}

	return decodeAndStoreOverride(entry, frontmatter, trimmedBody, resourceName, config.Overrides)
}

// decodeAndStoreOverride decodes the frontmatter into the appropriate config type
// and stores it in the ResourceOverrides map based on the resource kind.
func decodeAndStoreOverride(entry overrideFileEntry, frontmatter []byte, body, name string, overrides *ast.ResourceOverrides) error {
	switch entry.Kind {
	case "agent":
		var cfg ast.AgentConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode agent override %s: %w", entry.Path, err)
		}
		cfg.Body = body
		overrides.AddAgent(name, entry.Provider, cfg)
	case "skill":
		var cfg ast.SkillConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode skill override %s: %w", entry.Path, err)
		}
		cfg.Body = body
		overrides.AddSkill(name, entry.Provider, cfg)
	case "rule":
		var cfg ast.RuleConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode rule override %s: %w", entry.Path, err)
		}
		cfg.Body = body
		overrides.AddRule(name, entry.Provider, cfg)
	case "workflow":
		var cfg ast.WorkflowConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode workflow override %s: %w", entry.Path, err)
		}
		cfg.Body = body
		overrides.AddWorkflow(name, entry.Provider, cfg)
	case "mcp":
		var cfg ast.MCPConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode mcp override %s: %w", entry.Path, err)
		}
		overrides.AddMCP(name, entry.Provider, cfg)
	case "hooks":
		var cfg ast.NamedHookConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode hooks override %s: %w", entry.Path, err)
		}
		overrides.AddHooks(name, entry.Provider, cfg)
	case "settings":
		var cfg ast.SettingsConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode settings override %s: %w", entry.Path, err)
		}
		overrides.AddSettings(name, entry.Provider, cfg)
	case "policy":
		var cfg ast.PolicyConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode policy override %s: %w", entry.Path, err)
		}
		overrides.AddPolicy(name, entry.Provider, cfg)
	case "template":
		var cfg ast.TemplateConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode template override %s: %w", entry.Path, err)
		}
		cfg.Body = body
		overrides.AddTemplate(name, entry.Provider, cfg)
	default:
		return fmt.Errorf("override file %s: unsupported kind %q for overrides", entry.Path, entry.Kind)
	}
	return nil
}

// mapKeys extracts the keys from a map[string]map[string]T, returning them as a slice.
// This helper converts typed maps in ResourceOverrides to a uniform key list for
// table-driven validation.
func mapKeys[K comparable, V any](m map[string]map[K]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// validateOverrideBasesExist ensures that every override file has a corresponding
// base resource. Override files without bases cannot be applied.
func validateOverrideBasesExist(config *ast.XcaffoldConfig) error {
	if config.Overrides == nil {
		return nil
	}

	type overrideCheck struct {
		kind    string
		names   []string
		hasBase func(string) bool
	}

	checks := []overrideCheck{
		{"agent", mapKeys(config.Overrides.Agent), func(n string) bool { _, ok := config.Agents[n]; return ok }},
		{"skill", mapKeys(config.Overrides.Skill), func(n string) bool { _, ok := config.Skills[n]; return ok }},
		{"rule", mapKeys(config.Overrides.Rule), func(n string) bool { _, ok := config.Rules[n]; return ok }},
		{"workflow", mapKeys(config.Overrides.Workflow), func(n string) bool { _, ok := config.Workflows[n]; return ok }},
		{"mcp", mapKeys(config.Overrides.MCP), func(n string) bool { _, ok := config.MCP[n]; return ok }},
		{"hooks", mapKeys(config.Overrides.Hooks), func(n string) bool { _, ok := config.Hooks[n]; return ok }},
		{"settings", mapKeys(config.Overrides.Settings), func(n string) bool { _, ok := config.Settings[n]; return ok }},
		{"policy", mapKeys(config.Overrides.Policy), func(n string) bool { _, ok := config.Policies[n]; return ok }},
		{"template", mapKeys(config.Overrides.Template), func(n string) bool { _, ok := config.Templates[n]; return ok }},
	}

	for _, c := range checks {
		for _, name := range c.names {
			if !c.hasBase(name) {
				return fmt.Errorf("override file for %s %q has no base resource", c.kind, name)
			}
		}
	}
	return nil
}
