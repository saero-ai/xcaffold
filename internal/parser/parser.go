package parser

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// parseOption controls parsing behaviour per invocation.
type parseOption struct {
	globalScope bool
}

// parseOptionFunc configures a parseOption.
type parseOptionFunc func(*parseOption)

// withGlobalScope marks the parse as global scope, which allows absolute
// instructions_file paths (global configs reference files like ~/.claude/agents/*.md).
func withGlobalScope() parseOptionFunc {
	return func(o *parseOption) { o.globalScope = true }
}

func resolveParseOptions(opts []parseOptionFunc) parseOption {
	var o parseOption
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// Parse reads a .xcf YAML configuration from the given reader and returns a
// validated XcaffoldConfig. It treats the configuration as a complete, standalone file.
func Parse(r io.Reader) (*ast.XcaffoldConfig, error) {
	config, err := parsePartial(r)
	if err != nil {
		return nil, err
	}
	if err := validateMerged(config); err != nil {
		return nil, fmt.Errorf("invalid .xcf configuration: %w", err)
	}
	return config, nil
}

func parsePartial(r io.Reader, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read .xcf input: %w", err)
	}

	config := &ast.XcaffoldConfig{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	docIndex := 0

	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to parse .xcf YAML document %d: %w", docIndex, err)
		}

		// yaml.Decoder wraps each document in a DocumentNode; unwrap it.
		docNode := &node
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			docNode = node.Content[0]
		}

		kind := extractKind(docNode)

		switch kind {
		case "", "config":
			// Existing path: decode the node as a full/partial XcaffoldConfig.
			b, marshalErr := nodeToBytes(docNode)
			if marshalErr != nil {
				return nil, fmt.Errorf("failed to re-encode document %d: %w", docIndex, marshalErr)
			}
			dec := yaml.NewDecoder(bytes.NewReader(b))
			dec.KnownFields(true)

			if docIndex == 0 {
				// First document: decode directly into config.
				if decErr := dec.Decode(config); decErr != nil {
					return nil, fmt.Errorf("failed to parse .xcf YAML: %w", decErr)
				}
			} else {
				// Subsequent config document: decode into a partial and merge.
				var partial ast.XcaffoldConfig
				if decErr := dec.Decode(&partial); decErr != nil {
					return nil, fmt.Errorf("failed to parse .xcf YAML document %d: %w", docIndex, decErr)
				}
				if partial.Version != "" {
					config.Version = partial.Version
				}
				if partial.Project != nil {
					config.Project = partial.Project
				}
				for k, v := range partial.Agents {
					if config.Agents == nil {
						config.Agents = make(map[string]ast.AgentConfig)
					}
					config.Agents[k] = v
				}
				for k, v := range partial.Skills {
					if config.Skills == nil {
						config.Skills = make(map[string]ast.SkillConfig)
					}
					config.Skills[k] = v
				}
				for k, v := range partial.Rules {
					if config.Rules == nil {
						config.Rules = make(map[string]ast.RuleConfig)
					}
					config.Rules[k] = v
				}
				for k, v := range partial.Workflows {
					if config.Workflows == nil {
						config.Workflows = make(map[string]ast.WorkflowConfig)
					}
					config.Workflows[k] = v
				}
				for k, v := range partial.MCP {
					if config.MCP == nil {
						config.MCP = make(map[string]ast.MCPConfig)
					}
					config.MCP[k] = v
				}
			}

		case "agent", "skill", "rule", "workflow", "mcp", "project", "hooks", "settings":
			// Resource-kind document: route to the kind-aware parser.
			// Propagate the resource version to config.Version if not already set.
			if config.Version == "" {
				config.Version = extractVersion(docNode)
			}
			if parseErr := parseResourceDocument(docNode, kind, config, ""); parseErr != nil {
				return nil, parseErr
			}

		default:
			return nil, fmt.Errorf("unknown resource kind %q in document %d", kind, docIndex)
		}

		docIndex++
	}

	if docIndex == 0 {
		return nil, fmt.Errorf("failed to parse .xcf YAML: EOF")
	}

	o := resolveParseOptions(opts)
	if err := validatePartial(config, o.globalScope); err != nil {
		return nil, fmt.Errorf("invalid .xcf configuration part: %w", err)
	}
	return config, nil
}

