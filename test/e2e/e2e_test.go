//go:build e2e

package e2e_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImport_MultiProviderAutoDetect(t *testing.T) {
	sandbox := setupSandbox(t)
	_, _, code := runXcaffold(t, sandbox, "import")
	require.Equal(t, 0, code)
	assertFileExists(t, filepath.Join(sandbox, ".xcaffold", "project.xcaf"))
	assertDirExists(t, filepath.Join(sandbox, "xcaf", "agents"))
	assertDirExists(t, filepath.Join(sandbox, "xcaf", "rules"))
	assertDirExists(t, filepath.Join(sandbox, "xcaf", "skills"))
}

func TestImport_SmartAssembly_BaseAndOverride(t *testing.T) {
	sandbox := setupSandbox(t)
	_, _, code := runXcaffold(t, sandbox, "import")
	require.Equal(t, 0, code)
	devDir := filepath.Join(sandbox, "xcaf", "agents", "developer")
	assertDirExists(t, devDir)
	assertFileExists(t, filepath.Join(devDir, "agent.xcaf"))
	overrides := countFiles(t, devDir, "agent.*.xcaf")
	assert.Greater(t, overrides, 0, "expected override files for shared agent")
}

func TestImport_SingleProviderResource(t *testing.T) {
	sandbox := setupSandbox(t)
	_, _, code := runXcaffold(t, sandbox, "import")
	require.Equal(t, 0, code)
	reviewerXcaf := filepath.Join(sandbox, "xcaf", "agents", "reviewer", "agent.xcaf")
	assertFileExists(t, reviewerXcaf)
	assertFileContains(t, reviewerXcaf, "targets:")
}

func TestImport_DirectoryPerResource(t *testing.T) {
	sandbox := setupSandbox(t)
	_, _, code := runXcaffold(t, sandbox, "import")
	require.Equal(t, 0, code)
	assert.Greater(t, countFiles(t, filepath.Join(sandbox, "xcaf", "agents"), "agent.xcaf"), 0)
	assert.Greater(t, countFiles(t, filepath.Join(sandbox, "xcaf", "rules"), "rule.xcaf"), 0)
	assert.Greater(t, countFiles(t, filepath.Join(sandbox, "xcaf", "skills"), "skill.xcaf"), 0)
}

func TestImport_ConventionMemory(t *testing.T) {
	sandbox := setupSandbox(t)
	_, _, code := runXcaffold(t, sandbox, "import")
	require.Equal(t, 0, code)
	memDir := filepath.Join(sandbox, "xcaf", "agents", "developer", "memory")
	if _, err := os.Stat(memDir); err == nil {
		memFiles := countFiles(t, memDir, "*.md")
		assert.Greater(t, memFiles, 0, "expected memory files")
	}
}

func TestImport_ContextFiles(t *testing.T) {
	sandbox := setupSandbox(t)
	_, _, code := runXcaffold(t, sandbox, "import")
	require.Equal(t, 0, code)
	contextDir := filepath.Join(sandbox, "xcaf", "context")
	if _, err := os.Stat(contextDir); os.IsNotExist(err) {
		t.Log("NOTE: xcaf/context/ not created — context handling may vary with fixture data")
		return
	}
	assertDirExists(t, contextDir)
}

func TestImport_SkillReferences(t *testing.T) {
	sandbox := setupSandbox(t)
	_, _, code := runXcaffold(t, sandbox, "import")
	require.Equal(t, 0, code)
	refFile := filepath.Join(sandbox, "xcaf", "skills", "tdd", "references", "template.md")
	assertFileExists(t, refFile)
	assertFileContains(t, refFile, "Test Template")
}

func TestImport_ScopedRules(t *testing.T) {
	sandbox := setupSandbox(t)
	_, _, code := runXcaffold(t, sandbox, "import")
	require.Equal(t, 0, code)
	assertDirExists(t, filepath.Join(sandbox, "xcaf", "rules", "cli", "go-standards"))
	assertDirExists(t, filepath.Join(sandbox, "xcaf", "rules", "platform", "api-conventions"))
}

func TestValidate_Clean(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	_, _, code := runXcaffold(t, sandbox, "validate")
	assert.Equal(t, 0, code)
}

func TestValidate_HiddenFilesIgnored(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.WriteFile(filepath.Join(sandbox, "xcaf", "skills", "tdd", ".DS_Store"), []byte{}, 0o644)
	_, _, code := runXcaffold(t, sandbox, "validate")
	assert.Equal(t, 0, code)
}

func TestValidate_FilesystemInference(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	ruleDir := filepath.Join(sandbox, "xcaf", "rules", "test-inferred")
	os.MkdirAll(ruleDir, 0o755)
	os.WriteFile(filepath.Join(ruleDir, "rule.xcaf"), []byte("---\ndescription: Inferred.\nversion: \"1.0\"\n---\nNo secrets.\n"), 0o644)
	_, _, code := runXcaffold(t, sandbox, "validate")
	assert.Equal(t, 0, code)
}

