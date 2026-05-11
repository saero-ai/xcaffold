package policy

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeOutput(files map[string]string) *output.Output {
	return &output.Output{Files: files}
}

func makeConfigWithAgentHook(hookURL string) *ast.XcaffoldConfig {
	handler := ast.HookHandler{URL: hookURL}
	group := ast.HookMatcherGroup{Hooks: []ast.HookHandler{handler}}
	hookCfg := ast.HookConfig{"PostToolUse": []ast.HookMatcherGroup{group}}
	return &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"test-agent": {Hooks: hookCfg},
			},
		},
	}
}

func emptyConfig() *ast.XcaffoldConfig {
	return &ast.XcaffoldConfig{}
}

func TestRunInvariants_CleanOutput_NoErrors(t *testing.T) {
	compiled := makeOutput(map[string]string{
		".claude/agents/dev.md": "agent content",
		".claude/settings.json": `{"permissions": {"allow": []}}`,
	})
	errs := RunInvariants(emptyConfig(), compiled)
	assert.Empty(t, errs)
}

func TestRunInvariants_PathTraversal(t *testing.T) {
	compiled := makeOutput(map[string]string{
		"agents/../../../etc/passwd": "malicious",
	})
	errs := RunInvariants(emptyConfig(), compiled)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "directory traversal")
}

func TestRunInvariants_AbsolutePath(t *testing.T) {
	compiled := makeOutput(map[string]string{
		"/etc/passwd": "malicious",
	})
	errs := RunInvariants(emptyConfig(), compiled)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "absolute")
}

func TestRunInvariants_NullPermissions(t *testing.T) {
	compiled := makeOutput(map[string]string{
		"settings.json": `{"permissions": null}`,
	})
	errs := RunInvariants(emptyConfig(), compiled)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "null permissions")
}

func TestRunInvariants_ValidSettings(t *testing.T) {
	compiled := makeOutput(map[string]string{
		"settings.json": `{"permissions": {"allow": ["Bash"]}}`,
	})
	errs := RunInvariants(emptyConfig(), compiled)
	assert.Empty(t, errs)
}

func TestRunInvariants_HookHTTPURL(t *testing.T) {
	cfg := makeConfigWithAgentHook("http://evil.com")
	errs := RunInvariants(cfg, makeOutput(nil))
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "HTTPS")
}

func TestRunInvariants_HookHTTPSURL(t *testing.T) {
	cfg := makeConfigWithAgentHook("https://good.com")
	errs := RunInvariants(cfg, makeOutput(nil))
	assert.Empty(t, errs)
}

func TestRunInvariants_HookNoURL(t *testing.T) {
	handler := ast.HookHandler{Command: "echo hello"}
	group := ast.HookMatcherGroup{Hooks: []ast.HookHandler{handler}}
	hookCfg := ast.HookConfig{"PostToolUse": []ast.HookMatcherGroup{group}}
	cfg := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"test-agent": {Hooks: hookCfg},
			},
		},
	}
	errs := RunInvariants(cfg, makeOutput(nil))
	assert.Empty(t, errs)
}

func TestRunInvariants_MultipleViolations(t *testing.T) {
	compiled := makeOutput(map[string]string{
		"agents/../../../etc/passwd": "malicious",
	})
	cfg := makeConfigWithAgentHook("http://evil.com")
	errs := RunInvariants(cfg, compiled)
	require.Len(t, errs, 2)

	errTexts := make([]string, len(errs))
	for i, e := range errs {
		errTexts[i] = e.Error()
	}
	assert.Contains(t, errTexts[0], "directory traversal")
	assert.Contains(t, errTexts[1], "HTTPS")
}
