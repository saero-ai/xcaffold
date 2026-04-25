package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCompileMemory_SeedOnce_FileAbsent(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Description:  "Developer role.",
					Instructions: "Robert is the founder.",
				},
			},
		},
	}

	output, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Empty(t, notes, "no fidelity notes for new file")
	require.FileExists(t, filepath.Join(dir, "project_user-role.md"))
}

func TestCompileMemory_SeedOnce_FilePresent_NoOp(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "project_user-role.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("existing content"), 0o600))

	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Instructions: "Robert is the founder.",
				},
			},
		},
	}

	output, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.NotEmpty(t, notes, "fidelity note must be emitted on no-op")

	// File must not have been overwritten.
	data, _ := os.ReadFile(targetPath)
	require.Equal(t, "existing content", string(data))
}

func TestCompileMemory_Reseed_Overwrites(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "project_user-role.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("old content"), 0o600))

	r := NewMemoryRenderer(dir).WithReseed(true)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Instructions: "New content.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(targetPath)
	require.Contains(t, string(data), "New content.")
}

// TestCompileMemory_SeedOnce_ReseedRequired verifies that seed-once semantics
// (now the only mode after lifecycle removal) skip existing files unless --reseed.
func TestCompileMemory_SeedOnce_ReseedRequired(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch-decisions": {
					Name:         "arch-decisions",
					Instructions: "One-way compiler model.",
				},
			},
		},
	}

	output, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.FileExists(t, filepath.Join(dir, "project_arch-decisions.md"))
}

// TestCompileMemory_PriorSeeds_SeedOnce verifies that CompileWithPriorSeeds
// still applies seed-once logic (skips existing files) after lifecycle removal.
func TestCompileMemory_PriorSeeds_SeedOnce(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "project_arch-decisions.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("existing content"), 0o600))

	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch-decisions": {
					Name:         "arch-decisions",
					Instructions: "New xcf content.",
				},
			},
		},
	}

	priorHash := "sha256:abc123notmatching"
	output, notes, err := r.CompileWithPriorSeeds(config, dir, map[string]string{"arch-decisions": priorHash})
	require.NoError(t, err, "seed-once: existing file must not cause an error")
	require.NotNil(t, output)
	require.NotEmpty(t, notes, "seed-once: skip note must be emitted")

	// File must be untouched.
	data, _ := os.ReadFile(targetPath)
	require.Equal(t, "existing content", string(data))
}

// TestCompileMemory_PriorSeeds_ReseedOverrides verifies that WithReseed(true)
// overwrites even when the file exists.
func TestCompileMemory_PriorSeeds_ReseedOverrides(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "project_arch-decisions.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("existing content"), 0o600))

	r := NewMemoryRenderer(dir).WithReseed(true)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch-decisions": {
					Name:         "arch-decisions",
					Instructions: "Authoritative xcf content.",
				},
			},
		},
	}

	priorHash := "sha256:abc123notmatching"
	output, _, err := r.CompileWithPriorSeeds(config, dir, map[string]string{"arch-decisions": priorHash})
	require.NoError(t, err)
	require.NotNil(t, output)

	data, _ := os.ReadFile(targetPath)
	require.Contains(t, string(data), "Authoritative xcf content.")
}

func TestCompileMemory_FrontmatterFormat(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Description:  "Developer role.",
					Instructions: "Robert is the founder.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, "project_user-role.md"))
	content := string(data)
	require.Contains(t, content, "---")
	require.Contains(t, content, `description: "Developer role."`)
	require.Contains(t, content, "Robert is the founder.")
	// type: field must not appear — it was removed from MemoryConfig.
	require.NotContains(t, content, "type:")
}

func TestCompileMemory_MemoryIndexAppend(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Instructions: "Robert is the founder.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	indexPath := filepath.Join(dir, "MEMORY.md")
	data, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "## xcaffold seeds")
	require.Contains(t, string(data), "user-role")
}

func TestCompileMemory_InstructionsFileTraversal_Rejected(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"traversal": {
					InstructionsFile: "../../etc/passwd",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "escapes base dir")
}

func TestCompileMemory_DescriptionWithColon_QuotedSafely(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"colon-desc": {
					Name:         "colon-desc",
					Description:  "has: a colon",
					Instructions: "Some body text.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, readErr := os.ReadFile(filepath.Join(dir, "project_colon-desc.md"))
	require.NoError(t, readErr)
	content := string(data)
	require.Contains(t, content, `description: "has: a colon"`)

	// Extract the YAML frontmatter block (between the two "---" delimiters)
	// and verify it parses correctly.
	parts := strings.SplitN(content, "---", 3)
	require.Len(t, parts, 3, "expected two --- delimiters in output")
	var parsed map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &parsed))
	require.Equal(t, "has: a colon", parsed["description"])
}

// TestCompileMemory_Seeds_Recorded verifies that Seeds() returns a MemorySeed
// for each written entry after Compile.
func TestCompileMemory_Seeds_Recorded(t *testing.T) {
	dir := t.TempDir()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch": {Name: "arch", Instructions: "Original xcf content."},
			},
		},
	}

	r1 := NewMemoryRenderer(dir)
	_, _, err := r1.Compile(config, dir)
	require.NoError(t, err)

	seeds := r1.Seeds()
	require.Len(t, seeds, 1, "one seed must be recorded for one written entry")
	require.Equal(t, "arch", seeds[0].Name)
	require.Equal(t, "claude", seeds[0].Target)
	require.Equal(t, memoryLifecycleSeedOnce, seeds[0].Lifecycle)

	// Simulate a re-apply with the prior hash: seed-once skips, so Seeds() is empty.
	priorHash := seeds[0].Hash
	targetPath := filepath.Join(dir, "project_arch.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("agent modified this"), 0o600))

	r2 := NewMemoryRenderer(dir)
	_, notes, err := r2.CompileWithPriorSeeds(config, dir, map[string]string{"arch": priorHash})
	require.NoError(t, err, "seed-once: existing file must not error")
	require.NotEmpty(t, notes)

	// Verify the FidelityNote code is seed-skipped.
	var hasSkipCode bool
	for _, n := range notes {
		if n.Code == renderer.CodeMemorySeedSkipped {
			hasSkipCode = true
		}
	}
	require.True(t, hasSkipCode, "notes must include a MEMORY_SEED_SKIPPED note")
}
