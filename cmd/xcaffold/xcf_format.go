package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/compiler"
	"gopkg.in/yaml.v3"
)

func agentMemoryIndex(rootDir string) map[string][]string {
	discovered := compiler.DiscoverAgentMemory(rootDir)
	idx := make(map[string][]string)
	for key := range discovered {
		parts := strings.SplitN(key, "/", 2)
		if len(parts) == 2 {
			idx[parts[0]] = append(idx[parts[0]], parts[1])
		}
	}
	for id := range idx {
		sort.Strings(idx[id])
	}
	return idx
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
// to child files under xcf/, and project-level instruction references.
type projectSplitDoc struct {
	Kind         string                   `yaml:"kind"`
	Version      string                   `yaml:"version"`
	Name         string                   `yaml:"name"`
	Description  string                   `yaml:"description,omitempty"`
	Author       string                   `yaml:"author,omitempty"`
	Homepage     string                   `yaml:"homepage,omitempty"`
	Repository   string                   `yaml:"repository,omitempty"`
	License      string                   `yaml:"license,omitempty"`
	BackupDir    string                   `yaml:"backup-dir,omitempty"`
	Targets      []string                 `yaml:"targets,omitempty"`
	AgentRefs    []ast.AgentManifestEntry `yaml:"agents,omitempty"`
	SkillRefs    []string                 `yaml:"skills,omitempty"`
	RuleRefs     []string                 `yaml:"rules,omitempty"`
	WorkflowRefs []string                 `yaml:"workflows,omitempty"`
	MCPRefs      []string                 `yaml:"mcp,omitempty"`
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

// WriteProjectFile writes only the project.xcf file for rootDir from config.
// Use this instead of WriteSplitFiles when only the project metadata block needs
// updating (e.g. on re-import) and resource files should be left untouched.
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
	agentRefs := proj.AgentRefs
	agentMemMap := agentMemoryIndex(rootDir)
	if len(agentRefs) == 0 && len(config.Agents) > 0 {
		sortedAgents := sortedMapKeys(config.Agents)
		for _, sa := range sortedAgents {
			agentRefs = append(agentRefs, ast.AgentManifestEntry{
				ID:     sa,
				Memory: agentMemMap[sa],
			})
		}
	} else {
		for i, ref := range agentRefs {
			if mem, ok := agentMemMap[ref.ID]; ok && len(ref.Memory) == 0 {
				agentRefs[i].Memory = mem
			}
		}
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
	outDir := filepath.Join(rootDir, ".xcaffold")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	return writeYAMLFile(filepath.Join(outDir, "project.xcf"), projDoc)
}

// WriteSplitFiles writes an XcaffoldConfig to rootDir as individual .xcf files:
//
//   - rootDir/project.xcf        — kind: project (metadata + ref lists)
//   - rootDir/xcf/agents/<n>/<n>.xcf — kind: agent (one per agent, in its own subdirectory)
//   - rootDir/xcf/skills/<n>.xcf  — kind: skill   (one per skill)
//   - rootDir/xcf/rules/<n>/rule.xcf — kind: rule  (one per rule, in its own subdirectory)
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
	agentMemMap2 := agentMemoryIndex(rootDir)
	if len(agentRefs) == 0 && len(config.Agents) > 0 {
		sortedAgents := sortedMapKeys(config.Agents)
		for _, sa := range sortedAgents {
			agentRefs = append(agentRefs, ast.AgentManifestEntry{
				ID:     sa,
				Memory: agentMemMap2[sa],
			})
		}
	} else {
		for i, ref := range agentRefs {
			if mem, ok := agentMemMap2[ref.ID]; ok && len(ref.Memory) == 0 {
				agentRefs[i].Memory = mem
			}
		}
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

	// Build filter sets from the explicit ref lists. A nil map means "write all".
	agentRefIds := make([]string, 0, len(proj.AgentRefs))
	for _, ref := range proj.AgentRefs {
		agentRefIds = append(agentRefIds, ref.ID)
	}
	agentFilter := refSet(agentRefIds)
	skillFilter := refSet(proj.SkillRefs)
	ruleFilter := refSet(proj.RuleRefs)
	workflowFilter := refSet(proj.WorkflowRefs)
	mcpFilter := refSet(proj.MCPRefs)

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
	outDir := filepath.Join(rootDir, ".xcaffold")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	if err := writeYAMLFile(filepath.Join(outDir, "project.xcf"), projDoc); err != nil {
		return err
	}

	xcfDir := filepath.Join(rootDir, "xcf")

	// ── kind: agent ──────────────────────────────────────────────────────────
	// Each agent lives in its own subdirectory: xcf/agents/<id>/<id>.xcf
	// Flat files under xcf/agents/<id>.xcf are rejected by the parser.
	if len(config.Agents) > 0 {
		for _, k := range sortedMapKeys(config.Agents) {
			if agentFilter != nil && !agentFilter[k] {
				continue
			}
			agentSubDir := filepath.Join(xcfDir, "agents", k)
			if err := os.MkdirAll(agentSubDir, 0755); err != nil {
				return err
			}
			agent := config.Agents[k]
			if agent.Name == "" {
				agent.Name = k
			}
			body := strings.TrimSpace(agent.Body)
			doc := agentDoc{Kind: "agent", Version: version, AgentConfig: agent}
			if err := writeFrontmatterFile(filepath.Join(agentSubDir, k+".xcf"), doc, body); err != nil {
				return err
			}

			// Write agent overrides: agent.<provider>.xcf
			if config.Overrides != nil {
				if providers := config.Overrides.AgentProviders(k); len(providers) > 0 {
					for _, provider := range providers {
						overrideCfg, _ := config.Overrides.GetAgent(k, provider)
						overrideBody := strings.TrimSpace(overrideCfg.Body)
						overrideCfg.Body = "" // zero before YAML serialization
						overrideCfg.Name = "" // name is inferred from directory
						overridePath := filepath.Join(agentSubDir, fmt.Sprintf("agent.%s.xcf", provider))
						if err := writeFrontmatterFile(overridePath, overrideCfg, overrideBody); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// ── kind: skill ──────────────────────────────────────────────────────────
	// Each skill lives in its own subdirectory: xcf/skills/<name>/skill.xcf
	if len(config.Skills) > 0 {
		dir := filepath.Join(xcfDir, "skills")
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
			body := strings.TrimSpace(skill.Body)
			doc := skillDoc{Kind: "skill", Version: version, SkillConfig: skill}

			skillSubDir := filepath.Join(dir, k)
			if err := os.MkdirAll(skillSubDir, 0755); err != nil {
				return err
			}
			outPath := filepath.Join(skillSubDir, "skill.xcf")

			if err := writeFrontmatterFile(outPath, doc, body); err != nil {
				return err
			}

			// Write skill overrides: xcf/skills/<name>/skill.<provider>.xcf
			if config.Overrides != nil {
				if providers := config.Overrides.SkillProviders(k); len(providers) > 0 {
					for _, provider := range providers {
						overrideCfg, _ := config.Overrides.GetSkill(k, provider)
						overrideBody := strings.TrimSpace(overrideCfg.Body)
						overrideCfg.Body = ""
						overrideCfg.Name = ""
						overridePath := filepath.Join(skillSubDir, fmt.Sprintf("skill.%s.xcf", provider))
						if err := writeFrontmatterFile(overridePath, overrideCfg, overrideBody); err != nil {
							return err
						}
					}
				}
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
			if ruleFilter != nil && !ruleFilter[k] {
				continue
			}
			rule := config.Rules[k]
			if rule.Name == "" {
				rule.Name = k
			}
			body := strings.TrimSpace(rule.Body)
			doc := ruleDoc{Kind: "rule", Version: version, RuleConfig: rule}

			// Directory layout: xcf/rules/<name>/rule.xcf
			ruleSubDir := filepath.Join(dir, k)
			if err := os.MkdirAll(ruleSubDir, 0755); err != nil {
				return err
			}
			outPath := filepath.Join(ruleSubDir, "rule.xcf")

			if err := writeFrontmatterFile(outPath, doc, body); err != nil {
				return err
			}

			// Write rule overrides: rule.<provider>.xcf
			if config.Overrides != nil {
				if providers := config.Overrides.RuleProviders(k); len(providers) > 0 {
					for _, provider := range providers {
						overrideCfg, _ := config.Overrides.GetRule(k, provider)
						overrideBody := strings.TrimSpace(overrideCfg.Body)
						overrideCfg.Body = ""
						overrideCfg.Name = ""
						overridePath := filepath.Join(ruleSubDir, fmt.Sprintf("rule.%s.xcf", provider))
						if err := writeFrontmatterFile(overridePath, overrideCfg, overrideBody); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// ── kind: workflow ───────────────────────────────────────────────────────
	// Each workflow lives in its own subdirectory: xcf/workflows/<name>/workflow.xcf
	if len(config.Workflows) > 0 {
		dir := filepath.Join(xcfDir, "workflows")
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
			doc := workflowDoc{Kind: "workflow", Version: version, WorkflowConfig: wf}

			// Directory layout: xcf/workflows/<name>/workflow.xcf
			workflowSubDir := filepath.Join(dir, k)
			if err := os.MkdirAll(workflowSubDir, 0755); err != nil {
				return err
			}
			outPath := filepath.Join(workflowSubDir, "workflow.xcf")

			if err := writeYAMLFile(outPath, doc); err != nil {
				return err
			}

			// Write workflow overrides: workflow.<provider>.xcf
			if config.Overrides != nil {
				if providers := config.Overrides.WorkflowProviders(k); len(providers) > 0 {
					for _, provider := range providers {
						overrideCfg, _ := config.Overrides.GetWorkflow(k, provider)
						overrideCfg.Name = ""
						overridePath := filepath.Join(workflowSubDir, fmt.Sprintf("workflow.%s.xcf", provider))
						if err := writeYAMLFile(overridePath, overrideCfg); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// ── kind: mcp ────────────────────────────────────────────────────────────
	// Each MCP server lives in its own subdirectory: xcf/mcp/<name>/mcp.xcf
	if len(config.MCP) > 0 {
		dir := filepath.Join(xcfDir, "mcp")
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

			// Directory layout: xcf/mcp/<name>/mcp.xcf
			mcpSubDir := filepath.Join(dir, k)
			if err := os.MkdirAll(mcpSubDir, 0755); err != nil {
				return err
			}
			outPath := filepath.Join(mcpSubDir, "mcp.xcf")

			if err := writeYAMLFile(outPath, doc); err != nil {
				return err
			}

			// Write MCP overrides: mcp.<provider>.xcf
			if config.Overrides != nil {
				if providers := config.Overrides.MCPProviders(k); len(providers) > 0 {
					for _, provider := range providers {
						overrideCfg, _ := config.Overrides.GetMCP(k, provider)
						overrideCfg.Name = ""
						overridePath := filepath.Join(mcpSubDir, fmt.Sprintf("mcp.%s.xcf", provider))
						if err := writeYAMLFile(overridePath, overrideCfg); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// ── kind: policy ────────────────────────────────────────────────────────
	// Each policy lives in its own subdirectory: xcf/policy/<name>/policy.xcf
	if len(config.Policies) > 0 {
		dir := filepath.Join(xcfDir, "policy")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for _, k := range sortedMapKeys(config.Policies) {
			policy := config.Policies[k]
			if policy.Name == "" {
				policy.Name = k
			}
			doc := policyDoc{Kind: "policy", Version: version, PolicyConfig: policy}

			// Directory layout: xcf/policy/<name>/policy.xcf
			policySubDir := filepath.Join(dir, k)
			if err := os.MkdirAll(policySubDir, 0755); err != nil {
				return err
			}
			outPath := filepath.Join(policySubDir, "policy.xcf")

			if err := writeYAMLFile(outPath, doc); err != nil {
				return err
			}
		}
	}

	// ── kind: context ────────────────────────────────────────────────────────
	if len(config.Contexts) > 0 {
		dir := filepath.Join(xcfDir, "context")
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
			if err := writeFrontmatterFile(filepath.Join(dir, k+".xcf"), doc, body); err != nil {
				return err
			}
		}
	}

	// ── kind: hooks ──────────────────────────────────────────────────────────
	// Each named hook config lives in its own subdirectory: xcf/hooks/<name>/hooks.xcf
	if len(config.Hooks) > 0 {
		dir := filepath.Join(xcfDir, "hooks")
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
			}

			// Directory layout: xcf/hooks/<name>/hooks.xcf
			hookSubDir := filepath.Join(dir, k)
			if err := os.MkdirAll(hookSubDir, 0755); err != nil {
				return err
			}
			outPath := filepath.Join(hookSubDir, "hooks.xcf")

			if err := writeYAMLFile(outPath, doc); err != nil {
				return err
			}
		}
	}

	// ── kind: settings ───────────────────────────────────────────────────────
	// Each named settings config lives in its own subdirectory: xcf/settings/<name>/settings.xcf
	if len(config.Settings) > 0 {
		dir := filepath.Join(xcfDir, "settings")
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

			// Directory layout: xcf/settings/<name>/settings.xcf
			settingsSubDir := filepath.Join(dir, k)
			if err := os.MkdirAll(settingsSubDir, 0755); err != nil {
				return err
			}
			outPath := filepath.Join(settingsSubDir, "settings.xcf")

			if err := writeYAMLFile(outPath, doc); err != nil {
				return err
			}
		}
	}

	// ── provider passthrough (provider-specific files) ─────────────────────────
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
			var destRoot string
			if provider == "xcf" {
				// Canonical passthrough for hooks
				destRoot = filepath.Join(rootDir, "xcf")
			} else {
				destRoot = filepath.Join(rootDir, "xcf", "provider", provider)
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

// sortedAgentRefs returns AgentManifestEntry values sorted by ID for any
// string-keyed agent map. Used to build the AgentRefs list in kind: project.
func sortedAgentRefs[V any](m map[string]V) []ast.AgentManifestEntry {
	keys := sortedMapKeys(m)
	res := make([]ast.AgentManifestEntry, 0, len(keys))
	for _, k := range keys {
		res = append(res, ast.AgentManifestEntry{ID: k})
	}
	return res
}

// refSet builds a lookup set from a list of resource reference names.
// Returns nil when refs is empty, which callers interpret as "no filter — write all".
func refSet(refs []string) map[string]bool {
	if len(refs) == 0 {
		return nil
	}
	s := make(map[string]bool, len(refs))
	for _, r := range refs {
		s[r] = true
	}
	return s
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
