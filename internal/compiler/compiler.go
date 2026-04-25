package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/antigravity"
	"github.com/saero-ai/xcaffold/internal/renderer/claude"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
	"github.com/saero-ai/xcaffold/internal/renderer/cursor"
	"github.com/saero-ai/xcaffold/internal/renderer/gemini"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"gopkg.in/yaml.v3"
)

const (
	TargetClaude      = "claude"
	TargetCursor      = "cursor"
	TargetAntigravity = "antigravity"
	TargetCopilot     = "copilot"
	TargetGemini      = "gemini"
)

// Output is an alias for output.Output, preserved for backward compatibility.
// All callers that reference compiler.Output continue to work without changes.
type Output = output.Output

// Compile translates an XcaffoldConfig AST into platform-native files.
// target selects the output platform: "claude", "cursor", "antigravity", "copilot", "gemini".
// An empty target defaults to "claude" for backward compatibility.
// blueprintName narrows compilation to the named blueprint's resource subset.
// If blueprintName is empty, all resources are compiled.
//
// When a project config has both root-level resources (global scope, from extends
// or implicit global loading) and project-level resources (inside the project: block),
// the compiler merges them before rendering. Project resources override global
// resources by ID. After merging, inherited resources are stripped so global
// configurations are not physically duplicated into local project directories.
func Compile(config *ast.XcaffoldConfig, baseDir string, target string, blueprintName string) (*Output, []renderer.FidelityNote, error) {
	if target == "" {
		return nil, nil, fmt.Errorf("target is required; supported: claude, cursor, antigravity, copilot, gemini")
	}

	if config.Project != nil {
		mergeResourceScope(&config.ResourceScope, &config.Project.ResourceScope)
	}

	if err := resolver.ResolveAttributes(config); err != nil {
		return nil, nil, fmt.Errorf("attribute resolution failed: %w", err)
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
			blueprint.ResolveTransitiveDeps(&p, &config.ResourceScope)
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

	config.StripInherited()

	r, err := resolveRenderer(target)
	if err != nil {
		return nil, nil, err
	}
	return renderer.Orchestrate(r, config, baseDir)
}

// resolveRenderer returns the TargetRenderer for the given target name.
func resolveRenderer(target string) (renderer.TargetRenderer, error) {
	switch target {
	case TargetClaude:
		return claude.New(), nil
	case TargetCursor:
		return cursor.New(), nil
	case TargetAntigravity:
		return antigravity.New(), nil
	case TargetCopilot:
		return copilot.New(), nil
	case TargetGemini:
		return gemini.New(), nil
	default:
		return nil, fmt.Errorf("unsupported target %q: supported targets are \"claude\", \"cursor\", \"antigravity\", \"copilot\", \"gemini\"", target)
	}
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
	r, err := resolveRenderer(target)
	if err != nil {
		return ""
	}
	return r.OutputDir()
}

// DiscoverAgentMemory walks xcf/agents/<id>/memory/ directories to discover
// memory entries from .md files. Returns a mapping of "agentID/memName" -> MemoryConfig.
// MEMORY.md index files and non-.md files are skipped. Optional YAML frontmatter
// (name, description) is parsed; missing fields fall back to the filename and
// first line of content respectively.
func DiscoverAgentMemory(baseDir string) map[string]ast.MemoryConfig {
	result := make(map[string]ast.MemoryConfig)
	agentsDir := filepath.Join(baseDir, "xcf", "agents")

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
