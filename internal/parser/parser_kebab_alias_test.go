package parser

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRewriteLegacyKeys_AgentCamelCase verifies that pre-migration camelCase
// agent fields at the top level are rewritten to kebab-case.
func TestRewriteLegacyKeys_AgentCamelCase(t *testing.T) {
	in := []byte(`kind: global
version: "1.0"
agents:
  dev:
    maxTurns: 5
    disallowedTools: [Bash]
    permissionMode: restricted
    disableModelInvocation: true
    userInvocable: false
    initialPrompt: "hi"
`)
	out := rewriteLegacyKeys(in)
	s := string(out)
	assert.Contains(t, s, "max-turns:")
	assert.Contains(t, s, "disallowed-tools:")
	assert.Contains(t, s, "permission-mode:")
	assert.Contains(t, s, "disable-model-invocation:")
	assert.Contains(t, s, "user-invocable:")
	assert.Contains(t, s, "initial-prompt:")
	assert.NotContains(t, s, "maxTurns:")
	assert.NotContains(t, s, "disallowedTools:")
}

// TestRewriteLegacyKeys_IndentedKeysRewritten verifies that legacy keys are
// rewritten regardless of indentation level. instructions_file typically appears
// under an agent/skill map entry.
func TestRewriteLegacyKeys_IndentedKeysRewritten(t *testing.T) {
	in := []byte(`kind: global
version: "1.0"
agents:
  backend:
    description: "backend"
    instructions_file: "backend.md"
  frontend:
    description: "frontend"
    instructions_file: "frontend.md"
`)
	out := rewriteLegacyKeys(in)
	s := string(out)
	assert.Equal(t, 2, strings.Count(s, "instructions-file:"))
	assert.NotContains(t, s, "instructions_file:")
}

// TestRewriteLegacyKeys_HookHandlerCamelCase verifies the new HookHandler
// alias entries (statusMessage, allowedEnvVars) are rewritten to kebab-case.
func TestRewriteLegacyKeys_HookHandlerCamelCase(t *testing.T) {
	in := []byte(`kind: global
version: "1.0"
hooks:
  PreToolUse:
    - matcher: Bash
      hooks:
        - type: command
          command: echo hi
          statusMessage: "running"
          allowedEnvVars: [HOME, PATH]
`)
	out := rewriteLegacyKeys(in)
	s := string(out)
	assert.Contains(t, s, "status-message:")
	assert.Contains(t, s, "allowed-env-vars:")
	assert.NotContains(t, s, "statusMessage:")
	assert.NotContains(t, s, "allowedEnvVars:")
}

// TestRewriteLegacyKeys_SkipsStandaloneSettingsDocument verifies that a
// standalone kind: settings document is exempt — its provider wire-format keys
// (mcpServers, permissions) must pass through untouched.
func TestRewriteLegacyKeys_SkipsStandaloneSettingsDocument(t *testing.T) {
	in := []byte(`kind: settings
version: "1.0"
mcpServers:
  foo:
    command: npx
permissions:
  allow: [Bash]
`)
	out := rewriteLegacyKeys(in)
	s := string(out)
	assert.Contains(t, s, "mcpServers:", "provider wire-format key must survive inside kind: settings")
	assert.NotContains(t, s, "mcp-servers:")
}

// TestRewriteLegacyKeys_SettingsInsideGlobal verifies that the legacy camelCase
// mcpServers key inside the settings sub-block of a kind: global document is
// rewritten to mcp-servers (kebab-case). kind: global is not exempt from
// rewriting — only standalone kind: settings documents are exempt.
func TestRewriteLegacyKeys_SettingsInsideGlobal_MCPServersUntouched(t *testing.T) {
	in := []byte(`kind: global
version: "1.0"
settings:
  mcpServers:
    foo:
      command: npx
agents:
  dev:
    description: "dev"
`)
	out := rewriteLegacyKeys(in)
	s := string(out)
	// mcpServers inside a kind: global doc is rewritten to mcp-servers so the
	// SettingsConfig struct (yaml:"mcp-servers") can parse it correctly.
	assert.Contains(t, s, "mcp-servers:", "settings.mcpServers inside kind: global must be rewritten to mcp-servers")
	assert.NotContains(t, s, "mcpServers:")
}