// ParsedFile pairs a parsed partial config with its source file path.
type ParsedFile struct {
	Config   *ast.XcaffoldConfig
	FilePath string
}

// ParseDirectory recursively scans the given directory for all *.xcf files,
// parses them, merges them strictly (erroring on duplicate IDs), and then
// resolves 'extends:' chains.
func ParseDirectory(dir string) (*ast.XcaffoldConfig, error) {
	merged, err := parseDirectoryUnvalidated(dir)
	if err != nil {
		return nil, err
	}

	if err := validateMerged(merged); err != nil {
		return nil, fmt.Errorf("validation failed for project configuration: %w", err)
	}

	return merged, nil
}

// parseableKinds lists the kind values that isConfigFile accepts. An empty
// kind is treated as "config" for backward compatibility with legacy files
// that predate the kind: envelope field.
var parseableKinds = map[string]bool{
	"":         true,
	"config":   true,
	"project":  true,
	"agent":    true,
	"skill":    true,
	"rule":     true,
	"workflow": true,
	"mcp":      true,
	"hooks":    true,
	"settings": true,
}

// isConfigFile reads the kind: field from an .xcf file to determine if it
// should be parsed by the compiler. Returns true for config, resource-kind,
// and legacy (no-kind) files. Returns false for non-parseable kinds such as
// "registry" or "settings".
func isConfigFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var header struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &header); err != nil {
		return false
	}
	return parseableKinds[header.Kind]
}

func parseDirectoryUnvalidated(dir string) (*ast.XcaffoldConfig, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".xcf") {
			if isConfigFile(path) {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %q: %w", dir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *.xcf files found in directory %q", dir)
	}

	var parsedFiles []ParsedFile
	for _, f := range files {
		cfg, err := parseFileExact(f)
		if err != nil {
			return nil, err
		}
		parsedFiles = append(parsedFiles, ParsedFile{Config: cfg, FilePath: f})
	}

	globalConfig, err := loadGlobalBase()
	if err != nil {
		return nil, fmt.Errorf("failed to load implicit global configuration: %w", err)
	}

	merged, err := mergeAllStrict(parsedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config files in %q: %w", dir, err)
	}

	if merged.Extends != "" {
		merged, err = resolveExtends(dir, merged)
		if err != nil {
			return nil, err
		}
	}

	// Implicitly overlay the project configuration on top of the global base
	merged = mergeConfigOverride(globalConfig, merged)

	return merged, nil
}

func parseDirectoryRaw(dir string, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".xcf") {
			if isConfigFile(path) {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %q: %w", dir, err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *.xcf files found in directory %q", dir)
	}

	var parsedFiles []ParsedFile
	for _, f := range files {
		cfg, err := parseFileExact(f, opts...)
		if err != nil {
			return nil, err
		}
		parsedFiles = append(parsedFiles, ParsedFile{Config: cfg, FilePath: f})
	}

	merged, err := mergeAllStrict(parsedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config files in %q: %w", dir, err)
	}

	return merged, nil
}

func parseFileExact(path string, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config %q: %w", path, err)
	}
	defer f.Close()

	config, err := parsePartial(f, opts...)
	if err != nil {
		return nil, fmt.Errorf("error in %q: %w", path, err)
	}
	return config, nil
}

// loadGlobalBase implicitly discovers and loads the global configuration
// from ~/.xcaffold/ (or falls back to legacy ~/.claude/global.xcf).
// It returns an empty config if no global config is found.
// Resources loaded from this base are tagged as Inherited=true during merge.
func loadGlobalBase() (*ast.XcaffoldConfig, error) {
	if os.Getenv("XCAFFOLD_SKIP_GLOBAL") == "true" {
		return &ast.XcaffoldConfig{}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return &ast.XcaffoldConfig{}, nil // ignore errors, just no global
	}

	xcaffoldDir := filepath.Join(home, ".xcaffold")
	if stat, err := os.Stat(xcaffoldDir); err == nil && stat.IsDir() {
		// Parse the dir, but disable global loading to avoid infinite recursion!
		// parseDirectoryRaw natively parses a dir without applying global base.
		cfg, err := parseDirectoryRaw(xcaffoldDir, withGlobalScope())
		if err != nil {
			return nil, fmt.Errorf("failed to parse global directory %q: %w", xcaffoldDir, err)
		}
		// If the global config itself extends something, resolve it!
		if cfg.Extends != "" {
			visited := map[string]bool{xcaffoldDir: true}
			cfg, err = resolveExtendsRecursive(xcaffoldDir, cfg, visited)
			if err != nil {
				return nil, err
			}
		}
		return cfg, nil
	}

	return &ast.XcaffoldConfig{}, nil
}