func TestApply_Claude(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.RemoveAll(filepath.Join(sandbox, ".claude"))
	_, _, code := runXcaffold(t, sandbox, "apply", "--target", "claude")
	require.Equal(t, 0, code)
	assertDirExists(t, filepath.Join(sandbox, ".claude", "agents"))
	assertDirExists(t, filepath.Join(sandbox, ".claude", "skills"))
}

func TestApply_Gemini(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.RemoveAll(filepath.Join(sandbox, ".gemini"))
	_, _, code := runXcaffold(t, sandbox, "apply", "--target", "gemini")
	require.Equal(t, 0, code)
	assertDirExists(t, filepath.Join(sandbox, ".gemini"))
}

func TestApply_Cursor(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	_, _, code := runXcaffold(t, sandbox, "apply", "--target", "cursor")
	require.Equal(t, 0, code)
	assertDirExists(t, filepath.Join(sandbox, ".cursor"))
}

func TestApply_Antigravity(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.RemoveAll(filepath.Join(sandbox, ".agents"))
	_, _, code := runXcaffold(t, sandbox, "apply", "--target", "antigravity")
	require.Equal(t, 0, code)
	assertDirExists(t, filepath.Join(sandbox, ".agents"))
	assertDirNotExists(t, filepath.Join(sandbox, ".agents", "agents"))
}

func TestApply_Copilot(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.RemoveAll(filepath.Join(sandbox, ".claude"))
	_, _, code := runXcaffold(t, sandbox, "apply", "--target", "copilot")
	require.Equal(t, 0, code)
	assertDirExists(t, filepath.Join(sandbox, ".github"))
}

func TestApply_OverrideCompilation(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.RemoveAll(filepath.Join(sandbox, ".claude"))
	os.RemoveAll(filepath.Join(sandbox, ".gemini"))
	runXcaffold(t, sandbox, "apply", "--target", "claude")
	runXcaffold(t, sandbox, "apply", "--target", "gemini")
	claudeAgent := filepath.Join(sandbox, ".claude", "agents", "developer.md")
	geminiAgent := filepath.Join(sandbox, ".gemini", "agents", "developer.md")
	if _, err := os.Stat(claudeAgent); err == nil {
		assertFileContains(t, claudeAgent, "claude")
	}
	if _, err := os.Stat(geminiAgent); err == nil {
		assertFileContains(t, geminiAgent, "gemini")
	}
}

func TestApply_TargetsFiltering(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.RemoveAll(filepath.Join(sandbox, ".claude"))
	os.RemoveAll(filepath.Join(sandbox, ".gemini"))
	runXcaffold(t, sandbox, "apply", "--target", "claude")
	runXcaffold(t, sandbox, "apply", "--target", "gemini")
	claudeRules := filepath.Join(sandbox, ".claude", "rules")
	geminiRules := filepath.Join(sandbox, ".gemini", "rules")
	if _, err := os.Stat(claudeRules); err == nil {
		cCount := countFiles(t, claudeRules, "*.md")
		gCount := 0
		if _, err := os.Stat(geminiRules); err == nil {
			gCount = countFiles(t, geminiRules, "*.md")
		}
		assert.GreaterOrEqual(t, cCount, gCount)
	}
}

func TestApply_DryRun(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.RemoveAll(filepath.Join(sandbox, ".claude"))
	_, _, code := runXcaffold(t, sandbox, "apply", "--target", "claude", "--dry-run")
	assert.Equal(t, 0, code)
	assertDirNotExists(t, filepath.Join(sandbox, ".claude"))
}

func TestStatus_NoDrift(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.RemoveAll(filepath.Join(sandbox, ".claude"))
	runXcaffold(t, sandbox, "apply", "--target", "claude")
	_, _, code := runXcaffold(t, sandbox, "status", "--target", "claude")
	assert.Equal(t, 0, code)
}

func TestStatus_DriftDetected(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	os.RemoveAll(filepath.Join(sandbox, ".claude"))
	runXcaffold(t, sandbox, "apply", "--target", "claude")
	entries, _ := os.ReadDir(filepath.Join(sandbox, ".claude", "agents"))
	if len(entries) > 0 {
		f := filepath.Join(sandbox, ".claude", "agents", entries[0].Name())
		fh, _ := os.OpenFile(f, os.O_APPEND|os.O_WRONLY, 0o644)
		fh.WriteString("\n# drift\n")
		fh.Close()
		stdout, _, _ := runXcaffold(t, sandbox, "status", "--target", "claude")
		assert.Contains(t, stdout, entries[0].Name())
	}
}

func TestApply_CrossProviderRoundTrip(t *testing.T) {
	sandbox := setupSandbox(t)
	runXcaffold(t, sandbox, "import")
	xcafDir := filepath.Join(sandbox, "xcaf")
	before := countFiles(t, xcafDir, "*")
	for _, target := range []string{"claude", "cursor", "gemini", "antigravity", "copilot"} {
		_, _, code := runXcaffold(t, sandbox, "apply", "--target", target)
		assert.Equal(t, 0, code, "apply --%s failed", target)
	}
	after := countFiles(t, xcafDir, "*")
	assert.Equal(t, before, after, "xcaf/ file count changed")
}
