package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// ParseFileExact reads a .xcaf YAML configuration from the given path without
// loading the global base. This is the internal entry point called by Parse* functions.
func ParseFileExact(path string, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config %q: %w", path, err)
	}
	defer f.Close()

	// Prepend source path so kind-specific parsers can derive contextual
	// metadata from the file's on-disk location (e.g., xcaf/agents/<agentID>/memory/).
	// Caller-supplied opts override this by appearing later in the slice.
	opts = append([]parseOptionFunc{withSourcePath(path)}, opts...)

	config, err := parsePartial(f, opts...)
	if err != nil {
		return nil, fmt.Errorf("error in %q: %w", path, err)
	}
	return config, nil
}

// loadGlobalBase implicitly discovers and loads the global configuration
// from ~/.xcaffold/. It returns an empty config if no global config is found.
// Resources loaded from this base are tagged as Inherited=true during merge.
func loadGlobalBase() (*ast.XcaffoldConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &ast.XcaffoldConfig{}, nil // ignore errors, just no global
	}

	xcaffoldDir := filepath.Join(home, ".xcaffold")
	if stat, err := os.Stat(xcaffoldDir); err == nil && stat.IsDir() {
		// Parse the dir, but disable global loading to avoid infinite recursion!
		// parseDirectoryRaw natively parses a dir without applying global base.
		cfg, err := parseDirectoryRaw(xcaffoldDir, nil, nil, withGlobalScope())
		if err != nil {
			// TODO: surface global scope parse errors once the schema is finalized.
			return &ast.XcaffoldConfig{}, nil
		}
		// If the global config itself extends something, resolve it!
		if cfg.Extends != "" {
			visited := map[string]bool{xcaffoldDir: true}
			cfg, err = resolveExtendsRecursive(xcaffoldDir, cfg, nil, nil, visited)
			if err != nil {
				// TODO: surface extends resolution errors once global scope ships.
				return &ast.XcaffoldConfig{}, nil
			}
		}
		return cfg, nil
	}

	return &ast.XcaffoldConfig{}, nil
}

// resolveExtends resolves the extends: field in a configuration by recursively
// loading and merging base configurations. It detects circular dependencies.
func resolveExtends(contextDir string, config *ast.XcaffoldConfig, vars map[string]interface{}, envs map[string]string) (*ast.XcaffoldConfig, error) {
	visited := make(map[string]bool)
	return resolveExtendsRecursive(contextDir, config, vars, envs, visited)
}

// resolveExtendsRecursive recursively resolves extends: directives, tracking visited
// paths to detect circular dependencies. Base configurations are merged into the
// child configuration using mergeConfigOverride.
//
//nolint:gocyclo
func resolveExtendsRecursive(contextDir string, config *ast.XcaffoldConfig, vars map[string]interface{}, envs map[string]string, visited map[string]bool) (*ast.XcaffoldConfig, error) {
	if config.Extends == "" {
		return config, nil
	}

	var basePath string
	if config.Extends == "global" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not resolve 'extends: global': %w", err)
		}

		xcaffoldDir := filepath.Join(home, ".xcaffold")
		if stat, err := os.Stat(xcaffoldDir); err == nil && stat.IsDir() {
			if visited[xcaffoldDir] {
				return nil, fmt.Errorf("circular dependency detected: global setup extends itself")
			}
			visited[xcaffoldDir] = true

			baseConfig, err := parseDirectoryRaw(xcaffoldDir, vars, envs, withGlobalScope())
			if err != nil {
				return nil, fmt.Errorf("failed to parse global directory %q: %w", xcaffoldDir, err)
			}
			if baseConfig.Extends != "" {
				baseConfig, err = resolveExtendsRecursive(xcaffoldDir, baseConfig, vars, envs, visited)
				if err != nil {
					return nil, err
				}
			}
			return mergeConfigOverride(baseConfig, config), nil
		}

		return nil, fmt.Errorf("could not resolve 'extends: global': no global config found")
	} else if filepath.IsAbs(config.Extends) {
		basePath = config.Extends
	} else {
		basePath = filepath.Join(contextDir, config.Extends)
	}

	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("could not resolve extends path %q: %w", basePath, err)
	}

	if visited[absPath] {
		return nil, fmt.Errorf("circular extends detected: %q", absPath)
	}
	visited[absPath] = true

	parsed, err := ParseFileExact(absPath, withVars(vars), withEnvs(envs))
	if err != nil {
		return nil, fmt.Errorf("failed to load base config %q: %w", config.Extends, err)
	}

	baseConfig, err := resolveExtendsRecursive(filepath.Dir(absPath), parsed, vars, envs, visited)
	if err != nil {
		return nil, err
	}

	return mergeConfigOverride(baseConfig, config), nil
}

