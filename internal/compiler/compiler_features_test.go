package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Feature 1A: instructions-file: external file references
// ---------------------------------------------------------------------------

// TestCompile_AgentInstructionsFile_ReadsExternalFile verifies that an agent
// with instructions-file: uses the file body as its system prompt.
func TestCompile_AgentInstructionsFile_ReadsExternalFile(t *testing.T) {
	dir := t.TempDir()
	instrPath := filepath.Join(dir, "agents", "cto.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(instrPath), 0755))
	require.NoError(t, os.WriteFile(instrPath, []byte("You are the Chief Technology Officer.\nLead with clarity."), 0600))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"cto": {
					Description:      "Chief Technology Officer",
					InstructionsFile: "agents/cto.md",
				},
			},
		},
	}

	out, err := Compile(config, dir, "")
	require.NoError(t, err)

	content, ok := out.Files["agents/cto.md"]
	require.True(t, ok, "agents/cto.md should be compiled")
	assert.Contains(t, content, "You are the Chief Technology Officer.")
	assert.Contains(t, content, "Lead with clarity.")
	assert.Contains(t, content, "description: Chief Technology Officer")
}

// TestCompile_AgentInstructionsFile_StripsFrontmatter verifies that frontmatter
// in an instructions-file is stripped before being used as the prompt body.
func TestCompile_AgentInstructionsFile_StripsFrontmatter(t *testing.T) {
	dir := t.TempDir()
	instrPath := filepath.Join(dir, "agents", "dev.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(instrPath), 0755))
	content := "---\nname: Developer\nmodel: claude-sonnet\n---\n\nWrite clean code.\nAlways test."
	require.NoError(t, os.WriteFile(instrPath, []byte(content), 0600))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"developer": {InstructionsFile: "agents/dev.md"},
			},
		},
	}

	out, err := Compile(config, dir, "")
	require.NoError(t, err)

	compiled := out.Files["agents/developer.md"]
	assert.Contains(t, compiled, "Write clean code.")
	assert.NotContains(t, compiled, "name: Developer", "frontmatter should be stripped from file body")
}

// TestCompile_AgentInstructionsFile_Missing_ReturnsError verifies that a
// missing instructions-file causes a compile error, not silent empty content.
func TestCompile_AgentInstructionsFile_Missing_ReturnsError(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"cto": {InstructionsFile: "nonexistent/cto.md"},
			},
		},
	}

	_, err := Compile(config, t.TempDir(), "")
	require.Error(t, err, "missing instructions-file must return an error")
	assert.Contains(t, err.Error(), "nonexistent/cto.md")
}

// TestCompile_InstructionsFile_PathTraversal_Rejected verifies that
// instructions-file paths that escape the project root are rejected.
func TestCompile_InstructionsFile_PathTraversal_Rejected(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"evil": {InstructionsFile: "../../etc/passwd"},
			},
		},
	}

	_, err := Compile(config, t.TempDir(), "")
	require.Error(t, err, "traversal paths in instructions-file must be rejected")
}

// TestCompile_InstructionsFile_InlineWins verifies that inline "instructions:"
// takes priority over "instructions-file:" when both are set in the AST
// (this case is normally caught by the parser, but the compiler should also
// be defensive and honour the priority ordering).
func TestCompile_InstructionsFile_InlinePriority(t *testing.T) {
	dir := t.TempDir()
	fPath := filepath.Join(dir, "file.md")
	require.NoError(t, os.WriteFile(fPath, []byte("From file."), 0600))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent": {
					Instructions:     "From inline.",
					InstructionsFile: "file.md",
				},
			},
		},
	}

	// Parser would reject this, but we test the compiler directly.
	out, err := Compile(config, dir, "")
	require.NoError(t, err)
	content := out.Files["agents/agent.md"]
	assert.Contains(t, content, "From inline.", "inline instructions must take priority")
	assert.NotContains(t, content, "From file.")
}

// ---------------------------------------------------------------------------
// Feature 1B: references: skill supplementary files
// ---------------------------------------------------------------------------

