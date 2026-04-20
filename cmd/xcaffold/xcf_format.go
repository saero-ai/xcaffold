package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// projectMarshalDoc is the serialization envelope for the kind: project first
// document in MarshalMultiKind output. It replaces the legacy configDocument.
// Settings and Hooks are emitted as separate kind: settings and kind: hooks
// documents, matching the WriteSplitFiles pattern.
type projectMarshalDoc struct {
	Kind                string                  `yaml:"kind"`
	Version             string                  `yaml:"version"`
	Extends             string                  `yaml:"extends,omitempty"`
	Name                string                  `yaml:"name,omitempty"`
	Description         string                  `yaml:"description,omitempty"`
	Author              string                  `yaml:"author,omitempty"`
	Homepage            string                  `yaml:"homepage,omitempty"`
	Repository          string                  `yaml:"repository,omitempty"`
	License             string                  `yaml:"license,omitempty"`
	BackupDir           string                  `yaml:"backup-dir,omitempty"`
	Targets             []string                `yaml:"targets,omitempty"`
	AgentRefs           []string                `yaml:"agents,omitempty"`
	SkillRefs           []string                `yaml:"skills,omitempty"`
	RuleRefs            []string                `yaml:"rules,omitempty"`
	WorkflowRefs        []string                `yaml:"workflows,omitempty"`
	MCPRefs             []string                `yaml:"mcp,omitempty"`
	Instructions        string                  `yaml:"instructions,omitempty"`
	InstructionsFile    string                  `yaml:"instructions-file,omitempty"`
	InstructionsImports []string                `yaml:"instructions-imports,omitempty"`
	InstructionsScopes  []ast.InstructionsScope `yaml:"instructions-scopes,omitempty"`
	Test                ast.TestConfig          `yaml:"test,omitempty"`
	Local               ast.SettingsConfig      `yaml:"local,omitempty"`
}

// agentDoc is the serialization envelope for a kind: agent document.
type agentDoc struct {
	Kind            string `yaml:"kind"`
	Version         string `yaml:"version"`
	ast.AgentConfig `yaml:",inline"`
}

// skillDoc is the serialization envelope for a kind: skill document.
type skillDoc struct {
	Kind            string `yaml:"kind"`
	Version         string `yaml:"version"`
	ast.SkillConfig `yaml:",inline"`
}

// ruleDoc is the serialization envelope for a kind: rule document.
type ruleDoc struct {
	Kind           string `yaml:"kind"`
	Version        string `yaml:"version"`
	ast.RuleConfig `yaml:",inline"`
}

// workflowDoc is the serialization envelope for a kind: workflow document.
type workflowDoc struct {
	Kind               string `yaml:"kind"`
	Version            string `yaml:"version"`
	ast.WorkflowConfig `yaml:",inline"`
}

// mcpDoc is the serialization envelope for a kind: mcp document.
type mcpDoc struct {
	Kind          string `yaml:"kind"`
	Version       string `yaml:"version"`
	ast.MCPConfig `yaml:",inline"`
}

// memoryDoc is the serialization envelope for a kind: memory document.
type memoryDoc struct {
	Kind             string `yaml:"kind"`
	Version          string `yaml:"version"`
	ast.MemoryConfig `yaml:",inline"`
}

