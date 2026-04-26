package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDirContents_SkipsMEMORYmd(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Write a regular memory file and a MEMORY.md index.
	require.NoError(t, os.WriteFile(filepath.Join(src, "note.md"), []byte("content"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(src, "MEMORY.md"), []byte("# index"), 0600))

	require.NoError(t, copyDirContents(src, dst))

	// note.md must be copied.
	_, err := os.Stat(filepath.Join(dst, "note.md"))
	assert.NoError(t, err, "note.md should have been copied")

	// MEMORY.md must NOT be copied.
	_, err = os.Stat(filepath.Join(dst, "MEMORY.md"))
	assert.True(t, os.IsNotExist(err), "MEMORY.md should have been skipped")
}

func TestImportScope_XcfDirAlreadyExists(t *testing.T) {
	tmp := t.TempDir()

	// Create xcf/ directory inside the temp dir
	if err := os.MkdirAll(filepath.Join(tmp, "xcf"), 0755); err != nil {
		t.Fatalf("failed to create xcf/ dir: %v", err)
	}

	// Create .claude/agents/ with a dummy agent file so importScope has content to scan
	agentsDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/agents/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "test-agent.md"), []byte("# Test Agent\n"), 0600); err != nil {
		t.Fatalf("failed to write dummy agent: %v", err)
	}

	// Change into the temp dir
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir to tmp: %v", err)
	}

	err = importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude")
	if err == nil {
		t.Fatal("expected error when xcf/ directory already exists, got nil")
	}
	if !strings.Contains(err.Error(), "xcf/ directory already exists") {
		t.Errorf("expected error to contain %q, got: %v", "xcf/ directory already exists", err)
	}
}

// TestExtractBodyAfterFrontmatter verifies the helper function handles all edge cases.
func TestExtractBodyAfterFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with frontmatter and body",
			input:    "---\nname: dev\n---\n\nDev instructions here",
			expected: "Dev instructions here",
		},
		{
			name:     "no frontmatter returns full content",
			input:    "# Plain markdown\n\nSome content",
			expected: "# Plain markdown\n\nSome content",
		},
		{
			name:     "frontmatter with empty body",
			input:    "---\nname: dev\n---\n",
			expected: "",
		},
		{
			name:     "frontmatter with leading newline in body",
			input:    "---\nname: dev\n---\n\n# Header\n\nBody content",
			expected: "# Header\n\nBody content",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractBodyAfterFrontmatter([]byte(tc.input))
			if got != tc.expected {
				t.Errorf("extractBodyAfterFrontmatter(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestImportScope_Messaging_NoReferencedInPlace(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()

	// Create .claude/agents/dev.md with minimal content
	agentsDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/agents/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "dev.md"), []byte("# Dev Agent\n"), 0600); err != nil {
		t.Fatalf("failed to write dev.md: %v", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir to tmp: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	importErr := importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude")

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if importErr != nil {
		t.Fatalf("importScope returned error: %v", importErr)
	}

	if strings.Contains(output, "referenced in-place") {
		t.Errorf("output should not contain 'referenced in-place', got: %q", output)
	}
}

func TestImport_RoundTrip_SplitFiles(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir to tmp: %v", err)
	}

	// Create .claude/agents/dev.md
	agentsDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/agents/: %v", err)
	}
	devContent := "---\nname: dev\ndescription: Development agent\nmodel: claude-sonnet-4-5\ntools:\n  - Read\n  - Edit\n---\n\nYou are the dev agent. Write clean, well-tested code."
	if err := os.WriteFile(filepath.Join(agentsDir, "dev.md"), []byte(devContent), 0600); err != nil {
		t.Fatalf("failed to write dev.md: %v", err)
	}

	// Create .claude/agents/reviewer.md
	reviewerContent := "---\nname: reviewer\ndescription: Code review agent\nmodel: claude-opus-4-5\n---\n\nYou are the reviewer agent. Perform thorough code reviews."
	if err := os.WriteFile(filepath.Join(agentsDir, "reviewer.md"), []byte(reviewerContent), 0600); err != nil {
		t.Fatalf("failed to write reviewer.md: %v", err)
	}

	// Create .claude/skills/tdd/SKILL.md
	skillDir := filepath.Join(tmp, ".claude", "skills", "tdd")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	skillContent := "---\nname: tdd\ndescription: Test-driven development skill\n---\n\n# TDD Skill\n\nWrite tests first, then implementation."
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0600); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create .claude/skills/tdd/references/patterns.md (should still be copied)
	refsDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		t.Fatalf("failed to create references dir: %v", err)
	}
	refContent := "# TDD Patterns\n\nRed-Green-Refactor cycle."
	if err := os.WriteFile(filepath.Join(refsDir, "patterns.md"), []byte(refContent), 0600); err != nil {
		t.Fatalf("failed to write patterns.md: %v", err)
	}

	// Create .claude/rules/security.md
	rulesDir := filepath.Join(tmp, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/rules/: %v", err)
	}
	ruleContent := "---\nname: security\ndescription: Security rules\n---\n\nNever expose secrets. Always validate input."
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), []byte(ruleContent), 0600); err != nil {
		t.Fatalf("failed to write security.md: %v", err)
	}

	// Create .claude/workflows/deploy.md
	workflowsDir := filepath.Join(tmp, ".claude", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/workflows/: %v", err)
	}
	workflowContent := "---\nname: deploy\ndescription: Deployment workflow\n---\n\n# Deploy\n\nRun pre-flight checks, then deploy."
	if err := os.WriteFile(filepath.Join(workflowsDir, "deploy.md"), []byte(workflowContent), 0600); err != nil {
		t.Fatalf("failed to write deploy.md: %v", err)
	}

	// Create .claude/settings.json
	if err := os.WriteFile(filepath.Join(tmp, ".claude", "settings.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to write settings.json: %v", err)
	}

	// Run importScope
	if err := importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude"); err != nil {
		t.Fatalf("importScope returned unexpected error: %v", err)
	}

	// project.xcf must exist
	if _, err := os.Stat(filepath.Join(tmp, filepath.Join(".xcaffold", "project.xcf"))); err != nil {
		t.Fatalf("project.xcf was not created: %v", err)
	}

	// Read project.xcf — must contain kind: project
	scaffoldData, err := os.ReadFile(filepath.Join(tmp, filepath.Join(".xcaffold", "project.xcf")))
	require.NoError(t, err)
	scaffoldStr := string(scaffoldData)
	assert.Contains(t, scaffoldStr, "kind: project", "project.xcf must use kind: project (split-file format)")

	// Split .xcf files must exist for each resource
	// Agents live in their own subdirectory: xcf/agents/<id>/<id>.xcf
	expectedXcfFiles := []string{
		filepath.Join(tmp, "xcf", "agents", "dev", "dev.xcf"),
		filepath.Join(tmp, "xcf", "agents", "reviewer", "reviewer.xcf"),
		filepath.Join(tmp, "xcf", "skills", "tdd.xcf"),
		filepath.Join(tmp, "xcf", "rules", "security.xcf"),
		filepath.Join(tmp, "xcf", "workflows", "deploy.xcf"),
	}
	for _, f := range expectedXcfFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected split xcf file %q to exist: %v", f, err)
		}
	}

	// Skill reference file must still be copied
	xcfRefPath := filepath.Join(tmp, "xcf", "skills", "tdd", "references", "patterns.md")
	if _, err := os.Stat(xcfRefPath); err != nil {
		t.Errorf("expected skill reference file to be copied to %q: %v", xcfRefPath, err)
	}

	// .md files must NOT be copied (inline mode — no instructions-file references)
	unexpectedMdFiles := []string{
		filepath.Join(tmp, "xcf", "agents", "dev", "dev.md"),
		filepath.Join(tmp, "xcf", "agents", "reviewer", "reviewer.md"),
		filepath.Join(tmp, "xcf", "rules", "security.md"),
		filepath.Join(tmp, "xcf", "workflows", "deploy.md"),
	}
	for _, f := range unexpectedMdFiles {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("file %q should NOT exist (instructions are inlined, not copied)", f)
		}
	}

	// Agent .xcf files must use frontmatter format for inline instructions, not instructions-file.
	// Instructions content moves into the markdown body (after the closing ---), not as a YAML field.
	devXcf, err := os.ReadFile(filepath.Join(tmp, "xcf", "agents", "dev", "dev.xcf"))
	require.NoError(t, err)
	devXcfStr := string(devXcf)
	assert.Contains(t, devXcfStr, "kind: agent")
	assert.True(t, strings.HasPrefix(devXcfStr, "---\n"), "agent xcf must use frontmatter format for inline instructions")
	assert.NotContains(t, devXcfStr, "instructions:", "instructions must be in the markdown body, not as a YAML field")
	assert.NotContains(t, devXcfStr, "instructions-file:", "agent xcf must not use instructions-file")
	assert.Contains(t, devXcfStr, "Write clean, well-tested code", "agent xcf must contain body text")
}

