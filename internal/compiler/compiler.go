package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"github.com/saero-ai/xcaffold/providers"
	_ "github.com/saero-ai/xcaffold/providers/all" // registers all providers
	"gopkg.in/yaml.v3"
)

// Compile translates an XcaffoldConfig AST into platform-native files.
// target selects the output platform (see providers.PrimaryNames() for supported values).
// An empty target returns an error.
// blueprintName narrows compilation to the named blueprint's resource subset.
// If blueprintName is empty, all resources are compiled.
//
// When a project config has both root-level resources (global scope, from extends
// or implicit global loading) and project-level resources (inside the project: block),
// the compiler merges them before rendering. Project resources override global
// resources by ID. After merging, inherited resources are stripped so global
// configurations are not physically duplicated into local project directories.
func Compile(config *ast.XcaffoldConfig, baseDir string, target string, blueprintName string, varFile string) (*output.Output, []renderer.FidelityNote, error) {
	if target == "" {
		return nil, nil, fmt.Errorf("target is required; supported: %s", strings.Join(providers.PrimaryNames(), ", "))
	}

	if config.Project != nil {
		mergeResourceScope(&config.ResourceScope, &config.Project.ResourceScope)
	}

	if err := resolver.ResolveAttributes(config); err != nil {
		return nil, nil, fmt.Errorf("attribute resolution failed: %w", err)
	}

	// Load variables for DiscoverAgentMemory expansion
	vars, err := parser.LoadVariableStack(baseDir, target, varFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load variables: %w", err)
	}
	var envs map[string]string
	if config.Project != nil {
		envs = parser.LoadEnv(config.Project.AllowedEnvVars)
	}

	// Blueprint resolution: resolve extends chains and transitive deps before filtering.
	if len(config.Blueprints) > 0 {
		if err := blueprint.ResolveBlueprintExtends(config.Blueprints); err != nil {
			return nil, nil, fmt.Errorf("blueprint extends resolution failed: %w", err)
		}
		if errs := blueprint.ValidateBlueprintRefs(config.Blueprints, &config.ResourceScope); len(errs) > 0 {
			msgs := make([]string, len(errs))
			for i, e := range errs {
				msgs[i] = e.Error()
			}
			return nil, nil, fmt.Errorf("blueprint validation errors:\n%s", strings.Join(msgs, "\n"))
		}
	}
	if blueprintName != "" {
		if p, ok := config.Blueprints[blueprintName]; ok {
			if err := blueprint.ResolveTransitiveDeps(&p, &config.ResourceScope); err != nil {
				return nil, nil, fmt.Errorf("blueprint transitive dependency resolution failed: %w", err)
			}
			config.Blueprints[blueprintName] = p
		}
	}

	// Blueprint filtering: narrow the resource scope to the named blueprint's subset.
	if blueprintName != "" {
		var err error
		config, err = blueprint.ApplyBlueprint(config, blueprintName)
		if err != nil {
			return nil, nil, fmt.Errorf("blueprint filter failed: %w", err)
		}
	}

	r, err := ResolveRenderer(target)
	if err != nil {
		return nil, nil, err
	}

	// Context uniqueness validation: only when no blueprint is active. When a
	// blueprint is provided, its contexts: selector already narrows the set
	// to an explicit composition list and no ambiguity check is needed.
	if blueprintName == "" {
		if err := renderer.ValidateContextUniqueness(config.Contexts, []string{target}); err != nil {
			return nil, nil, fmt.Errorf("context validation failed: %w", err)
		}
	}

	overrideNotes := resolveTargetOverrides(config, target)

	config.Memory = DiscoverAgentMemory(baseDir, vars, envs)

	config.StripInherited()

	out, fidelityNotes, err := renderer.Orchestrate(r, config, baseDir)
	if err != nil {
		return nil, nil, err
	}
	return out, append(overrideNotes, fidelityNotes...), nil
}

// ResolveRenderer returns the TargetRenderer for the given target name.
func ResolveRenderer(target string) (renderer.TargetRenderer, error) {
	return providers.ResolveRenderer(target)
}

