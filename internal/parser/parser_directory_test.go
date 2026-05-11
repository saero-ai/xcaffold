package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestXCAF(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0600))
	return p
}

func TestParseDirectory_DuplicateAgentID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "project.xcaf", `kind: project
version: "1.0"
name: "test-project"
`)
	writeTestXCAF(t, dir, "agents.xcaf", `kind: global
version: "1.0"
agents:
  developer:
    description: "First developer"

`)
	writeTestXCAF(t, dir, "tools.xcaf", `kind: global
version: "1.0"
agents:
  developer:
    description: "Duplicate developer"

`)

	_, err := ParseDirectory(dir)
	require.Error(t, err, "duplicate agent ID across files must error")
	assert.Contains(t, err.Error(), "developer")
	assert.Contains(t, err.Error(), "agents.xcaf")
	assert.Contains(t, err.Error(), "tools.xcaf")
}

func TestParseDirectory_DuplicateSkillID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "skills1.xcaf", `kind: global
version: "1.0"
skills:
  git:
    description: "Git skill"
`)
	writeTestXCAF(t, dir, "skills2.xcaf", `kind: global
version: "1.0"
skills:
  git:
    description: "Duplicate git skill"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate skill ID \"git\"")
	assert.Contains(t, err.Error(), "skills1.xcaf")
	assert.Contains(t, err.Error(), "skills2.xcaf")
}

func TestParseDirectory_DuplicateRuleID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "rules_a.xcaf", `kind: global
version: "1.0"
rules:
  no-panics:
    description: "No panics"
`)
	writeTestXCAF(t, dir, "rules_b.xcaf", `kind: global
version: "1.0"
rules:
  no-panics:
    description: "Return errors instead"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate rule ID \"no-panics\"")
	assert.Contains(t, err.Error(), "rules_a.xcaf")
	assert.Contains(t, err.Error(), "rules_b.xcaf")
}

func TestParseDirectory_DuplicateMCPID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "mcp1.xcaf", `kind: global
version: "1.0"
mcp:
  postgres:
    command: "npx"
`)
	writeTestXCAF(t, dir, "mcp2.xcaf", `kind: global
version: "1.0"
mcp:
  postgres:
    command: "docker"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate mcp ID \"postgres\"")
	assert.Contains(t, err.Error(), "mcp1.xcaf")
	assert.Contains(t, err.Error(), "mcp2.xcaf")
}

func TestParseDirectory_DuplicateWorkflowID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "flows1.xcaf", `kind: global
version: "1.0"
workflows:
  launch:
    description: "Launch flow"
`)
	writeTestXCAF(t, dir, "flows2.xcaf", `kind: global
version: "1.0"
workflows:
  launch:
    description: "Another launch flow"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate workflow ID \"launch\"")
	assert.Contains(t, err.Error(), "flows1.xcaf")
	assert.Contains(t, err.Error(), "flows2.xcaf")
}

func TestParseDirectory_RecursiveSubdirectories(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "root.xcaf", `kind: project
version: "1.0"
name: "recursive-test"
`)
	writeTestXCAF(t, dir, "sub/dir1/agent.xcaf", `kind: global
version: "1.0"
agents:
  sub-agent:
    description: "I am nested"
`)
	writeTestXCAF(t, dir, "sub/dir2/deep/skill.xcaf", `kind: global
version: "1.0"
skills:
  deep-skill:
    description: "I am very nested"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	assert.Equal(t, "recursive-test", cfg.Project.Name)

	require.Contains(t, cfg.Agents, "sub-agent")
	assert.Equal(t, "I am nested", cfg.Agents["sub-agent"].Description)

	require.Contains(t, cfg.Skills, "deep-skill")
	assert.Equal(t, "I am very nested", cfg.Skills["deep-skill"].Description)
}

func TestParseDirectory_HiddenDirectoriesSkipped(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "root.xcaf", `kind: project
version: "1.0"
name: "visible-project"
`)
	writeTestXCAF(t, dir, ".hidden/agent.xcaf", `kind: global
version: "1.0"
agents:
  hidden-agent:
    description: "Should not be read"
`)
	writeTestXCAF(t, dir, "node_modules/pkg/agent.xcaf", `kind: global
version: "1.0"
agents:
  node-agent:
    description: "Should not be read"
`)
	writeTestXCAF(t, dir, "sub/.git/agent.xcaf", `kind: global
version: "1.0"
agents:
  git-agent:
    description: "Should not be read"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	assert.Equal(t, "visible-project", cfg.Project.Name)
	assert.NotContains(t, cfg.Agents, "hidden-agent")
	assert.NotContains(t, cfg.Agents, "node-agent")
	assert.NotContains(t, cfg.Agents, "git-agent")
}

