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

	return decodeAndStoreOverride(decodeOverrideOpts{
		Path:        entry.Path,
		Kind:        entry.Kind,
		Frontmatter: frontmatter,
		Body:        trimmedBody,
		Name:        resourceName,
		Provider:    entry.Provider,
		Overrides:   config.Overrides,
	})
}

// decodeOverrideOpts groups parameters for decoding override resources.
type decodeOverrideOpts struct {
	Path        string // file path for error reporting
	Kind        string
	Frontmatter []byte
	Body        string
	Name        string
	Provider    string
	Overrides   *ast.ResourceOverrides
}

// decodeOverrideCtx groups parameters for decoding override resources.
type decodeOverrideCtx struct {
	fm        []byte
	body      string
	name      string
	provider  string
	overrides *ast.ResourceOverrides
}

// decodeAndStoreOverride decodes the frontmatter into the appropriate config type
// and stores it in the ResourceOverrides map based on the resource kind.
// decodeAgent decodes and stores an agent override.
func decodeAgent(ctx decodeOverrideCtx) error {
	var cfg ast.AgentConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	cfg.Body = ctx.body
	ctx.overrides.AddAgent(ctx.name, ctx.provider, cfg)
	return nil
}

// decodeSkill decodes and stores a skill override.
func decodeSkill(ctx decodeOverrideCtx) error {
	var cfg ast.SkillConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	cfg.Body = ctx.body
	ctx.overrides.AddSkill(ctx.name, ctx.provider, cfg)
	return nil
}

// decodeRule decodes and stores a rule override.
func decodeRule(ctx decodeOverrideCtx) error {
	var cfg ast.RuleConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	cfg.Body = ctx.body
	ctx.overrides.AddRule(ctx.name, ctx.provider, cfg)
	return nil
}

// decodeWorkflow decodes and stores a workflow override.
func decodeWorkflow(ctx decodeOverrideCtx) error {
	var cfg ast.WorkflowConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	cfg.Body = ctx.body
	ctx.overrides.AddWorkflow(ctx.name, ctx.provider, cfg)
	return nil
}

// decodeMCP decodes and stores an MCP override.
func decodeMCP(ctx decodeOverrideCtx) error {
	var cfg ast.MCPConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	ctx.overrides.AddMCP(ctx.name, ctx.provider, cfg)
	return nil
}

// decodeHooks decodes and stores a hooks override.
func decodeHooks(ctx decodeOverrideCtx) error {
	var cfg ast.NamedHookConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	ctx.overrides.AddHooks(ctx.name, ctx.provider, cfg)
	return nil
}

// decodeSettings decodes and stores a settings override.
func decodeSettings(ctx decodeOverrideCtx) error {
	var cfg ast.SettingsConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	ctx.overrides.AddSettings(ctx.name, ctx.provider, cfg)
	return nil
}

// decodePolicy decodes and stores a policy override.
func decodePolicy(ctx decodeOverrideCtx) error {
	var cfg ast.PolicyConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	ctx.overrides.AddPolicy(ctx.name, ctx.provider, cfg)
	return nil
}

// decodeTemplate decodes and stores a template override.
func decodeTemplate(ctx decodeOverrideCtx) error {
	var cfg ast.TemplateConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	cfg.Body = ctx.body
	ctx.overrides.AddTemplate(ctx.name, ctx.provider, cfg)
	return nil
}

// decodeContext decodes and stores a context override.
func decodeContext(ctx decodeOverrideCtx) error {
	var cfg ast.ContextConfig
	if err := yaml.Unmarshal(ctx.fm, &cfg); err != nil {
		return err
	}
	cfg.Body = ctx.body
	ctx.overrides.AddContext(ctx.name, ctx.provider, cfg)
	return nil
}

func decodeAndStoreOverride(opts decodeOverrideOpts) error {
	ctx := decodeOverrideCtx{
		fm:        opts.Frontmatter,
		body:      opts.Body,
		name:      opts.Name,
		provider:  opts.Provider,
		overrides: opts.Overrides,
	}
	var err error
	switch opts.Kind {
	case "agent":
		err = decodeAgent(ctx)
	case "skill":
		err = decodeSkill(ctx)
	case "rule":
		err = decodeRule(ctx)
	case "workflow":
		err = decodeWorkflow(ctx)
	case "mcp":
		err = decodeMCP(ctx)
	case "hooks":
		err = decodeHooks(ctx)
	case "settings":
		err = decodeSettings(ctx)
	case "policy":
		err = decodePolicy(ctx)
	case "template":
		err = decodeTemplate(ctx)
	case "context":
		err = decodeContext(ctx)
	default:
		return fmt.Errorf("override file %s: unsupported kind %q for overrides", opts.Path, opts.Kind)
	}
	if err != nil {
		return fmt.Errorf("decode %s override %s: %w", opts.Kind, opts.Path, err)
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
