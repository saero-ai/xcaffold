package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	err = importScope(".claude", "scaffold.xcf", "project")
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

// TestExtractAgents_InlinesInstructions verifies agents use instructions: inline
// instead of copying .md files and setting instructions-file:.
func TestExtractAgents_InlinesInstructions(t *testing.T) {
	tmp := t.TempDir()

	// Create .claude/agents/dev.md
	agentsDir := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/agents/: %v", err)
	}
	agentContent := "---\nname: dev\ndescription: Dev agent\nmodel: sonnet\n---\n\nDev instructions here"
	if err := os.WriteFile(filepath.Join(agentsDir, "dev.md"), []byte(agentContent), 0600); err != nil {
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

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: make(map[string]ast.AgentConfig),
		},
	}
	count := 0
	var warnings []string

	if err := extractAgents(".claude", "project", config, &count, &warnings); err != nil {
		t.Fatalf("extractAgents returned error: %v", err)
	}

	agentCfg, ok := config.Agents["dev"]
	if !ok {
		t.Fatal("expected agent 'dev' to be in config.Agents")
	}

	// Instructions must be inlined — not an instructions-file reference
	if agentCfg.InstructionsFile != "" {
		t.Errorf("expected InstructionsFile to be empty (inlined), got %q", agentCfg.InstructionsFile)
	}

	// Body content must be embedded in Instructions
	if !strings.Contains(agentCfg.Instructions, "Dev instructions here") {
		t.Errorf("expected Instructions to contain body text, got %q", agentCfg.Instructions)
	}

	// The .md file must NOT be copied to xcf/agents/
	xcfPath := filepath.Join(tmp, "xcf", "agents", "dev.md")
	if _, err := os.Stat(xcfPath); err == nil {
		t.Errorf("xcf/agents/dev.md should NOT exist on disk (no file copy in inline mode)")
	}
}

func TestExtractRules_InlinesInstructions(t *testing.T) {
	tmp := t.TempDir()

	// Create .claude/rules/security.md
	rulesDir := filepath.Join(tmp, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/rules/: %v", err)
	}
	ruleContent := "---\nname: security\ndescription: Security rules\n---\n\nNever leak secrets."
	if err := os.WriteFile(filepath.Join(rulesDir, "security.md"), []byte(ruleContent), 0600); err != nil {
		t.Fatalf("failed to write security.md: %v", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir to tmp: %v", err)
	}

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Rules: make(map[string]ast.RuleConfig),
		},
	}
	count := 0
	var warnings []string

	if err := extractRules(".claude", "project", config, &count, &warnings); err != nil {
		t.Fatalf("extractRules returned error: %v", err)
	}

	ruleCfg, ok := config.Rules["security"]
	if !ok {
		t.Fatal("expected rule 'security' to be in config.Rules")
	}

	// Instructions must be inlined
	if ruleCfg.InstructionsFile != "" {
		t.Errorf("expected InstructionsFile to be empty (inlined), got %q", ruleCfg.InstructionsFile)
	}

	if !strings.Contains(ruleCfg.Instructions, "Never leak secrets.") {
		t.Errorf("expected Instructions to contain body text, got %q", ruleCfg.Instructions)
	}

	// The .md file must NOT be copied to xcf/rules/
	xcfPath := filepath.Join(tmp, "xcf", "rules", "security.md")
	if _, err := os.Stat(xcfPath); err == nil {
		t.Errorf("xcf/rules/security.md should NOT exist on disk (no file copy in inline mode)")
	}
}