// TestCompile_SkillWithReferences_CopiesFiles verifies that reference files
// declared in skills.references are copied into skills/<id>/references/.
func TestCompile_SkillWithReferences_CopiesFiles(t *testing.T) {
	dir := t.TempDir()
	refDir := filepath.Join(dir, "skills", "flutter-integration", "references")
	require.NoError(t, os.MkdirAll(refDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(refDir, "advanced-patterns.md"), []byte("# Advanced Patterns"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(refDir, "lottie-guide.md"), []byte("# Lottie Guide"), 0600))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"flutter-integration": {
					Description:  "Flutter SVG and Lottie integration",
					Instructions: "Integrate SVG and Lottie into Flutter apps.",
					References: []string{
						"skills/flutter-integration/references/advanced-patterns.md",
						"skills/flutter-integration/references/lottie-guide.md",
					},
				},
			},
		},
	}

	out, err := Compile(config, dir, "")
	require.NoError(t, err)

	_, hasSkill := out.Files["skills/flutter-integration/SKILL.md"]
	assert.True(t, hasSkill, "SKILL.md must be compiled")

	refContent, hasRef := out.Files["skills/flutter-integration/references/advanced-patterns.md"]
	assert.True(t, hasRef, "reference file must be in output")
	assert.Contains(t, refContent, "Advanced Patterns")

	_, hasRef2 := out.Files["skills/flutter-integration/references/lottie-guide.md"]
	assert.True(t, hasRef2, "second reference file must be in output")
}

// TestCompile_SkillReferences_Glob_ExpandsCorrectly verifies that glob patterns
// in references: expand to multiple files.
func TestCompile_SkillReferences_Glob_ExpandsCorrectly(t *testing.T) {
	dir := t.TempDir()
	refDir := filepath.Join(dir, "skills", "design", "refs")
	require.NoError(t, os.MkdirAll(refDir, 0755))
	for _, name := range []string{"colors.md", "typography.md", "layout.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(refDir, name), []byte("# "+name), 0600))
	}

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"design": {
					Instructions: "Design system patterns.",
					References:   []string{"skills/design/refs/*.md"},
				},
			},
		},
	}

	out, err := Compile(config, dir, "")
	require.NoError(t, err)

	refCount := 0
	for key := range out.Files {
		if filepath.Dir(key) == filepath.Clean("skills/design/references") {
			refCount++
		}
	}
	assert.Equal(t, 3, refCount, "glob should expand to all 3 reference files")
}

// TestCompile_SkillReferences_PathTraversal_Rejected verifies traversal is blocked.
func TestCompile_SkillReferences_PathTraversal_Rejected(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"evil": {
					Instructions: "Some skill.",
					References:   []string{"../../etc/shadow"},
				},
			},
		},
	}
	_, err := Compile(config, t.TempDir(), "")
	require.Error(t, err, "traversal references must be rejected")
}

// ---------------------------------------------------------------------------
// Feature 2A: settings type fixes
// ---------------------------------------------------------------------------

// TestCompile_Settings_StatusLine_IsObject verifies that statusLine emits as
// a JSON object ({"type":"command","command":"..."}) not a plain string.
func TestCompile_Settings_StatusLine_IsObject(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			StatusLine: &ast.StatusLineConfig{
				Type:    "command",
				Command: "bash ~/.claude/statusline.sh",
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	slAny, has := parsed["statusLine"]
	require.True(t, has, "statusLine must be present")
	slMap, ok := slAny.(map[string]any)
	require.True(t, ok, "statusLine must be an object, not a string")
	assert.Equal(t, "command", slMap["type"])
	assert.Equal(t, "bash ~/.claude/statusline.sh", slMap["command"])
}

// TestCompile_Settings_EnabledPlugins_IsMap verifies that enabledPlugins emits
// as a JSON object (map[string]bool) not an array.
func TestCompile_Settings_EnabledPlugins_IsMap(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			EnabledPlugins: map[string]bool{
				"plugin-a": true,
				"plugin-b": false,
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	epAny, has := parsed["enabledPlugins"]
	require.True(t, has, "enabledPlugins must be present")
	epMap, ok := epAny.(map[string]any)
	require.True(t, ok, "enabledPlugins must be an object (map), not an array")
	assert.Equal(t, true, epMap["plugin-a"])
	assert.Equal(t, false, epMap["plugin-b"])
}

// TestCompile_Settings_Schema_IsFirstKey verifies that $schema is emitted
// as the first key in settings.json.
func TestCompile_Settings_Schema_IsFirstKey(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			StatusLine: &ast.StatusLineConfig{Type: "command"},
		},
	}
	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw := out.Files["settings.json"]
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	_, has := parsed["$schema"]
	assert.True(t, has, "settings.json must contain $schema key")
}