// mergeAllStrict merges multiple ParsedFile objects from the same directory,
// detecting duplicate resource IDs and raising an error if found.
// This is distinct from mergeConfigOverride which is used for extends resolution.
//
// mergeAgentsStrict merges agent maps from parsed into merged config with duplicate checking.
//
//nolint:gocyclo
func mergeAgentsStrict(merged, parsed map[string]ast.AgentConfig, origins map[string]string, filePath string) (map[string]ast.AgentConfig, map[string]string, error) {
	return mergeMapStrict(merged, parsed, "agent", origins, filePath)
}

// mergeSkillsStrict merges skill maps from parsed into merged config with duplicate checking.
func mergeSkillsStrict(merged, parsed map[string]ast.SkillConfig, origins map[string]string, filePath string) (map[string]ast.SkillConfig, map[string]string, error) {
	return mergeMapStrict(merged, parsed, "skill", origins, filePath)
}

// mergeRulesStrict merges rule maps from parsed into merged config with duplicate checking.
func mergeRulesStrict(merged, parsed map[string]ast.RuleConfig, origins map[string]string, filePath string) (map[string]ast.RuleConfig, map[string]string, error) {
	return mergeMapStrict(merged, parsed, "rule", origins, filePath)
}

// mergeMCPStrict merges MCP maps from parsed into merged config with duplicate checking.
func mergeMCPStrict(merged, parsed map[string]ast.MCPConfig, origins map[string]string, filePath string) (map[string]ast.MCPConfig, map[string]string, error) {
	return mergeMapStrict(merged, parsed, "mcp", origins, filePath)
}

// mergeWorkflowsStrict merges workflow maps from parsed into merged config with duplicate checking.
func mergeWorkflowsStrict(merged, parsed map[string]ast.WorkflowConfig, origins map[string]string, filePath string) (map[string]ast.WorkflowConfig, map[string]string, error) {
	return mergeMapStrict(merged, parsed, "workflow", origins, filePath)
}

// mergePoliciesStrict merges policy maps from parsed into merged config with duplicate checking.
func mergePoliciesStrict(merged, parsed map[string]ast.PolicyConfig, origins map[string]string, filePath string) (map[string]ast.PolicyConfig, map[string]string, error) {
	return mergeMapStrict(merged, parsed, "policy", origins, filePath)
}

// printBlueprintsStrict merges blueprint maps from parsed into merged config with duplicate checking.
func mergeBlueprintsStrict(merged, parsed map[string]ast.BlueprintConfig, origins map[string]string, filePath string) (map[string]ast.BlueprintConfig, map[string]string, error) {
	return mergeMapStrict(merged, parsed, "blueprint name", origins, filePath)
}

// mergeContextsStrict merges context maps from parsed into merged config with duplicate checking.
func mergeContextsStrict(merged, parsed map[string]ast.ContextConfig, origins map[string]string, filePath string) (map[string]ast.ContextConfig, map[string]string, error) {
	return mergeMapStrict(merged, parsed, "context", origins, filePath)
}

// mergeOriginTrackingMaps is a helper type for managing all origin tracking maps.
type mergeOriginTrackingMaps struct {
	Agents     map[string]string
	Skills     map[string]string
	Rules      map[string]string
	MCP        map[string]string
	Workflows  map[string]string
	Policies   map[string]string
	Blueprints map[string]string
	Contexts   map[string]string
	Settings   string // The file that first contributed settings
}

// copyProjectMetadata copies project metadata fields from src to dst if non-empty.
func copyProjectMetadata(dst, src *ast.ProjectConfig) {
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.Description != "" {
		dst.Description = src.Description
	}
	if src.Version != "" {
		dst.Version = src.Version
	}
	if src.Author != "" {
		dst.Author = src.Author
	}
	if src.Homepage != "" {
		dst.Homepage = src.Homepage
	}
	if src.Repository != "" {
		dst.Repository = src.Repository
	}
	if src.License != "" {
		dst.License = src.License
	}
	if src.BackupDir != "" {
		dst.BackupDir = src.BackupDir
	}
	// Propagate targets declared by kind: project documents.
	// This field uses yaml:"-" so it is not decoded from YAML
	// directly; only kind: project documents populate it.
	if len(src.Targets) > 0 {
		dst.Targets = src.Targets
	}
}