func TestMergeImportDirs_XcfDirAlreadyExists(t *testing.T) {
	tmp := t.TempDir()

	// Create xcf/ directory inside the temp dir
	if err := os.MkdirAll(filepath.Join(tmp, "xcf"), 0755); err != nil {
		t.Fatalf("failed to create xcf/ dir: %v", err)
	}

	// Create .claude/agents/ with a dummy agent file
	agentsDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/agents/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "test-agent.md"), []byte("# Test Agent\n"), 0600); err != nil {
		t.Fatalf("failed to write dummy agent: %v", err)
	}

	// Change into the temp dir
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(origCwd) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir to tmp: %v", err)
	}

	err = mergeImportDirs([]platformDirInfo{{dirName: ".claude", platform: "claude", exists: true}}, filepath.Join(".xcaffold", "project.xcf"))
	if err == nil {
		t.Fatal("expected error when xcf/ directory already exists, got nil")
	}
	if !strings.Contains(err.Error(), "xcf/ directory already exists") {
		t.Errorf("expected error to contain %q, got: %v", "xcf/ directory already exists", err)
	}
}

func TestImportScope_EmitsSplitFileFormat(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	dir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(dir))

	// Create .claude/ with an agent and a skill
	require.NoError(t, os.MkdirAll(".claude/agents", 0755))
	require.NoError(t, os.MkdirAll(".claude/skills/tdd", 0755))
	require.NoError(t, os.WriteFile(".claude/agents/dev.md",
		[]byte("---\nname: dev\ndescription: Dev agent\nmodel: sonnet\n---\n\nDev instructions"), 0644))
	require.NoError(t, os.WriteFile(".claude/skills/tdd/SKILL.md",
		[]byte("---\nname: tdd\ndescription: TDD\n---\n\nTDD instructions"), 0644))

	err = importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude")
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(".xcaffold", "project.xcf"))
	require.NoError(t, err)

	s := string(content)

	// project.xcf must use kind: project (split-file format, not multi-kind)
	assert.Contains(t, s, "kind: project")

	// Must NOT contain multi-kind documents inline in project.xcf
	// (they are split into xcf/agents/<id>/<id>.xcf, xcf/skills/*.xcf, etc.)
	assert.NotContains(t, s, "kind: agent", "agent must be in xcf/agents/dev/dev.xcf, not project.xcf")
	assert.NotContains(t, s, "kind: skill", "skill must be in xcf/skills/tdd.xcf, not project.xcf")

	// Split files must exist — agents live in their own subdirectory
	assert.FileExists(t, filepath.Join(dir, "xcf", "agents", "dev", "dev.xcf"))
	assert.FileExists(t, filepath.Join(dir, "xcf", "skills", "tdd.xcf"))

	// Agent split file must use frontmatter format for inline instructions.
	// Instructions content moves into the markdown body, not as a YAML field.
	devXcf, err := os.ReadFile(filepath.Join(dir, "xcf", "agents", "dev", "dev.xcf"))
	require.NoError(t, err)
	devXcfContent := string(devXcf)
	assert.True(t, strings.HasPrefix(devXcfContent, "---\n"), "agent xcf must use frontmatter format")
	assert.NotContains(t, devXcfContent, "instructions:", "instructions must be in the markdown body, not as a YAML field")
	assert.Contains(t, devXcfContent, "Dev instructions")
	assert.NotContains(t, devXcfContent, "instructions-file:")
}

