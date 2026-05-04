package copilot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers/copilot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompile_Copilot_Agents_Minimal verifies that an agent with only name and
// description produces a file at .github/agents/<id>.agent.md with a valid
// frontmatter block that includes the description field.
func TestCompile_Copilot_Agents_Minimal(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"my-agent": {
					Name:        "My Agent",
					Description: "A minimal test agent.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	const wantPath = "agents/my-agent.agent.md"
	content, ok := out.Files[wantPath]
	require.True(t, ok, "expected output file %s; got files: %v", wantPath, out.Files)
	assert.Contains(t, content, "---", "frontmatter delimiters must be present")
	assert.Contains(t, content, "description:", "description is a required Copilot agent field")
	assert.Contains(t, content, "A minimal test agent.")
}

// TestCompile_Copilot_Agents_FullSchema verifies that supported agent fields —
// tools and model — are rendered into the agent frontmatter without fidelity
// notes. Fields unsupported by copilot (disable-model-invocation,
// user-invocable) are excluded from this test.
func TestCompile_Copilot_Agents_FullSchema(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"full-agent": {
					Name:        "Full Agent",
					Description: "Agent with full schema.",
					Model:       "gpt-4o",
					Tools:       []string{"read_file", "write_file"},
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	const wantPath = "agents/full-agent.agent.md"
	content, ok := out.Files[wantPath]
	require.True(t, ok, "expected output file %s", wantPath)

	assert.Contains(t, content, "description:")
	assert.Contains(t, content, "Full Agent")
	assert.Contains(t, content, "model:")
	assert.Contains(t, content, "gpt-4o")
	assert.Contains(t, content, "read_file")
	assert.Contains(t, content, "write_file")

	// Supported fields must not produce fidelity notes for this agent.
	agentNotes := filterNotes(notes, renderer.CodeFieldUnsupported)
	for _, n := range agentNotes {
		assert.NotEqual(t, "full-agent", n.Resource,
			"unexpected FIELD_UNSUPPORTED note for supported field: %+v", n)
	}
}

// TestCompile_Copilot_Agents_ProviderPassthrough verifies that a value nested
// under targets.copilot.provider is passed through verbatim into the agent
// frontmatter. The test uses the "target" key.
func TestCompile_Copilot_Agents_ProviderPassthrough(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"passthrough-agent": {
					Name:        "Passthrough Agent",
					Description: "Agent with provider passthrough.",
					Targets: map[string]ast.TargetOverride{
						"copilot": {
							Provider: map[string]any{
								"target": "my-custom-target",
							},
						},
					},
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	const wantPath = "agents/passthrough-agent.agent.md"
	content, ok := out.Files[wantPath]
	require.True(t, ok, "expected output file %s", wantPath)

	assert.Contains(t, content, "target:", "provider passthrough key 'target' must appear in frontmatter")
	assert.Contains(t, content, "my-custom-target", "provider passthrough value must appear in frontmatter")
}

// TestCompile_Copilot_Agents_UnsupportedFields verifies that agent fields with
// no Copilot equivalent — effort, permission-mode, disallowed-tools, isolation,
// skills, memory, max-turns, background, color, initial-prompt, and readonly —
// are silently dropped and produce FIELD_UNSUPPORTED fidelity notes.
// Security fields (permission-mode, disallowed-tools, isolation) are now checked
// by the orchestrator's CheckFieldSupport and emit FIELD_UNSUPPORTED like all
// other unsupported fields.
func TestCompile_Copilot_Agents_UnsupportedFields(t *testing.T) {
	r := copilot.New()
	readonly := true
	background := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"unsupported-agent": {
					Name:            "Unsupported Agent",
					Description:     "Agent with unsupported fields.",
					Effort:          "high",
					PermissionMode:  "restricted",
					DisallowedTools: []string{"bash"},
					Isolation:       "sandbox",
					Skills:          []string{"my-skill"},
					Memory:          ast.FlexStringSlice{"long-term"},
					MaxTurns:        10,
					Background:      &background,
					Color:           "blue",
					InitialPrompt:   "Hello!",
					Readonly:        &readonly,
				},
			},
		},
	}
	out, notes, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	const wantPath = "agents/unsupported-agent.agent.md"
	_, ok := out.Files[wantPath]
	require.True(t, ok, "expected output file %s", wantPath)

	// All unsupported fields — including security fields — must produce
	// FIELD_UNSUPPORTED now that checks are centralized in the orchestrator.
	unsupportedNotes := filterNotes(notes, renderer.CodeFieldUnsupported)
	require.NotEmpty(t, unsupportedNotes,
		"expected FIELD_UNSUPPORTED for permission-mode, disallowed-tools, isolation, effort, skills, memory, max-turns, background, color, initial-prompt, readonly")

	// Verify security fields specifically appear in the FIELD_UNSUPPORTED notes.
	securityFields := map[string]bool{"permission-mode": false, "disallowed-tools": false, "isolation": false}
	for _, n := range unsupportedNotes {
		if _, ok := securityFields[n.Field]; ok {
			securityFields[n.Field] = true
		}
	}
	for field, found := range securityFields {
		assert.True(t, found, "FIELD_UNSUPPORTED note must be emitted for security field %q", field)
	}
}