// mergeVersionAndProject merges version and project metadata from parsed into merged config.
func mergeVersionAndProject(merged *ast.XcaffoldConfig, p *ast.XcaffoldConfig) error {
	// Version merging
	if merged.Version != "" && p.Version != "" && merged.Version != p.Version {
		return fmt.Errorf("conflicting versions declared: %q vs %q", merged.Version, p.Version)
	}
	if p.Version != "" {
		merged.Version = p.Version
	}

	// Project metadata merging
	if p.Project != nil && p.Project.Name != "" {
		if merged.Project != nil && merged.Project.Name != "" && merged.Project.Name != p.Project.Name {
			return fmt.Errorf("multiple files declare project.name: %q vs %q", merged.Project.Name, p.Project.Name)
		}
		if merged.Project == nil {
			merged.Project = &ast.ProjectConfig{}
		}
		// Copy scalar metadata fields; ResourceScope is merged separately below.
		copyProjectMetadata(merged.Project, p.Project)
	}

	return nil
}

// mergeAllResourceMaps merges all resource kind maps from parsed into merged config.
func mergeAllResourceMaps(merged *ast.XcaffoldConfig, p *ast.XcaffoldConfig, f string, origins *mergeOriginTrackingMaps) error {
	var err error

	// Resource map merging
	merged.Agents, origins.Agents, err = mergeAgentsStrict(merged.Agents, p.Agents, origins.Agents, f)
	if err != nil {
		return err
	}

	merged.Skills, origins.Skills, err = mergeSkillsStrict(merged.Skills, p.Skills, origins.Skills, f)
	if err != nil {
		return err
	}

	merged.Rules, origins.Rules, err = mergeRulesStrict(merged.Rules, p.Rules, origins.Rules, f)
	if err != nil {
		return err
	}

	merged.MCP, origins.MCP, err = mergeMCPStrict(merged.MCP, p.MCP, origins.MCP, f)
	if err != nil {
		return err
	}

	merged.Workflows, origins.Workflows, err = mergeWorkflowsStrict(merged.Workflows, p.Workflows, origins.Workflows, f)
	if err != nil {
		return err
	}

	merged.Policies, origins.Policies, err = mergePoliciesStrict(merged.Policies, p.Policies, origins.Policies, f)
	if err != nil {
		return err
	}

	merged.Blueprints, origins.Blueprints, err = mergeBlueprintsStrict(merged.Blueprints, p.Blueprints, origins.Blueprints, f)
	if err != nil {
		return err
	}

	merged.Contexts, origins.Contexts, err = mergeContextsStrict(merged.Contexts, p.Contexts, origins.Contexts, f)
	if err != nil {
		return err
	}

	return nil
}

// mergeTestAndWarnings merges test config and parse warnings from parsed into merged config.
func mergeTestAndWarnings(merged *ast.XcaffoldConfig, p *ast.XcaffoldConfig) {
	// Accumulate parse warnings from each individual file parse.
	merged.ParseWarnings = append(merged.ParseWarnings, p.ParseWarnings...)

	// Overwrite test blocks (assuming only one file declares test config).
	// Test now lives in ProjectConfig.
	if p.Project != nil {
		pTest := p.Project.Test
		if pTest.CliPath != "" || pTest.JudgeModel != "" || pTest.Task != "" || pTest.MaxTurns != nil {
			if merged.Project == nil {
				merged.Project = &ast.ProjectConfig{}
			}
			merged.Project.Test = pTest
		}
	}
}

// mergeFileIntoAll merges a single parsed file into the merged configuration.
func mergeFileIntoAll(merged *ast.XcaffoldConfig, pf ParsedFile, origins *mergeOriginTrackingMaps) error {
	p := pf.Config
	f := pf.FilePath

	if err := mergeVersionAndProject(merged, p); err != nil {
		return err
	}

	// Extends field merging
	if p.Extends != "" {
		if merged.Extends != "" && merged.Extends != p.Extends {
			return fmt.Errorf("multiple files declare extends: %q vs %q", merged.Extends, p.Extends)
		}
		merged.Extends = p.Extends
	}

	if err := mergeAllResourceMaps(merged, p, f, origins); err != nil {
		return err
	}

	// Hooks are additive (merge named hook blocks).
	merged.Hooks = mergeNamedHooksAdditive(merged.Hooks, p.Hooks)

	mergeTestAndWarnings(merged, p)

	// Track which file first contributed non-empty settings.
	if origins.Settings == "" && len(p.Settings) > 0 {
		origins.Settings = f
	}

	// Deep merge settings map (conflicting scalar keys within the same named entry -> error).
	var err error
	merged.Settings, err = mergeSettingsMapStrict(merged.Settings, p.Settings, origins.Settings, f)
	if err != nil {
		return err
	}

	return nil
}

