package analyzer

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanProject(t *testing.T) {
	mockFS := fstest.MapFS{
		"package.json":             &fstest.MapFile{Data: []byte(`{"name": "test-app"}`)},
		"CLAUDE.md":                &fstest.MapFile{Data: []byte("Always run formatting")},
		"src/main.go":              &fstest.MapFile{Data: []byte("package main")},  // Should be depth 2, file recorded
		"src/utils/calc.go":        &fstest.MapFile{Data: []byte("package utils")}, // Depth 3, should be ignored
		"node_modules/a.js":        &fstest.MapFile{Data: []byte("ignore me")},     // Ignored dir
		".github/workflows/ci.yml": &fstest.MapFile{Data: []byte("on: push")},      // Depth 3, ignored
		"Makefile":                 &fstest.MapFile{Data: []byte("build: go build")},
		".gitlab-ci.yml":           &fstest.MapFile{Data: []byte("stages: test")},
	}

	sig, err := ScanProject(mockFS)
	require.NoError(t, err)

	// Check files list (should NOT include node_modules or utils/calc.go)
	expectedFiles := []string{
		".gitlab-ci.yml",
		"CLAUDE.md",
		"Makefile",
		"package.json",
		"src/",
		"src/main.go",
	}
	for _, expected := range expectedFiles {
		assert.Contains(t, sig.Files, expected)
	}
	assert.NotContains(t, sig.Files, "node_modules/")
	assert.NotContains(t, sig.Files, "node_modules/a.js")
	assert.NotContains(t, sig.Files, "src/utils/")
	assert.NotContains(t, sig.Files, "src/utils/calc.go")
	assert.NotContains(t, sig.Files, ".github/workflows/ci.yml")

	// Check manifests
	assert.Equal(t, `{"name": "test-app"}`, sig.DependencyManifests["package.json"])
	assert.Equal(t, "build: go build", sig.DependencyManifests["Makefile"])
	assert.Equal(t, "stages: test", sig.DependencyManifests[".gitlab-ci.yml"])

	// Check CLAUDE context
	assert.Equal(t, "Always run formatting", sig.ClaudeConfig)
}

func TestReadTruncated(t *testing.T) {
	mockFS := fstest.MapFS{
		"huge.txt":  &fstest.MapFile{Data: []byte(strings.Repeat("A", 10000))},
		"small.txt": &fstest.MapFile{Data: []byte("hello")},
	}

	huge := readTruncated(mockFS, "huge.txt", 5000)
	assert.True(t, strings.HasPrefix(huge, strings.Repeat("A", 5000)))
	assert.True(t, strings.HasSuffix(huge, "... (truncated)"))
	assert.Equal(t, 5000+16, len(huge)) // 5000 + "\n... (truncated)"

	small := readTruncated(mockFS, "small.txt", 5000)
	assert.Equal(t, "hello", small)
}