func TestDetectPlatformDirs_Empty(t *testing.T) {
	// Point to a temp dir with no provider directories
	tmp := t.TempDir()

	dirs := detectPlatformDirs(tmp, true)
	if len(dirs) != 0 {
		t.Errorf("expected 0 dirs when no provider dirs exist, got %d: %v", len(dirs), dirs)
	}
}

func TestDetectPlatformDirs_SingleProvider(t *testing.T) {
	tmp := t.TempDir()

	// Create <tmp>/.claude/agents/dev.md and <tmp>/.claude/rules/sec.md
	claudeAgents := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(claudeAgents, 0755); err != nil {
		t.Fatalf("failed to create .claude/agents/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeAgents, "dev.md"), []byte("# Dev\n"), 0600); err != nil {
		t.Fatalf("failed to write dev.md: %v", err)
	}
	claudeRules := filepath.Join(tmp, ".claude", "rules")
	if err := os.MkdirAll(claudeRules, 0755); err != nil {
		t.Fatalf("failed to create .claude/rules/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeRules, "sec.md"), []byte("# Sec\n"), 0600); err != nil {
		t.Fatalf("failed to write sec.md: %v", err)
	}

	dirs := detectPlatformDirs(tmp, true)
	if len(dirs) != 1 {
		t.Fatalf("expected 1 dir, got %d: %v", len(dirs), dirs)
	}
	if dirs[0].platform != "claude" {
		t.Errorf("expected platform %q, got %q", "claude", dirs[0].platform)
	}
	if dirs[0].agents != 1 {
		t.Errorf("expected 1 agent, got %d", dirs[0].agents)
	}
	if dirs[0].rules != 1 {
		t.Errorf("expected 1 rule, got %d", dirs[0].rules)
	}
	// dirName must be the absolute path to <tmp>/.claude
	expected := filepath.Join(tmp, ".claude")
	if dirs[0].dirName != expected {
		t.Errorf("expected dirName %q, got %q", expected, dirs[0].dirName)
	}
}

func TestDetectPlatformDirs_MultiProvider_SortedBySize(t *testing.T) {
	tmp := t.TempDir()

	// <tmp>/.claude — 1 agent
	claudeAgents := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(claudeAgents, 0755); err != nil {
		t.Fatalf("failed to create .claude/agents/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeAgents, "dev.md"), []byte("# Dev\n"), 0600); err != nil {
		t.Fatalf("failed to write dev.md: %v", err)
	}

	// <tmp>/.cursor — 2 rules (richer)
	cursorRules := filepath.Join(tmp, ".cursor", "rules")
	if err := os.MkdirAll(cursorRules, 0755); err != nil {
		t.Fatalf("failed to create .cursor/rules/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cursorRules, "r1.mdc"), []byte("rule1"), 0600); err != nil {
		t.Fatalf("failed to write r1.mdc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cursorRules, "r2.mdc"), []byte("rule2"), 0600); err != nil {
		t.Fatalf("failed to write r2.mdc: %v", err)
	}

	dirs := detectPlatformDirs(tmp, true)
	if len(dirs) != 2 {
		t.Fatalf("expected 2 dirs, got %d: %v", len(dirs), dirs)
	}
	// cursor has 2 rules vs claude's 1 agent — cursor must be first
	if dirs[0].platform != "cursor" {
		t.Errorf("expected richest provider first (cursor), got %q", dirs[0].platform)
	}
	if dirs[1].platform != "claude" {
		t.Errorf("expected second provider to be claude, got %q", dirs[1].platform)
	}
}

func TestDetectPlatformDirs_SkipEmpty_False_IncludesEmptyDirs(t *testing.T) {
	tmp := t.TempDir()

	// Create <tmp>/.claude with no resources (directory exists but empty)
	if err := os.MkdirAll(filepath.Join(tmp, ".claude"), 0755); err != nil {
		t.Fatalf("failed to create .claude/: %v", err)
	}

	// skipEmpty=false must include the empty dir
	dirs := detectPlatformDirs(tmp, false)
	if len(dirs) != 1 {
		t.Fatalf("expected 1 dir with skipEmpty=false, got %d", len(dirs))
	}
	if dirs[0].platform != "claude" {
		t.Errorf("expected platform claude, got %q", dirs[0].platform)
	}

	// skipEmpty=true must exclude the empty dir
	dirs = detectPlatformDirs(tmp, true)
	if len(dirs) != 0 {
		t.Errorf("expected 0 dirs with skipEmpty=true, got %d", len(dirs))
	}
}

func TestImportCmd_WithMemoryFlag_Registered(t *testing.T) {
	flag := importCmd.Flags().Lookup("with-memory")
	require.NotNil(t, flag, "--with-memory flag must be registered on importCmd")
	require.Equal(t, "false", flag.DefValue)
}

func TestRunImport_WithMemory_UsesSourceDir(t *testing.T) {
	memDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(memDir, "user-role.md"),
		[]byte("---\ntype: user\n---\nRobert."),
		0o600,
	))

	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)

	summary, err := runMemorySnapshot(importCmd, memDir, "claude", false)
	require.NoError(t, err)
	require.Equal(t, 1, summary.Imported)
	require.FileExists(t, filepath.Join(tmp, "xcf", "agents", "user-role.md"))
}

func TestImport_WithMemory_Gemini_ExtractsBlocks(t *testing.T) {
	// Prepare a GEMINI.md file with an xcaffold-seeded memory block.
	geminiDir := t.TempDir()
	geminiMD := `## Gemini Added Memories

<!-- xcaffold:memory name="user-role" type="user" seeded-at="2026-04-15T00:00:00Z" -->
**user-role** (user): Developer role.

Robert is the founder.
<!-- xcaffold:/memory -->
`
	require.NoError(t, os.WriteFile(filepath.Join(geminiDir, "GEMINI.md"), []byte(geminiMD), 0o600))
	t.Setenv("XCAFFOLD_GEMINI_DIR", geminiDir)

	// Set up a temp working directory for the sidecar output.
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origWd)

	summary, err := runMemorySnapshot(importCmd, "", "gemini", false)
	require.NoError(t, err)
	require.Equal(t, 1, summary.Imported, "one Gemini memory block must be imported")
	require.FileExists(t, filepath.Join(tmp, "xcf", "agents", "user-role.md"))
}

func TestDetectTargets(t *testing.T) {
	tests := []struct {
		name     string
		dirs     []string
		expected []string
	}{
		{
			name:     "claude dir",
			dirs:     []string{".claude"},
			expected: []string{"claude"},
		},
		{
			name:     "agents dir",
			dirs:     []string{".agents"},
			expected: []string{"antigravity"},
		},
		{
			name:     "cursor dir",
			dirs:     []string{".cursor"},
			expected: []string{"cursor"},
		},
		{
			name:     "multiple dirs sorted",
			dirs:     []string{".claude", ".agents"},
			expected: []string{"antigravity", "claude"},
		},
		{
			name:     "unknown dir ignored",
			dirs:     []string{".unknown"},
			expected: []string{},
		},
		{
			name:     "empty",
			dirs:     []string{},
			expected: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectTargets(tc.dirs...)
			if len(got) != len(tc.expected) {
				t.Errorf("detectTargets(%v) = %v, want %v", tc.dirs, got, tc.expected)
				return
			}
			for i, v := range got {
				if v != tc.expected[i] {
					t.Errorf("detectTargets(%v)[%d] = %q, want %q", tc.dirs, i, v, tc.expected[i])
				}
			}
		})
	}
}