func mergeAllStrict(parsedFiles []ParsedFile) (*ast.XcaffoldConfig, error) {
	if len(parsedFiles) == 0 {
		return &ast.XcaffoldConfig{}, nil
	}
	merged := &ast.XcaffoldConfig{}

	origins := &mergeOriginTrackingMaps{
		Agents:     make(map[string]string),
		Skills:     make(map[string]string),
		Rules:      make(map[string]string),
		MCP:        make(map[string]string),
		Workflows:  make(map[string]string),
		Policies:   make(map[string]string),
		Blueprints: make(map[string]string),
		Contexts:   make(map[string]string),
	}

	for _, pf := range parsedFiles {
		if err := mergeFileIntoAll(merged, pf, origins); err != nil {
			return nil, err
		}
	}

	return merged, nil
}

// mergeMapStrict merges two maps, raising an error if the same key appears in both.
// Returns the merged map, origin tracking map, and any error.
func mergeMapStrict[K comparable, V any](base, child map[K]V, kind string, baseOrigins map[K]string, childFile string) (map[K]V, map[K]string, error) {
	if base == nil && child == nil {
		return nil, baseOrigins, nil
	}
	if base == nil {
		origins := make(map[K]string, len(child))
		for k := range child {
			origins[k] = childFile
		}
		return child, origins, nil
	}
	if child == nil {
		return base, baseOrigins, nil
	}
	merged := make(map[K]V, len(base)+len(child))
	origins := make(map[K]string, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
		origins[k] = baseOrigins[k]
	}
	for k, v := range child {
		if _, exists := merged[k]; exists {
			return nil, nil, fmt.Errorf("duplicate %s ID \"%v\" found in %s and %s", kind, k, filepath.Base(origins[k]), filepath.Base(childFile))
		}
		merged[k] = v
		origins[k] = childFile
	}
	return merged, origins, nil
}

// mergeHooksAdditive merges two HookConfig maps additively, appending handlers.
func mergeHooksAdditive(base, child ast.HookConfig) ast.HookConfig {
	if base == nil {
		return child
	}
	if child == nil {
		return base
	}
	merged := make(ast.HookConfig)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = append(merged[k], v...)
	}
	return merged
}

