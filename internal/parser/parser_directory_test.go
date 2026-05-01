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

	writeTestXCF(t, dir, "project.xcf", `kind: project
version: "1.0"
name: "test-project"
`)
	writeTestXCF(t, dir, "agents.xcf", `kind: global
version: "1.0"
agents:
  developer:
    description: "First developer"

`)
	writeTestXCF(t, dir, "tools.xcf", `kind: global
version: "1.0"
agents:
  developer:
    description: "Duplicate developer"

`)

	_, err := ParseDirectory(dir)
	require.Error(t, err, "duplicate agent ID across files must error")
	assert.Contains(t, err.Error(), "developer")
	assert.Contains(t, err.Error(), "agents.xcf")
	assert.Contains(t, err.Error(), "tools.xcf")
}

func TestParseDirectory_DuplicateSkillID_ReportsBothFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "skills1.xcf", `kind: global
version: "1.0"
skills:
  git:
    description: "Git skill"
`)
	writeTestXCF(t, dir, "skills2.xcf", `kind: global
version: "1.0"
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

	writeTestXCF(t, dir, "rules_a.xcf", `kind: global
version: "1.0"
rules:
  no-panics:
    description: "No panics"
`)
	writeTestXCF(t, dir, "rules_b.xcf", `kind: global
version: "1.0"
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

	writeTestXCF(t, dir, "mcp1.xcf", `kind: global
version: "1.0"
mcp:
  postgres:
    command: "npx"
`)
	writeTestXCF(t, dir, "mcp2.xcf", `kind: global
version: "1.0"
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

	writeTestXCF(t, dir, "flows1.xcf", `kind: global
version: "1.0"
workflows:
  launch:
    description: "Launch flow"
`)
	writeTestXCF(t, dir, "flows2.xcf", `kind: global
version: "1.0"
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

	writeTestXCF(t, dir, "root.xcf", `kind: project
version: "1.0"
name: "recursive-test"
`)
	writeTestXCF(t, dir, "sub/dir1/agent.xcf", `kind: global
version: "1.0"
agents:
  sub-agent:
    description: "I am nested"
`)
	writeTestXCF(t, dir, "sub/dir2/deep/skill.xcf", `kind: global
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

	writeTestXCF(t, dir, "root.xcf", `kind: project
version: "1.0"
name: "visible-project"
`)
	writeTestXCF(t, dir, ".hidden/agent.xcf", `kind: global
version: "1.0"
agents:
  hidden-agent:
    description: "Should not be read"
`)
	writeTestXCF(t, dir, "node_modules/pkg/agent.xcf", `kind: global
version: "1.0"
agents:
  node-agent:
    description: "Should not be read"
`)
	writeTestXCF(t, dir, "sub/.git/agent.xcf", `kind: global
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
	// Multi-doc .xcf files are no longer supported; split into separate files.
	dir := t.TempDir()
	writeTestXCF(t, dir, "project.xcf", `kind: project
version: "1.0"
name: "fallback-project"
`)
	writeTestXCF(t, dir, "global.xcf", `kind: global
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
	assert.Contains(t, strings.ToLower(err.Error()), "no *.xcf files found")
}

func TestParseDirectory_ConflictingVersionAndProject_Errors(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "a.xcf", `kind: project
version: "1.0"
name: "project-a"
`)
	writeTestXCF(t, dir, "b.xcf", `kind: project
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

	writeTestXCF(t, dir, "global.xcf", `kind: project
version: "1.0"
name: "happy-path-project"
`)
	writeTestXCF(t, dir, "agents_front.xcf", `kind: global
version: "1.0"
agents:
  frontend:
    description: "Frontend developer"
`)
	writeTestXCF(t, dir, "agents_back.xcf", `kind: global
version: "1.0"
agents:
  backend:
    description: "Backend developer"
`)
	writeTestXCF(t, dir, "skills/utils.xcf", `kind: global
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

	writeTestXCF(t, dir, "project.xcf", `kind: project
version: "1.0"
name: "settings-merge-test"
`)
	writeTestXCF(t, dir, "project-settings.xcf", `kind: settings
version: "1.0"
model: "sonnet-4"
`)
	writeTestXCF(t, dir, "settings.xcf", `kind: settings
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

	writeTestXCF(t, dir, "a.xcf", `kind: project
version: "1.0"
name: "conflict-test"
`)
	writeTestXCF(t, dir, "a-settings.xcf", `kind: settings
version: "1.0"
model: "sonnet-4"
`)
	writeTestXCF(t, dir, "b.xcf", `kind: settings
version: "1.0"
model: "opus-4"
`)

	_, err := ParseDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model")
}

func TestParseDirectory_LocalDeepMerge_NonConflicting(t *testing.T) {
	dir := t.TempDir()

	writeTestXCF(t, dir, "project.xcf", `kind: project
version: "1.0"
name: "local-merge-test"
local:
  env:
    SECRET: "abc"
`)
	writeTestXCF(t, dir, "local-overrides.xcf", `kind: project
version: "1.0"
name: "local-merge-test"
local:
  effortLevel: "low"
`)

	cfg, err := ParseDirectory(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg.Project)
	assert.Equal(t, "low", cfg.Project.Local.EffortLevel)
	assert.Equal(t, "abc", cfg.Project.Local.Env["SECRET"])
}

// TestParse_Profile_IsParseableFile verifies that a kind: profile document
// is accepted by the parser without error (profile routing is handled separately).
func TestParse_Profile_IsParseableFile(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "project.xcf")
	require.NoError(t, os.WriteFile(proj, []byte("kind: project\nname: myproject\nversion: \"1.0\"\n"), 0600))
	pf := filepath.Join(dir, "backend.xcf")
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
	writeTestXCF(t, globalDir, "global.xcf", `kind: global
version: "1.0"
settings:
  model: "sonnet-4"
  effortLevel: "high"
  env:
    GLOBAL_KEY: "from-global"
`)

	// Create project config that extends global
	projectDir := t.TempDir()
	writeTestXCF(t, projectDir, "project.xcf", `kind: global
version: "1.0"
extends: global
settings:
  effortLevel: "low"
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