func TestExtractProjectInstructions_ClaudeRoot(t *testing.T) {
	tmp := t.TempDir()
	// Create a mock Claude project tree with a root CLAUDE.md.
	claudeMd := filepath.Join(tmp, "CLAUDE.md")
	require.NoError(t, os.WriteFile(claudeMd, []byte("Use pnpm. PostgreSQL 16."), 0o600))

	cfg := &ast.XcaffoldConfig{}
	require.NoError(t, extractProjectInstructions(tmp, "claude", cfg))

	require.Equal(t, "xcf/instructions/root.xcf", cfg.Project.InstructionsFile)
	sidecar := filepath.Join(tmp, "xcf", "instructions", "root.xcf")
	_, err := os.Stat(sidecar)
	require.NoError(t, err, "root sidecar must exist at %s", sidecar)
}

func TestExtractProjectInstructions_ClaudeNestedScopes(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "CLAUDE.md"), []byte("Root context."), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "packages", "worker"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "packages", "worker", "CLAUDE.md"), []byte("Worker context."), 0o600))

	cfg := &ast.XcaffoldConfig{}
	require.NoError(t, extractProjectInstructions(tmp, "claude", cfg))

	require.Len(t, cfg.Project.InstructionsScopes, 1)
	scope := cfg.Project.InstructionsScopes[0]
	require.Equal(t, "packages/worker", scope.Path)
	require.Equal(t, "concat", scope.MergeStrategy)
	require.Equal(t, "claude", scope.SourceProvider)
	// instructions-file must NOT point at the original CLAUDE.md.
	require.NotEqual(t, "packages/worker/CLAUDE.md", scope.InstructionsFile)
	require.Contains(t, scope.InstructionsFile, "xcf/instructions/scopes/")
}

func TestExtractProjectInstructions_CopilotFlatMode(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".github"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, ".github", "copilot-instructions.md"),
		[]byte("Copilot flat instructions."),
		0o600,
	))

	cfg := &ast.XcaffoldConfig{}
	require.NoError(t, extractProjectInstructions(tmp, "copilot", cfg))

	require.Equal(t, "xcf/instructions/root.xcf", cfg.Project.InstructionsFile)
	require.Empty(t, cfg.Project.InstructionsScopes, "flat Copilot mode must not create scope entries")
}

