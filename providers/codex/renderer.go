// Package codex compiles an XcaffoldConfig AST into Codex output files.
// Project instructions are written to AGENTS.md at the project root.
// Agents are written to .codex/agents/<id>.toml using the Codex native TOML format.
// Skills are written to .agents/skills/<id>/SKILL.md (moved from .codex/ by Finalize).
// MCP servers are written to .codex/config.toml.
// Rules, workflows, settings, and memory are unsupported; the orchestrator emits
// RENDERER_KIND_UNSUPPORTED notes based on Capabilities().
package codex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/resolver"
)

const targetName = "codex"

// codexAgent is the TOML representation of a single Codex agent definition.
type codexAgent struct {
	Name                  string `toml:"name"`
	Description           string `toml:"description"`
	DeveloperInstructions string `toml:"developer_instructions"`
	Model                 string `toml:"model,omitempty"`
	ModelReasoningEffort  string `toml:"model_reasoning_effort,omitempty"`
	SandboxMode           string `toml:"sandbox_mode,omitempty"`
}

// codexMCPServer is the TOML representation of a single MCP server entry.
type codexMCPServer struct {
	Env     map[string]string `toml:"env,omitempty"`
	Command string            `toml:"command,omitempty"`
	URL     string            `toml:"url,omitempty"`
	Type    string            `toml:"type,omitempty"`
	Args    []string          `toml:"args,omitempty"`
}

// Renderer compiles an XcaffoldConfig AST into Codex output files.
type Renderer struct{}

// New returns a new Renderer instance.
func New() *Renderer { return &Renderer{} }

// Target returns the canonical name of this renderer.
func (r *Renderer) Target() string { return targetName }

// OutputDir returns the base output directory for Codex.
func (r *Renderer) OutputDir() string { return ".codex" }

// Capabilities declares the resource kinds the Codex renderer supports.
func (r *Renderer) Capabilities() renderer.CapabilitySet {
	return renderer.CapabilitySet{
		Agents:               true,
		Skills:               true,
		Rules:                false,
		Workflows:            false,
		Hooks:                true,
		Settings:             false,
		MCP:                  true,
		Memory:               false,
		ProjectInstructions:  true,
		AgentNativeToolsOnly: false,
		SkillArtifactDirs: map[string]string{
			"references": "references",
			"scripts":    "scripts",
			"assets":     "assets",
		},
		RuleActivations: []string{},
		RuleEncoding:    renderer.RuleEncodingCapabilities{},
	}
}

// CompileAgents compiles all agent configs to agents/<id>.toml files.
func (r *Renderer) CompileAgents(agents map[string]ast.AgentConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	if len(agents) == 0 {
		return map[string]string{}, nil, nil
	}
	files := make(map[string]string)
	var notes []renderer.FidelityNote
	for _, id := range renderer.SortedKeys(agents) {
		agent := agents[id]
		if agent.Inherited {
			continue
		}
		content, agentNotes, err := compileCodexAgent(id, agent)
		if err != nil {
			return nil, nil, fmt.Errorf("codex: agent %q: %w", id, err)
		}
		files[filepath.Clean(fmt.Sprintf("agents/%s.toml", id))] = content
		notes = append(notes, agentNotes...)
	}
	return files, notes, nil
}

// CompileSkills compiles all skill configs to skills/<id>/SKILL.md files.
// Finalize moves these to .agents/skills/<id>/SKILL.md in rootFiles.
func (r *Renderer) CompileSkills(skills map[string]ast.SkillConfig, baseDir string) (map[string]string, []renderer.FidelityNote, error) {
	if len(skills) == 0 {
		return map[string]string{}, nil, nil
	}
	files := make(map[string]string)
	var notes []renderer.FidelityNote
	caps := r.Capabilities()
	for _, id := range renderer.SortedKeys(skills) {
		skill := skills[id]
		files[filepath.Clean(fmt.Sprintf("skills/%s/SKILL.md", id))] = buildCodexSkillContent(skill)
		artifactNotes := renderer.CompileArtifactsDemoted(targetName, renderer.ArtifactJob{
			ID: id, BaseDir: baseDir, Caps: caps, Files: files,
		}, skill.Artifacts)
		notes = append(notes, artifactNotes...)
	}
	return files, notes, nil
}

// CompileRules is a no-op. Codex does not support rules; the orchestrator
// emits RENDERER_KIND_UNSUPPORTED based on Capabilities().Rules == false.
func (r *Renderer) CompileRules(_ map[string]ast.RuleConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	return map[string]string{}, nil, nil
}

// CompileWorkflows is a no-op. Codex does not support workflows; the orchestrator
// emits RENDERER_KIND_UNSUPPORTED based on Capabilities().Workflows == false.
func (r *Renderer) CompileWorkflows(_ map[string]ast.WorkflowConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	return map[string]string{}, nil, nil
}

// CompileHooks encodes the hook config as JSON to hooks.json.
// The output is wrapped in a "hooks" key to match the Codex native format
// that the importer expects.
func (r *Renderer) CompileHooks(hooks ast.HookConfig, _ string) (map[string]string, []renderer.FidelityNote, error) {
	if len(hooks) == 0 {
		return map[string]string{}, nil, nil
	}
	wrapper := struct {
		Hooks ast.HookConfig `json:"hooks"`
	}{Hooks: hooks}
	b, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("codex: marshal hooks: %w", err)
	}
	return map[string]string{"hooks.json": string(b)}, nil, nil
}

// CompileSettings is a no-op. Codex does not support settings; the orchestrator
// emits RENDERER_KIND_UNSUPPORTED based on Capabilities().Settings == false.
func (r *Renderer) CompileSettings(_ ast.SettingsConfig) (map[string]string, []renderer.FidelityNote, error) {
	return map[string]string{}, nil, nil
}