// mergeNamedHooksAdditive merges two map[string]NamedHookConfig values additively.
// Events within each named block are appended across base and child.
func mergeNamedHooksAdditive(base, child map[string]ast.NamedHookConfig) map[string]ast.NamedHookConfig {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	merged := make(map[string]ast.NamedHookConfig, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for name, nh := range child {
		if existing, ok := merged[name]; ok {
			existing.Events = mergeHooksAdditive(existing.Events, nh.Events)
			merged[name] = existing
		} else {
			merged[name] = nh
		}
	}
	return merged
}

// mergeSettingsMapStrict merges two map[string]SettingsConfig values from the same
// directory. Named entries are merged per-name using mergeSettingsStrict.
func mergeSettingsMapStrict(base, child map[string]ast.SettingsConfig, baseFile, childFile string) (map[string]ast.SettingsConfig, error) {
	if len(child) == 0 {
		return base, nil
	}
	if len(base) == 0 {
		return child, nil
	}
	merged := make(map[string]ast.SettingsConfig, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for name, cs := range child {
		if bs, ok := merged[name]; ok {
			result, err := mergeSettingsStrict(bs, cs, baseFile, childFile)
			if err != nil {
				return nil, err
			}
			merged[name] = result
		} else {
			merged[name] = cs
		}
	}
	return merged, nil
}

// mergeSettingsMapOverride merges two map[string]SettingsConfig for extends
// resolution. Child entries override base entries with the same name.
func mergeSettingsMapOverride(base, child map[string]ast.SettingsConfig) map[string]ast.SettingsConfig {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	merged := make(map[string]ast.SettingsConfig, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for name, cs := range child {
		if bs, ok := merged[name]; ok {
			merged[name] = mergeSettingsOverride(bs, cs)
		} else {
			merged[name] = cs
		}
	}
	return merged
}

// mergeConfigOverride is used for extends resolution where the child overrides the base entirely.
// Base resources (those not overridden by the child) are tagged Inherited=true so renderers
// can skip them during project-scope compilation - they are already compiled at global scope.
func mergeConfigOverride(base, child *ast.XcaffoldConfig) *ast.XcaffoldConfig {
	merged := &ast.XcaffoldConfig{
		Version: child.Version, // child overrides version
	}

	if merged.Version == "" {
		merged.Version = base.Version
	}

	if base.Project != nil || child.Project != nil {
		merged.Project = &ast.ProjectConfig{}
		if base.Project != nil {
			*merged.Project = *base.Project
		}
		if child.Project != nil {
			if child.Project.Name != "" {
				merged.Project.Name = child.Project.Name
			}
			if child.Project.Description != "" {
				merged.Project.Description = child.Project.Description
			}
			if child.Project.BackupDir != "" {
				merged.Project.BackupDir = child.Project.BackupDir
			}
			// Propagate targets from kind: project documents.
			if len(child.Project.Targets) > 0 {
				merged.Project.Targets = child.Project.Targets
			}
			// Test override
			if child.Project.Test.CliPath != "" {
				merged.Project.Test.CliPath = child.Project.Test.CliPath
			}
			if child.Project.Test.JudgeModel != "" {
				merged.Project.Test.JudgeModel = child.Project.Test.JudgeModel
			}
			if child.Project.Test.Task != "" {
				merged.Project.Test.Task = child.Project.Test.Task
			}
			if child.Project.Test.MaxTurns != nil {
				merged.Project.Test.MaxTurns = child.Project.Test.MaxTurns
			}

			// Project instructions fields. A set field on the child wins; an empty
			// field on the child preserves the base value (matches the same
			// convention applied to Name, Description, and other scalar fields above).
		}
	}

	merged.Extends = "" // after resolving, extends is empty

	// Tag all base resources as inherited so renderers skip them during project-scope
	// compilation. Resources the child declares (same ID) are child-owned and NOT tagged.
	merged.Agents = mergeAgentsOverrideInherited(base.Agents, child.Agents)
	merged.Skills = mergeSkillsOverrideInherited(base.Skills, child.Skills)
	merged.Rules = mergeRulesOverrideInherited(base.Rules, child.Rules)
	merged.MCP = mergeMCPOverrideInherited(base.MCP, child.MCP)
	merged.Workflows = mergeWorkflowsOverrideInherited(base.Workflows, child.Workflows)
	merged.Policies = mergeMapOverride(base.Policies, child.Policies)
	merged.Blueprints = mergeMapOverride(base.Blueprints, child.Blueprints)
	merged.Contexts = mergeContextsOverrideInherited(base.Contexts, child.Contexts)
	merged.Hooks = mergeNamedHooksAdditive(base.Hooks, child.Hooks)

	merged.Settings = mergeSettingsMapOverride(base.Settings, child.Settings)

	// Preserve parse warnings from the child (project-level); base (global) warnings are discarded.
	merged.ParseWarnings = append(merged.ParseWarnings, child.ParseWarnings...)

	return merged
}

// mergeMapOverride merges two maps where child values override base values completely.
func mergeMapOverride[K comparable, V any](base, child map[K]V) map[K]V {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[K]V)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v // child overrides base completely
	}
	return merged
}

// mergeMapOverrideInherited merges two maps where base resources are tagged
// Inherited=true. Child resources (which override base) take precedence and are
// NOT tagged. This is implemented per concrete type because Go generics cannot
// assign to struct fields through a type parameter without reflection.
func mergeAgentsOverrideInherited(base, child map[string]ast.AgentConfig) map[string]ast.AgentConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.AgentConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeContextsOverrideInherited(base, child map[string]ast.ContextConfig) map[string]ast.ContextConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.ContextConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeSkillsOverrideInherited(base, child map[string]ast.SkillConfig) map[string]ast.SkillConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.SkillConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeRulesOverrideInherited(base, child map[string]ast.RuleConfig) map[string]ast.RuleConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.RuleConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeMCPOverrideInherited(base, child map[string]ast.MCPConfig) map[string]ast.MCPConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.MCPConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeWorkflowsOverrideInherited(base, child map[string]ast.WorkflowConfig) map[string]ast.WorkflowConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.WorkflowConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

// ParseFile reads a .xcaf YAML configuration from the given path, resolving
// 'extends:' references recursively. Evaluated as a strict, single file entry point.
func ParseFile(path string) (*ast.XcaffoldConfig, error) {
	globalConfig, err := loadGlobalBase()
	if err != nil {
		return nil, fmt.Errorf("failed to load implicit global configuration: %w", err)
	}

	config, err := ParseFileExact(path)
	if err != nil {
		return nil, err
	}
	if config.Extends != "" {
		config, err = resolveExtends(filepath.Dir(path), config, nil, nil)
		if err != nil {
			return nil, err
		}
	}

	// Implicitly overlay the project configuration on top of the global base
	merged := mergeConfigOverride(globalConfig, config)

	if err := validateMerged(merged); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return merged, nil
}