func TestExtractProjectInstructions_NeverSetsProviderFilename(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "CLAUDE.md"), []byte("Root."), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "src", "CLAUDE.md"), []byte("Src."), 0o600))

	cfg := &ast.XcaffoldConfig{}
	require.NoError(t, extractProjectInstructions(tmp, "claude", cfg))

	// instructions-file must never be set to a provider output filename.
	reservedNames := []string{"CLAUDE.md", "AGENTS.md", "GEMINI.md"}
	for _, scope := range cfg.Project.InstructionsScopes {
		for _, name := range reservedNames {
			require.NotContains(t, scope.InstructionsFile, name,
				"instructions-file must never point at provider output file %s", name)
		}
	}
}

func TestDetectDivergence_IdenticalContent_Collapsed(t *testing.T) {
	tmp := t.TempDir()
	// Same path, same content from two providers → single entry.
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "packages", "worker"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "worker", "CLAUDE.md"),
		[]byte("Worker context."), 0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "worker", "AGENTS.md"),
		[]byte("Worker context."), 0o600,
	))

	cfg := &ast.XcaffoldConfig{}
	require.NoError(t, extractProjectInstructions(tmp, "claude", cfg))
	require.NoError(t, detectAndMergeVariants(tmp, "cursor", cfg, false))

	require.Len(t, cfg.Project.InstructionsScopes, 1)
	require.Empty(t, cfg.Project.InstructionsScopes[0].Variants,
		"identical content must be collapsed to single entry")
}

func TestDetectDivergence_DifferentContent_VariantsSet(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "packages", "api"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "api", "CLAUDE.md"),
		[]byte("Claude API context — 42 lines of content here."), 0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "api", "AGENTS.md"),
		[]byte("Cursor API context — 31 lines here."), 0o600,
	))

	cfg := &ast.XcaffoldConfig{}
	require.NoError(t, extractProjectInstructions(tmp, "claude", cfg))
	require.NoError(t, detectAndMergeVariants(tmp, "cursor", cfg, false))

	require.Len(t, cfg.Project.InstructionsScopes, 1)
	scope := cfg.Project.InstructionsScopes[0]
	require.NotEmpty(t, scope.Variants, "divergent content must produce variant entries")
	require.Contains(t, scope.Variants, "claude")
	require.Contains(t, scope.Variants, "cursor")
	require.NotNil(t, scope.Reconciliation)
	require.Equal(t, "per-target", scope.Reconciliation.Strategy)
}

func TestDetectDivergence_AutoMergeUnion(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "packages", "api"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "api", "CLAUDE.md"),
		[]byte("Claude content."), 0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "api", "AGENTS.md"),
		[]byte("Cursor content."), 0o600,
	))

	cfg := &ast.XcaffoldConfig{}
	require.NoError(t, extractProjectInstructions(tmp, "claude", cfg))
	require.NoError(t, detectAndMergeVariants(tmp, "cursor", cfg, true /* autoMergeUnion */))

	require.Len(t, cfg.Project.InstructionsScopes, 1)
	scope := cfg.Project.InstructionsScopes[0]
	require.Empty(t, scope.Variants, "--auto-merge=union must clear variants map")
	require.NotNil(t, scope.Reconciliation)
	require.Equal(t, "union", scope.Reconciliation.Strategy)
}

func TestParseProvenanceMarkers_ReconstructsScopes(t *testing.T) {
	input := `Root content here.

<!-- xcaffold:scope path="packages/worker" merge="concat" origin="claude:CLAUDE.md" -->
Use BullMQ. Never call DB from worker code.
<!-- xcaffold:/scope -->

<!-- xcaffold:scope path="packages/api" merge="closest-wins" origin="cursor:AGENTS.md" -->
REST conventions only.
<!-- xcaffold:/scope -->
`
	scopes, rootContent, err := parseProvenanceMarkers(input)
	require.NoError(t, err)
	require.Equal(t, "Root content here.\n", rootContent)
	require.Len(t, scopes, 2)
	require.Equal(t, "packages/worker", scopes[0].Path)
	require.Equal(t, "concat", scopes[0].MergeStrategy)
	require.Equal(t, "claude", scopes[0].SourceProvider)
	require.Equal(t, "CLAUDE.md", scopes[0].SourceFilename)
	require.Contains(t, scopes[0].Instructions, "BullMQ")
	require.Equal(t, "packages/api", scopes[1].Path)
	require.Equal(t, "closest-wins", scopes[1].MergeStrategy)
}

func TestParseProvenanceMarkers_MalformedMarker_SkippedWithWarning(t *testing.T) {
	// Missing path attribute — treated as regular content, no error.
	input := `Root.

<!-- xcaffold:scope merge="concat" -->
Content without path.
<!-- xcaffold:/scope -->
`
	scopes, _, err := parseProvenanceMarkers(input)
	require.NoError(t, err)
	require.Empty(t, scopes, "malformed marker without path must be skipped")
}

func TestParseProvenanceMarkers_NoMarkers_ReturnsEmptyScopes(t *testing.T) {
	input := "Plain instructions with no markers.\n"
	scopes, rootContent, err := parseProvenanceMarkers(input)
	require.NoError(t, err)
	require.Empty(t, scopes)
	require.Equal(t, input, rootContent)
}

