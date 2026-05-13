package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/analyzer"
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/blueprint"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/registry"
	"github.com/spf13/cobra"
)

var graphFormat string
var graphAgent string
var graphProject string
var graphFull bool
var graphScanOutput bool
var graphAll bool
var graphBlueprintFlag string

const (
	kindAgent  = "agent"
	kindSkill  = "skill"
	kindRule   = "rule"
	kindMCP    = "mcp"
	kindPolicy = "policy"
)

var graphCmd = &cobra.Command{
	Use:   "graph [file]",
	Short: "Visualize the resource dependency graph",
	Long: `Renders a visual map of your agent team and resource dependencies.

  - Shows agents with their model, tools, skills, rules, and MCP servers
  - Supports multiple output formats: terminal, mermaid, dot, json

Formats:
  terminal  Default. ASCII topology printed to stdout.
  mermaid   Mermaid graph syntax for embedding in markdown.
  dot       Graphviz DOT language for rendering with graphviz.
  json      Machine-readable JSON graph for programmatic use.`,
	Example: `  $ xcaffold graph
  $ xcaffold graph --format mermaid > topology.md
  $ xcaffold graph --format dot | dot -Tsvg > topology.svg
  $ xcaffold graph --format json | jq .`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().StringVar(&graphFormat, "format", "terminal", "Output format: terminal, mermaid, dot, json")
	graphCmd.Flags().StringVarP(&graphAgent, "agent", "a", "", "Target a specific agent (shows only its topology)")
	graphCmd.Flags().StringVarP(&graphProject, "project", "p", "", "Target a specific managed project by registered name or path")
	graphCmd.Flags().BoolVarP(&graphFull, "full", "f", false, "Show the fully expanded topology tree (always true if targeting an agent)")
	graphCmd.Flags().BoolVar(&graphScanOutput, "scan-output", false, "Scan compiled output directories for undeclared artifacts")
	graphCmd.Flags().BoolVar(&graphAll, "all", false, "Show global topology and all registered projects")
	graphCmd.Flags().StringVar(&graphBlueprintFlag, "blueprint", "", "Show graph for the named blueprint only")
	_ = graphCmd.Flags().MarkHidden("blueprint")
	rootCmd.AddCommand(graphCmd)
}

// graphNode represents a node in the topology graph.
type graphNode struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`  // "agent", "skill", "rule", "mcp", "hook", "settings"
	Label        string            `json:"label"` // display label
	Meta         map[string]string `json:"meta,omitempty"`
	Tools        []string          `json:"tools,omitempty"`
	BlockedTools []string          `json:"blocked_tools,omitempty"`
	Paths        []string          `json:"paths,omitempty"`
	Plugins      []string          `json:"plugins,omitempty"`
}

// graphEdge represents a directed edge.
type graphEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"` // "skill", "rule", "mcp"
}

type graphData struct {
	Project     string                   `json:"project"`
	Scope       string                   `json:"scope"`
	ConfigPath  string                   `json:"config_path"`
	DiskEntries []analyzer.ArtifactEntry `json:"disk_entries,omitempty"`
	Nodes       []graphNode              `json:"nodes"`
	Edges       []graphEdge              `json:"edges"`
}

func runGraph(cmd *cobra.Command, args []string) error {
	if err := validateGraphFlags(); err != nil {
		return err
	}

	if strings.ToLower(graphFormat) == "terminal" || graphFormat == "" {
		return runGraphTerminalMode()
	}

	scopes, err := resolveGraphScopes(args)
	if err != nil {
		return err
	}

	if len(scopes) == 0 {
		return fmt.Errorf("no topology generated")
	}

	return printGraphOutput(scopes)
}

// validateGraphFlags checks for mutually exclusive graph flags.
func validateGraphFlags() error {
	if graphBlueprintFlag != "" && globalFlag {
		return fmt.Errorf("--blueprint cannot be used with --global (blueprints are project-scoped)")
	}
	if graphAll && globalFlag {
		return fmt.Errorf("--all and --global are mutually exclusive")
	}
	if graphAll && graphProject != "" {
		return fmt.Errorf("--all and --project are mutually exclusive")
	}
	return nil
}

// resolveGraphScopes determines which graph scopes to render based on flags and arguments.
func resolveGraphScopes(args []string) ([]*graphData, error) {
	var scopes []*graphData

	if graphAll {
		return resolveAllScopes()
	} else if graphProject != "" {
		return resolveProjectScope(graphProject)
	} else if len(args) > 0 {
		g, err := parseGraphData(args[0], "")
		if err != nil {
			return nil, err
		}
		return append(scopes, g), nil
	} else if globalFlag {
		g, err := parseGraphData(globalXcafPath, "global")
		if err != nil {
			return nil, err
		}
		return append(scopes, g), nil
	} else {
		g, err := parseGraphData(xcafPath, "project")
		if err != nil {
			return nil, err
		}
		return append(scopes, g), nil
	}
}

