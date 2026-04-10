package parser

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

// ---------------------------------------------------------------------------
// mergeSettingsStrict
// ---------------------------------------------------------------------------

func TestMergeSettingsStrict_EmptyBaseReturnsChild(t *testing.T) {
	child := ast.SettingsConfig{Model: "opus"}
	got, err := mergeSettingsStrict(ast.SettingsConfig{}, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, "opus", got.Model)
}

func TestMergeSettingsStrict_EmptyChildReturnsBase(t *testing.T) {
	base := ast.SettingsConfig{Model: "opus"}
	got, err := mergeSettingsStrict(base, ast.SettingsConfig{}, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, "opus", got.Model)
}

func TestMergeSettingsStrict_ScalarConflictModel(t *testing.T) {
	base := ast.SettingsConfig{Model: "opus"}
	child := ast.SettingsConfig{Model: "sonnet"}
	_, err := mergeSettingsStrict(base, child, "/dir/a.xcf", "/dir/b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model")
	assert.Contains(t, err.Error(), "a.xcf")
	assert.Contains(t, err.Error(), "b.xcf")
}

func TestMergeSettingsStrict_ScalarConflictEffortLevel(t *testing.T) {
	base := ast.SettingsConfig{EffortLevel: "low"}
	child := ast.SettingsConfig{EffortLevel: "high"}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "effortLevel")
}

func TestMergeSettingsStrict_NonConflictingScalarsMerge(t *testing.T) {
	base := ast.SettingsConfig{Model: "opus"}
	child := ast.SettingsConfig{Language: "en"}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, "opus", got.Model)
	assert.Equal(t, "en", got.Language)
}

func TestMergeSettingsStrict_SameScalarValueNoConflict(t *testing.T) {
	base := ast.SettingsConfig{Model: "opus"}
	child := ast.SettingsConfig{Model: "opus"}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, "opus", got.Model)
}

func TestMergeSettingsStrict_BoolPointerConflict(t *testing.T) {
	base := ast.SettingsConfig{Attribution: boolPtr(true)}
	child := ast.SettingsConfig{Attribution: boolPtr(false)}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "attribution")
}

func TestMergeSettingsStrict_BoolPointerNonConflicting(t *testing.T) {
	base := ast.SettingsConfig{Attribution: boolPtr(true)}
	child := ast.SettingsConfig{RespectGitignore: boolPtr(false)}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, true, *got.Attribution)
	assert.Equal(t, false, *got.RespectGitignore)
}

func TestMergeSettingsStrict_BoolPointerSameValueNoConflict(t *testing.T) {
	base := ast.SettingsConfig{Attribution: boolPtr(true)}
	child := ast.SettingsConfig{Attribution: boolPtr(true)}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, true, *got.Attribution)
}

func TestMergeSettingsStrict_IntPointerConflict(t *testing.T) {
	base := ast.SettingsConfig{CleanupPeriodDays: intPtr(7)}
	child := ast.SettingsConfig{CleanupPeriodDays: intPtr(30)}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleanupPeriodDays")
}

func TestMergeSettingsStrict_EnvAdditive(t *testing.T) {
	base := ast.SettingsConfig{Env: map[string]string{"A": "1"}}
	child := ast.SettingsConfig{Env: map[string]string{"B": "2"}}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"A": "1", "B": "2"}, got.Env)
}

func TestMergeSettingsStrict_EnvDuplicateKeyError(t *testing.T) {
	base := ast.SettingsConfig{Env: map[string]string{"A": "1"}}
	child := ast.SettingsConfig{Env: map[string]string{"A": "2"}}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "env")
	assert.Contains(t, err.Error(), "A")
}

func TestMergeSettingsStrict_EnvSameKeyValueNoError(t *testing.T) {
	base := ast.SettingsConfig{Env: map[string]string{"A": "1"}}
	child := ast.SettingsConfig{Env: map[string]string{"A": "1"}}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, "1", got.Env["A"])
}

func TestMergeSettingsStrict_EnabledPluginsAdditive(t *testing.T) {
	base := ast.SettingsConfig{EnabledPlugins: map[string]bool{"x": true}}
	child := ast.SettingsConfig{EnabledPlugins: map[string]bool{"y": false}}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"x": true, "y": false}, got.EnabledPlugins)
}

