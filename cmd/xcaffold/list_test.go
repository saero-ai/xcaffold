package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestList_SingleColumn_EachNameOnOwnLine(t *testing.T) {
	// Tests single-column rendering where each name is on its own line
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-1": {}, "agent-2": {}, "agent-3": {},
			},
		},
	}

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	assert.Contains(t, out, "AGENTS  (3)")
	// Verify each agent is on its own line with indent
	assert.Contains(t, out, "  agent-1")
	assert.Contains(t, out, "  agent-2")
	assert.Contains(t, out, "  agent-3")
}

func TestList_Header_OmitsZeroCounts(t *testing.T) {
	// Config with only agents, no skills/rules/mcp/etc
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-1": {},
			},
		},
	}

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	// Should contain agent count but no mention of zero skills/rules/mcp
	assert.Contains(t, out, "1 agent")
	assert.NotContains(t, out, "0 skills")
	assert.NotContains(t, out, "0 rules")
	assert.NotContains(t, out, "0 mcp")
}

func TestList_KindFilter_Agent(t *testing.T) {
	// Config with multiple kinds
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-1": {},
			},
			Skills: map[string]ast.SkillConfig{
				"skill-1": {},
			},
			Rules: map[string]ast.RuleConfig{
				"rule-1": {},
			},
		},
	}

	// Set filter to agents only
	listFilterAgent = "agent-1"
	defer func() { listFilterAgent = "" }()

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	// Should only show AGENTS section
	assert.Contains(t, out, "AGENTS  (1)")
	assert.Contains(t, out, "agent-1")
	assert.NotContains(t, out, "SKILLS")
	assert.NotContains(t, out, "RULES")
}

func TestList_AddedSections_Contexts_Hooks_Settings(t *testing.T) {
	// Config with new resource types
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-1": {},
			},
			Contexts: map[string]ast.ContextConfig{
				"context-1": {},
			},
		},
		Hooks: map[string]ast.NamedHookConfig{
			"hook-1": {},
		},
		Settings: map[string]ast.SettingsConfig{
			"setting-1": {},
		},
	}

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	// Should show new sections
	assert.Contains(t, out, "AGENTS  (1)")
	assert.Contains(t, out, "CONTEXTS  (1)")
	assert.Contains(t, out, "HOOKS  (1)")
	assert.Contains(t, out, "SETTINGS  (1)")
}

func TestList_VerboseMemory_ShowsEntries(t *testing.T) {
	dir := t.TempDir()

	// Create xcaf/agents/agent1/memory/file1.md
	memDir := filepath.Join(dir, "xcaf", "agents", "agent1", "memory")
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
				Agents: ast.ClearableList{Values: []string{"nestjs"}},
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

func TestList_ArgsValidator_RejectsPositionalArgs(t *testing.T) {
	err := listCmd.Args(listCmd, []string{"dev"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected argument")
	assert.Contains(t, err.Error(), "--flag=dev")
}

func TestList_ArgsValidator_AcceptsNoArgs(t *testing.T) {
	err := listCmd.Args(listCmd, []string{})
	assert.NoError(t, err)
}

func TestList_ZeroMatchFilter_ShowsMessage(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-1": {},
			},
		},
	}

	listFilterAgent = "nonexistent"
	defer func() { listFilterAgent = "" }()

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	assert.Contains(t, out, `No agents matching "nonexistent"`)
	assert.NotContains(t, out, "SKILLS")
}

func TestList_Header_SingularCounts(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:   map[string]ast.AgentConfig{"a": {}},
			Contexts: map[string]ast.ContextConfig{"c": {}},
		},
		Hooks:    map[string]ast.NamedHookConfig{"h": {}},
		Settings: map[string]ast.SettingsConfig{"s": {}},
	}

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	assert.Contains(t, out, "1 agent")
	assert.NotContains(t, out, "1 agents")
	assert.Contains(t, out, "1 context")
	assert.NotContains(t, out, "1 contexts")
	assert.Contains(t, out, "1 hook")
	assert.NotContains(t, out, "1 hooks")
	assert.Contains(t, out, "1 setting")
	assert.NotContains(t, out, "1 settings")
}

func TestList_KindFilter_HeaderShowsOnlyFilteredCounts(t *testing.T) {
	// Config with agents, skills, and rules
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-1": {},
				"agent-2": {},
			},
			Skills: map[string]ast.SkillConfig{
				"skill-1": {},
			},
			Rules: map[string]ast.RuleConfig{
				"rule-1": {},
			},
		},
	}

	// Set filter to agents only
	listFilterAgent = "*"
	defer func() { listFilterAgent = "" }()

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	// Header should only show agent count
	assert.Contains(t, out, "2 agents")
	assert.NotContains(t, out, "skill")
	assert.NotContains(t, out, "rule")
}

func TestList_KindFilter_HeaderShowsMultipleFilters(t *testing.T) {
	// Config with agents, skills, and rules
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"agent-1": {},
			},
			Skills: map[string]ast.SkillConfig{
				"skill-1": {},
				"skill-2": {},
			},
			Rules: map[string]ast.RuleConfig{
				"rule-1": {},
			},
		},
	}

	// Set filter to agents and skills
	listFilterAgent = "*"
	listFilterSkill = "*"
	defer func() {
		listFilterAgent = ""
		listFilterSkill = ""
	}()

	out, _ := captureListStdout(func() error {
		listCmd.SetOut(os.Stdout)
		printAllResources(listCmd, config, "/tmp/proj")
		return nil
	})

	// Header should show agent and skill counts but not rules
	assert.Contains(t, out, "1 agent")
	assert.Contains(t, out, "2 skills")
	assert.NotContains(t, out, "rule")
}

func TestListCmd_BlueprintFlagsVisible(t *testing.T) {
	f := listCmd.Flags()
	bp := f.Lookup("blueprint")
	require.NotNil(t, bp, "--blueprint flag must exist")
	assert.False(t, bp.Hidden, "--blueprint should not be hidden")

	res := f.Lookup("resolved")
	require.NotNil(t, res, "--resolved flag must exist")
	assert.False(t, res.Hidden, "--resolved should not be hidden")
}
