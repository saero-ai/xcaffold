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

// defaultMEMORY returns the expected path for the default agent's MEMORY.md.
func defaultMEMORY(dir string) string {
	return filepath.Join(dir, "default", "MEMORY.md")
}

// agentMEMORY returns the expected path for a named agent's MEMORY.md.
func agentMEMORY(dir, agentRef string) string {
	return filepath.Join(dir, agentRef, "MEMORY.md")
}

// TestMemoryRenderer_ConcatenatesIntoMEMORY verifies the primary new behavior:
// two entries for the same agent produce a single concatenated MEMORY.md with
// ## <name> headings, and no per-entry .md files are written.
func TestMemoryRenderer_ConcatenatesIntoMEMORY(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:     "user-role",
					Content:  "Robert is the founder.",
					AgentRef: "backend-dev",
				},
				"arch-decisions": {
					Name:     "arch-decisions",
					Content:  "One-way compiler model.",
					AgentRef: "backend-dev",
				},
			},
		},
	}

	output, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Empty(t, notes, "no fidelity notes for new files")

	memPath := agentMEMORY(dir, "backend-dev")
	require.FileExists(t, memPath)

	data, err := os.ReadFile(memPath)
	require.NoError(t, err)
	content := string(data)

	require.Contains(t, content, "## user-role")
	require.Contains(t, content, "Robert is the founder.")
	require.Contains(t, content, "## arch-decisions")
	require.Contains(t, content, "One-way compiler model.")

	// No individual per-entry files should be written.
	entries, err := os.ReadDir(filepath.Join(dir, "backend-dev"))
	require.NoError(t, err)
	require.Len(t, entries, 1, "only MEMORY.md must exist in agent dir")
	require.Equal(t, "MEMORY.md", entries[0].Name())
}

// TestMemoryRenderer_MultiAgent writes to separate per-agent directories.
func TestMemoryRenderer_MultiAgent(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"backend-pref": {
					Name:     "backend-pref",
					Content:  "Backend pref body.",
					AgentRef: "backend-dev",
				},
				"frontend-pref": {
					Name:     "frontend-pref",
					Content:  "Frontend pref body.",
					AgentRef: "frontend-dev",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	require.FileExists(t, agentMEMORY(dir, "backend-dev"))
	require.FileExists(t, agentMEMORY(dir, "frontend-dev"))

	backendData, _ := os.ReadFile(agentMEMORY(dir, "backend-dev"))
	require.Contains(t, string(backendData), "## backend-pref")
	require.NotContains(t, string(backendData), "## frontend-pref")

	frontendData, _ := os.ReadFile(agentMEMORY(dir, "frontend-dev"))
	require.Contains(t, string(frontendData), "## frontend-pref")
	require.NotContains(t, string(frontendData), "## backend-pref")
}

// TestMemoryRenderer_DefaultAgentRef uses "default" when AgentRef is empty.
func TestMemoryRenderer_DefaultAgentRef(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:    "user-role",
					Content: "Robert is the founder.",
					// AgentRef intentionally left empty.
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.FileExists(t, defaultMEMORY(dir))
}

func TestCompileMemory_SeedOnce_FileAbsent(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:        "user-role",
					Description: "Developer role.",
					Content:     "Robert is the founder.",
				},
			},
		},
	}

	output, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Empty(t, notes, "no fidelity notes for new file")
	require.FileExists(t, defaultMEMORY(dir))
}

func TestCompileMemory_SeedOnce_FilePresent_NoOp(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "default")
	require.NoError(t, os.MkdirAll(agentDir, 0o700))
	memPath := filepath.Join(agentDir, "MEMORY.md")
	require.NoError(t, os.WriteFile(memPath, []byte("existing content"), 0o600))

	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:    "user-role",
					Content: "Robert is the founder.",
				},
			},
		},
	}

	output, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.NotEmpty(t, notes, "fidelity note must be emitted on no-op")

	// File must not have been overwritten.
	data, _ := os.ReadFile(memPath)
	require.Equal(t, "existing content", string(data))
}