// resolveAllScopes loads the global topology and all registered projects.
func resolveAllScopes() ([]*graphData, error) {
	var scopes []*graphData

	gGlobal, err := parseGraphData(globalXcafPath, "global")
	if err != nil {
		return nil, err
	}
	scopes = append(scopes, gGlobal)

	projects, err := registry.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not list registered projects: %v\n", err)
	} else {
		for _, p := range projects {
			projXcaf := filepath.Join(p.Path, "project.xcaf")
			if p.ConfigDir != "" && p.ConfigDir != "." {
				projXcaf = filepath.Join(p.Path, p.ConfigDir, "project.xcaf")
			}
			g, err := parseGraphData(projXcaf, fmt.Sprintf("project:%s", p.Name))
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping project %q: %v\n", p.Name, err)
				continue
			}
			scopes = append(scopes, g)
		}
	}

	return scopes, nil
}

// resolveProjectScope loads a specific project's graph.
func resolveProjectScope(projectName string) ([]*graphData, error) {
	p, err := registry.Resolve(projectName)
	if err != nil {
		return nil, err
	}

	projXcaf := filepath.Join(p.Path, "project.xcaf")
	g, err := parseGraphData(projXcaf, fmt.Sprintf("project:%s", p.Name))
	if err != nil {
		return nil, err
	}

	return []*graphData{g}, nil
}

//nolint:gocyclo
func printGraphOutput(scopes []*graphData) error {
	switch strings.ToLower(graphFormat) {
	case "mermaid":
		for _, g := range scopes {
			fmt.Print(renderMermaid(g))
		}
	case "dot":
		for _, g := range scopes {
			fmt.Print(renderDOT(g))
		}
	case "json":
		b, err := json.MarshalIndent(scopes, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	default:
		return fmt.Errorf("unknown format %q", graphFormat)
	}
	return nil
}

func parseGraphData(configPath, scopeName string) (*graphData, error) {
	config, err := parser.ParseDirectory(projectParseRoot())
	if err != nil {
		if scopeName != "" {
			return nil, fmt.Errorf("[%s] parse error: %w", scopeName, err)
		}
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if graphAgent != "" {
		if err := filterGraphToAgent(config, graphAgent, configPath); err != nil {
			return nil, err
		}
	}

	if graphBlueprintFlag != "" {
		if err := applyGraphBlueprint(config, graphBlueprintFlag); err != nil {
			return nil, err
		}
	}

	if scopeName != "global" {
		config.StripInherited()
	}
	g := buildGraph(config)
	g.Scope = scopeName
	g.ConfigPath = configPath

	if graphScanOutput {
		scanGraphOutput(configPath, g)
	}

	return g, nil
}

// filterGraphToAgent restricts the config to a single agent and its dependencies.
func filterGraphToAgent(config *ast.XcaffoldConfig, agentID string, configPath string) error {
	if _, ok := config.Agents[agentID]; !ok {
		return fmt.Errorf("agent %q not found in %s", agentID, configPath)
	}

	targetAgent := config.Agents[agentID]
	config.Agents = map[string]ast.AgentConfig{agentID: targetAgent}

	// Filter to only referenced skills.
	filteredSkills := make(map[string]ast.SkillConfig)
	for _, s := range targetAgent.Skills.Values {
		if sk, ok := config.Skills[s]; ok {
			filteredSkills[s] = sk
		}
	}
	config.Skills = filteredSkills

	// Filter to only referenced rules.
	filteredRules := make(map[string]ast.RuleConfig)
	for _, r := range targetAgent.Rules.Values {
		if ru, ok := config.Rules[r]; ok {
			filteredRules[r] = ru
		}
	}
	config.Rules = filteredRules

	// Filter to only referenced MCP servers.
	filteredMCP := make(map[string]ast.MCPConfig)
	for _, m := range targetAgent.MCP.Values {
		if mcp, ok := config.MCP[m]; ok {
			filteredMCP[m] = mcp
		}
	}
	config.MCP = filteredMCP

	return nil
}

// applyGraphBlueprint applies blueprint filtering to the config.
func applyGraphBlueprint(config *ast.XcaffoldConfig, bpName string) error {
	if err := blueprint.ResolveBlueprintExtends(config.Blueprints); err != nil {
		return fmt.Errorf("blueprint extends resolution failed: %w", err)
	}
	if errs := blueprint.ValidateBlueprintRefs(config.Blueprints, &config.ResourceScope); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("blueprint validation errors:\n%s", strings.Join(msgs, "\n"))
	}
	if p, ok := config.Blueprints[bpName]; ok {
		if err := blueprint.ResolveTransitiveDeps(&p, &config.ResourceScope); err != nil {
			return fmt.Errorf("blueprint transitive dependency resolution failed: %w", err)
		}
		config.Blueprints[bpName] = p
	}
	filtered, err := blueprint.ApplyBlueprint(config, bpName)
	if err != nil {
		return fmt.Errorf("blueprint %q: %w", bpName, err)
	}
	*config = *filtered
	return nil
}

// scanGraphOutput scans the compiled output directory for undeclared artifacts.
func scanGraphOutput(configPath string, g *graphData) {
	a := analyzer.New()
	declared := make(map[string]bool)
	for _, n := range g.Nodes {
		if n.Kind != "settings" {
			declared[fmt.Sprintf("%s:%s", n.Kind, n.ID)] = true
		}
	}

	baseDir := filepath.Dir(configPath)
	targetDir := filepath.Join(baseDir, compiler.OutputDir(targetFlag))

	entries, err := a.ScanOutputDir(targetDir, declared)
	if err == nil {
		g.DiskEntries = entries
	}
}

func buildGraph(config *ast.XcaffoldConfig) *graphData {
	var projectName string
	if config.Project != nil {
		projectName = config.Project.Name
	}
	g := &graphData{Project: projectName}
	appendGraphSettings(config, g)
	appendGraphSkills(config, g)
	appendGraphRules(config, g)
	appendGraphMCP(config, g)
	appendGraphPolicies(config, g)
	appendGraphAgents(config, g)
	appendGraphHooks(config, g)
	appendGraphWorkflows(config, g)
	return g
}

func appendGraphSettings(config *ast.XcaffoldConfig, g *graphData) {
	// Get the active settings (first available key after blueprint filtering).
	var es ast.SettingsConfig
	for _, s := range config.Settings {
		es = s
		break
	}
	if len(es.EnabledPlugins) > 0 {
		plugins := make([]string, 0, len(es.EnabledPlugins))
		for p, enabled := range es.EnabledPlugins {
			if enabled {
				plugins = append(plugins, p)
			}
		}
		sort.Strings(plugins)
		g.Nodes = append(g.Nodes, graphNode{
			ID:      "settings:global",
			Kind:    "settings",
			Label:   "Settings",
			Plugins: plugins,
		})
	}
}

func appendGraphSkills(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Skills) {
		skill := config.Skills[id]
		label := id
		if skill.Name != "" {
			label = skill.Name
		}
		var tools []string
		if len(skill.AllowedTools.Values) > 0 {
			tools = skill.AllowedTools.Values
		}
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "skill:" + id,
			Kind:  kindSkill,
			Label: label,
			Tools: tools,
		})
	}
}