// TestRewriteLegacyKeys_MultiDocumentMixedKinds verifies a multi-document file
// with both kind: global (rewritten) and kind: settings (exempt) behaves
// correctly per-document.
func TestRewriteLegacyKeys_MultiDocumentMixedKinds(t *testing.T) {
	in := []byte(`---
kind: settings
version: "1.0"
mcpServers:
  wire: {command: npx}
---
kind: global
version: "1.0"
agents:
  dev:
    maxTurns: 3
`)
	out := rewriteLegacyKeys(in)
	s := string(out)
	assert.Contains(t, s, "mcpServers:", "settings doc wire-format key must survive")
	assert.Contains(t, s, "max-turns:", "global doc legacy key must be rewritten")
	assert.NotContains(t, s, "maxTurns:")
}

// TestRewriteLegacyKeys_StringValueNotRewritten verifies that the rewriter
// matches only key positions (start-of-trimmed-line), not substrings inside
// YAML scalar values.
func TestRewriteLegacyKeys_StringValueNotRewritten(t *testing.T) {
	in := []byte(`kind: global
version: "1.0"
agents:
  dev:
    description: "documented here: instructions_file is a field"
    instructions: "see alwaysApply for details"
`)
	out := rewriteLegacyKeys(in)
	s := string(out)
	// String values must remain verbatim.
	assert.Contains(t, s, `"documented here: instructions_file is a field"`)
	assert.Contains(t, s, `"see alwaysApply for details"`)
}

// TestRewriteLegacyKeys_CommentLinesNotRewritten verifies that comment lines
// beginning with # are not touched.
func TestRewriteLegacyKeys_CommentLinesNotRewritten(t *testing.T) {
	in := []byte(`kind: global
version: "1.0"
# Historical note: the field was once written as instructions_file and
# has since been renamed to instructions-file.
agents:
  dev:
    description: "dev"
`)
	out := rewriteLegacyKeys(in)
	s := string(out)
	assert.Contains(t, s, "# Historical note: the field was once written as instructions_file")
	assert.Contains(t, s, "# has since been renamed to instructions-file.")
}

// TestRewriteLegacyKeys_PolicySnakeCase verifies policy match/require/deny
// snake_case fields are rewritten.
func TestRewriteLegacyKeys_PolicySnakeCase(t *testing.T) {
	in := []byte(`kind: policy
version: "1.0"
name: p1
severity: error
target: output
match:
  has_tool: Bash
  name_matches: "deploy*"
  target_includes: claude
require:
  - field: description
    is_present: true
    min_length: 10
deny:
  - content_contains: ["secret"]
  - path_contains: ".."
`)
	out := rewriteLegacyKeys(in)
	s := string(out)
	for _, want := range []string{
		"has-tool:", "name-matches:", "target-includes:",
		"is-present:", "min-length:",
		"content-contains:", "path-contains:",
	} {
		assert.Contains(t, s, want, "expected kebab form %q", want)
	}
	for _, notWant := range []string{
		"has_tool:", "name_matches:", "target_includes:",
		"is_present:", "min_length:",
		"content_contains:", "path_contains:",
	} {
		assert.NotContains(t, s, notWant, "legacy key %q must be rewritten", notWant)
	}
}

// TestIsSettingsDocument_TopLevelOnly verifies the exemption check matches
// only top-level "kind: settings" declarations. An indented "kind: settings"
// value inside a nested map must NOT cause false-positive exemption.
func TestIsSettingsDocument_TopLevelOnly(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want bool
	}{
		{
			name: "top-level kind: settings",
			doc:  "kind: settings\nversion: \"1.0\"\n",
			want: true,
		},
		{
			name: "top-level kind: global",
			doc:  "kind: global\nversion: \"1.0\"\n",
			want: false,
		},
		{
			name: "indented kind: settings inside a map must not match",
			doc:  "kind: global\nversion: \"1.0\"\nmetadata:\n  kind: settings\n",
			want: false,
		},
		{
			name: "kind declaration preceded by comments",
			doc:  "# header\n# note\nkind: settings\n",
			want: true,
		},
		{
			name: "kind declaration preceded by blank lines",
			doc:  "\n\nkind: settings\n",
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isSettingsDocument([]byte(tc.doc))
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestRewriteLegacyKeys_IdempotentOnKebabInput verifies that already-canonical
// kebab-case input passes through unchanged.
func TestRewriteLegacyKeys_IdempotentOnKebabInput(t *testing.T) {
	in := []byte(`kind: global
version: "1.0"
agents:
  dev:
    max-turns: 3
    disallowed-tools: [Bash]

`)
	out := rewriteLegacyKeys(in)
	require.True(t, bytes.Equal(in, out), "kebab-case input must be unchanged by the rewriter")
}