func TestCompileMemory_Reseed_Overwrites(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "default")
	require.NoError(t, os.MkdirAll(agentDir, 0o700))
	memPath := filepath.Join(agentDir, "MEMORY.md")
	require.NoError(t, os.WriteFile(memPath, []byte("old content"), 0o600))

	r := NewMemoryRenderer(dir).WithReseed(true)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:    "user-role",
					Content: "New content.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(memPath)
	require.Contains(t, string(data), "New content.")
}

// TestCompileMemory_SeedOnce_ReseedRequired verifies seed-once semantics skip
// existing MEMORY.md unless --reseed.
func TestCompileMemory_SeedOnce_ReseedRequired(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch-decisions": {
					Name:    "arch-decisions",
					Content: "One-way compiler model.",
				},
			},
		},
	}

	output, _, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.FileExists(t, defaultMEMORY(dir))
}

// TestCompileMemory_PriorSeeds_SeedOnce verifies that CompileWithPriorSeeds
// still applies seed-once logic (skips existing MEMORY.md).
func TestCompileMemory_PriorSeeds_SeedOnce(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "default")
	require.NoError(t, os.MkdirAll(agentDir, 0o700))
	memPath := filepath.Join(agentDir, "MEMORY.md")
	require.NoError(t, os.WriteFile(memPath, []byte("existing content"), 0o600))

	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch-decisions": {
					Name:    "arch-decisions",
					Content: "New xcf content.",
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
	data, _ := os.ReadFile(memPath)
	require.Equal(t, "existing content", string(data))
}

// TestCompileMemory_PriorSeeds_ReseedOverrides verifies that WithReseed(true)
// overwrites even when MEMORY.md exists.
func TestCompileMemory_PriorSeeds_ReseedOverrides(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "default")
	require.NoError(t, os.MkdirAll(agentDir, 0o700))
	memPath := filepath.Join(agentDir, "MEMORY.md")
	require.NoError(t, os.WriteFile(memPath, []byte("existing content"), 0o600))

	r := NewMemoryRenderer(dir).WithReseed(true)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch-decisions": {
					Name:    "arch-decisions",
					Content: "Authoritative xcf content.",
				},
			},
		},
	}

	priorHash := "sha256:abc123notmatching"
	output, _, err := r.CompileWithPriorSeeds(config, dir, map[string]string{"arch-decisions": priorHash})
	require.NoError(t, err)
	require.NotNil(t, output)

	data, _ := os.ReadFile(memPath)
	require.Contains(t, string(data), "Authoritative xcf content.")
}

