package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestXCF(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0600))
	return p
}

func TestParseDirectory_DuplicateAgentID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "project.xcf", `
version: "1.0"
project:
  name: "test-project"
`)
	writeTestXCF(t, dir, "agents.xcf", `
agents:
  developer:
    description: "First developer"
    instructions: "Do stuff"
`)
	writeTestXCF(t, dir, "tools.xcf", `
agents:
  developer:
    description: "Duplicate developer"
    instructions: "Do other stuff"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err, "duplicate agent ID across files must error")
	assert.Contains(t, err.Error(), "developer")
	assert.Contains(t, err.Error(), "agents.xcf")
	assert.Contains(t, err.Error(), "tools.xcf")
}

func TestParseDirectory_DuplicateSkillID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "skills1.xcf", `
skills:
  git:
    description: "Git skill"
`)
	writeTestXCF(t, dir, "skills2.xcf", `
skills:
  git:
    description: "Duplicate git skill"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate skill ID \"git\"")
	assert.Contains(t, err.Error(), "skills1.xcf")
	assert.Contains(t, err.Error(), "skills2.xcf")
}

func TestParseDirectory_DuplicateRuleID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "rules_a.xcf", `
rules:
  no-panics:
    description: "No panics"
`)
	writeTestXCF(t, dir, "rules_b.xcf", `
rules:
  no-panics:
    description: "Return errors instead"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate rule ID \"no-panics\"")
	assert.Contains(t, err.Error(), "rules_a.xcf")
	assert.Contains(t, err.Error(), "rules_b.xcf")
}

func TestParseDirectory_DuplicateMCPID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "mcp1.xcf", `
mcp:
  postgres:
    command: "npx"
`)
	writeTestXCF(t, dir, "mcp2.xcf", `
mcp:
  postgres:
    command: "docker"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate mcp ID \"postgres\"")
	assert.Contains(t, err.Error(), "mcp1.xcf")
	assert.Contains(t, err.Error(), "mcp2.xcf")
}

func TestParseDirectory_DuplicateWorkflowID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "flows1.xcf", `
workflows:
  launch:
    description: "Launch flow"
`)
	writeTestXCF(t, dir, "flows2.xcf", `
workflows:
  launch:
    description: "Another launch flow"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate workflow ID \"launch\"")
	assert.Contains(t, err.Error(), "flows1.xcf")
	assert.Contains(t, err.Error(), "flows2.xcf")
}

func TestParseDirectory_RecursiveSubdirectories(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "root.xcf", `
version: "1.0"
project:
  name: "recursive-test"
`)
	writeTestXCF(t, dir, "sub/dir1/agent.xcf", `
agents:
  sub-agent:
    description: "I am nested"
`)
	writeTestXCF(t, dir, "sub/dir2/deep/skill.xcf", `
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

	writeTestXCF(t, dir, "root.xcf", `
version: "1.0"
project:
  name: "visible-project"
`)
	writeTestXCF(t, dir, ".hidden/agent.xcf", `
agents:
  hidden-agent:
    description: "Should not be read"
`)
	writeTestXCF(t, dir, "node_modules/pkg/agent.xcf", `
agents:
  node-agent:
    description: "Should not be read"
`)
	writeTestXCF(t, dir, "sub/.git/agent.xcf", `
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
	dir := t.TempDir()
	path := writeTestXCF(t, dir, "scaffold.xcf", `
version: "1.0"
project:
  name: "fallback-project"
agents:
  fallback-agent:
    description: "Fallback"
`)

	cfg, err := ParseDirectory(path)
	require.NoError(t, err)
	assert.Equal(t, "fallback-project", cfg.Project.Name)
	require.Contains(t, cfg.Agents, "fallback-agent")
}

func TestParseDirectory_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, strings.ToLower(err.Error()), "no *.xcf files found")
}

func TestParseDirectory_ConflictingVersionAndProject_Errors(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "a.xcf", `
version: "1.0"
project:
  name: "project-a"
`)
	writeTestXCF(t, dir, "b.xcf", `
version: "2.0"
project:
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

	writeTestXCF(t, dir, "global.xcf", `
version: "1.0"
project:
  name: "happy-path-project"
`)
	writeTestXCF(t, dir, "agents_front.xcf", `
agents:
  frontend:
    description: "Frontend developer"
`)
	writeTestXCF(t, dir, "agents_back.xcf", `
agents:
  backend:
    description: "Backend developer"
`)
	writeTestXCF(t, dir, "skills/utils.xcf", `
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

	writeTestXCF(t, dir, "project.xcf", `
version: "1.0"
project:
  name: "settings-merge-test"
settings:
  model: "sonnet-4"
`)
	writeTestXCF(t, dir, "settings.xcf", `
settings:
  effortLevel: "high"
  env:
    API_KEY: "test"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, "sonnet-4", cfg.Settings.Model)
	assert.Equal(t, "high", cfg.Settings.EffortLevel)
	assert.Equal(t, "test", cfg.Settings.Env["API_KEY"])
}

func TestParseDirectory_SettingsConflict_Errors(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "a.xcf", `
version: "1.0"
project:
  name: "conflict-test"
settings:
  model: "sonnet-4"
`)
	writeTestXCF(t, dir, "b.xcf", `
settings:
  model: "opus-4"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model")
}

func TestParseDirectory_LocalDeepMerge_NonConflicting(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "project.xcf", `
version: "1.0"
project:
  name: "local-merge-test"
local:
  env:
    SECRET: "abc"
`)
	writeTestXCF(t, dir, "local-overrides.xcf", `
local:
  effortLevel: "low"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	assert.Equal(t, "low", cfg.Local.EffortLevel)
	assert.Equal(t, "abc", cfg.Local.Env["SECRET"])
}

func TestParseDirectory_ExtendsGlobal_InheritsSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create global config
	globalDir := filepath.Join(home, ".xcaffold")
	require.NoError(t, os.MkdirAll(globalDir, 0755))
	writeTestXCF(t, globalDir, "global.xcf", `
version: "1.0"
project:
  name: "global"
settings:
  model: "sonnet-4"
  effortLevel: "high"
  env:
    GLOBAL_KEY: "from-global"
`)

	// Create project config that extends global
	projectDir := t.TempDir()
	writeTestXCF(t, projectDir, "scaffold.xcf", `
version: "1.0"
project:
  name: "my-project"
extends: global
settings:
  effortLevel: "low"
  env:
    PROJECT_KEY: "from-project"
`)

	cfg, err := ParseDirectory(projectDir)
	require.NoError(t, err)
	assert.Equal(t, "sonnet-4", cfg.Settings.Model)
	assert.Equal(t, "low", cfg.Settings.EffortLevel)
	assert.Equal(t, "from-global", cfg.Settings.Env["GLOBAL_KEY"])
	assert.Equal(t, "from-project", cfg.Settings.Env["PROJECT_KEY"])
}