// TestCompile_Copilot_Agents_InlineMCP verifies that inline mcp-servers declared
// on an agent are rendered as an mcp-servers: key in the agent frontmatter.
func TestCompile_Copilot_Agents_InlineMCP(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"mcp-agent": {
					Name:        "MCP Agent",
					Description: "Agent with inline MCP.",
					MCPServers: map[string]ast.MCPConfig{
						"filesystem": {
							Type:    "stdio",
							Command: "mcp-server-filesystem",
							Args:    []string{"/workspace"},
						},
					},
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	const wantPath = "agents/mcp-agent.agent.md"
	content, ok := out.Files[wantPath]
	require.True(t, ok, "expected output file %s", wantPath)

	assert.Contains(t, content, "mcp-servers:", "inline MCP must appear as mcp-servers: in frontmatter")
	assert.Contains(t, content, "filesystem", "MCP server id must appear in frontmatter")
}

// TestCompile_Copilot_Agents_WithBody verifies that an agent with instructions
// produces a file whose content includes both the YAML frontmatter block and the
// instructions as the markdown body after the closing --- delimiter.
func TestCompile_Copilot_Agents_WithBody(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"body-agent": {
					Name:        "Body Agent",
					Description: "Agent with a body.",
					Body:        "Always prefer idiomatic Go.",
				},
			},
		},
	}
	out, _, err := renderer.Orchestrate(r, config, "")
	require.NoError(t, err)

	const wantPath = "agents/body-agent.agent.md"
	content, ok := out.Files[wantPath]
	require.True(t, ok, "expected output file %s", wantPath)

	assert.Contains(t, content, "---", "frontmatter delimiters must be present")
	assert.Contains(t, content, "description:")
	assert.Contains(t, content, "Always prefer idiomatic Go.",
		"instructions must appear in the markdown body after frontmatter")
}

// TestCompileAgents_Copilot_ClaudeDirPresent_EmitsPassthroughNotes verifies that
// when a .claude/ directory is present, CompileAgents returns no output files and
// emits one CLAUDE_NATIVE_PASSTHROUGH info note per agent.
func TestCompileAgents_Copilot_ClaudeDirPresent_EmitsPassthroughNotes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))

	r := copilot.New()
	agents := map[string]ast.AgentConfig{
		"auth-specialist": {Name: "Auth Specialist", Description: "Handles auth."},
	}
	files, notes, err := r.CompileAgents(agents, dir)
	require.NoError(t, err)
	assert.Empty(t, files, "no .github/agents/ files should be written when .claude/ is present")
	require.Len(t, notes, 1)
	assert.Equal(t, renderer.CodeClaudeNativePassthrough, notes[0].Code)
	assert.Equal(t, renderer.LevelInfo, notes[0].Level)
	assert.Equal(t, "auth-specialist", notes[0].Resource)
	assert.Contains(t, notes[0].Reason, ".claude/agents/auth-specialist.md")
}

// TestCompileAgents_Copilot_NoClaude_FullTranslation verifies that when no
// .claude/ directory exists, CompileAgents falls through to full translation
// and writes the .github/agents/<id>.agent.md file.
func TestCompileAgents_Copilot_NoClaude_FullTranslation(t *testing.T) {
	dir := t.TempDir() // no .claude/ subdirectory

	r := copilot.New()
	agents := map[string]ast.AgentConfig{
		"my-agent": {Name: "My Agent", Description: "A test agent."},
	}
	files, notes, err := r.CompileAgents(agents, dir)
	require.NoError(t, err)
	assert.Contains(t, files, "agents/my-agent.agent.md",
		"full translation must write .github/agents/ when .claude/ is absent")
	for _, n := range notes {
		assert.NotEqual(t, renderer.CodeClaudeNativePassthrough, n.Code,
			"no CLAUDE_NATIVE_PASSTHROUGH notes expected when .claude/ is absent")
	}
}
