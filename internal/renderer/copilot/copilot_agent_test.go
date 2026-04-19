package copilot_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/internal/renderer/copilot"
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
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	const wantPath = "agents/my-agent.agent.md"
	content, ok := out.Files[wantPath]
	require.True(t, ok, "expected output file %s; got files: %v", wantPath, out.Files)
	assert.Contains(t, content, "---", "frontmatter delimiters must be present")
	assert.Contains(t, content, "description:", "description is a required Copilot agent field")
	assert.Contains(t, content, "A minimal test agent.")
}

// TestCompile_Copilot_Agents_FullSchema verifies that supported agent fields —
// tools, model, disable-model-invocation, and user-invocable — are rendered
// into the agent frontmatter without fidelity notes.
func TestCompile_Copilot_Agents_FullSchema(t *testing.T) {
	r := copilot.New()
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"full-agent": {
					Name:                   "Full Agent",
					Description:            "Agent with full schema.",
					Model:                  "gpt-4o",
					Tools:                  []string{"read_file", "write_file"},
					DisableModelInvocation: boolPtr(true),
					UserInvocable:          boolPtr(true),
				},
			},
		},
	}
	out, notes, err := r.Compile(config, "")
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
	out, _, err := r.Compile(config, "")
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
// are silently dropped and produce fidelity notes.
// Security fields (permission-mode, disallowed-tools, isolation) must produce a
// AGENT_SECURITY_FIELDS_DROPPED note. All others produce FIELD_UNSUPPORTED.
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
					Memory:          "long-term",
					MaxTurns:        10,
					Background:      &background,
					Color:           "blue",
					InitialPrompt:   "Hello!",
					Readonly:        &readonly,
				},
			},
		},
	}
	out, notes, err := r.Compile(config, "")
	require.NoError(t, err)

	const wantPath = "agents/unsupported-agent.agent.md"
	_, ok := out.Files[wantPath]
	require.True(t, ok, "expected output file %s", wantPath)

	// Security fields must produce AGENT_SECURITY_FIELDS_DROPPED.
	securityNotes := filterNotes(notes, renderer.CodeAgentSecurityFieldsDropped)
	require.NotEmpty(t, securityNotes,
		"expected AGENT_SECURITY_FIELDS_DROPPED for permission-mode, disallowed-tools, isolation")

	// Other unsupported fields must produce FIELD_UNSUPPORTED.
	unsupportedNotes := filterNotes(notes, renderer.CodeFieldUnsupported)
	require.NotEmpty(t, unsupportedNotes,
		"expected FIELD_UNSUPPORTED for effort, skills, memory, max-turns, background, color, initial-prompt, readonly")
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
	out, _, err := r.Compile(config, "")
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
					Name:         "Body Agent",
					Description:  "Agent with a body.",
					Instructions: "Always prefer idiomatic Go.",
				},
			},
		},
	}
	out, _, err := r.Compile(config, "")
	require.NoError(t, err)

	const wantPath = "agents/body-agent.agent.md"
	content, ok := out.Files[wantPath]
	require.True(t, ok, "expected output file %s", wantPath)

	assert.Contains(t, content, "---", "frontmatter delimiters must be present")
	assert.Contains(t, content, "description:")
	assert.Contains(t, content, "Always prefer idiomatic Go.",
		"instructions must appear in the markdown body after frontmatter")
}