// ParseFile reads a .xcf YAML configuration from the given path, resolving
// 'extends:' references recursively. Evaluated as a strict, single file entry point.
func ParseFile(path string) (*ast.XcaffoldConfig, error) {
	globalConfig, err := loadGlobalBase()
	if err != nil {
		return nil, fmt.Errorf("failed to load implicit global configuration: %w", err)
	}

	config, err := parseFileExact(path)
	if err != nil {
		return nil, err
	}
	if config.Extends != "" {
		config, err = resolveExtends(filepath.Dir(path), config)
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

func resolveExtends(contextDir string, config *ast.XcaffoldConfig) (*ast.XcaffoldConfig, error) {
	visited := make(map[string]bool)
	return resolveExtendsRecursive(contextDir, config, visited)
}

//nolint:gocyclo
func resolveExtendsRecursive(contextDir string, config *ast.XcaffoldConfig, visited map[string]bool) (*ast.XcaffoldConfig, error) {
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

			baseConfig, err := parseDirectoryRaw(xcaffoldDir, withGlobalScope())
			if err != nil {
				return nil, fmt.Errorf("failed to parse global directory %q: %w", xcaffoldDir, err)
			}
			if baseConfig.Extends != "" {
				baseConfig, err = resolveExtendsRecursive(xcaffoldDir, baseConfig, visited)
				if err != nil {
					return nil, err
				}
			}
			return mergeConfigOverride(baseConfig, config), nil
		}

		legacyPath := filepath.Join(home, ".claude", "global.xcf")
		if _, err := os.Stat(legacyPath); err == nil {
			fmt.Fprintf(os.Stderr, "WARNING: extends: global resolved from legacy path %s -- run 'xcaffold migrate' to move to %s\n", legacyPath, xcaffoldDir)
			basePath = legacyPath
		} else {
			return nil, fmt.Errorf("could not resolve 'extends: global': no global config found")
		}
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

	parsed, err := parseFileExact(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load base config %q: %w", config.Extends, err)
	}

	baseConfig, err := resolveExtendsRecursive(filepath.Dir(absPath), parsed, visited)
	if err != nil {
		return nil, err
	}

	return mergeConfigOverride(baseConfig, config), nil
}

// Merge operations

// mergeAllStrict is used to merge files living in the same directory.
// Duplicate maps (like Agents, Skills, etc.) cause errors.
//
//nolint:gocyclo
func mergeAllStrict(parsedFiles []ParsedFile) (*ast.XcaffoldConfig, error) {
	if len(parsedFiles) == 0 {
		return &ast.XcaffoldConfig{}, nil
	}
	merged := &ast.XcaffoldConfig{}

	agentOrigins := map[string]string{}
	skillOrigins := map[string]string{}
	ruleOrigins := map[string]string{}
	mcpOrigins := map[string]string{}
	workflowOrigins := map[string]string{}
	settingsOrigin := ""
	localOrigin := ""

	for _, pf := range parsedFiles {
		p := pf.Config
		f := pf.FilePath
		var err error

		if merged.Version != "" && p.Version != "" && merged.Version != p.Version {
			return nil, fmt.Errorf("conflicting versions declared: %q vs %q", merged.Version, p.Version)
		}
		if p.Version != "" {
			merged.Version = p.Version
		}

		if p.Project != nil && p.Project.Name != "" {
			if merged.Project != nil && merged.Project.Name != "" && merged.Project.Name != p.Project.Name {
				return nil, fmt.Errorf("multiple files declare project.name: %q vs %q", merged.Project.Name, p.Project.Name)
			}
			if merged.Project == nil {
				merged.Project = &ast.ProjectConfig{}
			}
			// Copy scalar metadata fields; Local and ResourceScope are merged separately below.
			if p.Project.Name != "" {
				merged.Project.Name = p.Project.Name
			}
			if p.Project.Description != "" {
				merged.Project.Description = p.Project.Description
			}
			if p.Project.Version != "" {
				merged.Project.Version = p.Project.Version
			}
			if p.Project.Author != "" {
				merged.Project.Author = p.Project.Author
			}
			if p.Project.Homepage != "" {
				merged.Project.Homepage = p.Project.Homepage
			}
			if p.Project.Repository != "" {
				merged.Project.Repository = p.Project.Repository
			}
			if p.Project.License != "" {
				merged.Project.License = p.Project.License
			}
			if p.Project.BackupDir != "" {
				merged.Project.BackupDir = p.Project.BackupDir
			}
			// Propagate targets and ref lists declared by kind: project documents.
			// These fields use yaml:"-" so they are not decoded by the legacy
			// kind: config path; only kind: project documents populate them.
			if len(p.Project.Targets) > 0 {
				merged.Project.Targets = p.Project.Targets
			}
			if len(p.Project.AgentRefs) > 0 {
				merged.Project.AgentRefs = p.Project.AgentRefs
			}
			if len(p.Project.SkillRefs) > 0 {
				merged.Project.SkillRefs = p.Project.SkillRefs
			}
			if len(p.Project.RuleRefs) > 0 {
				merged.Project.RuleRefs = p.Project.RuleRefs
			}
			if len(p.Project.WorkflowRefs) > 0 {
				merged.Project.WorkflowRefs = p.Project.WorkflowRefs
			}
			if len(p.Project.MCPRefs) > 0 {
				merged.Project.MCPRefs = p.Project.MCPRefs
			}
		}

		if p.Extends != "" {
			if merged.Extends != "" && merged.Extends != p.Extends {
				return nil, fmt.Errorf("multiple files declare extends: %q vs %q", merged.Extends, p.Extends)
			}
			merged.Extends = p.Extends
		}

		merged.Agents, agentOrigins, err = mergeMapStrict(merged.Agents, p.Agents, "agent", agentOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.Skills, skillOrigins, err = mergeMapStrict(merged.Skills, p.Skills, "skill", skillOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.Rules, ruleOrigins, err = mergeMapStrict(merged.Rules, p.Rules, "rule", ruleOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.MCP, mcpOrigins, err = mergeMapStrict(merged.MCP, p.MCP, "mcp", mcpOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.Workflows, workflowOrigins, err = mergeMapStrict(merged.Workflows, p.Workflows, "workflow", workflowOrigins, f)
		if err != nil {
			return nil, err
		}

		// Hooks are additive (append handlers)
		merged.Hooks = mergeHooksAdditive(merged.Hooks, p.Hooks)

		// Overwrite test blocks (assuming only one file declares test config).
		// Test now lives in ProjectConfig.
		if p.Project != nil {
			pTest := p.Project.Test
			if pTest.CliPath != "" || pTest.ClaudePath != "" || pTest.JudgeModel != "" {
				if merged.Project == nil {
					merged.Project = &ast.ProjectConfig{}
				}
				merged.Project.Test = pTest
			}
		}

		// Track which file first contributed non-empty settings/local.
		if settingsOrigin == "" && !isEmptySettings(p.Settings) {
			settingsOrigin = f
		}
		if p.Project != nil && localOrigin == "" && !isEmptySettings(p.Project.Local) {
			localOrigin = f
		}

		// Deep merge settings block (conflicting keys → error).
		merged.Settings, err = mergeSettingsStrict(merged.Settings, p.Settings, settingsOrigin, f)
		if err != nil {
			return nil, err
		}
		// Deep merge local block (now lives in ProjectConfig).
		if p.Project != nil {
			if merged.Project == nil {
				merged.Project = &ast.ProjectConfig{}
			}
			merged.Project.Local, err = mergeSettingsStrict(merged.Project.Local, p.Project.Local, localOrigin, f)
			if err != nil {
				return nil, err
			}
		}
	}
	return merged, nil
}

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

// mergeConfigOverride is used for extends resolution where the child overrides the base entirely.
// Base resources (those not overridden by the child) are tagged Inherited=true so renderers
// can skip them during project-scope compilation — they are already compiled at global scope.
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
			// Propagate targets and ref lists from kind: project documents.
			if len(child.Project.Targets) > 0 {
				merged.Project.Targets = child.Project.Targets
			}
			if len(child.Project.AgentRefs) > 0 {
				merged.Project.AgentRefs = child.Project.AgentRefs
			}
			if len(child.Project.SkillRefs) > 0 {
				merged.Project.SkillRefs = child.Project.SkillRefs
			}
			if len(child.Project.RuleRefs) > 0 {
				merged.Project.RuleRefs = child.Project.RuleRefs
			}
			if len(child.Project.WorkflowRefs) > 0 {
				merged.Project.WorkflowRefs = child.Project.WorkflowRefs
			}
			if len(child.Project.MCPRefs) > 0 {
				merged.Project.MCPRefs = child.Project.MCPRefs
			}
			// Test override
			if child.Project.Test.CliPath != "" {
				merged.Project.Test.CliPath = child.Project.Test.CliPath
			}
			if child.Project.Test.ClaudePath != "" {
				merged.Project.Test.ClaudePath = child.Project.Test.ClaudePath
			}
			if child.Project.Test.JudgeModel != "" {
				merged.Project.Test.JudgeModel = child.Project.Test.JudgeModel
			}
			// Local settings override
			var baseLocal ast.SettingsConfig
			if base.Project != nil {
				baseLocal = base.Project.Local
			}
			merged.Project.Local = mergeSettingsOverride(baseLocal, child.Project.Local)
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
	merged.Hooks = mergeHooksAdditive(base.Hooks, child.Hooks)

	merged.Settings = mergeSettingsOverride(base.Settings, child.Settings)

	return merged
}

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

// Validations

func validateID(kind, id string) error {
	if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return fmt.Errorf("%s id contains invalid characters: %q", kind, id)
	}
	return nil
}

var knownTools = map[string]bool{
	"Read": true, "Write": true, "Edit": true, "MultiEdit": true,
	"Bash": true, "Glob": true, "Grep": true, "LS": true,
	"WebFetch": true, "WebSearch": true,
	"TodoRead": true, "TodoWrite": true,
	"NotebookRead": true, "NotebookEdit": true,
	"Task": true, "Computer": true, "AskUserQuestion": true,
	"Agent": true, "ExitPlanMode": true, "EnterPlanMode": true,
	"mcp": true,
}

var validHookEvents = map[string]bool{
	"PreToolUse": true, "PostToolUse": true, "PostToolUseFailure": true,
	"PermissionRequest": true, "PermissionDenied": true,
	"SessionStart": true, "SessionEnd": true,
	"UserPromptSubmit": true, "Stop": true, "StopFailure": true,
	"SubagentStart": true, "SubagentStop": true, "TeammateIdle": true,
	"TaskCreated": true, "TaskCompleted": true,
	"PreCompact": true, "PostCompact": true,
	"InstructionsLoaded": true, "ConfigChange": true,
	"CwdChanged": true, "FileChanged": true,
	"WorktreeCreate": true, "WorktreeRemove": true,
	"Elicitation": true, "ElicitationResult": true,
	"Notification": true,
}

func validatePartial(c *ast.XcaffoldConfig, globalScope bool) error {
	if err := validateIDs(c); err != nil {
		return err
	}
	if err := validateHookEvents(c.Hooks); err != nil {
		return err
	}
	if err := validateInstructions(c, globalScope); err != nil {
		return err
	}
	return nil
}

func validateMerged(c *ast.XcaffoldConfig) error {
	if err := validateBase(c); err != nil {
		return err
	}
	if err := validateCrossReferences(c); err != nil {
		return err
	}
	if err := validatePermissions(c); err != nil {
		return err
	}
	return nil
}

// parsePermissionRule parses a permission rule string of the form "ToolName" or
// "ToolName(pattern)". It applies strings.TrimSpace to both the tool name and
// the pattern. Returns (toolName, pattern, nil) on success, or ("", "", err).
func parsePermissionRule(rule string) (toolName, pattern string, err error) {
	idx := strings.Index(rule, "(")
	if idx == -1 {
		// bare tool name
		name := strings.TrimSpace(rule)
		if name == "" {
			return "", "", fmt.Errorf("permissions: empty rule string")
		}
		return name, "", nil
	}
	// has a pattern
	name := strings.TrimSpace(rule[:idx])
	rest := rule[idx+1:]
	if !strings.HasSuffix(rest, ")") {
		return "", "", fmt.Errorf("permissions: malformed rule %q — missing closing parenthesis", rule)
	}
	pat := strings.TrimSpace(rest[:len(rest)-1])
	if pat == "" {
		return "", "", fmt.Errorf("permissions: malformed rule %q — empty pattern", rule)
	}
	return name, pat, nil
}

// validatePermissions validates permission rule strings in settings.permissions
// and checks for agent/settings contradictions.
//
//nolint:gocyclo
func validatePermissions(c *ast.XcaffoldConfig) error {
	if c.Settings.Permissions == nil {
		return nil
	}
	p := c.Settings.Permissions

	allowSet := make(map[string]bool)
	denySet := make(map[string]bool)
	askSet := make(map[string]bool)

	for _, rule := range p.Allow {
		name, _, err := parsePermissionRule(rule)
		if err != nil {
			return fmt.Errorf("invalid .xcf configuration: %w", err)
		}
		if !knownTools[name] {
			return fmt.Errorf("permissions: unknown tool %q in allow rule %q", name, rule)
		}
		allowSet[rule] = true
	}
	for _, rule := range p.Deny {
		name, _, err := parsePermissionRule(rule)
		if err != nil {
			return fmt.Errorf("invalid .xcf configuration: %w", err)
		}
		if !knownTools[name] {
			return fmt.Errorf("permissions: unknown tool %q in deny rule %q", name, rule)
		}
		denySet[rule] = true
	}
	for _, rule := range p.Ask {
		name, _, err := parsePermissionRule(rule)
		if err != nil {
			return fmt.Errorf("invalid .xcf configuration: %w", err)
		}
		if !knownTools[name] {
			return fmt.Errorf("permissions: unknown tool %q in ask rule %q", name, rule)
		}
		askSet[rule] = true
	}

	// Contradiction checks
	for rule := range allowSet {
		if denySet[rule] {
			return fmt.Errorf("permissions: rule %q appears in both allow and deny", rule)
		}
		if askSet[rule] {
			return fmt.Errorf("permissions: rule %q appears in both allow and ask", rule)
		}
	}

	// Agent cross-reference checks
	for agentID, agent := range c.Agents {
		// disallowedTools vs settings.permissions.allow
		for _, tool := range agent.DisallowedTools {
			for rule := range allowSet {
				ruleName, _, _ := parsePermissionRule(rule)
				if ruleName == tool {
					return fmt.Errorf("agent %q: tool %q is in disallowedTools but also in settings.permissions.allow", agentID, tool)
				}
			}
		}
		// agent.tools vs settings.permissions.deny (bare deny only)
		for _, tool := range agent.Tools {
			if denySet[tool] {
				return fmt.Errorf("agent %q: tool %q is required by agent but is unconditionally denied in settings.permissions.deny", agentID, tool)
			}
		}
	}

	return nil
}

func validateBase(c *ast.XcaffoldConfig) error {
	if c.Version == "" {
		return fmt.Errorf("version is required (e.g. \"1.0\")")
	}

	if c.Extends == "" && c.Project != nil {
		name := strings.TrimSpace(c.Project.Name)
		if name == "" {
			return fmt.Errorf("project.name is required and must not be empty unless extending another config")
		}
	}
	return nil
}

func validateResourceIDs[T any](resources map[string]T, kind string) error {
	for id := range resources {
		if err := validateID(kind, id); err != nil {
			return err
		}
	}
	return nil
}

func validateIDs(c *ast.XcaffoldConfig) error {
	if err := validateResourceIDs(c.Agents, "agent"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Skills, "skill"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Rules, "rule"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Hooks, "hook"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.MCP, "mcp"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Workflows, "workflow"); err != nil {
		return err
	}
	return nil
}

func validateHookEvents(hooks ast.HookConfig) error {
	for event := range hooks {
		if !validHookEvents[event] {
			return fmt.Errorf("unknown hook event %q; see documentation for supported lifecycle events", event)
		}
	}
	return nil
}

func validateInstructions(c *ast.XcaffoldConfig, globalScope bool) error {
	for id, agent := range c.Agents {
		if err := validateInstructionOrFile("agent", id, agent.Instructions, agent.InstructionsFile, globalScope); err != nil {
			return err
		}
	}
	for id, skill := range c.Skills {
		if err := validateInstructionOrFile("skill", id, skill.Instructions, skill.InstructionsFile, globalScope); err != nil {
			return err
		}
	}
	for id, rule := range c.Rules {
		if err := validateInstructionOrFile("rule", id, rule.Instructions, rule.InstructionsFile, globalScope); err != nil {
			return err
		}
	}
	for id, wf := range c.Workflows {
		if err := validateInstructionOrFile("workflow", id, wf.Instructions, wf.InstructionsFile, globalScope); err != nil {
			return err
		}
	}
	return nil
}

func validateInstructionOrFile(kind, id, inst, file string, globalScope bool) error {
	if inst != "" && file != "" {
		return fmt.Errorf("%s %q: instructions and instructions_file are mutually exclusive; set one or the other", kind, id)
	}
	return validateInstructionsFile(kind, id, file, globalScope)
}

func validateCrossReferences(c *ast.XcaffoldConfig) error {
	for agentID, agent := range c.Agents {
		for _, skillID := range agent.Skills {
			if _, ok := c.Skills[skillID]; !ok {
				return fmt.Errorf("agent %q references undefined skill %q", agentID, skillID)
			}
		}
		for _, ruleID := range agent.Rules {
			if _, ok := c.Rules[ruleID]; !ok {
				return fmt.Errorf("agent %q references undefined rule %q", agentID, ruleID)
			}
		}
		for _, mcpID := range agent.MCP {
			if _, ok := c.MCP[mcpID]; !ok {
				return fmt.Errorf("agent %q references undefined mcp server %q", agentID, mcpID)
			}
		}
	}

	return nil
}

// Diagnostic represents a single validation finding returned by ValidateFile.
// Severity is either "error" or "warning". Errors cause non-zero exits in
// xcaffold validate; warnings are informational only.
type Diagnostic struct {
	Severity string // "error" or "warning"
	Message  string
}

// knownPlugins is the hardcoded registry of officially supported plugin IDs.
// Plugin validation produces warnings only — custom plugins are not errors.
var knownPlugins = map[string]bool{
	"commit-commands":   true,
	"security-guidance": true,
	"code-review":       true,
	"pr-review-toolkit": true,
}

// ValidateFile parses the .xcf file at path, runs file-existence checks and
// plugin validation, and returns all diagnostics. ParseFile already runs
// validateCrossReferences internally, so this function does not duplicate it.
func ValidateFile(path string) []Diagnostic {
	config, err := ParseFile(path)
	if err != nil {
		return []Diagnostic{{Severity: "error", Message: err.Error()}}
	}
	var diags []Diagnostic
	diags = append(diags, validateFileRefs(config, filepath.Dir(path))...)
	diags = append(diags, validatePlugins(config)...)
	return diags
}

// validateFileRefs checks that instructions_file paths and skill references
// exist on disk, and detects duplicate IDs across resource types.
//
//nolint:gocyclo
func validateFileRefs(c *ast.XcaffoldConfig, baseDir string) []Diagnostic {
	var diags []Diagnostic

	// Skill subdirectory file sets: warn on missing files for references, scripts, assets
	for id, skill := range c.Skills {
		for _, subdirPaths := range []struct {
			subdir string
			paths  []string
		}{
			{"references", skill.References},
			{"scripts", skill.Scripts},
			{"assets", skill.Assets},
		} {
			for _, ref := range subdirPaths.paths {
				if ref == "" {
					continue
				}
				abs := filepath.Join(baseDir, ref)
				if _, err := os.Stat(abs); os.IsNotExist(err) {
					diags = append(diags, Diagnostic{
						Severity: "warning",
						Message:  fmt.Sprintf("skill %q %s file that does not exist: %q", id, subdirPaths.subdir, ref),
					})
				}
			}
		}
	}

	// instructions_file existence: error on missing files
	checkInstrFile := func(kind, id, instrFile string) {
		if instrFile == "" {
			return
		}
		abs := filepath.Join(baseDir, instrFile)
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			diags = append(diags, Diagnostic{
				Severity: "error",
				Message:  fmt.Sprintf("%s %q instructions_file not found: %q", kind, id, instrFile),
			})
		}
	}

	for id, agent := range c.Agents {
		checkInstrFile("agent", id, agent.InstructionsFile)
	}
	for id, skill := range c.Skills {
		checkInstrFile("skill", id, skill.InstructionsFile)
	}
	for id, rule := range c.Rules {
		checkInstrFile("rule", id, rule.InstructionsFile)
	}
	for id, wf := range c.Workflows {
		checkInstrFile("workflow", id, wf.InstructionsFile)
	}

	// Duplicate ID check across resource types
	seen := make(map[string][]string) // id -> []resourceType
	for id := range c.Agents {
		seen[id] = append(seen[id], "agent")
	}
	for id := range c.Skills {
		seen[id] = append(seen[id], "skill")
	}
	for id := range c.Rules {
		seen[id] = append(seen[id], "rule")
	}
	for id := range c.Workflows {
		seen[id] = append(seen[id], "workflow")
	}
	for id, types := range seen {
		if len(types) > 1 {
			diags = append(diags, Diagnostic{
				Severity: "warning",
				Message:  fmt.Sprintf("ID %q is used in both %s and %s; this may cause confusion", id, types[0], types[1]),
			})
		}
	}

	return diags
}

// validatePlugins checks settings.enabledPlugins and local.enabledPlugins
// against the knownPlugins registry. Unknown plugins produce warnings only.
func validatePlugins(c *ast.XcaffoldConfig) []Diagnostic {
	var diags []Diagnostic
	check := func(plugins map[string]bool, block string) {
		for id := range plugins {
			if !knownPlugins[id] {
				diags = append(diags, Diagnostic{
					Severity: "warning",
					Message: fmt.Sprintf(
						"%s.enabledPlugins: unknown plugin %q; known plugins: commit-commands, security-guidance, code-review, pr-review-toolkit",
						block, id,
					),
				})
			}
		}
	}
	check(c.Settings.EnabledPlugins, "settings")
	if c.Project != nil {
		check(c.Project.Local.EnabledPlugins, "local")
	}
	return diags
}

// reservedOutputPrefixes are compiler output directories. instructions_file paths
// starting with these prefixes create circular dependencies where the compiler
// reads its own output.
var reservedOutputPrefixes = []string{".claude/", ".cursor/", ".agents/", ".antigravity/"}

func validateInstructionsFile(kind, id, path string, globalScope bool) error {
	if path == "" {
		return nil
	}
	if filepath.IsAbs(path) && !globalScope {
		return fmt.Errorf("%s %q: instructions_file must be a relative path, got absolute path %q", kind, id, path)
	}
	if strings.ContainsAny(path, "\\") || strings.Contains(path, "..") {
		return fmt.Errorf("%s %q: instructions_file contains invalid path characters: %q", kind, id, path)
	}
	// Skip reserved-output-prefix check for absolute paths (they are outside project dir).
	if filepath.IsAbs(path) {
		return nil
	}
	cleaned := filepath.Clean(path)
	for _, prefix := range reservedOutputPrefixes {
		if strings.HasPrefix(cleaned, filepath.Clean(prefix)) {
			return fmt.Errorf("%s %q: instructions_file %q references compiler output directory %s — this creates a circular dependency", kind, id, path, prefix)
		}
	}
	return nil
}