func appendGraphRules(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Rules) {
		rule := config.Rules[id]
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "rule:" + id,
			Kind:  kindRule,
			Label: id,
			Paths: rule.Paths.Values,
		})
	}
}

func appendGraphMCP(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.MCP) {
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "mcp:" + id,
			Kind:  kindMCP,
			Label: id,
		})
	}
}

func appendGraphAgents(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Agents) {
		agent := config.Agents[id]
		meta := map[string]string{}
		if agent.Model != "" {
			meta["model"] = agent.Model
		}
		if agent.Effort != "" {
			meta["effort"] = agent.Effort
		}
		if len(agent.Memory) > 0 {
			meta["memory"] = strings.Join([]string(agent.Memory), ", ")
		}

		label := id
		if agent.Name != "" {
			label = agent.Name
		}

		g.Nodes = append(g.Nodes, graphNode{
			ID:           "agent:" + id,
			Kind:         kindAgent,
			Label:        label,
			Meta:         meta,
			Tools:        agent.Tools.Values,
			BlockedTools: agent.DisallowedTools.Values,
		})

		for _, skillID := range agent.Skills.Values {
			g.Edges = append(g.Edges, graphEdge{From: "agent:" + id, To: "skill:" + skillID, Label: "skill"})
		}
		for _, ruleID := range agent.Rules.Values {
			g.Edges = append(g.Edges, graphEdge{From: "agent:" + id, To: "rule:" + ruleID, Label: "rule"})
		}
		for _, mcpID := range agent.MCP.Values {
			g.Edges = append(g.Edges, graphEdge{From: "agent:" + id, To: "mcp:" + mcpID, Label: "mcp"})
		}
	}
}

func appendGraphPolicies(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Policies) {
		p := config.Policies[id]
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "policy:" + id,
			Kind:  kindPolicy,
			Label: id,
			Meta:  map[string]string{"severity": p.Severity, "target": p.Target},
		})
	}
}

func appendGraphHooks(config *ast.XcaffoldConfig, g *graphData) {
	var effectiveHooks ast.HookConfig
	if dh, ok := config.Hooks["default"]; ok {
		effectiveHooks = dh.Events
	}
	for _, event := range sortedKeys(effectiveHooks) {
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "hook:" + event,
			Kind:  "hook",
			Label: event,
		})
	}
}

func appendGraphWorkflows(config *ast.XcaffoldConfig, g *graphData) {
	for _, id := range sortedKeys(config.Workflows) {
		wf := config.Workflows[id]
		label := id
		if wf.Description != "" {
			label = wf.Description
		}
		g.Nodes = append(g.Nodes, graphNode{
			ID:    "workflow:" + id,
			Kind:  "workflow",
			Label: label,
		})
	}
}

// sortedKeys returns sorted keys of any map[string]V.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