func TestExtractSkills_InlinesInstructionsButCopiesRefs(t *testing.T) {
	tmp := t.TempDir()

	// Create .claude/skills/tdd/SKILL.md
	skillDir := filepath.Join(tmp, ".claude", "skills", "tdd")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	skillContent := "---\nname: tdd\ndescription: Test-driven development\n---\n\n# TDD Skill\n\nWrite tests first."
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0600); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	// Create .claude/skills/tdd/references/example.txt
	refsDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		t.Fatalf("failed to create references dir: %v", err)
	}
	refContent := "example reference content"
	if err := os.WriteFile(filepath.Join(refsDir, "example.txt"), []byte(refContent), 0600); err != nil {
		t.Fatalf("failed to write example.txt: %v", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir to tmp: %v", err)
	}

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Skills: make(map[string]ast.SkillConfig),
		},
	}
	count := 0
	var warnings []string

	if err := extractSkills(".claude", "project", config, &count, &warnings); err != nil {
		t.Fatalf("extractSkills returned error: %v", err)
	}

	skillCfg, ok := config.Skills["tdd"]
	if !ok {
		t.Fatal("expected skill 'tdd' to be in config.Skills")
	}

	// Instructions must be inlined — not an instructions-file reference
	if skillCfg.InstructionsFile != "" {
		t.Errorf("expected InstructionsFile to be empty (inlined), got %q", skillCfg.InstructionsFile)
	}

	if !strings.Contains(skillCfg.Instructions, "TDD Skill") {
		t.Errorf("expected Instructions to contain body text, got %q", skillCfg.Instructions)
	}

	// SKILL.md must NOT be copied to xcf/skills/tdd/SKILL.md
	xcfSkillPath := filepath.Join(tmp, "xcf", "skills", "tdd", "SKILL.md")
	if _, err := os.Stat(xcfSkillPath); err == nil {
		t.Errorf("xcf/skills/tdd/SKILL.md should NOT exist on disk (no file copy for SKILL.md)")
	}

	// references/example.txt MUST still be copied to xcf/skills/tdd/references/example.txt
	xcfRefPath := filepath.Join(tmp, "xcf", "skills", "tdd", "references", "example.txt")
	refData, err := os.ReadFile(xcfRefPath)
	if err != nil {
		t.Fatalf("expected xcf/skills/tdd/references/example.txt to exist on disk: %v", err)
	}
	if string(refData) != refContent {
		t.Errorf("xcf/skills/tdd/references/example.txt content mismatch: got %q, want %q", string(refData), refContent)
	}

	// References in config must point to xcf/ paths
	for _, ref := range skillCfg.References {
		if strings.HasPrefix(ref, ".claude/") {
			t.Errorf("reference %q should not start with .claude/", ref)
		}
		if !strings.HasPrefix(ref, "xcf/") {
			t.Errorf("reference %q should start with xcf/", ref)
		}
	}
}

func TestExtractWorkflows_InlinesInstructions(t *testing.T) {
	tmp := t.TempDir()

	// Create .claude/workflows/deploy.md
	workflowsDir := filepath.Join(tmp, ".claude", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/workflows/: %v", err)
	}
	workflowContent := "---\nname: deploy\ndescription: Deploy workflow\n---\n\n# Deploy\n\nRun deploy steps."
	if err := os.WriteFile(filepath.Join(workflowsDir, "deploy.md"), []byte(workflowContent), 0600); err != nil {
		t.Fatalf("failed to write deploy.md: %v", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("failed to chdir to tmp: %v", err)
	}

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Workflows: make(map[string]ast.WorkflowConfig),
		},
	}
	count := 0
	var warnings []string

	if err := extractWorkflows(".claude", "project", config, &count, &warnings); err != nil {
		t.Fatalf("extractWorkflows returned error: %v", err)
	}

	workflowCfg, ok := config.Workflows["deploy"]
	if !ok {
		t.Fatal("expected workflow 'deploy' to be in config.Workflows")
	}

	// Instructions must be inlined
	if workflowCfg.InstructionsFile != "" {
		t.Errorf("expected InstructionsFile to be empty (inlined), got %q", workflowCfg.InstructionsFile)
	}

	if !strings.Contains(workflowCfg.Instructions, "Run deploy steps.") {
		t.Errorf("expected Instructions to contain body text, got %q", workflowCfg.Instructions)
	}

	// The .md file must NOT be copied to xcf/workflows/
	xcfPath := filepath.Join(tmp, "xcf", "workflows", "deploy.md")
	if _, err := os.Stat(xcfPath); err == nil {
		t.Errorf("xcf/workflows/deploy.md should NOT exist on disk (no file copy in inline mode)")
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

	importErr := importScope(".claude", "scaffold.xcf", "project")

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
	if err := importScope(".claude", "scaffold.xcf", "project"); err != nil {
		t.Fatalf("importScope returned unexpected error: %v", err)
	}

	// scaffold.xcf must exist
	if _, err := os.Stat(filepath.Join(tmp, "scaffold.xcf")); err != nil {
		t.Fatalf("scaffold.xcf was not created: %v", err)
	}

	// Read scaffold.xcf — must contain kind: project
	scaffoldData, err := os.ReadFile(filepath.Join(tmp, "scaffold.xcf"))
	require.NoError(t, err)
	scaffoldStr := string(scaffoldData)
	assert.Contains(t, scaffoldStr, "kind: project", "scaffold.xcf must use kind: project (split-file format)")

	// Split .xcf files must exist for each resource
	expectedXcfFiles := []string{
		filepath.Join(tmp, "xcf", "agents", "dev.xcf"),
		filepath.Join(tmp, "xcf", "agents", "reviewer.xcf"),
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
		filepath.Join(tmp, "xcf", "agents", "dev.md"),
		filepath.Join(tmp, "xcf", "agents", "reviewer.md"),
		filepath.Join(tmp, "xcf", "rules", "security.md"),
		filepath.Join(tmp, "xcf", "workflows", "deploy.md"),
	}
	for _, f := range unexpectedMdFiles {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("file %q should NOT exist (instructions are inlined, not copied)", f)
		}
	}

	// Agent .xcf files must contain inline instructions, not instructions-file
	devXcf, err := os.ReadFile(filepath.Join(tmp, "xcf", "agents", "dev.xcf"))
	require.NoError(t, err)
	devXcfStr := string(devXcf)
	assert.Contains(t, devXcfStr, "kind: agent")
	assert.Contains(t, devXcfStr, "instructions:", "agent xcf must have inline instructions")
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

	err = mergeImportDirs([]string{".claude"}, "scaffold.xcf")
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

	err = importScope(".claude", "scaffold.xcf", "project")
	require.NoError(t, err)

	content, err := os.ReadFile("scaffold.xcf")
	require.NoError(t, err)

	s := string(content)

	// scaffold.xcf must use kind: project (split-file format, not multi-kind)
	assert.Contains(t, s, "kind: project")

	// Must NOT contain multi-kind documents inline in scaffold.xcf
	// (they are split into xcf/agents/*.xcf, xcf/skills/*.xcf, etc.)
	assert.NotContains(t, s, "kind: agent", "agent must be in xcf/agents/dev.xcf, not scaffold.xcf")
	assert.NotContains(t, s, "kind: skill", "skill must be in xcf/skills/tdd.xcf, not scaffold.xcf")

	// Split files must exist
	assert.FileExists(t, filepath.Join(dir, "xcf", "agents", "dev.xcf"))
	assert.FileExists(t, filepath.Join(dir, "xcf", "skills", "tdd.xcf"))

	// Agent split file must have inline instructions
	devXcf, err := os.ReadFile(filepath.Join(dir, "xcf", "agents", "dev.xcf"))
	require.NoError(t, err)
	assert.Contains(t, string(devXcf), "instructions:")
	assert.Contains(t, string(devXcf), "Dev instructions")
	assert.NotContains(t, string(devXcf), "instructions-file:")
}