// TestDetectDivergence_ExistingSidecarMissing_ReturnsError verifies that
// detectAndMergeVariants surfaces a read error when the existing sidecar
// path on disk does not exist, rather than silently treating it as empty.
func TestDetectDivergence_ExistingSidecarMissing_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "packages", "api"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "api", "CLAUDE.md"),
		[]byte("Claude API context."), 0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "api", "AGENTS.md"),
		[]byte("Cursor API context."), 0o600,
	))

	cfg := &ast.XcaffoldConfig{}
	require.NoError(t, extractProjectInstructions(tmp, "claude", cfg))

	// Sabotage: remove the existing sidecar so the read will fail.
	require.Len(t, cfg.Project.InstructionsScopes, 1)
	sidecarPath := filepath.Join(tmp, cfg.Project.InstructionsScopes[0].InstructionsFile)
	require.NoError(t, os.Remove(sidecarPath))

	err := detectAndMergeVariants(tmp, "cursor", cfg, false)
	require.Error(t, err, "detectAndMergeVariants must return an error when existing sidecar is missing")
	require.Contains(t, err.Error(), "read existing sidecar")
}

// TestParseProvenanceMarkers_CRLFLineEndings verifies that the close marker
// is recognised even when the file uses CRLF line endings.
func TestParseProvenanceMarkers_CRLFLineEndings(t *testing.T) {
	// Construct the input with CRLF line endings.
	input := "Root content.\r\n" +
		"<!-- xcaffold:scope path=\"packages/worker\" merge=\"concat\" origin=\"claude:CLAUDE.md\" -->\r\n" +
		"Worker instructions.\r\n" +
		"<!-- xcaffold:/scope -->\r\n"

	scopes, _, err := parseProvenanceMarkers(input)
	require.NoError(t, err)
	require.Len(t, scopes, 1, "CRLF close marker must be recognised")
	require.Equal(t, "packages/worker", scopes[0].Path)
	require.Contains(t, scopes[0].Instructions, "Worker instructions.")
}

// TestImport_ClaudeAndCursor_DetectsDivergence creates a temp project with both
// CLAUDE.md and AGENTS.md (different content) and verifies that after the import
// flow the resulting config has Variants populated on the overlapping scope.
func TestImport_ClaudeAndCursor_DetectsDivergence(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(tmp))

	// Root CLAUDE.md and AGENTS.md (different content for divergence detection).
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "CLAUDE.md"), []byte("Claude root instructions."), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("Cursor root instructions."), 0o600))

	// Nested scopes with differing content.
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "packages", "api"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "api", "CLAUDE.md"),
		[]byte("Claude API context — unique content."), 0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "packages", "api", "AGENTS.md"),
		[]byte("Cursor API context — different content."), 0o600,
	))

	// Simulate the importScope+runProjectInstructionsDiscovery sequence.
	// We call extractProjectInstructions directly and then detectAndMergeVariants
	// to exercise the wiring logic without touching the importScope preamble guards.
	cfg := &ast.XcaffoldConfig{
		Version: "1.0",
		Project: &ast.ProjectConfig{Name: "test-project"},
		ResourceScope: ast.ResourceScope{
			Agents: make(map[string]ast.AgentConfig),
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
			MCP:    make(map[string]ast.MCPConfig),
		},
	}

	require.NoError(t, extractProjectInstructions(tmp, "claude", cfg))
	require.NoError(t, detectAndMergeVariants(tmp, "cursor", cfg, false))

	require.Len(t, cfg.Project.InstructionsScopes, 1, "should have one overlapping scope")
	scope := cfg.Project.InstructionsScopes[0]
	require.NotEmpty(t, scope.Variants, "divergent CLAUDE.md vs AGENTS.md must populate Variants")
	require.Contains(t, scope.Variants, "claude")
	require.Contains(t, scope.Variants, "cursor")
}