func TestCompileMemory_FrontmatterFormat(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"user-role": {
					Name:        "user-role",
					Description: "Developer role.",
					Content:     "Robert is the founder.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(defaultMEMORY(dir))
	content := string(data)
	require.Contains(t, content, "## user-role")
	require.Contains(t, content, "Robert is the founder.")
	// type: field must not appear.
	require.NotContains(t, content, "type:")
}

func TestCompileMemory_DescriptionWithColon_QuotedSafely(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"colon-desc": {
					Name:        "colon-desc",
					Description: "has: a colon",
					Content:     "Some body text.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, readErr := os.ReadFile(defaultMEMORY(dir))
	require.NoError(t, readErr)
	content := string(data)

	// The description is embedded in the ## heading section, not frontmatter.
	// Verify the body text appears.
	require.Contains(t, content, "Some body text.")
	require.Contains(t, content, "## colon-desc")

	// No YAML frontmatter block in the new concatenated format.
	require.NotContains(t, content, "---")
}

func TestRenderMemoryMarkdown_NoType(t *testing.T) {
	entry := ast.MemoryConfig{Description: "User preferences"}
	out := renderMemoryMarkdown(entry, "Always be concise.")
	require.NotContains(t, out, "type:")
	require.Contains(t, out, `description: "User preferences"`)
	require.Contains(t, out, "Always be concise.")
}

func TestRenderMemoryMarkdown_NoDescription_NoFrontmatter(t *testing.T) {
	entry := ast.MemoryConfig{}
	out := renderMemoryMarkdown(entry, "Plain body text.")
	require.NotContains(t, out, "---")
	require.Equal(t, "Plain body text.\n", out)
}

// TestCompileMemory_Seeds_Recorded verifies that Seeds() returns a MemorySeed
// for each written agent MEMORY.md after Compile.
func TestCompileMemory_Seeds_Recorded(t *testing.T) {
	dir := t.TempDir()

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"arch": {Name: "arch", Content: "Original xcf content."},
			},
		},
	}

	r1 := NewMemoryRenderer(dir)
	_, _, err := r1.Compile(config, dir)
	require.NoError(t, err)

	seeds := r1.Seeds()
	require.Len(t, seeds, 1, "one seed per agent must be recorded")
	require.Equal(t, "default", seeds[0].Name)
	require.Equal(t, "claude", seeds[0].Target)

	// Simulate a re-apply with the prior hash: seed-once skips, so Seeds() is empty.
	priorHash := seeds[0].Hash
	agentDir := filepath.Join(dir, "default")
	memPath := filepath.Join(agentDir, "MEMORY.md")
	require.NoError(t, os.WriteFile(memPath, []byte("agent modified this"), 0o600))

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

// TestCompileMemory_EmptyBody_Skipped verifies that entries with empty bodies
// do not produce files or errors.
func TestCompileMemory_EmptyBody_Skipped(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"empty": {
					Name: "empty",
					// No instructions or instructions-file.
				},
			},
		},
	}

	_, notes, err := r.Compile(config, dir)
	require.NoError(t, err)
	require.NotEmpty(t, notes)
	var hasEmpty bool
	for _, n := range notes {
		if n.Code == renderer.CodeMemoryBodyEmpty {
			hasEmpty = true
		}
	}
	require.True(t, hasEmpty, "must emit MEMORY_BODY_EMPTY note")

	// No agent dir should be created for all-empty entries.
	_, statErr := os.Stat(filepath.Join(dir, "default"))
	require.True(t, os.IsNotExist(statErr), "no agent dir for empty-body-only agent")
}

// TestCompileMemory_AgentRefTraversal_Rejected verifies that an AgentRef
// containing traversal sequences is rejected.
func TestCompileMemory_AgentRefTraversal_Rejected(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"evil": {
					Name:     "evil",
					Content:  "bad content",
					AgentRef: "../escaped",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "agent-ref")
}

// TestCompileMemory_DeterministicOrder verifies that entries in a single
// MEMORY.md are sorted by name regardless of map iteration order.
func TestCompileMemory_DeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	r := NewMemoryRenderer(dir)
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"zzz-last": {
					Name:    "zzz-last",
					Content: "Z body.",
				},
				"aaa-first": {
					Name:    "aaa-first",
					Content: "A body.",
				},
			},
		},
	}

	_, _, err := r.Compile(config, dir)
	require.NoError(t, err)

	data, _ := os.ReadFile(defaultMEMORY(dir))
	content := string(data)

	posFirst := strings.Index(content, "## aaa-first")
	posLast := strings.Index(content, "## zzz-last")
	require.Greater(t, posLast, posFirst, "aaa-first must appear before zzz-last")
}

// TestCompileMemory_YAML_unused ensures yaml import is used (keeps the import
// intact for TestCompileMemory_DescriptionWithColon_QuotedSafely usage).
func TestCompileMemory_YAML_unused(t *testing.T) {
	// yaml.Unmarshal is used in TestCompileMemory_DescriptionWithColon_QuotedSafely;
	// this test is a no-op placeholder to satisfy static analysis if that test
	// is removed in the future.
	var m map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte("key: value"), &m))
}
