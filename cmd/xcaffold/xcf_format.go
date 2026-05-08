package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

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

// policyDoc is the serialization envelope for a kind: policy document.
type policyDoc struct {
	Kind             string `yaml:"kind"`
	Version          string `yaml:"version"`
	ast.PolicyConfig `yaml:",inline"`
}

// contextDoc is the serialization envelope for a kind: context document.
type contextDoc struct {
	Kind              string `yaml:"kind"`
	Version           string `yaml:"version"`
	ast.ContextConfig `yaml:",inline"`
}

// projectSplitDoc is the serialization envelope for kind: project in split-file mode.
// It does NOT contain resource maps — only metadata, targets, ref lists pointing
// to child files under xcaf/, and project-level instruction references.
type projectSplitDoc struct {
	Kind        string   `yaml:"kind"`
	Version     string   `yaml:"version"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Author      string   `yaml:"author,omitempty"`
	Homepage    string   `yaml:"homepage,omitempty"`
	Repository  string   `yaml:"repository,omitempty"`
	License     string   `yaml:"license,omitempty"`
	BackupDir   string   `yaml:"backup-dir,omitempty"`
	Targets     []string `yaml:"targets,omitempty"`
}

// hooksSplitDoc is the serialization envelope for kind: hooks in split-file mode.
type hooksSplitDoc struct {
	Kind    string                        `yaml:"kind"`
	Version string                        `yaml:"version"`
	Events  ast.HookConfig                `yaml:"events"`
	Targets map[string]ast.TargetOverride `yaml:"targets,omitempty"`
}

// settingsSplitDoc is the serialization envelope for kind: settings in split-file mode.
type settingsSplitDoc struct {
	Kind               string `yaml:"kind"`
	Version            string `yaml:"version"`
	ast.SettingsConfig `yaml:",inline"`
}

// hooksOverrideDoc is for serializing provider-specific hook overrides (no kind/version).
type hooksOverrideDoc struct {
	Events  ast.HookConfig                `yaml:"events"`
	Targets map[string]ast.TargetOverride `yaml:"targets,omitempty"`
}

// settingsOverrideDoc is for serializing provider-specific settings overrides (no kind/version).
type settingsOverrideDoc struct {
	ast.SettingsConfig `yaml:",inline"`
}

// WriteProjectFile writes only the project.xcaf file for rootDir from config.
// Use this instead of WriteSplitFiles when only the project metadata block needs
// updating (e.g. on re-import) and resource files should be left untouched.
// Project metadata is written to rootDir/project.xcaf (root level).
func WriteProjectFile(config *ast.XcaffoldConfig, rootDir string) error {
	rootDir = filepath.Clean(rootDir)
	version := config.Version
	if version == "" {
		version = "1.0"
	}
	proj := config.Project
	if proj == nil {
		proj = &ast.ProjectConfig{}
	}
	projDoc := projectSplitDoc{
		Kind:        "project",
		Version:     version,
		Name:        proj.Name,
		Description: proj.Description,
		Author:      proj.Author,
		Homepage:    proj.Homepage,
		Repository:  proj.Repository,
		License:     proj.License,
		BackupDir:   proj.BackupDir,
		Targets:     proj.Targets,
	}
	// Write project.xcaf to root level (preferred location)
	return writeYAMLFile(filepath.Join(rootDir, "project.xcaf"), projDoc)
}

// WriteSplitFiles writes an XcaffoldConfig to rootDir as individual .xcaf files:
//
//   - rootDir/project.xcaf              — kind: project (metadata + ref lists)
//   - rootDir/xcaf/agents/<name>/agent.xcaf      — kind: agent (one per agent, in its own subdirectory)
//   - rootDir/xcaf/skills/<name>/skill.xcaf      — kind: skill (one per skill, in its own subdirectory)
//   - rootDir/xcaf/rules/<name>/rule.xcaf        — kind: rule (one per rule, in its own subdirectory)
//   - rootDir/xcaf/workflows/<name>/workflow.xcaf — kind: workflow (one per workflow, in its own subdirectory)
//   - rootDir/xcaf/mcp/<name>/mcp.xcaf           — kind: mcp (one per MCP server, in its own subdirectory)
//   - rootDir/xcaf/hooks/<name>/hooks.xcaf       — kind: hooks (each named hook in its own subdirectory, only when non-empty)
//   - rootDir/xcaf/settings/<name>/settings.xcaf — kind: settings (each named settings in its own subdirectory, only when non-zero)
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

	// Write project.xcaf
	proj := config.Project
	if proj == nil {
		proj = &ast.ProjectConfig{}
	}

	projDoc := projectSplitDoc{
		Kind:        "project",
		Version:     version,
		Name:        proj.Name,
		Description: proj.Description,
		Author:      proj.Author,
		Homepage:    proj.Homepage,
		Repository:  proj.Repository,
		License:     proj.License,
		BackupDir:   proj.BackupDir,
		Targets:     proj.Targets,
	}

	// Since ref lists are no longer used, write all resources (nil filters mean "write all")
	var (
		agentFilter    map[string]bool
		skillFilter    map[string]bool
		ruleFilter     map[string]bool
		workflowFilter map[string]bool
		mcpFilter      map[string]bool
	)
	// Write project.xcaf to root level (preferred location)
	if err := writeYAMLFile(filepath.Join(rootDir, "project.xcaf"), projDoc); err != nil {
		return err
	}

	xcafDir := filepath.Join(rootDir, "xcaf")

	// Delegate to per-kind helpers
	if err := writeAgentFiles(config, xcafDir, version, agentFilter); err != nil {
		return err
	}
	if err := writeSkillFiles(config, xcafDir, version, skillFilter); err != nil {
		return err
	}
	if err := writeRuleFiles(config, xcafDir, version, ruleFilter); err != nil {
		return err
	}
	if err := writeWorkflowFiles(config, xcafDir, version, workflowFilter); err != nil {
		return err
	}
	if err := writeMCPFiles(config, xcafDir, version, mcpFilter); err != nil {
		return err
	}
	if err := writePolicyFiles(config, xcafDir, version); err != nil {
		return err
	}
	if err := writeContextFiles(config, xcafDir, version); err != nil {
		return err
	}
	if err := writeHooksFiles(config, xcafDir, version); err != nil {
		return err
	}
	if err := writeSettingsFiles(config, xcafDir, version); err != nil {
		return err
	}
	if err := writeProviderExtrasFiles(config, rootDir); err != nil {
		return err
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

// writeFrontmatterFile writes doc as YAML frontmatter (between --- delimiters)
// followed by body when body is non-empty. When body is empty it falls back to
// writeYAMLFile so the output remains plain YAML with no frontmatter wrapper.
func writeFrontmatterFile(path string, doc any, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return writeYAMLFile(path, doc)
	}

	b, err := marshalYAML2(doc)
	if err != nil {
		return err
	}
	content := strings.TrimRight(string(b), "\n")

	var out strings.Builder
	out.WriteString("---\n")
	out.WriteString(content)
	out.WriteString("\n---\n")
	out.WriteString(body)
	out.WriteString("\n")

	return os.WriteFile(filepath.Clean(path), []byte(out.String()), 0644)
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
		len(s.MdExcludes) == 0 &&
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

// writeAgentFiles writes all agents from config to xcaf/agents/ directory.
func writeAgentFiles(config *ast.XcaffoldConfig, xcafDir, version string, agentFilter map[string]bool) error {
	if len(config.Agents) == 0 {
		return nil
	}
	for _, k := range sortedMapKeys(config.Agents) {
		if agentFilter != nil && !agentFilter[k] {
			continue
		}
		agentSubDir := filepath.Join(xcafDir, "agents", k)
		if err := os.MkdirAll(agentSubDir, 0755); err != nil {
			return err
		}
		agent := config.Agents[k]
		if agent.Name == "" {
			agent.Name = k
		}
		body := strings.TrimSpace(agent.Body)
		doc := agentDoc{Kind: "agent", Version: version, AgentConfig: agent}
		if err := writeFrontmatterFile(filepath.Join(agentSubDir, "agent.xcaf"), doc, body); err != nil {
			return err
		}
		// Write agent overrides: agent.<provider>.xcaf
		if config.Overrides != nil {
			if providers := config.Overrides.AgentProviders(k); len(providers) > 0 {
				for _, provider := range providers {
					overrideCfg, _ := config.Overrides.GetAgent(k, provider)
					overrideBody := strings.TrimSpace(overrideCfg.Body)
					overrideCfg.Body = ""
					overrideCfg.Name = ""
					overridePath := filepath.Join(agentSubDir, fmt.Sprintf("agent.%s.xcaf", provider))
					if err := writeFrontmatterFile(overridePath, overrideCfg, overrideBody); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// writeSkillFiles writes all skills from config to xcaf/skills/ directory.
func writeSkillFiles(config *ast.XcaffoldConfig, xcafDir, version string, skillFilter map[string]bool) error {
	if len(config.Skills) == 0 {
		return nil
	}
	dir := filepath.Join(xcafDir, "skills")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, k := range sortedMapKeys(config.Skills) {
		if skillFilter != nil && !skillFilter[k] {
			continue
		}
		skill := config.Skills[k]
		if skill.Name == "" {
			skill.Name = k
		}

		// Normalize any legacy project-root-relative paths to skill-dir-relative.
		prefix := "xcaf/skills/" + k + "/"
		for i, ref := range skill.References.Values {
			skill.References.Values[i] = strings.TrimPrefix(ref, prefix)
		}
		for i, ref := range skill.Scripts.Values {
			skill.Scripts.Values[i] = strings.TrimPrefix(ref, prefix)
		}
		for i, ref := range skill.Assets.Values {
			skill.Assets.Values[i] = strings.TrimPrefix(ref, prefix)
		}
		for i, ref := range skill.Examples.Values {
			skill.Examples.Values[i] = strings.TrimPrefix(ref, prefix)
		}

		body := strings.TrimSpace(skill.Body)
		doc := skillDoc{Kind: "skill", Version: version, SkillConfig: skill}

		skillSubDir := filepath.Join(dir, k)
		if err := os.MkdirAll(skillSubDir, 0755); err != nil {
			return err
		}
		outPath := filepath.Join(skillSubDir, "skill.xcaf")

		if err := writeFrontmatterFile(outPath, doc, body); err != nil {
			return err
		}

		// Write skill overrides: xcaf/skills/<name>/skill.<provider>.xcaf
		if config.Overrides != nil {
			if providers := config.Overrides.SkillProviders(k); len(providers) > 0 {
				for _, provider := range providers {
					overrideCfg, _ := config.Overrides.GetSkill(k, provider)
					overrideBody := strings.TrimSpace(overrideCfg.Body)
					overrideCfg.Body = ""
					overrideCfg.Name = ""
					overridePath := filepath.Join(skillSubDir, fmt.Sprintf("skill.%s.xcaf", provider))
					if err := writeFrontmatterFile(overridePath, overrideCfg, overrideBody); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// writeRuleFiles writes all rules from config to xcaf/rules/ directory.
func writeRuleFiles(config *ast.XcaffoldConfig, xcafDir, version string, ruleFilter map[string]bool) error {
	if len(config.Rules) == 0 {
		return nil
	}
	dir := filepath.Join(xcafDir, "rules")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, k := range sortedMapKeys(config.Rules) {
		if ruleFilter != nil && !ruleFilter[k] {
			continue
		}
		rule := config.Rules[k]
		if rule.Name == "" {
			rule.Name = k
		}
		body := strings.TrimSpace(rule.Body)
		doc := ruleDoc{Kind: "rule", Version: version, RuleConfig: rule}

		// Directory layout: xcaf/rules/<name>/rule.xcaf
		ruleSubDir := filepath.Join(dir, k)
		if err := os.MkdirAll(ruleSubDir, 0755); err != nil {
			return err
		}
		outPath := filepath.Join(ruleSubDir, "rule.xcaf")

		if err := writeFrontmatterFile(outPath, doc, body); err != nil {
			return err
		}

		// Write rule overrides: rule.<provider>.xcaf
		if config.Overrides != nil {
			if providers := config.Overrides.RuleProviders(k); len(providers) > 0 {
				for _, provider := range providers {
					overrideCfg, _ := config.Overrides.GetRule(k, provider)
					overrideBody := strings.TrimSpace(overrideCfg.Body)
					overrideCfg.Body = ""
					overrideCfg.Name = ""
					overridePath := filepath.Join(ruleSubDir, fmt.Sprintf("rule.%s.xcaf", provider))
					if err := writeFrontmatterFile(overridePath, overrideCfg, overrideBody); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// writeWorkflowFiles writes all workflows from config to xcaf/workflows/ directory.
func writeWorkflowFiles(config *ast.XcaffoldConfig, xcafDir, version string, workflowFilter map[string]bool) error {
	if len(config.Workflows) == 0 {
		return nil
	}
	dir := filepath.Join(xcafDir, "workflows")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, k := range sortedMapKeys(config.Workflows) {
		if workflowFilter != nil && !workflowFilter[k] {
			continue
		}
		wf := config.Workflows[k]
		if wf.Name == "" {
			wf.Name = k
		}
		body := strings.TrimSpace(wf.Body)
		doc := workflowDoc{Kind: "workflow", Version: version, WorkflowConfig: wf}

		// Directory layout: xcaf/workflows/<name>/workflow.xcaf
		workflowSubDir := filepath.Join(dir, k)
		if err := os.MkdirAll(workflowSubDir, 0755); err != nil {
			return err
		}
		outPath := filepath.Join(workflowSubDir, "workflow.xcaf")

		if err := writeFrontmatterFile(outPath, doc, body); err != nil {
			return err
		}

		// Write workflow overrides: workflow.<provider>.xcaf
		if config.Overrides != nil {
			if providers := config.Overrides.WorkflowProviders(k); len(providers) > 0 {
				for _, provider := range providers {
					overrideCfg, _ := config.Overrides.GetWorkflow(k, provider)
					overrideBody := strings.TrimSpace(overrideCfg.Body)
					overrideCfg.Body = ""
					overrideCfg.Name = ""
					overridePath := filepath.Join(workflowSubDir, fmt.Sprintf("workflow.%s.xcaf", provider))
					if err := writeFrontmatterFile(overridePath, overrideCfg, overrideBody); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// writeMCPFiles writes all MCP servers from config to xcaf/mcp/ directory.
func writeMCPFiles(config *ast.XcaffoldConfig, xcafDir, version string, mcpFilter map[string]bool) error {
	if len(config.MCP) == 0 {
		return nil
	}
	dir := filepath.Join(xcafDir, "mcp")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, k := range sortedMapKeys(config.MCP) {
		if mcpFilter != nil && !mcpFilter[k] {
			continue
		}
		mcp := config.MCP[k]
		if mcp.Name == "" {
			mcp.Name = k
		}
		doc := mcpDoc{Kind: "mcp", Version: version, MCPConfig: mcp}

		// Directory layout: xcaf/mcp/<name>/mcp.xcaf
		mcpSubDir := filepath.Join(dir, k)
		if err := os.MkdirAll(mcpSubDir, 0755); err != nil {
			return err
		}
		outPath := filepath.Join(mcpSubDir, "mcp.xcaf")

		if err := writeYAMLFile(outPath, doc); err != nil {
			return err
		}

		// Write MCP overrides: mcp.<provider>.xcaf
		if config.Overrides != nil {
			if providers := config.Overrides.MCPProviders(k); len(providers) > 0 {
				for _, provider := range providers {
					overrideCfg, _ := config.Overrides.GetMCP(k, provider)
					overrideCfg.Name = ""
					overridePath := filepath.Join(mcpSubDir, fmt.Sprintf("mcp.%s.xcaf", provider))
					if err := writeYAMLFile(overridePath, overrideCfg); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// writePolicyFiles writes all policies from config to xcaf/policy/ directory.
func writePolicyFiles(config *ast.XcaffoldConfig, xcafDir, version string) error {
	if len(config.Policies) == 0 {
		return nil
	}
	dir := filepath.Join(xcafDir, "policy")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, k := range sortedMapKeys(config.Policies) {
		policy := config.Policies[k]
		if policy.Name == "" {
			policy.Name = k
		}
		doc := policyDoc{Kind: "policy", Version: version, PolicyConfig: policy}

		// Directory layout: xcaf/policy/<name>/policy.xcaf
		policySubDir := filepath.Join(dir, k)
		if err := os.MkdirAll(policySubDir, 0755); err != nil {
			return err
		}
		outPath := filepath.Join(policySubDir, "policy.xcaf")

		if err := writeYAMLFile(outPath, doc); err != nil {
			return err
		}
	}
	return nil
}

// writeContextFiles writes all contexts from config to xcaf/context/ directory.
func writeContextFiles(config *ast.XcaffoldConfig, xcafDir, version string) error {
	if len(config.Contexts) == 0 {
		return nil
	}
	dir := filepath.Join(xcafDir, "context")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, k := range sortedMapKeys(config.Contexts) {
		ctx := config.Contexts[k]
		if ctx.Name == "" {
			ctx.Name = k
		}
		body := strings.TrimSpace(ctx.Body)
		ctx.Body = "" // zero before YAML serialization
		doc := contextDoc{Kind: "context", Version: version, ContextConfig: ctx}

		ctxSubDir := filepath.Join(dir, k)
		if err := os.MkdirAll(ctxSubDir, 0755); err != nil {
			return err
		}
		outPath := filepath.Join(ctxSubDir, "context.xcaf")

		if err := writeFrontmatterFile(outPath, doc, body); err != nil {
			return err
		}
	}
	return nil
}

// writeHooksFiles writes all named hooks from config to xcaf/hooks/ directory.
func writeHooksFiles(config *ast.XcaffoldConfig, xcafDir, version string) error {
	if len(config.Hooks) == 0 {
		return nil
	}
	dir := filepath.Join(xcafDir, "hooks")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, k := range sortedMapKeys(config.Hooks) {
		hook := config.Hooks[k]
		if len(hook.Events) == 0 {
			continue
		}

		doc := hooksSplitDoc{
			Kind:    "hooks",
			Version: version,
			Events:  hook.Events,
			Targets: hook.Targets,
		}

		// Directory layout: xcaf/hooks/<name>/hooks.xcaf
		hookSubDir := filepath.Join(dir, k)
		if err := os.MkdirAll(hookSubDir, 0755); err != nil {
			return err
		}
		outPath := filepath.Join(hookSubDir, "hooks.xcaf")

		if err := writeYAMLFile(outPath, doc); err != nil {
			return err
		}

		// Write hooks overrides: hooks.<provider>.xcaf
		if config.Overrides != nil {
			if providers := config.Overrides.HooksProviders(k); len(providers) > 0 {
				for _, provider := range providers {
					overrideCfg, _ := config.Overrides.GetHooks(k, provider)
					overridePath := filepath.Join(hookSubDir, fmt.Sprintf("hooks.%s.xcaf", provider))
					overrideDoc := hooksOverrideDoc{
						Events:  overrideCfg.Events,
						Targets: overrideCfg.Targets,
					}
					if err := writeYAMLFile(overridePath, overrideDoc); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// writeSettingsFiles writes all named settings from config to xcaf/settings/ directory.
func writeSettingsFiles(config *ast.XcaffoldConfig, xcafDir, version string) error {
	if len(config.Settings) == 0 {
		return nil
	}
	dir := filepath.Join(xcafDir, "settings")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, k := range sortedMapKeys(config.Settings) {
		settings := config.Settings[k]
		if isZeroSettings(settings) {
			continue
		}

		doc := settingsSplitDoc{
			Kind:           "settings",
			Version:        version,
			SettingsConfig: settings,
		}

		// Directory layout: xcaf/settings/<name>/settings.xcaf
		settingsSubDir := filepath.Join(dir, k)
		if err := os.MkdirAll(settingsSubDir, 0755); err != nil {
			return err
		}
		outPath := filepath.Join(settingsSubDir, "settings.xcaf")

		if err := writeYAMLFile(outPath, doc); err != nil {
			return err
		}

		// Write settings overrides: settings.<provider>.xcaf
		if config.Overrides != nil {
			if providers := config.Overrides.SettingsProviders(k); len(providers) > 0 {
				for _, provider := range providers {
					overrideCfg, _ := config.Overrides.GetSettings(k, provider)
					overrideCfg.Name = ""
					overridePath := filepath.Join(settingsSubDir, fmt.Sprintf("settings.%s.xcaf", provider))
					overrideDoc := settingsOverrideDoc{
						SettingsConfig: overrideCfg,
					}
					if err := writeYAMLFile(overridePath, overrideDoc); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// writeProviderExtrasFiles writes provider-specific passthrough files.
func writeProviderExtrasFiles(config *ast.XcaffoldConfig, rootDir string) error {
	if len(config.ProviderExtras) == 0 {
		return nil
	}
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
		var destRoot string
		if provider == "xcaf" {
			// Canonical passthrough for hooks
			destRoot = filepath.Join(rootDir, "xcaf")
		} else {
			destRoot = filepath.Join(rootDir, "xcaf", "provider", provider)
		}

		for _, rel := range keys {
			data := files[rel]
			outPath := filepath.Join(destRoot, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}

			perm := os.FileMode(0644)
			if strings.HasSuffix(rel, ".sh") {
				perm = 0755
			}

			if err := os.WriteFile(filepath.Clean(outPath), data, perm); err != nil {
				return err
			}
		}
	}
	return nil
}