func TestParseDirectory_SingleFileFallback(t *testing.T) {
	// Multi-doc .xcaf files are no longer supported; split into separate files.
	dir := t.TempDir()
	writeTestXCAF(t, dir, "project.xcaf", `kind: project
version: "1.0"
name: "fallback-project"
`)
	writeTestXCAF(t, dir, "global.xcaf", `kind: global
version: "1.0"
agents:
  fallback-agent:
    description: "Fallback"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, "fallback-project", cfg.Project.Name)
	require.Contains(t, cfg.Agents, "fallback-agent")
}

func TestParseDirectory_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, strings.ToLower(err.Error()), "no *.xcaf files found")
}

func TestParseDirectory_ConflictingVersionAndProject_Errors(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "a.xcaf", `kind: project
version: "1.0"
name: "project-a"
`)
	writeTestXCAF(t, dir, "b.xcaf", `kind: project
version: "2.0"
name: "project-b"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	// We expect the parser to fail on the version conflict first or project conflict.
	// Either is fine, we just verify it correctly rejects.
	assert.True(t,
		strings.Contains(err.Error(), "conflicting versions") || strings.Contains(err.Error(), "multiple files declare project.name"),
		"Error should complain about version or project conflicts",
	)
}

func TestParseDirectory_MultiFileHappyPath(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "global.xcaf", `kind: project
version: "1.0"
name: "happy-path-project"
`)
	writeTestXCAF(t, dir, "agents_front.xcaf", `kind: global
version: "1.0"
agents:
  frontend:
    description: "Frontend developer"
`)
	writeTestXCAF(t, dir, "agents_back.xcaf", `kind: global
version: "1.0"
agents:
  backend:
    description: "Backend developer"
`)
	writeTestXCAF(t, dir, "skills/utils.xcaf", `kind: global
version: "1.0"
skills:
  git:
    description: "Git skill"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	assert.Equal(t, "happy-path-project", cfg.Project.Name)
	assert.Equal(t, "1.0", cfg.Version)
	require.Contains(t, cfg.Agents, "frontend")
	require.Contains(t, cfg.Agents, "backend")
	require.Contains(t, cfg.Skills, "git")
}

func TestParseDirectory_SettingsDeepMerge_NonConflicting(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "project.xcaf", `kind: project
version: "1.0"
name: "settings-merge-test"
`)
	writeTestXCAF(t, dir, "project-settings.xcaf", `kind: settings
version: "1.0"
model: "sonnet-4"
`)
	writeTestXCAF(t, dir, "settings.xcaf", `kind: settings
version: "1.0"
effort-level: "high"
env:
  API_KEY: "test"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, "sonnet-4", cfg.Settings["default"].Model)
	assert.Equal(t, "high", cfg.Settings["default"].EffortLevel)
	assert.Equal(t, "test", cfg.Settings["default"].Env["API_KEY"])
}

