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

func TestExtractAgents_CopiesToXcfDir(t *testing.T) {
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

	// Must point to xcf/agents/dev.md, not .claude/agents/dev.md
	if agentCfg.InstructionsFile != "xcf/agents/dev.md" {
		t.Errorf("expected InstructionsFile == %q, got %q", "xcf/agents/dev.md", agentCfg.InstructionsFile)
	}

	// The path must NOT start with .claude/
	if strings.HasPrefix(agentCfg.InstructionsFile, ".claude/") {
		t.Errorf("InstructionsFile should not start with .claude/, got: %q", agentCfg.InstructionsFile)
	}

	// The file must exist on disk at xcf/agents/dev.md
	xcfPath := filepath.Join(tmp, "xcf", "agents", "dev.md")
	data, err := os.ReadFile(xcfPath)
	if err != nil {
		t.Fatalf("expected xcf/agents/dev.md to exist on disk: %v", err)
	}
	if string(data) != agentContent {
		t.Errorf("xcf/agents/dev.md content mismatch: got %q, want %q", string(data), agentContent)
	}
}

func TestExtractRules_CopiesToXcfDir(t *testing.T) {
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

	// Must point to xcf/rules/security.md, not .claude/rules/security.md
	if ruleCfg.InstructionsFile != "xcf/rules/security.md" {
		t.Errorf("expected InstructionsFile == %q, got %q", "xcf/rules/security.md", ruleCfg.InstructionsFile)
	}

	// The path must NOT start with .claude/
	if strings.HasPrefix(ruleCfg.InstructionsFile, ".claude/") {
		t.Errorf("InstructionsFile should not start with .claude/, got: %q", ruleCfg.InstructionsFile)
	}

	// The file must exist on disk at xcf/rules/security.md
	xcfPath := filepath.Join(tmp, "xcf", "rules", "security.md")
	data, err := os.ReadFile(xcfPath)
	if err != nil {
		t.Fatalf("expected xcf/rules/security.md to exist on disk: %v", err)
	}
	if string(data) != ruleContent {
		t.Errorf("xcf/rules/security.md content mismatch: got %q, want %q", string(data), ruleContent)
	}
}

func TestExtractSkills_CopiesToXcfDir(t *testing.T) {
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

	// Must point to xcf/skills/tdd/SKILL.md, not .claude/skills/tdd/SKILL.md
	if skillCfg.InstructionsFile != "xcf/skills/tdd/SKILL.md" {
		t.Errorf("expected InstructionsFile == %q, got %q", "xcf/skills/tdd/SKILL.md", skillCfg.InstructionsFile)
	}

	// Must NOT start with .claude/
	if strings.HasPrefix(skillCfg.InstructionsFile, ".claude/") {
		t.Errorf("InstructionsFile should not start with .claude/, got: %q", skillCfg.InstructionsFile)
	}

	// SKILL.md must exist on disk at xcf/skills/tdd/SKILL.md
	xcfSkillPath := filepath.Join(tmp, "xcf", "skills", "tdd", "SKILL.md")
	data, err := os.ReadFile(xcfSkillPath)
	if err != nil {
		t.Fatalf("expected xcf/skills/tdd/SKILL.md to exist on disk: %v", err)
	}
	if string(data) != skillContent {
		t.Errorf("xcf/skills/tdd/SKILL.md content mismatch: got %q, want %q", string(data), skillContent)
	}

	// references/example.txt must exist on disk at xcf/skills/tdd/references/example.txt
	xcfRefPath := filepath.Join(tmp, "xcf", "skills", "tdd", "references", "example.txt")
	refData, err := os.ReadFile(xcfRefPath)
	if err != nil {
		t.Fatalf("expected xcf/skills/tdd/references/example.txt to exist on disk: %v", err)
	}
	if string(refData) != refContent {
		t.Errorf("xcf/skills/tdd/references/example.txt content mismatch: got %q, want %q", string(refData), refContent)
	}

	// References in config must point to xcf/ paths, not .claude/ paths
	for _, ref := range skillCfg.References {
		if strings.HasPrefix(ref, ".claude/") {
			t.Errorf("reference %q should not start with .claude/", ref)
		}
		if !strings.HasPrefix(ref, "xcf/") {
			t.Errorf("reference %q should start with xcf/", ref)
		}
	}
}