// FormatXCF serializes config to a multi-kind YAML string with no header comment.
// It is a thin wrapper around MarshalMultiKind for use in tests and tooling that
// need a plain string rather than a []byte with an optional header.
func FormatXCF(config *ast.XcaffoldConfig) (string, error) {
	b, err := MarshalMultiKind(config, "")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// MarshalMultiKind serializes an XcaffoldConfig as multi-kind YAML documents
// separated by "---". The first document is always kind: project, followed by
// individual kind: agent, kind: skill, kind: rule, kind: workflow, kind: mcp,
// kind: settings, and kind: hooks documents in alphabetical key order
// (deterministic output). Settings and hooks are emitted as separate documents.
//
// If header is non-empty it is prepended to the output as a comment block.
func MarshalMultiKind(config *ast.XcaffoldConfig, header string) ([]byte, error) {
	version := config.Version
	if version == "" {
		version = "1.0"
	}

	var docs [][]byte

	// ── kind: project document ──────────────────────────────────────────────
	proj := config.Project
	if proj == nil {
		proj = &ast.ProjectConfig{}
	}
	cfgDoc := projectMarshalDoc{
		Kind:                "project",
		Version:             version,
		Extends:             config.Extends,
		Name:                proj.Name,
		Description:         proj.Description,
		Author:              proj.Author,
		Homepage:            proj.Homepage,
		Repository:          proj.Repository,
		License:             proj.License,
		BackupDir:           proj.BackupDir,
		Targets:             proj.Targets,
		AgentRefs:           proj.AgentRefs,
		SkillRefs:           proj.SkillRefs,
		RuleRefs:            proj.RuleRefs,
		WorkflowRefs:        proj.WorkflowRefs,
		MCPRefs:             proj.MCPRefs,
		Instructions:        proj.Instructions,
		InstructionsFile:    proj.InstructionsFile,
		InstructionsImports: proj.InstructionsImports,
		InstructionsScopes:  proj.InstructionsScopes,
		Test:                proj.Test,
		Local:               proj.Local,
	}
	b, err := marshalYAML2(cfgDoc)
	if err != nil {
		return nil, err
	}
	docs = append(docs, bytes.TrimRight(b, "\n"))

	// ── kind: agent documents (sorted) ───────────────────────────────────────
	if len(config.Agents) > 0 {
		keys := make([]string, 0, len(config.Agents))
		for k := range config.Agents {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			agent := config.Agents[k]
			if agent.Name == "" {
				agent.Name = k
			}
			doc := agentDoc{
				Kind:        "agent",
				Version:     version,
				AgentConfig: agent,
			}
			b, err := marshalYAML2(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── kind: skill documents (sorted) ───────────────────────────────────────
	if len(config.Skills) > 0 {
		keys := make([]string, 0, len(config.Skills))
		for k := range config.Skills {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			skill := config.Skills[k]
			if skill.Name == "" {
				skill.Name = k
			}
			doc := skillDoc{
				Kind:        "skill",
				Version:     version,
				SkillConfig: skill,
			}
			b, err := marshalYAML2(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── kind: rule documents (sorted) ────────────────────────────────────────
	if len(config.Rules) > 0 {
		keys := make([]string, 0, len(config.Rules))
		for k := range config.Rules {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rule := config.Rules[k]
			if rule.Name == "" {
				rule.Name = k
			}
			doc := ruleDoc{
				Kind:       "rule",
				Version:    version,
				RuleConfig: rule,
			}
			b, err := marshalYAML2(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── kind: workflow documents (sorted) ─────────────────────────────────────
	if len(config.Workflows) > 0 {
		keys := make([]string, 0, len(config.Workflows))
		for k := range config.Workflows {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			wf := config.Workflows[k]
			if wf.Name == "" {
				wf.Name = k
			}
			doc := workflowDoc{
				Kind:           "workflow",
				Version:        version,
				WorkflowConfig: wf,
			}
			b, err := marshalYAML2(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── kind: mcp documents (sorted) ─────────────────────────────────────────
	if len(config.MCP) > 0 {
		keys := make([]string, 0, len(config.MCP))
		for k := range config.MCP {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			mcp := config.MCP[k]
			if mcp.Name == "" {
				mcp.Name = k
			}
			doc := mcpDoc{
				Kind:      "mcp",
				Version:   version,
				MCPConfig: mcp,
			}
			b, err := marshalYAML2(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── kind: settings document ─────────────────────────────────────────────
	if !isZeroSettings(config.Settings) {
		doc := settingsSplitDoc{
			Kind:           "settings",
			Version:        version,
			SettingsConfig: config.Settings,
		}
		b, err := marshalYAML2(doc)
		if err != nil {
			return nil, err
		}
		docs = append(docs, bytes.TrimRight(b, "\n"))
	}

	// ── kind: hooks document ────────────────────────────────────────────────
	if len(config.Hooks) > 0 {
		doc := hooksSplitDoc{
			Kind:    "hooks",
			Version: version,
			Events:  config.Hooks,
		}
		b, err := marshalYAML2(doc)
		if err != nil {
			return nil, err
		}
		docs = append(docs, bytes.TrimRight(b, "\n"))
	}

	// ── Assemble output ──────────────────────────────────────────────────────
	strs := make([]string, len(docs))
	for i, d := range docs {
		strs[i] = string(d)
	}
	joined := strings.Join(strs, "\n---\n")

	var out strings.Builder
	if header != "" {
		out.WriteString(header)
		out.WriteString("\n\n")
	}
	out.WriteString(joined)
	out.WriteString("\n")

	return []byte(out.String()), nil
}

// projectSplitDoc is the serialization envelope for kind: project in split-file mode.
// It does NOT contain resource maps — only metadata, targets, and ref lists pointing
// to child files under xcf/.
type projectSplitDoc struct {
	Kind         string   `yaml:"kind"`
	Version      string   `yaml:"version"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description,omitempty"`
	Author       string   `yaml:"author,omitempty"`
	Homepage     string   `yaml:"homepage,omitempty"`
	Repository   string   `yaml:"repository,omitempty"`
	License      string   `yaml:"license,omitempty"`
	BackupDir    string   `yaml:"backup-dir,omitempty"`
	Targets      []string `yaml:"targets,omitempty"`
	AgentRefs    []string `yaml:"agents,omitempty"`
	SkillRefs    []string `yaml:"skills,omitempty"`
	RuleRefs     []string `yaml:"rules,omitempty"`
	WorkflowRefs []string `yaml:"workflows,omitempty"`
	MCPRefs      []string `yaml:"mcp,omitempty"`
}

// hooksSplitDoc is the serialization envelope for kind: hooks in split-file mode.
type hooksSplitDoc struct {
	Kind    string         `yaml:"kind"`
	Version string         `yaml:"version"`
	Events  ast.HookConfig `yaml:"events"`
}

// settingsSplitDoc is the serialization envelope for kind: settings in split-file mode.
type settingsSplitDoc struct {
	Kind               string `yaml:"kind"`
	Version            string `yaml:"version"`
	ast.SettingsConfig `yaml:",inline"`
}

// WriteSplitFiles writes an XcaffoldConfig to rootDir as individual .xcf files:
//
//   - rootDir/project.xcf        — kind: project (metadata + ref lists)
//   - rootDir/xcf/agents/<n>.xcf  — kind: agent   (one per agent)
//   - rootDir/xcf/skills/<n>.xcf  — kind: skill   (one per skill)
//   - rootDir/xcf/rules/<n>.xcf   — kind: rule    (one per rule)
//   - rootDir/xcf/workflows/<n>.xcf — kind: workflow
//   - rootDir/xcf/mcp/<n>.xcf     — kind: mcp
//   - rootDir/xcf/hooks.xcf       — kind: hooks   (only when non-empty)
//   - rootDir/xcf/settings.xcf    — kind: settings (only when non-zero)
//
// Output is deterministic: all resource maps are emitted in sorted key order.
// All paths are cleaned via filepath.Clean. Directories are created with 0755,
// files written with 0644.
func WriteSplitFiles(config *ast.XcaffoldConfig, rootDir string) error {
	rootDir = filepath.Clean(rootDir)

	version := config.Version
	if version == "" {
		version = "1.0"
	}

	// ── kind: project (project.xcf) ────────────────────────────────────────
	proj := config.Project
	if proj == nil {
		proj = &ast.ProjectConfig{}
	}

	// Derive ref lists from the config maps when the explicit ref fields are empty.
	agentRefs := proj.AgentRefs
	if len(agentRefs) == 0 && len(config.Agents) > 0 {
		agentRefs = sortedMapKeys(config.Agents)
	}
	skillRefs := proj.SkillRefs
	if len(skillRefs) == 0 && len(config.Skills) > 0 {
		skillRefs = sortedMapKeys(config.Skills)
	}
	ruleRefs := proj.RuleRefs
	if len(ruleRefs) == 0 && len(config.Rules) > 0 {
		ruleRefs = sortedMapKeys(config.Rules)
	}
	workflowRefs := proj.WorkflowRefs
	if len(workflowRefs) == 0 && len(config.Workflows) > 0 {
		workflowRefs = sortedMapKeys(config.Workflows)
	}
	mcpRefs := proj.MCPRefs
	if len(mcpRefs) == 0 && len(config.MCP) > 0 {
		mcpRefs = sortedMapKeys(config.MCP)
	}

	projDoc := projectSplitDoc{
		Kind:         "project",
		Version:      version,
		Name:         proj.Name,
		Description:  proj.Description,
		Author:       proj.Author,
		Homepage:     proj.Homepage,
		Repository:   proj.Repository,
		License:      proj.License,
		BackupDir:    proj.BackupDir,
		Targets:      proj.Targets,
		AgentRefs:    agentRefs,
		SkillRefs:    skillRefs,
		RuleRefs:     ruleRefs,
		WorkflowRefs: workflowRefs,
		MCPRefs:      mcpRefs,
	}
	if err := writeYAMLFile(filepath.Join(rootDir, "project.xcf"), projDoc); err != nil {
		return err
	}

	xcfDir := filepath.Join(rootDir, "xcf")

	// ── kind: agent ──────────────────────────────────────────────────────────
	if len(config.Agents) > 0 {
		dir := filepath.Join(xcfDir, "agents")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for _, k := range sortedMapKeys(config.Agents) {
			agent := config.Agents[k]
			if agent.Name == "" {
				agent.Name = k
			}
			doc := agentDoc{Kind: "agent", Version: version, AgentConfig: agent}
			if err := writeYAMLFile(filepath.Join(dir, k+".xcf"), doc); err != nil {
				return err
			}
		}
	}

	// ── kind: skill ──────────────────────────────────────────────────────────
	if len(config.Skills) > 0 {
		dir := filepath.Join(xcfDir, "skills")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for _, k := range sortedMapKeys(config.Skills) {
			skill := config.Skills[k]
			if skill.Name == "" {
				skill.Name = k
			}
			doc := skillDoc{Kind: "skill", Version: version, SkillConfig: skill}
			if err := writeYAMLFile(filepath.Join(dir, k+".xcf"), doc); err != nil {
				return err
			}
		}
	}

	// ── kind: rule ───────────────────────────────────────────────────────────
	if len(config.Rules) > 0 {
		dir := filepath.Join(xcfDir, "rules")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for _, k := range sortedMapKeys(config.Rules) {
			rule := config.Rules[k]
			if rule.Name == "" {
				rule.Name = k
			}
			doc := ruleDoc{Kind: "rule", Version: version, RuleConfig: rule}
			outPath := filepath.Join(dir, filepath.FromSlash(k)+".xcf")
			// Rule IDs may contain forward-slash namespacing for subdirectory rules
			// (e.g. "cli/build-go-cli"). Ensure the parent directory exists before writing.
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}
			if err := writeYAMLFile(outPath, doc); err != nil {
				return err
			}
		}
	}

	// ── kind: workflow ───────────────────────────────────────────────────────
	if len(config.Workflows) > 0 {
		dir := filepath.Join(xcfDir, "workflows")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for _, k := range sortedMapKeys(config.Workflows) {
			wf := config.Workflows[k]
			if wf.Name == "" {
				wf.Name = k
			}
			doc := workflowDoc{Kind: "workflow", Version: version, WorkflowConfig: wf}
			if err := writeYAMLFile(filepath.Join(dir, k+".xcf"), doc); err != nil {
				return err
			}
		}
	}

	// ── kind: mcp ────────────────────────────────────────────────────────────
	if len(config.MCP) > 0 {
		dir := filepath.Join(xcfDir, "mcp")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for _, k := range sortedMapKeys(config.MCP) {
			mcp := config.MCP[k]
			if mcp.Name == "" {
				mcp.Name = k
			}
			doc := mcpDoc{Kind: "mcp", Version: version, MCPConfig: mcp}
			if err := writeYAMLFile(filepath.Join(dir, k+".xcf"), doc); err != nil {
				return err
			}
		}
	}

	// ── kind: hooks ──────────────────────────────────────────────────────────
	if len(config.Hooks) > 0 {
		if err := os.MkdirAll(xcfDir, 0755); err != nil {
			return err
		}
		doc := hooksSplitDoc{
			Kind:    "hooks",
			Version: version,
			Events:  config.Hooks,
		}
		if err := writeYAMLFile(filepath.Join(xcfDir, "hooks.xcf"), doc); err != nil {
			return err
		}
	}

	// ── kind: settings ───────────────────────────────────────────────────────
	if !isZeroSettings(config.Settings) {
		if err := os.MkdirAll(xcfDir, 0755); err != nil {
			return err
		}
		doc := settingsSplitDoc{
			Kind:           "settings",
			Version:        version,
			SettingsConfig: config.Settings,
		}
		if err := writeYAMLFile(filepath.Join(xcfDir, "settings.xcf"), doc); err != nil {
			return err
		}
	}

	// ── kind: memory ─────────────────────────────────────────────────────────
	if len(config.Memory) > 0 {
		dir := filepath.Join(xcfDir, "memory")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for _, k := range sortedMapKeys(config.Memory) {
			mem := config.Memory[k]
			if mem.Name == "" {
				mem.Name = k
			}
			doc := memoryDoc{Kind: "memory", Version: version, MemoryConfig: mem}
			outPath := filepath.Join(dir, filepath.FromSlash(k)+".xcf")
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}
			if err := writeYAMLFile(outPath, doc); err != nil {
				return err
			}
		}
	}

	// ── extras (provider-specific unrecognized files) ─────────────────────────
	if len(config.ProviderExtras) > 0 {
		providers := sortedMapKeys(config.ProviderExtras)
		for _, provider := range providers {
			files := config.ProviderExtras[provider]
			if len(files) == 0 {
				continue
			}
			keys := make([]string, 0, len(files))
			for k := range files {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, rel := range keys {
				data := files[rel]
				outPath := filepath.Join(xcfDir, "extras", provider, filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Clean(outPath), data, 0644); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// marshalYAML2 marshals v to YAML with 2-space indentation.
func marshalYAML2(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeYAMLFile marshals v to YAML with 2-space indentation and writes it to
// path with 0644 permissions.
func writeYAMLFile(path string, v any) error {
	b, err := marshalYAML2(v)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Clean(path), b, 0644)
}

// sortedMapKeys returns sorted keys for the common resource map types.
// Each overload uses a generic helper to avoid reflection.
func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// isZeroSettings reports whether s is the zero value of SettingsConfig,
// indicating that no settings file should be written.
func isZeroSettings(s ast.SettingsConfig) bool {
	return s.Model == "" &&
		s.EffortLevel == "" &&
		s.DefaultShell == "" &&
		s.Language == "" &&
		s.OutputStyle == "" &&
		s.PlansDirectory == "" &&
		s.OtelHeadersHelper == "" &&
		s.AutoMemoryDirectory == "" &&
		s.Permissions == nil &&
		s.Sandbox == nil &&
		s.StatusLine == nil &&
		s.MCPServers == nil &&
		len(s.Hooks) == 0 &&
		len(s.Env) == 0 &&
		len(s.EnabledPlugins) == 0 &&
		len(s.AvailableModels) == 0 &&
		len(s.ClaudeMdExcludes) == 0 &&
		s.CleanupPeriodDays == nil &&
		s.IncludeGitInstructions == nil &&
		s.SkipDangerousModePermissionPrompt == nil &&
		s.AutoMemoryEnabled == nil &&
		s.DisableAllHooks == nil &&
		s.Attribution == nil &&
		s.RespectGitignore == nil &&
		s.DisableSkillShellExecution == nil &&
		s.AlwaysThinkingEnabled == nil &&
		s.Agent == nil &&
		s.Worktree == nil &&
		s.AutoMode == nil
}
