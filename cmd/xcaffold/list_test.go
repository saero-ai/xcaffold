package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
)

func captureListStdout(f func() error) (string, error) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	err := f()

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), err
}

func TestListCmd_FlagsRegistered(t *testing.T) {
	assert.NotNil(t, listCmd.Flag("blueprint"))
	assert.NotNil(t, listCmd.Flag("resolved"))
	assert.NotNil(t, listCmd.Flag("verbose"))
}

func TestListCmd_IsRegistered(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "list" {
			return
		}
	}
	t.Fatalf("listCmd NOT registered on rootCmd")
}

func TestList_StripInherited_GlobalNotShown(t *testing.T) {
	// Not practically testing full CLI parsing because it requires full mock global,
	// but we can test the `printAllResources` format output assuming the config is stripped
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"local-agent": {},
			},
		},
	}

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	assert.Contains(t, out, "AGENTS  (1)")
	assert.Contains(t, out, "local-agent")
}

func TestList_RuleGrouping_MixedDepths(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"cli/a":      {},
				"platform/b": {},
				"root-rule":  {},
			},
		},
	}

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	assert.Contains(t, out, "cli/  (1)")
	assert.Contains(t, out, "a")
	assert.Contains(t, out, "platform/  (1)")
	assert.Contains(t, out, "b")
	assert.Contains(t, out, "(root)  (1)")
	assert.Contains(t, out, "root-rule")
}

func TestList_RuleGrouping_RootOnly(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: map[string]ast.RuleConfig{
				"root-rule-1": {},
				"root-rule-2": {},
			},
		},
	}

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	assert.Contains(t, out, "(root)  (2)")
	assert.Contains(t, out, "root-rule-1")
	assert.Contains(t, out, "root-rule-2")
}

func TestList_ColumnLayout_ThreePerRow(t *testing.T) {
	// Tests column rendering visually
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"a1": {}, "a2": {}, "a3": {}, "a4": {}, "a5": {}, "a6": {}, "a7": {},
			},
		},
	}

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	assert.Contains(t, out, "AGENTS  (7)")
}

func TestList_VerboseMemory_ShowsEntries(t *testing.T) {
	dir := t.TempDir()

	// Create xcf/agents/agent1/memory/file1.md
	memDir := filepath.Join(dir, "xcf", "agents", "agent1", "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "file1.md"), []byte("..."), 0644)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent1": {},
			},
		},
	}

	// Default
	listVerboseFlag = false
	outDefaults, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, dir)
		return nil
	})
	assert.Contains(t, outDefaults, "MEMORY  (1 entries across 1 agents)")
	assert.Contains(t, outDefaults, "agent1 (1)")

	// Verbose
	listVerboseFlag = true
	outVerbose, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, dir)
		return nil
	})
	assert.Contains(t, outVerbose, "MEMORY  (1 entries across 1 agents)")
	assert.Contains(t, outVerbose, "agent1  (1)")
	assert.Contains(t, outVerbose, "file1")
}

func TestList_Blueprint_FilteredOutput(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{},
		Blueprints: map[string]ast.BlueprintConfig{
			"backend": {
				Agents: []string{"nestjs"},
			},
		},
	}

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printBlueprintResources(listCmd, config, "backend", false)
		return nil
	})

	assert.Contains(t, out, "BLUEPRINT: backend")
	assert.Contains(t, out, "AGENTS  (1)")
	assert.Contains(t, out, "nestjs")
}