// mergeResourceScope overlays project-scoped resources onto the root scope.
// For map-based resources (agents, skills, rules, MCP, workflows), project
// entries override global entries with the same ID.
func mergeResourceScope(root, project *ast.ResourceScope) {
	root.Agents = mergeMap(root.Agents, project.Agents)
	root.Skills = mergeMap(root.Skills, project.Skills)
	root.Rules = mergeMap(root.Rules, project.Rules)
	root.MCP = mergeMap(root.MCP, project.MCP)
	root.Workflows = mergeMap(root.Workflows, project.Workflows)
}

// mergeMap copies base entries then overlays child entries (child wins on conflict).
func mergeMap[K comparable, V any](base, child map[K]V) map[K]V {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[K]V, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v
	}
	return merged
}

// OutputDir returns the target-specific root directory for compilation outputs
// (e.g. .claude, .cursor, .agents). Empty or unknown targets return an empty string.
func OutputDir(target string) string {
	if target == "" {
		return ""
	}
	r, err := ResolveRenderer(target)
	if err != nil {
		return ""
	}
	return r.OutputDir()
}

// DiscoverAgentMemory walks xcaf/agents/<id>/memory/ directories to discover
// memory entries from .md files. Returns a mapping of "agentID/memName" -> MemoryConfig.
// MEMORY.md index files and non-.md files are skipped. Optional YAML frontmatter
// (name, description) is parsed; missing fields fall back to the filename and
// first line of content respectively.
func DiscoverAgentMemory(baseDir string, vars map[string]interface{}, envs map[string]string) map[string]ast.MemoryConfig {
	result := make(map[string]ast.MemoryConfig)
	agentsDir := filepath.Join(baseDir, "xcaf", "agents")

	agentEntries, err := os.ReadDir(agentsDir)
	if err != nil {
		return result
	}

	for _, agentEntry := range agentEntries {
		if !agentEntry.IsDir() {
			continue
		}
		agentID := agentEntry.Name()
		memDir := filepath.Join(agentsDir, agentID, "memory")

		memFiles, err := os.ReadDir(memDir)
		if err != nil {
			continue
		}

		for _, memFile := range memFiles {
			if memFile.IsDir() {
				continue
			}
			fname := memFile.Name()
			if !strings.HasSuffix(fname, ".md") || fname == "MEMORY.md" {
				continue
			}

			data, err := os.ReadFile(filepath.Join(memDir, fname))
			if err != nil {
				continue
			}

			// Expand variables in memory content
			if len(vars) > 0 || len(envs) > 0 {
				expanded, err := resolver.ExpandVariables(data, vars, envs)
				if err == nil {
					data = expanded
				}
			}

			stem := strings.TrimSuffix(fname, ".md")
			name := stem
			desc := ""
			content := string(data)

			var front struct {
				Name        string `yaml:"name"`
				Description string `yaml:"description"`
			}
			if body, fmErr := parseFrontmatter(data, &front); fmErr == nil && front.Name != "" {
				name = front.Name
				desc = front.Description
				content = body
			} else {
				lines := strings.SplitN(strings.TrimSpace(content), "\n", 2)
				if len(lines) > 0 {
					desc = strings.TrimSpace(lines[0])
					runes := []rune(desc)
					if len(runes) > 120 {
						desc = string(runes[:120])
					}
				}
			}

			key := agentID + "/" + stem
			result[key] = ast.MemoryConfig{
				Name:        name,
				Description: desc,
				Content:     content,
				AgentRef:    agentID,
			}
		}
	}
	return result
}

// parseFrontmatter splits optional YAML frontmatter (delimited by "---\n") from
// the body of data, unmarshalling the frontmatter into v. Returns the trimmed
// body and nil on success. Returns an error when no frontmatter is present or
// when the YAML cannot be parsed.
func parseFrontmatter(data []byte, v interface{}) (string, error) {
	const delim = "---\n"
	s := string(data)
	if !strings.HasPrefix(s, delim) {
		return "", fmt.Errorf("no frontmatter")
	}
	rest := s[len(delim):]
	end := strings.Index(rest, delim)
	if end < 0 {
		return "", fmt.Errorf("unclosed frontmatter")
	}
	fm := rest[:end]
	body := strings.TrimSpace(rest[end+len(delim):])
	if err := yaml.Unmarshal([]byte(fm), v); err != nil {
		return "", err
	}
	return body, nil
}