func TestExtractWorkflows_CopiesToXcfDir(t *testing.T) {
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

	// Must point to xcf/workflows/deploy.md, not .claude/workflows/deploy.md
	if workflowCfg.InstructionsFile != "xcf/workflows/deploy.md" {
		t.Errorf("expected InstructionsFile == %q, got %q", "xcf/workflows/deploy.md", workflowCfg.InstructionsFile)
	}

	// Must NOT start with .claude/
	if strings.HasPrefix(workflowCfg.InstructionsFile, ".claude/") {
		t.Errorf("InstructionsFile should not start with .claude/, got: %q", workflowCfg.InstructionsFile)
	}

	// The file must exist on disk at xcf/workflows/deploy.md
	xcfPath := filepath.Join(tmp, "xcf", "workflows", "deploy.md")
	data, err := os.ReadFile(xcfPath)
	if err != nil {
		t.Fatalf("expected xcf/workflows/deploy.md to exist on disk: %v", err)
	}
	if string(data) != workflowContent {
		t.Errorf("xcf/workflows/deploy.md content mismatch: got %q, want %q", string(data), workflowContent)
	}
}

func TestImportScope_Messaging_NoReferencedInPlace(t *testing.T) {
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
	if !strings.Contains(output, "xcf/") {
		t.Errorf("output should contain 'xcf/', got: %q", output)
	}
}

func TestImport_RoundTrip_ValidatesPaths(t *testing.T) {
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

	// Create .claude/skills/tdd/references/patterns.md
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

	// Run importScope — this is the function under test.
	if err := importScope(".claude", "scaffold.xcf", "project"); err != nil {
		t.Fatalf("importScope returned unexpected error: %v", err)
	}

	// scaffold.xcf must exist on disk.
	if _, err := os.Stat(filepath.Join(tmp, "scaffold.xcf")); err != nil {
		t.Fatalf("scaffold.xcf was not created: %v", err)
	}

	// xcf/ directory must exist with expected subdirectories and files.
	expectedFiles := []string{
		filepath.Join(tmp, "xcf", "agents", "dev.md"),
		filepath.Join(tmp, "xcf", "agents", "reviewer.md"),
		filepath.Join(tmp, "xcf", "skills", "tdd", "SKILL.md"),
		filepath.Join(tmp, "xcf", "skills", "tdd", "references", "patterns.md"),
		filepath.Join(tmp, "xcf", "rules", "security.md"),
		filepath.Join(tmp, "xcf", "workflows", "deploy.md"),
	}
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected file %q to exist on disk: %v", f, err)
		}
	}

	// Parse the generated scaffold.xcf — ParseFile runs full validation including
	// validateInstructionsFile, so a successful parse proves no reserved-prefix paths
	// (e.g. .claude/, .cursor/) slipped through.
	parsed, err := parser.ParseFile(filepath.Join(tmp, "scaffold.xcf"))
	if err != nil {
		t.Fatalf("parser.ParseFile failed on generated scaffold.xcf: %v", err)
	}

	// Version must be "1.0".
	if parsed.Version != "1.0" {
		t.Errorf("expected version %q, got %q", "1.0", parsed.Version)
	}

	// Collect all InstructionsFile values from resources that originated from this
	// import (i.e. not inherited from the global base config) and assert:
	//   1. They do not start with any reserved output prefix.
	//   2. The file they reference actually exists on disk (relative to tmp).
	//
	// Inherited resources have absolute paths pointing to the user's global
	// ~/.xcaffold/ directory — those are valid by design and outside this test's scope.
	reservedPrefixes := []string{".claude/", ".cursor/", ".agents/", ".antigravity/"}

	checkPath := func(kind, id, path string, inherited bool) {
		if path == "" || inherited {
			return
		}
		// Absolute paths belong to the global scope — skip reserved-prefix and
		// disk-existence checks for them (validateInstructionsFile does the same).
		if filepath.IsAbs(path) {
			return
		}
		for _, prefix := range reservedPrefixes {
			if strings.HasPrefix(path, prefix) {
				t.Errorf("%s %q: InstructionsFile %q starts with reserved prefix %q", kind, id, path, prefix)
			}
		}
		abs := filepath.Join(tmp, path)
		if _, err := os.Stat(abs); err != nil {
			t.Errorf("%s %q: InstructionsFile %q does not exist on disk at %q: %v", kind, id, path, abs, err)
		}
	}

	for id, a := range parsed.Agents {
		checkPath("agent", id, a.InstructionsFile, a.Inherited)
	}
	for id, s := range parsed.Skills {
		checkPath("skill", id, s.InstructionsFile, s.Inherited)
	}
	for id, r := range parsed.Rules {
		checkPath("rule", id, r.InstructionsFile, r.Inherited)
	}
	for id, w := range parsed.Workflows {
		checkPath("workflow", id, w.InstructionsFile, w.Inherited)
	}
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
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()
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

func TestImportScope_EmitsMultiKindFormat(t *testing.T) {
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

	// Must contain multi-kind documents
	assert.Contains(t, s, "kind: config")
	assert.Contains(t, s, "kind: agent")
	assert.Contains(t, s, "kind: skill")
	assert.Contains(t, s, "---")

	// Must NOT be monolithic — agents must not be nested under kind: config
	// In multi-kind format the config document has no agents: key
	assert.NotContains(t, s, "agents:\n  dev:")
}