// ---------------------------------------------------------------------------
// Feature: stripFrontmatter helper
// ---------------------------------------------------------------------------

func TestStripFrontmatter_WithFrontmatter(t *testing.T) {
	input := "---\nname: CTO\nmodel: claude-sonnet\n---\n\nYou are the CTO.\nLead with clarity."
	result := stripFrontmatter(input)
	assert.Equal(t, "You are the CTO.\nLead with clarity.", result)
	assert.NotContains(t, result, "name:")
}

func TestStripFrontmatter_WithoutFrontmatter(t *testing.T) {
	input := "You are the CTO.\nLead with clarity."
	result := stripFrontmatter(input)
	assert.Equal(t, input, result)
}

func TestStripFrontmatter_EmptyFile(t *testing.T) {
	result := stripFrontmatter("")
	assert.Equal(t, "", result)
}

func TestStripFrontmatter_OnlyFrontmatter(t *testing.T) {
	input := "---\nname: CTO\n---\n"
	result := stripFrontmatter(input)
	assert.Equal(t, "", result)
}

// ---------------------------------------------------------------------------
// Feature 4A: Convention-over-configuration auto-discovery
// ---------------------------------------------------------------------------

// TestCompile_ConventionAutoDiscover_Agent verifies that when an agent has no
// instructions or instructions-file, the compiler auto-discovers agents/<id>.md.
func TestCompile_ConventionAutoDiscover_Agent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agents"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "agents", "cto.md"),
		[]byte("You are the CTO. Lead with clarity."),
		0600,
	))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"cto": {Description: "Chief Technology Officer"},
				// No instructions or instructions-file — relies on convention
			},
		},
	}

	out, err := Compile(config, dir, "")
	require.NoError(t, err)

	content := out.Files["agents/cto.md"]
	assert.Contains(t, content, "You are the CTO.")
}

// TestCompile_ConventionAutoDiscover_Skill verifies that skills/<id>/SKILL.md
// is auto-discovered by convention when no instructions fields are set.
func TestCompile_ConventionAutoDiscover_Skill(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "skills", "git-workflow"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "skills", "git-workflow", "SKILL.md"),
		[]byte("---\nname: Git Workflow\n---\n\nFollow the git workflow."),
		0600,
	))

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: map[string]ast.SkillConfig{
				"git-workflow": {Description: "Git workflow patterns"},
				// No instructions or instructions-file — relies on convention
			},
		},
	}

	out, err := Compile(config, dir, "")
	require.NoError(t, err)

	content := out.Files["skills/git-workflow/SKILL.md"]
	assert.Contains(t, content, "Follow the git workflow.")
	assert.NotContains(t, content, "name: Git Workflow", "frontmatter should be stripped")
}

// TestCompile_ConventionAutoDiscover_MissingFile_SilentEmpty verifies that
// when the convention file doesn't exist, the resource compiles with an empty
// body (not an error).
func TestCompile_ConventionAutoDiscover_MissingFile_SilentEmpty(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"cto": {Description: "CTO agent"},
				// No agents/cto.md exists in baseDir
			},
		},
	}

	out, err := Compile(config, t.TempDir(), "") // empty tempdir — no convention file
	require.NoError(t, err, "missing convention file must be silent, not an error")

	content := out.Files["agents/cto.md"]
	// Should compile the frontmatter only, with empty body
	assert.Contains(t, content, "description: CTO agent")
}

// ---------------------------------------------------------------------------
// Feature: Sandbox configuration emits to settings.json
// ---------------------------------------------------------------------------