func TestMergeSettingsStrict_EnabledPluginsDuplicateKeyError(t *testing.T) {
	base := ast.SettingsConfig{EnabledPlugins: map[string]bool{"x": true}}
	child := ast.SettingsConfig{EnabledPlugins: map[string]bool{"x": false}}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enabledPlugins")
}

func TestMergeSettingsStrict_MCPServersAdditive(t *testing.T) {
	base := ast.SettingsConfig{MCPServers: map[string]ast.MCPConfig{
		"srv1": {Command: "cmd1"},
	}}
	child := ast.SettingsConfig{MCPServers: map[string]ast.MCPConfig{
		"srv2": {Command: "cmd2"},
	}}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Len(t, got.MCPServers, 2)
	assert.Equal(t, "cmd1", got.MCPServers["srv1"].Command)
	assert.Equal(t, "cmd2", got.MCPServers["srv2"].Command)
}

func TestMergeSettingsStrict_MCPServersDuplicateKeyError(t *testing.T) {
	base := ast.SettingsConfig{MCPServers: map[string]ast.MCPConfig{
		"srv1": {Command: "cmd1"},
	}}
	child := ast.SettingsConfig{MCPServers: map[string]ast.MCPConfig{
		"srv1": {Command: "cmd2"},
	}}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mcpServers")
	assert.Contains(t, err.Error(), "srv1")
}

func TestMergeSettingsStrict_HooksAdditive(t *testing.T) {
	base := ast.SettingsConfig{Hooks: ast.HookConfig{
		"PreCommit": {{Matcher: "*.go", Hooks: []ast.HookHandler{{Command: "lint"}}}},
	}}
	child := ast.SettingsConfig{Hooks: ast.HookConfig{
		"PreCommit": {{Matcher: "*.ts", Hooks: []ast.HookHandler{{Command: "eslint"}}}},
	}}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Len(t, got.Hooks["PreCommit"], 2)
}

func TestMergeSettingsStrict_AnyFieldConflictAgent(t *testing.T) {
	base := ast.SettingsConfig{Agent: "agent-a"}
	child := ast.SettingsConfig{Agent: "agent-b"}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent")
}

func TestMergeSettingsStrict_AnyFieldConflictWorktree(t *testing.T) {
	base := ast.SettingsConfig{Worktree: true}
	child := ast.SettingsConfig{Worktree: false}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worktree")
}

func TestMergeSettingsStrict_AnyFieldConflictAutoMode(t *testing.T) {
	base := ast.SettingsConfig{AutoMode: "full"}
	child := ast.SettingsConfig{AutoMode: "limited"}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "autoMode")
}

func TestMergeSettingsStrict_StructPointerConflictPermissions(t *testing.T) {
	base := ast.SettingsConfig{Permissions: &ast.PermissionsConfig{Allow: []string{"a"}}}
	child := ast.SettingsConfig{Permissions: &ast.PermissionsConfig{Allow: []string{"b"}}}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permissions")
}

func TestMergeSettingsStrict_StructPointerConflictSandbox(t *testing.T) {
	base := ast.SettingsConfig{Sandbox: &ast.SandboxConfig{Enabled: boolPtr(true)}}
	child := ast.SettingsConfig{Sandbox: &ast.SandboxConfig{Enabled: boolPtr(false)}}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox")
}

func TestMergeSettingsStrict_StructPointerConflictStatusLine(t *testing.T) {
	base := ast.SettingsConfig{StatusLine: &ast.StatusLineConfig{Type: "command"}}
	child := ast.SettingsConfig{StatusLine: &ast.StatusLineConfig{Type: "static"}}
	_, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "statusLine")
}

func TestMergeSettingsStrict_SliceAppendUnique(t *testing.T) {
	base := ast.SettingsConfig{AvailableModels: []string{"opus", "sonnet"}}
	child := ast.SettingsConfig{AvailableModels: []string{"sonnet", "haiku"}}
	got, err := mergeSettingsStrict(base, child, "a.xcf", "b.xcf")
	require.NoError(t, err)
	assert.Equal(t, []string{"opus", "sonnet", "haiku"}, got.AvailableModels)
}

