package bir_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFixture writes content to a temp file and returns its path.
func writeFixture(t *testing.T, filename, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestImportWorkflow_StripsFrontmatter(t *testing.T) {
	content := `---
name: my-workflow
description: A test workflow
---
1. Step one
2. Step two
3. Step three`

	path := writeFixture(t, "my-workflow.md", content)
	unit, err := bir.ImportWorkflow(path, "antigravity")
	require.NoError(t, err)
	assert.NotContains(t, unit.ResolvedBody, "name: my-workflow")
	assert.NotContains(t, unit.ResolvedBody, "---")
	assert.Contains(t, unit.ResolvedBody, "1. Step one")
}

func TestImportWorkflow_IDFromFilename(t *testing.T) {
	content := `---
name: deploy
---
1. Build
2. Push`

	path := writeFixture(t, "deploy-workflow.md", content)
	unit, err := bir.ImportWorkflow(path, "antigravity")
	require.NoError(t, err)
	assert.Equal(t, "deploy-workflow", unit.ID)
}

func TestImportWorkflow_SourcePlatformIsAntigravity(t *testing.T) {
	content := "1. Step one\n2. Step two"
	path := writeFixture(t, "simple.md", content)

	unit, err := bir.ImportWorkflow(path, "antigravity")
	require.NoError(t, err)
	assert.Equal(t, "antigravity", unit.SourcePlatform)
}

func TestImportWorkflow_SourceKindIsWorkflow(t *testing.T) {
	content := "1. Step one\n2. Step two"
	path := writeFixture(t, "simple.md", content)

	unit, err := bir.ImportWorkflow(path, "antigravity")
	require.NoError(t, err)
	assert.Equal(t, bir.SourceWorkflow, unit.SourceKind)
}

func TestImportWorkflow_DetectsIntents(t *testing.T) {
	content := `---
name: build
---
1. Run go build
2. Run go test`

	path := writeFixture(t, "build.md", content)
	unit, err := bir.ImportWorkflow(path, "antigravity")
	require.NoError(t, err)
	require.NotEmpty(t, unit.Intents)

	var hasProcedure bool
	for _, intent := range unit.Intents {
		if intent.Type == bir.IntentProcedure {
			hasProcedure = true
		}
	}
	assert.True(t, hasProcedure, "expected procedure intent from numbered steps")
}

func TestImportWorkflow_SimpleProcedure_OnlyProcedureIntent(t *testing.T) {
	content := `1. Read the file
2. Parse YAML
3. Write output`

	path := writeFixture(t, "simple-proc.md", content)
	unit, err := bir.ImportWorkflow(path, "antigravity")
	require.NoError(t, err)
	require.Len(t, unit.Intents, 1)
	assert.Equal(t, bir.IntentProcedure, unit.Intents[0].Type)
}

func TestImportWorkflow_NoFrontmatter_BodyPreserved(t *testing.T) {
	content := "1. First\n2. Second"
	path := writeFixture(t, "no-fm.md", content)

	unit, err := bir.ImportWorkflow(path, "antigravity")
	require.NoError(t, err)
	assert.Contains(t, unit.ResolvedBody, "1. First")
	assert.Contains(t, unit.ResolvedBody, "2. Second")
}

func TestImportWorkflow_MissingFile_ReturnsError(t *testing.T) {
	_, err := bir.ImportWorkflow("/nonexistent/path/workflow.md", "antigravity")
	assert.Error(t, err)
}

func TestImportWorkflow_SourcePathSet(t *testing.T) {
	content := "1. Step one"
	path := writeFixture(t, "check-path.md", content)

	unit, err := bir.ImportWorkflow(path, "antigravity")
	require.NoError(t, err)
	assert.Equal(t, path, unit.SourcePath)
}

func TestImportWorkflow_WithConstraintAndProcedure(t *testing.T) {
	content := `---
name: secure-deploy
---
NEVER deploy without a review.

1. Run tests
2. Tag the release
3. Push to registry`

	path := writeFixture(t, "secure-deploy.md", content)
	unit, err := bir.ImportWorkflow(path, "antigravity")
	require.NoError(t, err)
	assert.Len(t, unit.Intents, 2)

	types := make(map[bir.IntentType]bool)
	for _, intent := range unit.Intents {
		types[intent.Type] = true
	}
	assert.True(t, types[bir.IntentProcedure])
	assert.True(t, types[bir.IntentConstraint])
}
