package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/spf13/cobra"
)

var listJSONFlag bool

type listResourceJSON struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Target string `json:"target"`
	Source string `json:"source"`
}

func init() {
	listCmd.Flags().BoolVar(&listJSONFlag, "json", false, "Output resources as a JSON array")
}

func listProviderTarget(config *ast.XcaffoldConfig) string {
	if targetFlag != "" {
		return targetFlag
	}
	if config.Project != nil && len(config.Project.Targets) > 0 {
		return config.Project.Targets[0]
	}
	return ""
}

func printListJSON(cmd *cobra.Command, config *ast.XcaffoldConfig, baseDir string) error {
	entries := collectListResourceEntries(config, baseDir, listProviderTarget(config))
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal list JSON: %w", err)
	}
	cmd.Println(string(b))
	return nil
}

func collectListResourceEntries(config *ast.XcaffoldConfig, baseDir, target string) []listResourceJSON {
	var entries []listResourceJSON
	add := func(kind string, names []string, providerFn func(string) string) {
		sort.Strings(names)
		for _, name := range names {
			provider := target
			if providerFn != nil {
				if p := providerFn(name); p != "" {
					provider = p
				}
			}
			entries = append(entries, listResourceJSON{
				Kind:   kind,
				Name:   name,
				Target: provider,
				Source: resolveListResourceSource(baseDir, kind, name, provider),
			})
		}
	}

	if listHasFilter() {
		if listFilterAgent != "" {
			add("agent", sortedMapKeys(filterMapByName(config.Agents, listFilterAgent)), agentSourceProvider(config))
		}
		if listFilterSkill != "" {
			add("skill", sortedMapKeys(filterMapByName(config.Skills, listFilterSkill)), skillSourceProvider(config))
		}
		if listFilterRule != "" {
			add("rule", sortedMapKeys(filterMapByName(config.Rules, listFilterRule)), ruleSourceProvider(config))
		}
		if listFilterWorkflow != "" {
			add("workflow", sortedMapKeys(filterMapByName(config.Workflows, listFilterWorkflow)), workflowSourceProvider(config))
		}
		if listFilterMCP != "" {
			add("mcp", sortedMapKeys(filterMapByName(config.MCP, listFilterMCP)), mcpSourceProvider(config))
		}
		if listFilterContext != "" {
			add("context", sortedMapKeys(filterMapByName(config.Contexts, listFilterContext)), contextSourceProvider(config))
		}
		if listFilterHook {
			add("hook", sortedMapKeys(config.Hooks), nil)
		}
		if listFilterSetting {
			add("setting", sortedMapKeys(config.Settings), nil)
		}
		return entries
	}

	add("agent", sortedMapKeys(config.Agents), agentSourceProvider(config))
	add("skill", sortedMapKeys(config.Skills), skillSourceProvider(config))
	add("rule", sortedMapKeys(config.Rules), ruleSourceProvider(config))
	add("workflow", sortedMapKeys(config.Workflows), workflowSourceProvider(config))
	add("mcp", sortedMapKeys(config.MCP), mcpSourceProvider(config))
	add("context", sortedMapKeys(config.Contexts), contextSourceProvider(config))
	add("hook", sortedMapKeys(config.Hooks), nil)
	add("setting", sortedMapKeys(config.Settings), nil)
	return entries
}

func agentSourceProvider(config *ast.XcaffoldConfig) func(string) string {
	return func(id string) string { return config.Agents[id].SourceProvider }
}

func skillSourceProvider(config *ast.XcaffoldConfig) func(string) string {
	return func(id string) string { return config.Skills[id].SourceProvider }
}

func ruleSourceProvider(config *ast.XcaffoldConfig) func(string) string {
	return func(id string) string { return config.Rules[id].SourceProvider }
}

func workflowSourceProvider(config *ast.XcaffoldConfig) func(string) string {
	return func(id string) string { return config.Workflows[id].SourceProvider }
}

func mcpSourceProvider(config *ast.XcaffoldConfig) func(string) string {
	return func(id string) string { return config.MCP[id].SourceProvider }
}

func contextSourceProvider(config *ast.XcaffoldConfig) func(string) string {
	return func(id string) string { return config.Contexts[id].SourceProvider }
}

func resolveListResourceSource(baseDir, kind, name, target string) string {
	for _, rel := range listResourceSourceCandidates(kind, name, target) {
		if _, err := os.Stat(filepath.Join(baseDir, rel)); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	candidates := listResourceSourceCandidates(kind, name, target)
	if len(candidates) == 0 {
		return ""
	}
	return filepath.ToSlash(candidates[0])
}

func listResourceSourceCandidates(kind, name, target string) []string {
	dir, baseFile := listResourceDirAndFile(kind, name)
	if dir == "" {
		return nil
	}
	var out []string
	if target != "" {
		stem := strings.TrimSuffix(baseFile, ".xcaf")
		out = append(out, filepath.Join(dir, stem+"."+target+".xcaf"))
	}
	out = append(out, filepath.Join(dir, baseFile))
	return out
}

func listResourceDirAndFile(kind, name string) (dir, file string) {
	switch kind {
	case "agent":
		return filepath.Join("xcaf", "agents", name), "agent.xcaf"
	case "skill":
		return filepath.Join("xcaf", "skills", name), "skill.xcaf"
	case "rule":
		return filepath.Join("xcaf", "rules", filepath.FromSlash(name)), "rule.xcaf"
	case "workflow":
		return filepath.Join("xcaf", "workflows", name), "workflow.xcaf"
	case "mcp":
		return filepath.Join("xcaf", "mcp", name), "mcp.xcaf"
	case "context":
		return filepath.Join("xcaf", "context", name), "context.xcaf"
	case "hook":
		return filepath.Join("xcaf", "hooks", name), "hooks.xcaf"
	case "setting":
		return filepath.Join("xcaf", "settings", name), "settings.xcaf"
	default:
		return "", ""
	}
}
