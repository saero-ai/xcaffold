package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"gopkg.in/yaml.v3"
)

// overrideFileEntry represents a detected override file with its parsed metadata.
type overrideFileEntry struct {
	Path     string
	Kind     string
	Provider string
}

// classifyOverrideFile parses a filename to detect <kind>.<provider>.xcaf pattern.
// Returns (kind, provider, isOverride). If not an override file, isOverride is false.
func classifyOverrideFile(filename string) (kind, provider string, isOverride bool) {
	name := strings.TrimSuffix(filename, ".xcaf")
	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	if !canonicalKindFilenames[parts[0]] {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// parseOverrideFile parses a single override file (.provider.xcaf) and stores
// the partial resource config in XcaffoldConfig.Overrides.
func parseOverrideFile(entry overrideFileEntry, config *ast.XcaffoldConfig, vars map[string]interface{}, envs map[string]string) error {
	data, err := os.ReadFile(entry.Path)
	if err != nil {
		return fmt.Errorf("read override %s: %w", entry.Path, err)
	}

	if len(vars) > 0 || len(envs) > 0 {
		data, err = resolver.ExpandVariables(data, vars, envs)
		if err != nil {
			return fmt.Errorf("expand variables in %s: %w", entry.Path, err)
		}
	}

	frontmatter, body, err := extractFrontmatterAndBody(data)
	if err != nil {
		return fmt.Errorf("parse override %s: %w", entry.Path, err)
	}

	// Memory does not participate in the override system.
	if entry.Kind == "memory" {
		return nil
	}

	// Infer resource name from directory: xcaf/agents/<name>/agent.claude.xcaf -> name
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
// decodeAgent decodes and stores an agent override.
func decodeAgent(fm []byte, body, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.AgentConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	cfg.Body = body
	overrides.AddAgent(name, provider, cfg)
	return nil
}

// decodeSkill decodes and stores a skill override.
func decodeSkill(fm []byte, body, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.SkillConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	cfg.Body = body
	overrides.AddSkill(name, provider, cfg)
	return nil
}

// decodeRule decodes and stores a rule override.
func decodeRule(fm []byte, body, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.RuleConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	cfg.Body = body
	overrides.AddRule(name, provider, cfg)
	return nil
}

// decodeWorkflow decodes and stores a workflow override.
func decodeWorkflow(fm []byte, body, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.WorkflowConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	cfg.Body = body
	overrides.AddWorkflow(name, provider, cfg)
	return nil
}

// decodeMCP decodes and stores an MCP override.
func decodeMCP(fm []byte, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.MCPConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	overrides.AddMCP(name, provider, cfg)
	return nil
}

// decodeHooks decodes and stores a hooks override.
func decodeHooks(fm []byte, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.NamedHookConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	overrides.AddHooks(name, provider, cfg)
	return nil
}

// decodeSettings decodes and stores a settings override.
func decodeSettings(fm []byte, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.SettingsConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	overrides.AddSettings(name, provider, cfg)
	return nil
}

// decodePolicy decodes and stores a policy override.
func decodePolicy(fm []byte, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.PolicyConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	overrides.AddPolicy(name, provider, cfg)
	return nil
}

// decodeTemplate decodes and stores a template override.
func decodeTemplate(fm []byte, body, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.TemplateConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	cfg.Body = body
	overrides.AddTemplate(name, provider, cfg)
	return nil
}

// decodeContext decodes and stores a context override.
func decodeContext(fm []byte, body, name, provider string, overrides *ast.ResourceOverrides) error {
	var cfg ast.ContextConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return err
	}
	cfg.Body = body
	overrides.AddContext(name, provider, cfg)
	return nil
}

func decodeAndStoreOverride(entry overrideFileEntry, frontmatter []byte, body, name string, overrides *ast.ResourceOverrides) error {
	var err error
	switch entry.Kind {
	case "agent":
		err = decodeAgent(frontmatter, body, name, entry.Provider, overrides)
	case "skill":
		err = decodeSkill(frontmatter, body, name, entry.Provider, overrides)
	case "rule":
		err = decodeRule(frontmatter, body, name, entry.Provider, overrides)
	case "workflow":
		err = decodeWorkflow(frontmatter, body, name, entry.Provider, overrides)
	case "mcp":
		err = decodeMCP(frontmatter, name, entry.Provider, overrides)
	case "hooks":
		err = decodeHooks(frontmatter, name, entry.Provider, overrides)
	case "settings":
		err = decodeSettings(frontmatter, name, entry.Provider, overrides)
	case "policy":
		err = decodePolicy(frontmatter, name, entry.Provider, overrides)
	case "template":
		err = decodeTemplate(frontmatter, body, name, entry.Provider, overrides)
	case "context":
		err = decodeContext(frontmatter, body, name, entry.Provider, overrides)
	default:
		return fmt.Errorf("override file %s: unsupported kind %q for overrides", entry.Path, entry.Kind)
	}
	if err != nil {
		return fmt.Errorf("decode %s override %s: %w", entry.Kind, entry.Path, err)
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
		{"context", mapKeys(config.Overrides.Context), func(n string) bool { _, ok := config.Contexts[n]; return ok }},
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