func TestParseDirectory_SettingsConflict_Errors(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "a.xcaf", `kind: project
version: "1.0"
name: "conflict-test"
`)
	writeTestXCAF(t, dir, "a-settings.xcaf", `kind: settings
version: "1.0"
model: "sonnet-4"
`)
	writeTestXCAF(t, dir, "b.xcaf", `kind: settings
version: "1.0"
model: "opus-4"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model")
}

func TestParse_Profile_IsParseableFile(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "project.xcaf")
	require.NoError(t, os.WriteFile(proj, []byte("kind: project\nname: myproject\nversion: \"1.0\"\n"), 0600))
	pf := filepath.Join(dir, "backend.xcaf")
	require.NoError(t, os.WriteFile(pf, []byte("kind: profile\nname: backend\nversion: \"1.0\"\n"), 0600))
	_, err := ParseDirectory(dir)
	require.NoError(t, err)
}

func TestParseDirectory_ExtendsGlobal_InheritsSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create global config
	globalDir := filepath.Join(home, ".xcaffold")
	require.NoError(t, os.MkdirAll(globalDir, 0755))
	writeTestXCAF(t, globalDir, "global.xcaf", `kind: global
version: "1.0"
settings:
  model: "sonnet-4"
  effort-level: "high"
  env:
    GLOBAL_KEY: "from-global"
`)

	// Create project config that extends global
	projectDir := t.TempDir()
	writeTestXCAF(t, projectDir, "project.xcaf", `kind: global
version: "1.0"
extends: global
settings:
  effort-level: "low"
  env:
    PROJECT_KEY: "from-project"
`)

	cfg, err := ParseDirectory(projectDir)
	require.NoError(t, err)
	assert.Equal(t, "sonnet-4", cfg.Settings["default"].Model)
	assert.Equal(t, "low", cfg.Settings["default"].EffortLevel)
	assert.Equal(t, "from-global", cfg.Settings["default"].Env["GLOBAL_KEY"])
	assert.Equal(t, "from-project", cfg.Settings["default"].Env["PROJECT_KEY"])
}

func TestParseDirectory_VariableExpansion(t *testing.T) {
	dir := t.TempDir()

	writeTestXCAF(t, dir, "xcaf/project.vars", `org = acme
`)
	writeTestXCAF(t, dir, "project.xcaf", `kind: project
version: "1.0"
name: test-proj
`)
	writeTestXCAF(t, dir, "xcaf/agents/dev/agent.xcaf", `kind: agent
version: "1.0"
name: "dev"
description: "Agent for ${var.org}"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	require.Contains(t, cfg.Agents, "dev")
	assert.Equal(t, "Agent for acme", cfg.Agents["dev"].Description)
}

func TestParseDirectory_IgnoresLegacyXCFFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a valid .xcaf file that SHOULD be parsed
	writeTestXCAF(t, dir, "project.xcaf", `kind: project
version: "1.0"
name: test
`)
	writeTestXCAF(t, dir, "xcaf/agents/dev/agent.xcaf", `---
kind: agent
version: "1.0"
name: dev
description: "Valid agent"
`)

	// Create a .xcf file (old extension) that SHOULD be ignored
	writeTestXCAF(t, dir, "xcaf/agents/legacy/agent.xcf", `---
kind: agent
version: "1.0"
name: legacy
description: "Should be ignored"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)

	// The .xcaf agent should be found
	assert.Contains(t, cfg.Agents, "dev")
	// The .xcf agent should NOT be found
	assert.NotContains(t, cfg.Agents, "legacy", ".xcf files must be ignored (clean break)")
}

func TestParseDirectory_ErrorsOnXCFOnlyDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create only .xcf files — parser should find nothing parseable
	writeTestXCAF(t, dir, "agents/dev.xcf", `---
kind: agent
version: "1.0"
name: dev
description: "Legacy"
`)

	_, err := ParseDirectory(dir)
	assert.Error(t, err, "directory with only .xcf files should produce an error (no parseable files)")
	assert.Contains(t, strings.ToLower(err.Error()), "no *.xcaf files found")
}