// ---------------------------------------------------------------------------
// mergeSettingsOverride
// ---------------------------------------------------------------------------

func TestMergeSettingsOverride_ChildWinsScalarConflict(t *testing.T) {
	base := ast.SettingsConfig{Model: "opus"}
	child := ast.SettingsConfig{Model: "sonnet"}
	got := mergeSettingsOverride(base, child)
	assert.Equal(t, "sonnet", got.Model)
}

func TestMergeSettingsOverride_EmptyChildPreservesBase(t *testing.T) {
	base := ast.SettingsConfig{Model: "opus", Language: "en"}
	got := mergeSettingsOverride(base, ast.SettingsConfig{})
	assert.Equal(t, "opus", got.Model)
	assert.Equal(t, "en", got.Language)
}

func TestMergeSettingsOverride_ChildEnvKeysOverrideBase(t *testing.T) {
	base := ast.SettingsConfig{Env: map[string]string{"A": "1", "B": "2"}}
	child := ast.SettingsConfig{Env: map[string]string{"A": "99", "C": "3"}}
	got := mergeSettingsOverride(base, child)
	assert.Equal(t, "99", got.Env["A"])
	assert.Equal(t, "2", got.Env["B"])
	assert.Equal(t, "3", got.Env["C"])
}

func TestMergeSettingsOverride_ChildBoolPointersOverrideBase(t *testing.T) {
	base := ast.SettingsConfig{Attribution: boolPtr(true)}
	child := ast.SettingsConfig{Attribution: boolPtr(false)}
	got := mergeSettingsOverride(base, child)
	assert.Equal(t, false, *got.Attribution)
}

func TestMergeSettingsOverride_ChildStructPointerWins(t *testing.T) {
	base := ast.SettingsConfig{Permissions: &ast.PermissionsConfig{Allow: []string{"a"}}}
	child := ast.SettingsConfig{Permissions: &ast.PermissionsConfig{Allow: []string{"b"}}}
	got := mergeSettingsOverride(base, child)
	assert.Equal(t, []string{"b"}, got.Permissions.Allow)
}

func TestMergeSettingsOverride_SlicesAdditive(t *testing.T) {
	base := ast.SettingsConfig{AvailableModels: []string{"opus"}}
	child := ast.SettingsConfig{AvailableModels: []string{"opus", "haiku"}}
	got := mergeSettingsOverride(base, child)
	assert.Equal(t, []string{"opus", "haiku"}, got.AvailableModels)
}

func TestMergeSettingsOverride_HooksAdditive(t *testing.T) {
	base := ast.SettingsConfig{Hooks: ast.HookConfig{
		"PreCommit": {{Matcher: "*.go", Hooks: []ast.HookHandler{{Command: "lint"}}}},
	}}
	child := ast.SettingsConfig{Hooks: ast.HookConfig{
		"PostCommit": {{Matcher: "*", Hooks: []ast.HookHandler{{Command: "notify"}}}},
	}}
	got := mergeSettingsOverride(base, child)
	assert.Len(t, got.Hooks["PreCommit"], 1)
	assert.Len(t, got.Hooks["PostCommit"], 1)
}

func TestMergeSettingsOverride_MCPServersChildOverrides(t *testing.T) {
	base := ast.SettingsConfig{MCPServers: map[string]ast.MCPConfig{
		"srv1": {Command: "cmd1"},
	}}
	child := ast.SettingsConfig{MCPServers: map[string]ast.MCPConfig{
		"srv1": {Command: "cmd-override"},
		"srv2": {Command: "cmd2"},
	}}
	got := mergeSettingsOverride(base, child)
	assert.Equal(t, "cmd-override", got.MCPServers["srv1"].Command)
	assert.Equal(t, "cmd2", got.MCPServers["srv2"].Command)
}

// ---------------------------------------------------------------------------
// isEmptySettings
// ---------------------------------------------------------------------------

func TestIsEmptySettings_Zero(t *testing.T) {
	assert.True(t, isEmptySettings(ast.SettingsConfig{}))
}

func TestIsEmptySettings_NonZero(t *testing.T) {
	assert.False(t, isEmptySettings(ast.SettingsConfig{Model: "x"}))
}