func TestCompile_Settings_SandboxConfig_EmitsCorrectly(t *testing.T) {
	trueVal := true
	falseVal := false
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Sandbox: &ast.SandboxConfig{
				Enabled:                  &trueVal,
				AutoAllowBashIfSandboxed: &trueVal,
				AllowUnsandboxedCommands: &falseVal,
				ExcludedCommands:         []string{"docker *"},
				Filesystem: &ast.SandboxFilesystem{
					AllowWrite: []string{"~/.kube", "/tmp/build"},
					DenyRead:   []string{"~/"},
					AllowRead:  []string{"."},
				},
				Network: &ast.SandboxNetwork{
					AllowedDomains: []string{"registry.npmjs.org", "github.com"},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	sandboxAny, has := parsed["sandbox"]
	require.True(t, has, "sandbox must be present")
	sandboxMap, ok := sandboxAny.(map[string]any)
	require.True(t, ok, "sandbox must be an object")

	assert.Equal(t, true, sandboxMap["enabled"])
	assert.Equal(t, true, sandboxMap["autoAllowBashIfSandboxed"])
	assert.Equal(t, false, sandboxMap["allowUnsandboxedCommands"])

	fsMap, ok := sandboxMap["filesystem"].(map[string]any)
	require.True(t, ok, "filesystem must be an object")
	allowWrite, ok := fsMap["allowWrite"].([]any)
	require.True(t, ok, "allowWrite must be an array")
	assert.Len(t, allowWrite, 2)

	netMap, ok := sandboxMap["network"].(map[string]any)
	require.True(t, ok, "network must be an object")
	domains, ok := netMap["allowedDomains"].([]any)
	require.True(t, ok, "allowedDomains must be an array")
	assert.Len(t, domains, 2)
}

// ---------------------------------------------------------------------------
// Feature: MCP HTTP transport support
// ---------------------------------------------------------------------------

func TestCompile_MCP_HTTPTransport_EmitsCorrectly(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			MCP: map[string]ast.MCPConfig{
				"github": {
					Type: "http",
					URL:  "https://api.github.com/mcp",
					Headers: map[string]string{
						"Authorization": "Bearer ${GITHUB_TOKEN}",
					},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["mcp.json"]
	require.True(t, ok, "mcp.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	mcpServers, ok := parsed["mcpServers"].(map[string]any)
	require.True(t, ok)

	gh, ok := mcpServers["github"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "http", gh["type"])
	assert.Equal(t, "https://api.github.com/mcp", gh["url"])

	headers, ok := gh["headers"].(map[string]any)
	require.True(t, ok, "headers must be an object")
	assert.Equal(t, "Bearer ${GITHUB_TOKEN}", headers["Authorization"])
}

// ---------------------------------------------------------------------------
// Feature: Typed Permissions (allow/deny/ask)
// ---------------------------------------------------------------------------

func TestCompile_Settings_TypedPermissions_EmitsCorrectly(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Permissions: &ast.PermissionsConfig{
				Allow: []string{"Bash(npm test *)", "Read(**/*.ts)"},
				Deny:  []string{"Bash(rm -rf *)"},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	permsAny, has := parsed["permissions"]
	require.True(t, has, "permissions must be present")
	permsMap, ok := permsAny.(map[string]any)
	require.True(t, ok, "permissions must be an object")

	allowAny, ok := permsMap["allow"].([]any)
	require.True(t, ok, "allow must be an array")
	assert.Len(t, allowAny, 2)

	denyAny, ok := permsMap["deny"].([]any)
	require.True(t, ok, "deny must be an array")
	assert.Len(t, denyAny, 1)

	// "ask" should be omitted when empty
	_, hasAsk := permsMap["ask"]
	assert.False(t, hasAsk, "empty ask list should be omitted from JSON")
}

// ---------------------------------------------------------------------------
// Feature: 3-level nested hooks JSON structure
// ---------------------------------------------------------------------------

func TestCompile_Hooks_ThreeLevelNested_StructureCorrect(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PostToolUse": []ast.HookMatcherGroup{
					{
						Matcher: "Write|Edit",
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "npx prettier --write $FILE", Timeout: intPtr(10000)},
						},
					},
				},
				"Notification": []ast.HookMatcherGroup{
					{
						Hooks: []ast.HookHandler{
							{Type: "command", Command: "echo 'notification received'"},
						},
					},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json should exist in output")

	// Must be valid JSON with {hooks: {event: [...]}} structure
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooksWrapper, ok := parsed["hooks"].(map[string]any)
	require.True(t, ok, "top-level must have 'hooks' key")

	// PostToolUse event
	postToolUse, ok := hooksWrapper["PostToolUse"].([]any)
	require.True(t, ok, "PostToolUse must be an array")
	require.Len(t, postToolUse, 1)

	matcherGroup, ok := postToolUse[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Write|Edit", matcherGroup["matcher"])

	handlers, ok := matcherGroup["hooks"].([]any)
	require.True(t, ok, "hooks must be an array")
	handler, ok := handlers[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "command", handler["type"])
	assert.Equal(t, "npx prettier --write $FILE", handler["command"])
	assert.Equal(t, float64(10000), handler["timeout"])

	// Notification event (no matcher)
	notification, ok := hooksWrapper["Notification"].([]any)
	require.True(t, ok, "Notification must be an array")
	require.Len(t, notification, 1)
}

// ---------------------------------------------------------------------------
// Feature: New agent frontmatter fields
// ---------------------------------------------------------------------------

func TestCompile_Agent_NewFields_EmitCorrectly(t *testing.T) {
	bgTrue := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"secure": {
					Description:    "Secure agent",
					PermissionMode: "plan",
					Background:     &bgTrue,
					Isolation:      "worktree",
					Color:          "blue",
					InitialPrompt:  "Hello, how can I help?",
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	content := out.Files["agents/secure.md"]
	assert.Contains(t, content, "permission-mode: plan")
	assert.Contains(t, content, "background: true")
	assert.Contains(t, content, "isolation: worktree")
	assert.Contains(t, content, "color: blue")
	assert.Contains(t, content, "initial-prompt: Hello, how can I help?")
}

// ---------------------------------------------------------------------------
// Feature: Agent-scoped hooks and mcpServers
// ---------------------------------------------------------------------------

func TestCompile_Agent_ScopedHooksAndMCP_EmitCorrectly(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"hooked": {
					Description: "Agent with hooks and MCP",
					Hooks: ast.HookConfig{
						"PreToolUse": []ast.HookMatcherGroup{
							{
								Matcher: "Bash",
								Hooks: []ast.HookHandler{
									{Type: "command", Command: "echo check"},
								},
							},
						},
					},
					MCPServers: map[string]ast.MCPConfig{
						"local-db": {
							Command: "npx",
							Args:    []string{"-y", "sqlite-mcp"},
						},
					},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	content := out.Files["agents/hooked.md"]
	assert.Contains(t, content, "hooks:")
	assert.Contains(t, content, "PreToolUse")
	assert.Contains(t, content, "mcp-servers:")
	assert.Contains(t, content, "local-db")
}

// ---------------------------------------------------------------------------
// Feature: OtelHeadersHelper, DisableAllHooks, Attribution settings
// ---------------------------------------------------------------------------

func TestCompile_Settings_NewFields_EmitCorrectly(t *testing.T) {
	trueVal := true
	falseVal := false
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			OtelHeadersHelper: "/bin/generate_headers.sh",
			DisableAllHooks:   &falseVal,
			Attribution:       &trueVal,
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	assert.Equal(t, "/bin/generate_headers.sh", parsed["otelHeadersHelper"])
	assert.Equal(t, false, parsed["disableAllHooks"])
	assert.Equal(t, true, parsed["attribution"])
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func intPtr(i int) *int { return &i }

// ---------------------------------------------------------------------------
// Feature: HookHandler — http, prompt, and full field coverage
// ---------------------------------------------------------------------------

func TestCompile_Hooks_HTTPHandler_EmitsCorrectly(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": []ast.HookMatcherGroup{
					{
						Matcher: "Bash",
						Hooks: []ast.HookHandler{
							{
								Type: "http",
								URL:  "https://hooks.internal/validate",
								Headers: map[string]string{
									"Authorization": "Bearer ${API_KEY}",
								},
								AllowedEnvVars: []string{"API_KEY"},
								Timeout:        intPtr(30),
							},
						},
					},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw := out.Files["settings.json"]
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooks := parsed["hooks"].(map[string]any)
	groups := hooks["PreToolUse"].([]any)
	handler := groups[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)

	assert.Equal(t, "http", handler["type"])
	assert.Equal(t, "https://hooks.internal/validate", handler["url"])
	assert.NotNil(t, handler["headers"])
	assert.NotNil(t, handler["allowedEnvVars"])
	assert.Equal(t, float64(30), handler["timeout"])
}

func TestCompile_Hooks_PromptHandler_EmitsCorrectly(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"PreToolUse": []ast.HookMatcherGroup{
					{
						Matcher: "Edit",
						Hooks: []ast.HookHandler{
							{
								Type:    "prompt",
								Prompt:  "Verify this edit follows our coding standards",
								Model:   "claude-haiku-4-5-20251001",
								Timeout: intPtr(30),
							},
						},
					},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw := out.Files["settings.json"]
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooks := parsed["hooks"].(map[string]any)
	groups := hooks["PreToolUse"].([]any)
	handler := groups[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)

	assert.Equal(t, "prompt", handler["type"])
	assert.Equal(t, "Verify this edit follows our coding standards", handler["prompt"])
	assert.Equal(t, "claude-haiku-4-5-20251001", handler["model"])
}

func TestCompile_Hooks_AllHandlerFields_EmitCorrectly(t *testing.T) {
	asyncTrue := true
	onceTrue := true
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Hooks: ast.HookConfig{
				"SessionStart": []ast.HookMatcherGroup{
					{
						Hooks: []ast.HookHandler{
							{
								Type:          "command",
								Command:       "./scripts/setup.sh",
								Timeout:       intPtr(120),
								Async:         &asyncTrue,
								Once:          &onceTrue,
								Shell:         "bash",
								StatusMessage: "Running setup...",
								If:            "Bash(./scripts/*)",
							},
						},
					},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw := out.Files["settings.json"]
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	hooks := parsed["hooks"].(map[string]any)
	groups := hooks["SessionStart"].([]any)
	handler := groups[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)

	assert.Equal(t, "command", handler["type"])
	assert.Equal(t, "./scripts/setup.sh", handler["command"])
	assert.Equal(t, float64(120), handler["timeout"])
	assert.Equal(t, true, handler["async"])
	assert.Equal(t, true, handler["once"])
	assert.Equal(t, "bash", handler["shell"])
	assert.Equal(t, "Running setup...", handler["statusMessage"])
	assert.Equal(t, "Bash(./scripts/*)", handler["if"])
}

// ---------------------------------------------------------------------------
// Feature: settings.json — new fields
// ---------------------------------------------------------------------------

func TestCompile_Settings_AllNewFields_EmitCorrectly(t *testing.T) {
	trueVal := true
	falseVal := false
	cleanupDays := 30
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Model:                      "claude-sonnet-4-6",
			OutputStyle:                "concise",
			Language:                   "en",
			IncludeGitInstructions:     &trueVal,
			DisableSkillShellExecution: &falseVal,
			DefaultShell:               "zsh",
			CleanupPeriodDays:          &cleanupDays,
			AvailableModels:            []string{"claude-sonnet-4-6", "claude-haiku-4-5-20251001"},
			RespectGitignore:           &trueVal,
			PlansDirectory:             "docs/plans",
			ClaudeMdExcludes:           []string{"vendor/**"},
			AutoMemoryEnabled:          &trueVal,
			AutoMemoryDirectory:        ".claude/memory",
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	assert.Equal(t, "claude-sonnet-4-6", parsed["model"])
	assert.Equal(t, "concise", parsed["outputStyle"])
	assert.Equal(t, "en", parsed["language"])
	assert.Equal(t, true, parsed["includeGitInstructions"])
	assert.Equal(t, false, parsed["disableSkillShellExecution"])
	assert.Equal(t, "zsh", parsed["defaultShell"])
	assert.Equal(t, float64(30), parsed["cleanupPeriodDays"])
	assert.Len(t, parsed["availableModels"], 2)
	assert.Equal(t, true, parsed["respectGitignore"])
	assert.Equal(t, "docs/plans", parsed["plansDirectory"])
	assert.Len(t, parsed["claudeMdExcludes"], 1)
	assert.Equal(t, true, parsed["autoMemoryEnabled"])
	assert.Equal(t, ".claude/memory", parsed["autoMemoryDirectory"])
}

// ---------------------------------------------------------------------------
// Local Settings: local: block compiles to settings.local.json
// ---------------------------------------------------------------------------

// TestCompile_LocalSettings_EmitsSettingsLocalJSON verifies that a local: block
// ---------------------------------------------------------------------------
// Feature: Permission regression tests
// ---------------------------------------------------------------------------

func TestCompile_Permissions_DenyPropagatestoSettings(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Permissions: &ast.PermissionsConfig{
				Deny: []string{"Write"},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	permsAny, has := parsed["permissions"]
	require.True(t, has, "permissions key must be present")
	permsMap, ok := permsAny.(map[string]any)
	require.True(t, ok)

	denyAny, ok := permsMap["deny"].([]any)
	require.True(t, ok, "deny must be an array")
	require.Len(t, denyAny, 1)
	assert.Equal(t, "Write", denyAny[0])
}

func TestCompile_Permissions_SandboxPropagatestoSettings(t *testing.T) {
	enabled := true
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Sandbox: &ast.SandboxConfig{
				Enabled: &enabled,
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.json"]
	require.True(t, ok, "settings.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	sandboxAny, has := parsed["sandbox"]
	require.True(t, has, "sandbox key must be present")
	sandboxMap, ok := sandboxAny.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, sandboxMap["enabled"])
}

func TestCompile_Permissions_CursorDropsPermissions(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Permissions: &ast.PermissionsConfig{
				Allow: []string{"Read"},
				Deny:  []string{"Bash"},
			},
		},
	}

	out, err := Compile(config, t.TempDir(), "cursor")
	require.NoError(t, err)

	// Cursor emits no settings.json — permissions must not appear in any output file
	for path, content := range out.Files {
		assert.NotContains(t, content, `"permissions"`, "cursor output file %q must not contain permissions key", path)
	}
	// Confirm no settings.json at all
	_, hasSettings := out.Files["settings.json"]
	assert.False(t, hasSettings, "cursor target must not emit settings.json")
}

func TestCompile_Permissions_CursorDropsSandbox(t *testing.T) {
	enabled := true
	config := &ast.XcaffoldConfig{
		Settings: ast.SettingsConfig{
			Sandbox: &ast.SandboxConfig{
				Enabled: &enabled,
			},
		},
	}

	out, err := Compile(config, t.TempDir(), "cursor")
	require.NoError(t, err)

	_, hasSettings := out.Files["settings.json"]
	assert.False(t, hasSettings, "cursor target must not emit settings.json")
	for path, content := range out.Files {
		assert.NotContains(t, content, `"sandbox"`, "cursor output file %q must not contain sandbox key", path)
	}
}

func TestCompile_Permissions_DisallowedToolsInAgentFrontmatter(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Description:     "Developer agent",
					Instructions:    "Build things.",
					DisallowedTools: []string{"Bash", "Write"},
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	content, ok := out.Files["agents/dev.md"]
	require.True(t, ok, "agents/dev.md must be generated")
	assert.Contains(t, content, "disallowed-tools:", "disallowed-tools must appear in Claude agent frontmatter")
}

func TestCompile_Permissions_DisallowedToolsNotInCursorOutput(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Description:     "Developer agent",
					Instructions:    "Build things.",
					DisallowedTools: []string{"Bash", "Write"},
				},
			},
		},
	}

	out, err := Compile(config, t.TempDir(), "cursor")
	require.NoError(t, err)

	content, ok := out.Files["agents/dev.md"]
	require.True(t, ok, "agents/dev.md must be generated for cursor target")
	assert.NotContains(t, content, "disallowed-tools:", "disallowed-tools must not appear in Cursor agent output")
}

// in the XcaffoldConfig compiles to settings.local.json.
func TestCompile_LocalSettings_EmitsSettingsLocalJSON(t *testing.T) {
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Local: ast.SettingsConfig{
				Model: "claude-opus-4-6",
				Env: map[string]string{
					"ANTHROPIC_API_KEY": "sk-test-key",
				},
			},
		},
	}

	out, err := Compile(config, "", "")
	require.NoError(t, err)

	raw, ok := out.Files["settings.local.json"]
	require.True(t, ok, "settings.local.json must be generated")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	assert.Equal(t, "claude-opus-4-6", parsed["model"])
	env, ok := parsed["env"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "sk-test-key", env["ANTHROPIC_API_KEY"])
}