// CompileMCP writes MCP server declarations to config.toml.
func (r *Renderer) CompileMCP(servers map[string]ast.MCPConfig) (map[string]string, []renderer.FidelityNote, error) {
	if len(servers) == 0 {
		return map[string]string{}, nil, nil
	}
	content, err := buildMCPConfig(servers)
	if err != nil {
		return nil, nil, fmt.Errorf("codex: encode MCP config: %w", err)
	}
	return map[string]string{"config.toml": content}, nil, nil
}

// CompileProjectInstructions writes AGENTS.md at the project root.
func (r *Renderer) CompileProjectInstructions(config *ast.XcaffoldConfig, _ string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	body := renderer.ResolveContextBody(config, targetName)
	if body == "" {
		return nil, map[string]string{}, nil, nil
	}
	return nil, map[string]string{"AGENTS.md": body}, nil, nil
}

// CompileMemory is a no-op. Codex does not support memory; the orchestrator
// emits RENDERER_KIND_UNSUPPORTED based on Capabilities().Memory == false.
func (r *Renderer) CompileMemory(_ *ast.XcaffoldConfig, _ string, _ renderer.MemoryOptions) (map[string]string, []renderer.FidelityNote, error) {
	return map[string]string{}, nil, nil
}

// Finalize moves skills from the .codex output dir into rootFiles under
// .agents/skills/ so they land alongside the AGENTS.md open-standard layout.
func (r *Renderer) Finalize(files map[string]string, rootFiles map[string]string) (map[string]string, map[string]string, []renderer.FidelityNote, error) {
	var toDelete []string
	for key, content := range files {
		if !strings.HasPrefix(key, "skills/") {
			continue
		}
		rootFiles[".agents/"+key] = content
		toDelete = append(toDelete, key)
	}
	for _, key := range toDelete {
		delete(files, key)
	}
	return files, rootFiles, nil, nil
}

// compileCodexAgent builds the TOML content for a single agent and collects
// fidelity notes for unsupported fields.
func compileCodexAgent(id string, agent ast.AgentConfig) (string, []renderer.FidelityNote, error) {
	ca := codexAgent{
		Name:        agent.Name,
		Description: agent.Description,
	}
	ca.DeveloperInstructions = strings.TrimRight(resolver.StripFrontmatter(agent.Body), "\n")

	if agent.Model != "" {
		if modelID, ok := renderer.ResolveModel(agent.Model, targetName); ok {
			ca.Model = modelID
		}
	}
	if agent.Effort != "" {
		ca.ModelReasoningEffort = agent.Effort
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(ca); err != nil {
		return "", nil, err
	}

	notes := codexAgentFidelityNotes(id, agent)
	return buf.String(), notes, nil
}

// codexAgentFidelityNotes returns FIELD_UNSUPPORTED notes for agent fields
// that Codex does not support natively.
func codexAgentFidelityNotes(id string, agent ast.AgentConfig) []renderer.FidelityNote {
	type field struct {
		name    string
		present bool
	}
	unsupported := []field{
		{"tools", len(agent.Tools.Values) > 0},
		{"disallowed-tools", len(agent.DisallowedTools.Values) > 0},
		{"max-turns", agent.MaxTurns != nil},
		{"readonly", agent.Readonly != nil},
		{"permission-mode", agent.PermissionMode != ""},
		{"disable-model-invocation", agent.DisableModelInvocation != nil},
		{"user-invocable", agent.UserInvocable != nil},
		{"background", agent.Background != nil},
		{"color", agent.Color != ""},
		{"initial-prompt", agent.InitialPrompt != ""},
		{"memory", len(agent.Memory) > 0},
		{"hooks", len(agent.Hooks) > 0},
	}
	var notes []renderer.FidelityNote
	for _, f := range unsupported {
		if !f.present {
			continue
		}
		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelWarning,
			Target:     targetName,
			Kind:       "agent",
			Resource:   id,
			Field:      f.name,
			Code:       renderer.CodeFieldUnsupported,
			Reason:     fmt.Sprintf("agent %q field %q has no Codex equivalent and was dropped", id, f.name),
			Mitigation: "Remove the field or use a provider that supports it",
		})
	}
	return notes
}

// buildCodexSkillContent renders the SKILL.md content for a single Codex skill.
// Codex supports name, description, and when-to-use in frontmatter.
func buildCodexSkillContent(skill ast.SkillConfig) string {
	body := resolver.StripFrontmatter(skill.Body)
	var sb strings.Builder
	sb.WriteString("---\n")
	if skill.Name != "" {
		fmt.Fprintf(&sb, "name: %s\n", skill.Name)
	}
	if skill.Description != "" {
		fmt.Fprintf(&sb, "description: %s\n", skill.Description)
	}
	if skill.WhenToUse != "" {
		fmt.Fprintf(&sb, "when-to-use: %s\n", skill.WhenToUse)
	}
	sb.WriteString("---\n")
	if body != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimRight(body, "\n"))
		sb.WriteString("\n")
	}
	return sb.String()
}

// buildMCPConfig encodes MCP servers as a TOML config file with [mcp_servers.*] sections.
func buildMCPConfig(servers map[string]ast.MCPConfig) (string, error) {
	type mcpDoc struct {
		MCPServers map[string]codexMCPServer `toml:"mcp_servers"`
	}
	doc := mcpDoc{MCPServers: make(map[string]codexMCPServer, len(servers))}
	for _, id := range renderer.SortedKeys(servers) {
		srv := servers[id]
		doc.MCPServers[id] = codexMCPServer{
			Command: srv.Command,
			Args:    srv.Args,
			URL:     srv.URL,
			Type:    srv.Type,
			Env:     srv.Env,
		}
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}