// TestImport_FlatFileWithProvenanceMarkers_ReconstructsScopes verifies that a
// previously flattened instructions file containing xcaffold:scope markers is
// parsed back into discrete scope entries by runProjectInstructionsDiscovery.
func TestImport_FlatFileWithProvenanceMarkers_ReconstructsScopes(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(tmp))

	// Flat CLAUDE.md with embedded provenance markers (as rendered by xcaffold).
	flat := `Root context.

<!-- xcaffold:scope path="packages/worker" merge="concat" origin="claude:CLAUDE.md" -->
Use BullMQ. Never call DB from worker code.
<!-- xcaffold:/scope -->

<!-- xcaffold:scope path="packages/api" merge="closest-wins" origin="cursor:AGENTS.md" -->
REST conventions only.
<!-- xcaffold:/scope -->
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "CLAUDE.md"), []byte(flat), 0o600))

	// Create a minimal .claude/ structure so importScope succeeds.
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude", "agents"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, ".claude", "agents", "dev.md"),
		[]byte("---\nname: dev\ndescription: Dev agent\n---\n\nDev instructions."),
		0o644,
	))

	// Run importScope to write project.xcf.
	require.NoError(t, importScope(".claude", filepath.Join(".xcaffold", "project.xcf"), "project", "claude"))

	// Now run the project instructions discovery which should detect markers.
	require.NoError(t, runProjectInstructionsDiscovery(tmp, "claude", filepath.Join(".xcaffold", "project.xcf")))

	// Read the updated project.xcf raw content — avoid parser.ParseFile which
	// merges global config and would pick up the user's real ~/.xcaffold/project.xcf.
	xcfData, err := os.ReadFile(filepath.Join(tmp, ".xcaffold", "project.xcf"))
	require.NoError(t, err)
	xcfStr := string(xcfData)

	// Scopes should have been reconstructed from the markers and serialised
	// back into project.xcf as instruction-scopes entries.
	require.Contains(t, xcfStr, "packages/worker",
		"xcf must contain the reconstructed packages/worker scope")
	require.Contains(t, xcfStr, "packages/api",
		"xcf must contain the reconstructed packages/api scope")
}

// TestImport_FromGemini_ExtractsInstructions verifies that GEMINI.md is
// discovered by extractProjectInstructions and written to a sidecar.
func TestImport_FromGemini_ExtractsInstructions(t *testing.T) {
	tmp := t.TempDir()

	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "GEMINI.md"),
		[]byte("Use Go 1.24. Never panic in library code."),
		0o600,
	))

	cfg := &ast.XcaffoldConfig{}
	require.NoError(t, extractProjectInstructions(tmp, "gemini", cfg))

	require.Equal(t, "xcf/instructions/root.xcf", cfg.Project.InstructionsFile)
	sidecar := filepath.Join(tmp, "xcf", "instructions", "root.xcf")
	data, err := os.ReadFile(sidecar)
	require.NoError(t, err)
	require.Contains(t, string(data), "Use Go 1.24.")
}

// TestResolveSourceFiles_WalksNestedDirs verifies that resolveSourceFiles recurses
// into subdirectories when the source is a directory (fixing the --source .claude/rules/
// regression where nested .md files were silently dropped).
func TestResolveSourceFiles_WalksNestedDirs(t *testing.T) {
	tmp := t.TempDir()

	// Flat file at root of the directory.
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "root.md"), []byte("# root"), 0o600))

	// File one level deep.
	sub := filepath.Join(tmp, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "nested.md"), []byte("# nested"), 0o600))

	// File two levels deep.
	deep := filepath.Join(tmp, "sub", "deep")
	require.NoError(t, os.MkdirAll(deep, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "very-nested.md"), []byte("# very nested"), 0o600))

	// Non-.md file must be excluded.
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "ignore.txt"), []byte("not a doc"), 0o600))

	files, err := resolveSourceFiles(tmp)
	require.NoError(t, err)

	// All three .md files must be discovered regardless of depth.
	assert.Len(t, files, 3, "resolveSourceFiles must walk nested subdirectories")

	// Results must be sorted (deterministic).
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = filepath.Base(f)
	}
	assert.Contains(t, names, "root.md")
	assert.Contains(t, names, "nested.md")
	assert.Contains(t, names, "very-nested.md")
}

func TestExtractSkillSubdirs_AntigravityResources(t *testing.T) {
	// Create a fake Antigravity skill directory with resources/ and examples/
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "resources"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "resources", "TEMPLATE.md"), []byte("# Template"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "examples"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "examples", "sample.md"), []byte("# Sample"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: my-skill\n---\nInstructions"), 0o644))

	outDir := t.TempDir()
	var warnings []string

	refs, scripts, assets, examples, err := extractSkillSubdirs(
		filepath.Join(skillDir, "SKILL.md"), "my-skill", "antigravity", outDir, &warnings,
	)
	require.NoError(t, err)

	// refs and scripts should be empty for this fixture
	_ = refs
	_ = scripts

	// Antigravity resources/ → canonical assets/
	if len(assets) == 0 {
		t.Error("expected resources/ to map to assets/, got empty")
	}
	// Antigravity examples/ → canonical examples/
	if len(examples) == 0 {
		t.Error("expected examples/ to map to examples/, got empty")
	}

	// Verify files were copied to canonical locations
	expectedAsset := filepath.Join(outDir, "xcf", "skills", "my-skill", "assets", "TEMPLATE.md")
	if _, err := os.Stat(expectedAsset); os.IsNotExist(err) {
		t.Errorf("expected asset copied to %s", expectedAsset)
	}

	expectedExample := filepath.Join(outDir, "xcf", "skills", "my-skill", "examples", "sample.md")
	if _, err := os.Stat(expectedExample); os.IsNotExist(err) {
		t.Errorf("expected example copied to %s", expectedExample)
	}
}

func TestExtractSkillSubdirs_ClaudeFlatMdFiles(t *testing.T) {
	// Claude flat .md files alongside SKILL.md should be treated as references.
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: my-skill\n---\nInstructions"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "helper.md"), []byte("# Helper"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "guide.md"), []byte("# Guide"), 0o644))

	outDir := t.TempDir()
	var warnings []string

	refs, scripts, assets, examples, err := extractSkillSubdirs(
		filepath.Join(skillDir, "SKILL.md"), "my-skill", "claude", outDir, &warnings,
	)
	require.NoError(t, err)

	_ = scripts
	_ = assets
	_ = examples

	if len(refs) != 2 {
		t.Errorf("expected 2 refs for flat .md files, got %d: %v", len(refs), refs)
	}

	// Verify files were copied to references/
	for _, name := range []string{"helper.md", "guide.md"} {
		dest := filepath.Join(outDir, "xcf", "skills", "my-skill", "references", name)
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			t.Errorf("expected reference copied to %s", dest)
		}
	}

	if len(warnings) != 0 {
		t.Errorf("expected no warnings for claude provider, got: %v", warnings)
	}
}

func TestExtractSkillSubdirs_UnknownProviderPassthrough(t *testing.T) {
	// Unknown providers should route all subdir files to passthrough and emit a warning.
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "extras"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "extras", "data.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: my-skill\n---\nInstructions"), 0o644))

	outDir := t.TempDir()
	var warnings []string

	refs, scripts, assets, examples, err := extractSkillSubdirs(
		filepath.Join(skillDir, "SKILL.md"), "my-skill", "unknown-provider", outDir, &warnings,
	)
	require.NoError(t, err)

	// All canonical slices should be empty — unknown provider has no canonical mapping.
	if len(refs)+len(scripts)+len(assets)+len(examples) != 0 {
		t.Errorf("expected empty canonical slices for unknown provider, got refs=%v scripts=%v assets=%v examples=%v",
			refs, scripts, assets, examples)
	}

	// File should appear in the passthrough directory.
	passthroughFile := filepath.Join(outDir, "xcf", "provider", "unknown-provider", "skills", "my-skill", "extras", "data.json")
	if _, err := os.Stat(passthroughFile); os.IsNotExist(err) {
		t.Errorf("expected passthrough file at %s", passthroughFile)
	}

	// A warning must have been emitted for the unknown provider.
	if len(warnings) == 0 {
		t.Error("expected a warning for unknown provider, got none")
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "unknown provider") && strings.Contains(w, "unknown-provider") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about unknown provider, got: %v", warnings)
	}
}

func TestWriteMemoryFiles_WritesMarkdownToDisk(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Memory: map[string]ast.MemoryConfig{
				"dev/context": {
					Name:        "context",
					Description: "Project context",
					Content:     "---\nname: context\ndescription: Project context\n---\n\nThis is memory content.",
					AgentRef:    "dev",
				},
			},
		},
	}

	n, err := writeMemoryFiles(config)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	data, err := os.ReadFile(filepath.Join("xcf", "agents", "dev", "memory", "context.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "This is memory content.")
}

func TestAllProviders_HaveRegisteredImporters(t *testing.T) {
	providers := []string{"claude", "cursor", "gemini", "copilot", "antigravity"}
	for _, p := range providers {
		imp := findImporterByProvider(p)
		require.NotNilf(t, imp, "provider %q must have a registered importer", p)
	}
}

func TestMergeImportDirs_DedupRicherWins(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	// Claude: dev agent with short instructions
	claudeDir := filepath.Join(tmp, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(claudeDir, "agents"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "agents", "dev.md"),
		[]byte("---\nname: dev\ndescription: Short\n---\n\nShort."),
		0o644,
	))

	// Cursor: dev agent with longer instructions
	cursorDir := filepath.Join(tmp, ".cursor")
	require.NoError(t, os.MkdirAll(filepath.Join(cursorDir, "agents"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(cursorDir, "agents", "dev.md"),
		[]byte("---\nname: dev\ndescription: Detailed developer\n---\n\nYou are a senior developer. Follow TDD strictly. Always write tests before implementation. Review code carefully."),
		0o644,
	))

	dirs := []platformDirInfo{
		{dirName: ".claude", platform: "claude", exists: true},
		{dirName: ".cursor", platform: "cursor", exists: true},
	}

	err := mergeImportDirs(dirs, filepath.Join(tmp, ".xcaffold", "project.xcf"))
	require.NoError(t, err)

	// The cursor version is longer — it should win
	config, parseErr := parser.ParseDirectory(".")
	require.NoError(t, parseErr)
	dev, ok := config.Agents["dev"]
	require.True(t, ok)
	require.Contains(t, dev.Instructions, "Follow TDD strictly")
}

func TestMergeImportDirs_ImportsHooksMCPSettings(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	// Claude: agents + settings.json with MCP + hooks
	claudeDir := filepath.Join(tmp, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(claudeDir, "agents"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "agents", "dev.md"),
		[]byte("---\nname: dev\ndescription: Developer\n---\n\nDevelop."),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"mcpServers":{"my-server":{"command":"node","args":["srv.js"]}}}`),
		0o644,
	))

	// Cursor: agents dir
	cursorDir := filepath.Join(tmp, ".cursor")
	require.NoError(t, os.MkdirAll(filepath.Join(cursorDir, "agents"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(cursorDir, "agents", "reviewer.md"),
		[]byte("---\nname: reviewer\n---\n\nReview."),
		0o644,
	))

	dirs := []platformDirInfo{
		{dirName: ".claude", platform: "claude", exists: true},
		{dirName: ".cursor", platform: "cursor", exists: true},
	}

	err := mergeImportDirs(dirs, filepath.Join(tmp, ".xcaffold", "project.xcf"))
	require.NoError(t, err)

	// MCP from .claude/ must be present
	config, parseErr := parser.ParseDirectory(".")
	require.NoError(t, parseErr)
	_, hasMCP := config.MCP["my-server"]
	require.True(t, hasMCP, "MCP server from .claude/ must be imported in multi-dir mode")
}

