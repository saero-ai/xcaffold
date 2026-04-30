//go:build stress

package stress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/saero-ai/xcaffold/internal/compiler"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/saero-ai/xcaffold/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStress_100Agents(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("version: \"1.0\"\nproject:\n  name: \"stress-test\"\nagents:\n")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&sb, "  agent-%03d:\n    description: \"Agent %d\"\n    instructions: \"Do task %d.\"\n    model: \"claude-3-7-sonnet-20250219\"\n    tools: [Read, Write, Bash]\n", i, i, i)
	}
	config, err := parser.Parse(strings.NewReader(sb.String()))
	require.NoError(t, err)
	assert.Len(t, config.Agents, 100)

	start := time.Now()
	out, _, err := compiler.Compile(config, "", "", "")
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, out.Files, 100)
	t.Logf("Compiled 100 agents in %v", elapsed)
	assert.Less(t, elapsed, 5*time.Second)
}

func TestStress_500Agents(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("version: \"1.0\"\nproject:\n  name: \"stress-test\"\nagents:\n")
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&sb, "  agent-%03d:\n    description: \"Agent %d\"\n    instructions: \"Do task %d with a longer instruction to simulate real-world usage. This paragraph is intentionally padded.\"\n    model: \"claude-3-7-sonnet-20250219\"\n    tools: [Read, Write, Bash, Glob, Grep, Edit]\n", i, i, i)
	}
	config, err := parser.Parse(strings.NewReader(sb.String()))
	require.NoError(t, err)
	assert.Len(t, config.Agents, 500)

	start := time.Now()
	out, _, err := compiler.Compile(config, "", "", "")
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, out.Files, 500)
	t.Logf("Compiled 500 agents in %v", elapsed)
	assert.Less(t, elapsed, 15*time.Second)
}

func TestStress_LargeInstructions(t *testing.T) {
	bigInstructions := strings.Repeat("Follow these instructions carefully. ", 3000)
	yaml := fmt.Sprintf("version: \"1.0\"\nproject:\n  name: \"stress-test\"\nagents:\n  big-agent:\n    description: \"Agent with huge instructions\"\n    instructions: |\n      %s\n", bigInstructions)

	config, err := parser.Parse(strings.NewReader(yaml))
	require.NoError(t, err)

	out, _, err := compiler.Compile(config, "", "", "")
	require.NoError(t, err)

	content := out.Files["agents/big-agent.md"]
	assert.Greater(t, len(content), 100000)
}

func TestStress_FullLifecycle_100Agents(t *testing.T) {
	dir := t.TempDir()

	var sb strings.Builder
	sb.WriteString("version: \"1.0\"\nproject:\n  name: \"lifecycle-stress\"\nagents:\n")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&sb, "  agent-%03d:\n    description: \"Agent %d\"\n    instructions: \"Do task %d.\"\n", i, i, i)
	}

	xcfPath := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(xcfPath, []byte(sb.String()), 0644))

	config, err := parser.ParseFile(xcfPath)
	require.NoError(t, err)

	out, _, err := compiler.Compile(config, "", "", "")
	require.NoError(t, err)

	claudeDir := filepath.Join(dir, ".claude")
	for _, subdir := range []string{"agents", "skills", "rules"} {
		require.NoError(t, os.MkdirAll(filepath.Join(claudeDir, subdir), 0755))
	}
	for path, content := range out.Files {
		absPath := filepath.Join(claudeDir, path)
		require.NoError(t, os.WriteFile(absPath, []byte(content), 0644))
	}

	manifest, err := state.GenerateState(out, state.StateOpts{Target: "claude", BaseDir: dir}, nil)
	require.NoError(t, err)
	statePath := state.StateFilePath(dir, "")
	require.NoError(t, state.WriteState(manifest, statePath))

	recovered, err := state.ReadState(statePath)
	require.NoError(t, err)
	assert.Len(t, recovered.Targets["claude"].Artifacts, 100)

	// Verify determinism
	manifest2, err := state.GenerateState(out, state.StateOpts{Target: "claude", BaseDir: dir}, nil)
	require.NoError(t, err)
	arts1 := manifest.Targets["claude"].Artifacts
	arts2 := manifest2.Targets["claude"].Artifacts
	for i := range arts1 {
		assert.Equal(t, arts1[i].Path, arts2[i].Path)
		assert.Equal(t, arts1[i].Hash, arts2[i].Hash)
	}
}

func TestStress_FullSchema_AllBlockTypes(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("version: \"1.0\"\nproject:\n  name: \"full-stress\"\nagents:\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "  agent-%02d:\n    description: \"Agent %d\"\n    instructions: \"Task %d.\"\n", i, i, i)
	}
	sb.WriteString("skills:\n")
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&sb, "  skill-%02d:\n    description: \"Skill %d\"\n    instructions: \"Do skill %d.\"\n", i, i, i)
	}
	sb.WriteString("rules:\n")
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&sb, "  rule-%02d:\n    instructions: \"Rule %d.\"\n", i, i)
	}

	config, err := parser.Parse(strings.NewReader(sb.String()))
	require.NoError(t, err)

	out, _, err := compiler.Compile(config, "", "", "")
	require.NoError(t, err)

	assert.Len(t, out.Files, 40) // 20 agents + 10 skills + 10 rules

	m1, err2 := state.GenerateState(out, state.StateOpts{Target: "claude"}, nil)
	require.NoError(t, err2)
	m2, err3 := state.GenerateState(out, state.StateOpts{Target: "claude"}, nil)
	require.NoError(t, err3)
	arts1 := m1.Targets["claude"].Artifacts
	arts2 := m2.Targets["claude"].Artifacts
	assert.Len(t, arts1, 40)
	for i := range arts1 {
		assert.Equal(t, arts1[i].Path, arts2[i].Path)
	}
}