func TestDetectAllGlobalPlatformDirs_Empty(t *testing.T) {
	// Point HOME to a temp dir with no provider directories
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dirs := detectAllGlobalPlatformDirs()
	if len(dirs) != 0 {
		t.Errorf("expected 0 dirs when no provider dirs exist, got %d: %v", len(dirs), dirs)
	}
}

func TestDetectAllGlobalPlatformDirs_SingleProvider(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create ~/.claude/agents/dev.md and ~/.claude/rules/sec.md
	claudeAgents := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(claudeAgents, 0755); err != nil {
		t.Fatalf("failed to create ~/.claude/agents/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeAgents, "dev.md"), []byte("# Dev\n"), 0600); err != nil {
		t.Fatalf("failed to write dev.md: %v", err)
	}
	claudeRules := filepath.Join(tmp, ".claude", "rules")
	if err := os.MkdirAll(claudeRules, 0755); err != nil {
		t.Fatalf("failed to create ~/.claude/rules/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeRules, "sec.md"), []byte("# Sec\n"), 0600); err != nil {
		t.Fatalf("failed to write sec.md: %v", err)
	}

	dirs := detectAllGlobalPlatformDirs()
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
	// dirName must be the absolute path to ~/.claude
	expected := filepath.Join(tmp, ".claude")
	if dirs[0].dirName != expected {
		t.Errorf("expected dirName %q, got %q", expected, dirs[0].dirName)
	}
}

func TestDetectAllGlobalPlatformDirs_MultiProvider_SortedBySize(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// ~/.claude — 1 agent
	claudeAgents := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(claudeAgents, 0755); err != nil {
		t.Fatalf("failed to create ~/.claude/agents/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeAgents, "dev.md"), []byte("# Dev\n"), 0600); err != nil {
		t.Fatalf("failed to write dev.md: %v", err)
	}

	// ~/.cursor — 2 rules (richer)
	cursorRules := filepath.Join(tmp, ".cursor", "rules")
	if err := os.MkdirAll(cursorRules, 0755); err != nil {
		t.Fatalf("failed to create ~/.cursor/rules/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cursorRules, "r1.mdc"), []byte("rule1"), 0600); err != nil {
		t.Fatalf("failed to write r1.mdc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cursorRules, "r2.mdc"), []byte("rule2"), 0600); err != nil {
		t.Fatalf("failed to write r2.mdc: %v", err)
	}

	dirs := detectAllGlobalPlatformDirs()
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
