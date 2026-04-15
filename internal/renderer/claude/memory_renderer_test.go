package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
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
					Type:         "user",
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
	require.FileExists(t, filepath.Join(dir, "user-role.md"))
}

func TestCompileMemory_SeedOnce_FilePresent_NoOp(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "user-role.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("existing content"), 0o600))

	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Type:         "user",
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
	targetPath := filepath.Join(dir, "user-role.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("old content"), 0o600))

	r := NewMemoryRenderer(dir).WithReseed(true)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Type:         "user",
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

func TestCompileMemory_Tracked_FirstApply(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch-decisions": {
					Name:         "arch-decisions",
					Type:         "reference",
					Lifecycle:    "tracked",
					Instructions: "One-way compiler model.",
				},
			},
		},
	}

	output, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.FileExists(t, filepath.Join(dir, "arch-decisions.md"))
}

func TestCompileMemory_Tracked_DriftDetected(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "arch-decisions.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("agent modified this"), 0o600))

	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch-decisions": {
					Name:         "arch-decisions",
					Type:         "reference",
					Lifecycle:    "tracked",
					Instructions: "Original xcf content.",
				},
			},
		},
	}

	// Prior seed hash does not match the current on-disk content
	priorHash := "sha256:abc123notmatching"

	_, _, err := r.CompileWithPriorSeeds(config, dir, map[string]string{"arch-decisions": priorHash})
	require.Error(t, err, "drift must produce an error")
	require.Contains(t, err.Error(), "memory drift detected")
}

func TestCompileMemory_Tracked_ReseedOverridesDrift(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "arch-decisions.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("agent modified this"), 0o600))

	r := NewMemoryRenderer(dir).WithReseed(true)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch-decisions": {
					Name:         "arch-decisions",
					Type:         "reference",
					Lifecycle:    "tracked",
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
					Type:         "user",
					Description:  "Developer role.",
					Instructions: "Robert is the founder.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, "user-role.md"))
	content := string(data)
	require.Contains(t, content, "---")
	require.Contains(t, content, "type: user")
	require.Contains(t, content, `description: "Developer role."`)
	require.Contains(t, content, "Robert is the founder.")
}

func TestCompileMemory_MemoryIndexAppend(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:         "user-role",
					Type:         "user",
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

func TestCompileMemory_EntryNameWithSlash_Rejected(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"bad/name": {
					Name:         "bad/name",
					Instructions: "some content",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must not contain path separators")
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

func TestCompileMemory_UnknownLifecycle_Rejected(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"bad-lifecycle": {
					Name:         "bad-lifecycle",
					Lifecycle:    "weird",
					Instructions: "some content",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown lifecycle")
}

func TestCompileMemory_DescriptionWithColon_QuotedSafely(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"colon-desc": {
					Name:         "colon-desc",
					Type:         "user",
					Description:  "has: a colon",
					Instructions: "Some body text.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, readErr := os.ReadFile(filepath.Join(dir, "colon-desc.md"))
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