func TestMergeImportDirs_ImportsMemory(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	// Create .claude/ with agent + memory
	claudeDir := filepath.Join(tmp, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(claudeDir, "agents"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "agents", "dev.md"),
		[]byte("---\nname: dev\ndescription: Developer agent\n---\n\nYou are a developer."),
		0o644,
	))
	require.NoError(t, os.MkdirAll(filepath.Join(claudeDir, "agent-memory", "dev"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "agent-memory", "dev", "context.md"),
		[]byte("---\nname: context\ndescription: Project context\n---\n\nAlways use Go 1.24."),
		0o644,
	))

	// Create .cursor/ with an agent
	cursorDir := filepath.Join(tmp, ".cursor")
	require.NoError(t, os.MkdirAll(filepath.Join(cursorDir, "agents"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(cursorDir, "agents", "reviewer.md"),
		[]byte("---\nname: reviewer\ndescription: Code reviewer\n---\n\nReview code carefully."),
		0o644,
	))

	dirs := []platformDirInfo{
		{dirName: ".claude", platform: "claude", exists: true},
		{dirName: ".cursor", platform: "cursor", exists: true},
	}

	err := mergeImportDirs(dirs, filepath.Join(tmp, ".xcaffold", "project.xcf"))
	require.NoError(t, err)

	// Memory must be written to disk
	memPath := filepath.Join("xcf", "agents", "dev", "memory", "context.md")
	_, statErr := os.Stat(memPath)
	require.NoError(t, statErr, "memory file must exist at %s", memPath)

	data, _ := os.ReadFile(memPath)
	require.Contains(t, string(data), "Always use Go 1.24.")
}
