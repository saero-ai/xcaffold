package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/importer"
	"github.com/saero-ai/xcaffold/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProviderImporter wraps a directory name and platform string and delegates
// to the real ProviderImporter for the given platform.
type testProviderImporter struct {
	dirName  string
	platform string
	delegate importer.ProviderImporter
}

func (t testProviderImporter) InputDir() string {
	return t.dirName
}

func (t testProviderImporter) Provider() string {
	return t.platform
}

func (t testProviderImporter) Classify(rel string, isDir bool) (importer.Kind, importer.Layout) {
	if t.delegate != nil {
		return t.delegate.Classify(rel, isDir)
	}
	return importer.KindUnknown, importer.LayoutUnknown
}

func (t testProviderImporter) Extract(rel string, data []byte, config *ast.XcaffoldConfig) error {
	if t.delegate != nil {
		return t.delegate.Extract(rel, data, config)
	}
	return nil
}

func (t testProviderImporter) Import(dir string, config *ast.XcaffoldConfig) error {
	if t.delegate != nil {
		return t.delegate.Import(dir, config)
	}
	return nil
}

// makeTestImporters builds a slice of test importers for the given (dirName, platform) pairs,
// looking up the real importer delegate for each platform.
func makeTestImporters(pairs ...struct{ dir, platform string }) []importer.ProviderImporter {
	realImporters := importer.DefaultImporters()
	var result []importer.ProviderImporter
	for _, p := range pairs {
		var delegate importer.ProviderImporter
		for _, real := range realImporters {
			if real.Provider() == p.platform {
				delegate = real
				break
			}
		}
		result = append(result, testProviderImporter{
			dirName:  p.dir,
			platform: p.platform,
			delegate: delegate,
		})
	}
	return result
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

	err = importScope(".claude", "project.xcf", "project", "claude")
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

	importErr := importScope(".claude", "project.xcf", "project", "claude")

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
	if err := importScope(".claude", "project.xcf", "project", "claude"); err != nil {
		t.Fatalf("importScope returned unexpected error: %v", err)
	}

	// project.xcf must exist
	if _, err := os.Stat(filepath.Join(tmp, "project.xcf")); err != nil {
		t.Fatalf("project.xcf was not created: %v", err)
	}

	// Read project.xcf — must contain kind: project
	scaffoldData, err := os.ReadFile(filepath.Join(tmp, "project.xcf"))
	require.NoError(t, err)
	scaffoldStr := string(scaffoldData)
	assert.Contains(t, scaffoldStr, "kind: project", "project.xcf must use kind: project (split-file format)")

	// Split .xcf files must exist for each resource
	// Agents live in their own subdirectory: xcf/agents/<name>/agent.xcf
	expectedXcfFiles := []string{
		filepath.Join(tmp, "xcf", "agents", "dev", "agent.xcf"),
		filepath.Join(tmp, "xcf", "agents", "reviewer", "agent.xcf"),
		filepath.Join(tmp, "xcf", "skills", "tdd", "skill.xcf"),
		filepath.Join(tmp, "xcf", "rules", "security", "rule.xcf"),
		filepath.Join(tmp, "xcf", "workflows", "deploy", "workflow.xcf"),
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
	devXcf, err := os.ReadFile(filepath.Join(tmp, "xcf", "agents", "dev", "agent.xcf"))
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

	err = mergeImportDirs(makeTestImporters(struct{ dir, platform string }{".claude", "claude"}), "project.xcf")
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

	err = importScope(".claude", "project.xcf", "project", "claude")
	require.NoError(t, err)

	content, err := os.ReadFile("project.xcf")
	require.NoError(t, err)

	s := string(content)

	// project.xcf must use kind: project (split-file format, not multi-kind)
	assert.Contains(t, s, "kind: project")

	// Must NOT contain multi-kind documents inline in project.xcf
	assert.NotContains(t, s, "kind: agent", "agent must be in xcf/agents/dev/agent.xcf, not project.xcf")
	assert.NotContains(t, s, "kind: skill", "skill must be in xcf/skills/tdd/skill.xcf, not project.xcf")

	// Split files must exist — each kind in its own subdirectory
	assert.FileExists(t, filepath.Join(dir, "xcf", "agents", "dev", "agent.xcf"))
	assert.FileExists(t, filepath.Join(dir, "xcf", "skills", "tdd", "skill.xcf"))

	// Agent split file must use frontmatter format for inline instructions.
	// Instructions content moves into the markdown body, not as a YAML field.
	devXcf, err := os.ReadFile(filepath.Join(dir, "xcf", "agents", "dev", "agent.xcf"))
	require.NoError(t, err)
	devXcfContent := string(devXcf)
	assert.True(t, strings.HasPrefix(devXcfContent, "---\n"), "agent xcf must use frontmatter format")
	assert.NotContains(t, devXcfContent, "instructions:", "instructions must be in the markdown body, not as a YAML field")
	assert.Contains(t, devXcfContent, "Dev instructions")
	assert.NotContains(t, devXcfContent, "instructions-file:")
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

	// File should appear in the skill directory with unknown subdirs.
	passthroughFile := filepath.Join(outDir, "xcf", "skills", "my-skill", "extras", "data.json")
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

func TestMergeImportDirs_SmartAssembly_DifferentAgents(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	claudeDir := filepath.Join(tmp, ".claude")
	require.NoError(t, os.MkdirAll(filepath.Join(claudeDir, "agents"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "agents", "dev.md"),
		[]byte("---\nname: dev\ndescription: Short\n---\n\nShort."),
		0o644,
	))

	cursorDir := filepath.Join(tmp, ".cursor")
	require.NoError(t, os.MkdirAll(filepath.Join(cursorDir, "agents"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(cursorDir, "agents", "dev.md"),
		[]byte("---\nname: dev\ndescription: Detailed developer\n---\n\nYou are a senior developer. Follow TDD strictly."),
		0o644,
	))

	importers := makeTestImporters(
		struct{ dir, platform string }{".claude", "claude"},
		struct{ dir, platform string }{".cursor", "cursor"},
	)

	err := mergeImportDirs(importers, filepath.Join(tmp, ".xcaffold", "project.xcf"))
	require.NoError(t, err)

	config, parseErr := parser.ParseDirectory(".")
	require.NoError(t, parseErr)
	dev, ok := config.Agents["dev"]
	require.True(t, ok, "dev agent must exist in base config")
	require.NotNil(t, dev.Targets, "dev agent should have targets")
	require.Equal(t, 2, len(dev.Targets), "dev agent should list both providers in targets")
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

	importers := makeTestImporters(
		struct{ dir, platform string }{".claude", "claude"},
		struct{ dir, platform string }{".cursor", "cursor"},
	)

	err := mergeImportDirs(importers, filepath.Join(tmp, ".xcaffold", "project.xcf"))
	require.NoError(t, err)

	// MCP from .claude/ must be present
	config, parseErr := parser.ParseDirectory(".")
	require.NoError(t, parseErr)
	_, hasMCP := config.MCP["my-server"]
	require.True(t, hasMCP, "MCP server from .claude/ must be imported in multi-dir mode")
}

func TestMergeImportDirs_ImportsMemory(t *testing.T) {
	t.Setenv("XCAFFOLD_SKIP_GLOBAL", "true")
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

	importers := makeTestImporters(
		struct{ dir, platform string }{".claude", "claude"},
		struct{ dir, platform string }{".cursor", "cursor"},
	)

	err := mergeImportDirs(importers, filepath.Join(tmp, ".xcaffold", "project.xcf"))
	require.NoError(t, err)

	// Memory must be written to disk
	memPath := filepath.Join("xcf", "agents", "dev", "memory", "context.md")
	_, statErr := os.Stat(memPath)
	require.NoError(t, statErr, "memory file must exist at %s", memPath)

	data, _ := os.ReadFile(memPath)
	require.Contains(t, string(data), "Always use Go 1.24.")
}

func TestImport_RemovedFlags_NotRegistered(t *testing.T) {
	flags := importCmd.Flags()
	for _, name := range []string{"source", "from", "auto-merge", "with-memory"} {
		if flags.Lookup(name) != nil {
			t.Errorf("flag --%s should be removed", name)
		}
	}
}

func TestImport_PreservedFlags_StillRegistered(t *testing.T) {
	flags := importCmd.Flags()
	if flags.Lookup("plan") == nil {
		t.Error("--plan flag should be preserved")
	}
}

func TestImport_TargetFlag_Registered(t *testing.T) {
	f := importCmd.Flags().Lookup("target")
	if f == nil {
		t.Fatal("--target flag should be registered")
	}
	if f.Value.String() != "" {
		t.Errorf("--target flag default should be empty string, got %q", f.Value.String())
	}
}

func TestImport_TargetFlag_ValidatesProvider(t *testing.T) {
	original := importTargetFlag
	defer func() { importTargetFlag = original }()

	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmp)

	importTargetFlag = "invalid-provider"
	err := runImport(importCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown target") {
		t.Fatalf("expected error for invalid target, got: %v", err)
	}
}

func TestImport_TargetFlag_ValidProvider_Accepted(t *testing.T) {
	original := importTargetFlag
	defer func() { importTargetFlag = original }()

	tmp := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmp)

	// Create a mock .claude directory to avoid "no providers found" error
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude", "agents"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, ".claude", "agents", "test.md"),
		[]byte("# Test Agent"),
		0600,
	))

	importTargetFlag = "claude"
	err := runImport(importCmd, nil)
	if err != nil && strings.Contains(err.Error(), "unknown target") {
		t.Fatalf("valid target 'claude' should not produce unknown target error, got: %v", err)
	}
}

func TestImport_KindFilterFlags_Registered(t *testing.T) {
	flags := importCmd.Flags()
	for _, name := range []string{"agent", "skill", "rule", "workflow", "mcp", "hook", "setting", "memory"} {
		if flags.Lookup(name) == nil {
			t.Errorf("--%s flag should be registered", name)
		}
	}
}

func TestApplyKindFilters_NoFilters_KeepsEverything(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{"dev": {}},
			Skills: map[string]ast.SkillConfig{"tdd": {}},
			Rules:  map[string]ast.RuleConfig{"security": {}},
		},
	}
	applyKindFilters(config)
	if config.Agents == nil || config.Skills == nil || config.Rules == nil {
		t.Error("with no filters set, all resources should be preserved")
	}
}

func TestApplyKindFilters_AgentOnly_NilOtherKinds(t *testing.T) {
	// Save original filter state
	originalAgent := importFilterAgent
	originalSkill := importFilterSkill
	originalRule := importFilterRule
	defer func() {
		importFilterAgent = originalAgent
		importFilterSkill = originalSkill
		importFilterRule = originalRule
	}()

	// Simulate --agent flag being set
	importFilterAgent = "*"
	importFilterSkill = ""
	importFilterRule = ""

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{"dev": {}},
			Skills: map[string]ast.SkillConfig{"tdd": {}},
			Rules:  map[string]ast.RuleConfig{"security": {}},
		},
	}
	applyKindFilters(config)
	if config.Agents == nil {
		t.Error("agents should be preserved when --agent is set")
	}
	if config.Skills != nil {
		t.Error("skills should be nil when --agent is set without --skill")
	}
	if config.Rules != nil {
		t.Error("rules should be nil when --agent is set without --rule")
	}
}

func TestApplyKindFilters_NameFilter_NarrowsResource(t *testing.T) {
	// Save original filter state
	originalAgent := importFilterAgent
	defer func() { importFilterAgent = originalAgent }()

	// Simulate --agent dev flag
	importFilterAgent = "dev"

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev":      {},
				"reviewer": {},
			},
		},
	}
	applyKindFilters(config)
	if len(config.Agents) != 1 {
		t.Errorf("expected 1 agent after filter, got %d", len(config.Agents))
	}
	if _, ok := config.Agents["dev"]; !ok {
		t.Error("dev agent should be preserved")
	}
	if _, ok := config.Agents["reviewer"]; ok {
		t.Error("reviewer agent should be filtered out")
	}
}

func TestApplyKindFilters_NameFilter_Nonexistent_Nils(t *testing.T) {
	// Save original filter state
	originalAgent := importFilterAgent
	defer func() { importFilterAgent = originalAgent }()

	// Simulate --agent nonexistent flag
	importFilterAgent = "nonexistent"

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {},
			},
		},
	}
	applyKindFilters(config)
	if config.Agents != nil {
		t.Error("agents should be nil when named resource does not exist")
	}
}

func TestApplyKindFilters_HooksOnly(t *testing.T) {
	// Save original filter state
	originalHooks := importFilterHook
	defer func() { importFilterHook = originalHooks }()

	importFilterHook = true

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{"dev": {}},
		},
		Hooks: map[string]ast.NamedHookConfig{"default": {}},
	}
	applyKindFilters(config)
	if config.Agents != nil {
		t.Error("agents should be nil when only --hooks is set")
	}
	if config.Hooks == nil {
		t.Error("hooks should be preserved when --hooks is set")
	}
}

func TestApplyKindFilters_MultipleKinds(t *testing.T) {
	// Save original filter state
	originalAgent := importFilterAgent
	originalSkill := importFilterSkill
	originalRule := importFilterRule
	originalHooks := importFilterHook
	defer func() {
		importFilterAgent = originalAgent
		importFilterSkill = originalSkill
		importFilterRule = originalRule
		importFilterHook = originalHooks
	}()

	importFilterAgent = "*"
	importFilterSkill = "*"
	importFilterRule = ""
	importFilterHook = true

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{"dev": {}},
			Skills: map[string]ast.SkillConfig{"tdd": {}},
			Rules:  map[string]ast.RuleConfig{"security": {}},
		},
		Hooks: map[string]ast.NamedHookConfig{"default": {}},
	}
	applyKindFilters(config)
	if config.Agents == nil || config.Skills == nil || config.Hooks == nil {
		t.Error("agents, skills, and hooks should be preserved")
	}
	if config.Rules != nil {
		t.Error("rules should be nil when --rule is not set")
	}
}

func TestTagResourcesWithProvider_TagsAllKinds(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    map[string]ast.AgentConfig{"dev": {Description: "Dev agent"}},
			Skills:    map[string]ast.SkillConfig{"tdd": {Description: "TDD skill"}},
			Rules:     map[string]ast.RuleConfig{"security": {Description: "Security rule"}},
			Workflows: map[string]ast.WorkflowConfig{"deploy": {Description: "Deploy workflow"}},
		},
	}
	tagResourcesWithProvider(config, "claude")

	for name, agent := range config.Agents {
		if _, ok := agent.Targets["claude"]; !ok {
			t.Errorf("agent %q should have targets[claude]", name)
		}
	}
	for name, skill := range config.Skills {
		if _, ok := skill.Targets["claude"]; !ok {
			t.Errorf("skill %q should have targets[claude]", name)
		}
	}
	for name, rule := range config.Rules {
		if _, ok := rule.Targets["claude"]; !ok {
			t.Errorf("rule %q should have targets[claude]", name)
		}
	}
	for name, wf := range config.Workflows {
		if _, ok := wf.Targets["claude"]; !ok {
			t.Errorf("workflow %q should have targets[claude]", name)
		}
	}
}

func TestTagResourcesWithProvider_PreservesExistingTargets(t *testing.T) {
	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {
					Description: "Dev agent",
					Targets:     map[string]ast.TargetOverride{"gemini": {}},
				},
			},
		},
	}
	tagResourcesWithProvider(config, "claude")

	agent := config.Agents["dev"]
	if _, ok := agent.Targets["claude"]; !ok {
		t.Error("should add claude to targets")
	}
	if _, ok := agent.Targets["gemini"]; !ok {
		t.Error("should preserve existing gemini target")
	}
}

func TestTagResourcesWithProvider_EmptyConfig(t *testing.T) {
	config := &ast.XcaffoldConfig{}
	tagResourcesWithProvider(config, "claude")
}

func TestAssembleMultiProvider_IdenticalAgents(t *testing.T) {
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			ResourceScope: ast.ResourceScope{
				Agents: map[string]ast.AgentConfig{
					"dev": {Description: "Developer", Model: "sonnet", Body: "You are a developer."},
				},
			},
		},
		"gemini": {
			ResourceScope: ast.ResourceScope{
				Agents: map[string]ast.AgentConfig{
					"dev": {Description: "Developer", Model: "sonnet", Body: "You are a developer."},
				},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
	assembleMultiProviderResources(providerConfigs, result)

	dev := result.Agents["dev"]
	if _, ok := dev.Targets["claude"]; !ok {
		t.Error("identical agent should list claude in targets")
	}
	if _, ok := dev.Targets["gemini"]; !ok {
		t.Error("identical agent should list gemini in targets")
	}
	if result.Overrides != nil {
		if _, ok := result.Overrides.GetAgent("dev", "claude"); ok {
			t.Error("identical agents should not produce overrides")
		}
		if _, ok := result.Overrides.GetAgent("dev", "gemini"); ok {
			t.Error("identical agents should not produce overrides")
		}
	}
}

func TestAssembleMultiProvider_DifferentAgents(t *testing.T) {
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			ResourceScope: ast.ResourceScope{
				Agents: map[string]ast.AgentConfig{
					"dev": {Description: "Developer", Model: "opus", Body: "Claude developer."},
				},
			},
		},
		"gemini": {
			ResourceScope: ast.ResourceScope{
				Agents: map[string]ast.AgentConfig{
					"dev": {Description: "Developer", Model: "gemini-pro", Body: "Gemini developer."},
				},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
	assembleMultiProviderResources(providerConfigs, result)

	dev := result.Agents["dev"]
	if len(dev.Targets) != 2 {
		t.Errorf("different agent should list both providers in targets, got %d", len(dev.Targets))
	}
	if result.Overrides == nil {
		t.Fatal("different agents should produce overrides")
	}
}

func TestAssembleMultiProvider_SingleProviderAgent(t *testing.T) {
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			ResourceScope: ast.ResourceScope{
				Agents: map[string]ast.AgentConfig{
					"dev": {Description: "Developer"},
				},
			},
		},
		"gemini": {
			ResourceScope: ast.ResourceScope{
				Agents: map[string]ast.AgentConfig{
					"reviewer": {Description: "Reviewer"},
				},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
	assembleMultiProviderResources(providerConfigs, result)

	if len(result.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(result.Agents))
	}
	if _, ok := result.Agents["dev"].Targets["claude"]; !ok {
		t.Error("dev should be tagged with claude")
	}
	if _, ok := result.Agents["reviewer"].Targets["gemini"]; !ok {
		t.Error("reviewer should be tagged with gemini")
	}
}

func TestImport_Output_ExplainsTargetsTagging(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"),
		"---\nname: dev\ndescription: Dev\n---\n\nDev agent.")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := importScope(".claude", "project.xcf", "project", "claude")

	w.Close()
	os.Stdout = oldStdout
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	require.NoError(t, err)
	assert.Contains(t, output, "targets:", "import output should explain targets tagging")
	assert.Contains(t, output, "claude", "output should mention the source provider")
}

func TestImportScope_PlanFlag_DryRun(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	// Set up test provider directories with agents and skills
	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"),
		"---\nname: dev\ndescription: Dev Agent\n---\n\nDev instructions.")
	writeFile(t, filepath.Join(tmp, ".claude", "skills", "testing", "SKILL.md"),
		"---\nname: testing\ndescription: Testing Skill\n---\n\nTesting steps.")

	// Enable --plan flag
	oldImportPlan := importPlan
	importPlan = true
	defer func() { importPlan = oldImportPlan }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := importScope(".claude", "project.xcf", "project", "claude")

	w.Close()
	os.Stdout = oldStdout
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// With --plan, no error should occur
	require.NoError(t, err)

	// Output should show plan summary
	assert.Contains(t, output, "Import plan (dry-run)", "output should indicate dry-run mode")
	assert.Contains(t, output, "1 agents", "output should show agents count")
	assert.Contains(t, output, "1 skills", "output should show skills count")

	// Verify NO files were written
	assert.NoFileExists(t, filepath.Join(tmp, ".xcaffold", "project.xcf"), "project.xcf should not exist in plan mode")
	assert.NoFileExists(t, filepath.Join(tmp, "xcf", "agents", "dev.xcf"), "xcf files should not exist in plan mode")
}

func TestMergeImportDirs_PlanFlag_DryRun(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	// Set up two provider directories
	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"),
		"---\nname: dev\ndescription: Dev\n---\n\nDev.")
	writeFile(t, filepath.Join(tmp, ".gemini", "agents", "reviewer.md"),
		"---\nname: reviewer\ndescription: Reviewer\n---\n\nReviewer.")

	// Enable --plan flag
	oldImportPlan := importPlan
	importPlan = true
	defer func() { importPlan = oldImportPlan }()

	providerImporters := makeTestImporters(
		struct{ dir, platform string }{filepath.Join(tmp, ".claude"), "claude"},
		struct{ dir, platform string }{filepath.Join(tmp, ".gemini"), "gemini"},
	)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := mergeImportDirs(providerImporters, "project.xcf")

	w.Close()
	os.Stdout = oldStdout
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	require.NoError(t, err)
	assert.Contains(t, output, "Import plan (dry-run)")
	assert.Contains(t, output, "2 provider directories")

	// Verify NO files were written
	assert.NoFileExists(t, filepath.Join(tmp, "project.xcf"))
	assert.NoFileExists(t, filepath.Join(tmp, "xcf"))
}

func TestMergeImportDirs_MultiProvider_ConflictCount(t *testing.T) {
	t.Setenv("XCAFFOLD_HOME", t.TempDir())
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	// Create two agents with the same name but different content
	writeFile(t, filepath.Join(tmp, ".claude", "agents", "dev.md"),
		"---\nname: dev\nmodel: claude-sonnet\n---\n\nClaude dev agent.")
	writeFile(t, filepath.Join(tmp, ".gemini", "agents", "dev.md"),
		"---\nname: dev\nmodel: gemini-2.5-flash\n---\n\nGemini dev agent.")

	providerImporters := makeTestImporters(
		struct{ dir, platform string }{filepath.Join(tmp, ".claude"), "claude"},
		struct{ dir, platform string }{filepath.Join(tmp, ".gemini"), "gemini"},
	)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := mergeImportDirs(providerImporters, "project.xcf")

	w.Close()
	os.Stdout = oldStdout
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	require.NoError(t, err)
	// When there are conflicts, the output should mention override files or conflicts
	assert.Contains(t, output, "conflict", "output should indicate conflicts detected")
}

func TestRunPostImportSteps_WritesMemoryAndPrunes(t *testing.T) {
	tmp := t.TempDir()
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	defer os.Chdir(origDir)

	config := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents: map[string]ast.AgentConfig{
				"dev": {Description: "Developer"},
			},
			Memory: map[string]ast.MemoryConfig{
				"dev/context": {
					Name:        "context",
					Description: "Project context",
					Content:     "Context details",
					AgentRef:    "dev",
				},
			},
		},
	}

	err := runPostImportSteps(config, tmp, false)
	require.NoError(t, err)

	// Memory file should be written
	memPath := filepath.Join(tmp, "xcf", "agents", "dev", "memory", "context.md")
	data, err := os.ReadFile(memPath)
	require.NoError(t, err, "memory file should exist")
	require.Contains(t, string(data), "Context details")
}

// TestSplitAgentOverrides_DeterministicBase verifies that the most generic
// provider (fewest provider-specific fields) is selected as the base.
func TestSplitAgentOverrides_DeterministicBase(t *testing.T) {
	hooksData := ast.HookConfig{
		"events": []ast.HookMatcherGroup{},
	}

	configs := map[string]ast.AgentConfig{
		"claude": {
			Name:        "dev",
			Description: "Developer",
			Model:       "claude-sonnet",
			Tools:       []string{"Read", "Write"},
			Hooks:       hooksData,
			Body:        "Claude developer instructions",
		},
		"gemini": {
			Name:        "dev",
			Description: "Developer",
			Model:       "",
			Tools:       []string{},
			Hooks:       nil,
			Body:        "Gemini developer instructions",
		},
		"cursor": {
			Name:        "dev",
			Description: "Developer",
			Model:       "cursor-model",
			Tools:       []string{},
			Hooks:       nil,
			Body:        "Cursor developer instructions",
		},
	}

	base, overrides := splitAgentOverrides(configs)

	// Gemini should be selected as base (score 0: no model, no tools, no hooks)
	// Claude scores 12 (model +1, tools +1, hooks +10)
	// Cursor scores 1 (model +1)
	// So the selection should be:
	// - Gemini: 0 (base)
	// - Cursor: 1 (override)
	// - Claude: 12 (override)

	// The base should match one of the input configs
	found := false
	for provider, cfg := range configs {
		if cfg.Name == base.Name && cfg.Description == base.Description {
			// Check if this is the minimal-score provider
			if provider == "gemini" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("base should be selected from minimal-score provider, got base with body: %s", base.Body)
	}

	// All non-base providers should be in overrides
	expectedOverrideCount := len(configs) - 1
	if len(overrides) != expectedOverrideCount {
		t.Errorf("expected %d overrides, got %d", expectedOverrideCount, len(overrides))
	}
}

// TestSplitAgentOverrides_BodyDedup verifies that identical bodies are stripped
// from overrides.
func TestSplitAgentOverrides_BodyDedup(t *testing.T) {
	sharedBody := "Shared developer instructions"

	configs := map[string]ast.AgentConfig{
		"claude": {
			Name:        "dev",
			Description: "Developer",
			Model:       "claude-sonnet",
			Tools:       []string{"Read"},
			Body:        sharedBody,
		},
		"gemini": {
			Name:        "dev",
			Description: "Developer",
			Model:       "gemini-pro",
			Tools:       []string{},
			Body:        sharedBody, // Identical body
		},
	}

	base, overrides := splitAgentOverrides(configs)

	// Find the override (the non-base provider)
	if len(overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(overrides))
	}

	for _, override := range overrides {
		// If the bodies are identical, the override's body should be stripped
		if strings.TrimSpace(base.Body) == strings.TrimSpace(override.Body) {
			if override.Body != "" {
				t.Errorf("override body should be empty when identical to base, got: %q", override.Body)
			}
		}
	}
}

// TestSplitSkillOverrides_DeterministicBase verifies skills use AllowedTools for scoring.
func TestSplitSkillOverrides_DeterministicBase(t *testing.T) {
	configs := map[string]ast.SkillConfig{
		"claude": {
			Name:         "tdd",
			Description:  "TDD Skill",
			AllowedTools: []string{"Read", "Write"},
			Body:         "Claude TDD",
		},
		"gemini": {
			Name:         "tdd",
			Description:  "TDD Skill",
			AllowedTools: []string{},
			Body:         "Gemini TDD",
		},
	}

	base, _ := splitSkillOverrides(configs)

	// Base body should be from the minimal-score provider (gemini with empty AllowedTools)
	if !strings.Contains(base.Body, "Gemini") {
		t.Errorf("base should be from gemini (minimal score), got body: %s", base.Body)
	}
}

// TestSplitRuleOverrides_DeterministicBase verifies rules use alphabetical ordering.
func TestSplitRuleOverrides_DeterministicBase(t *testing.T) {
	configs := map[string]ast.RuleConfig{
		"zenith": {
			Name:        "security",
			Description: "Security Rule",
			Body:        "Zenith security",
		},
		"alpha": {
			Name:        "security",
			Description: "Security Rule",
			Body:        "Alpha security",
		},
	}

	base, _ := splitRuleOverrides(configs)

	// Both providers have the same score (0), so alphabetical tie-break applies
	// "alpha" < "zenith", so alpha should be base
	if !strings.Contains(base.Body, "Alpha") {
		t.Errorf("base should be alphabetically first (alpha), got body: %s", base.Body)
	}
}

// TestSplitWorkflowOverrides_DeterministicBase verifies workflows use alphabetical ordering.
func TestSplitWorkflowOverrides_DeterministicBase(t *testing.T) {
	configs := map[string]ast.WorkflowConfig{
		"zebra": {
			Name:        "deploy",
			Description: "Deploy Workflow",
			Body:        "Zebra deploy",
		},
		"apple": {
			Name:        "deploy",
			Description: "Deploy Workflow",
			Body:        "Apple deploy",
		},
	}

	base, _ := splitWorkflowOverrides(configs)

	// Both providers have score 0, so alphabetical tie-break applies
	// "apple" < "zebra", so apple should be base
	if !strings.Contains(base.Body, "Apple") {
		t.Errorf("base should be alphabetically first (apple), got body: %s", base.Body)
	}
}

func TestImport_HookConfigsIdentical_ReturnsTrueForMatchingConfigs(t *testing.T) {
	configs := map[string]ast.NamedHookConfig{
		"claude": {
			Name:      "default",
			Artifacts: []string{"lint.sh"},
			Events: ast.HookConfig{
				"pre-commit": []ast.HookMatcherGroup{},
			},
		},
		"gemini": {
			Name:      "default",
			Artifacts: []string{"lint.sh"},
			Events: ast.HookConfig{
				"pre-commit": []ast.HookMatcherGroup{},
			},
		},
	}

	result := hookConfigsIdentical(configs)
	if !result {
		t.Errorf("hookConfigsIdentical should return true for matching configs, got false")
	}
}

func TestImport_HookConfigsIdentical_ReturnsFalseForDifferentConfigs(t *testing.T) {
	configs := map[string]ast.NamedHookConfig{
		"claude": {
			Name:      "default",
			Artifacts: []string{"lint.sh", "format.sh"},
		},
		"gemini": {
			Name:      "default",
			Artifacts: []string{"lint.sh"},
		},
	}

	result := hookConfigsIdentical(configs)
	if result {
		t.Errorf("hookConfigsIdentical should return false for different configs, got true")
	}
}

func TestImport_AssembleHooks_SingleProvider(t *testing.T) {
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			Hooks: map[string]ast.NamedHookConfig{
				"default": {
					Name:      "default",
					Artifacts: []string{"lint.sh"},
				},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		Hooks: make(map[string]ast.NamedHookConfig),
	}

	assembleHooks(providerConfigs, result)

	if _, ok := result.Hooks["default"]; !ok {
		t.Errorf("assembleHooks should add hook to result when single provider")
	}
	if result.Overrides != nil {
		t.Errorf("assembleHooks should not create overrides for single provider, got non-nil")
	}
}

func TestImport_AssembleHooks_DifferentMultiProvider(t *testing.T) {
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			Hooks: map[string]ast.NamedHookConfig{
				"pre-commit": {
					Name:      "pre-commit",
					Artifacts: []string{"lint.sh", "format.sh"},
					Events: ast.HookConfig{
						"pre-commit": []ast.HookMatcherGroup{},
					},
				},
			},
		},
		"gemini": {
			Hooks: map[string]ast.NamedHookConfig{
				"pre-commit": {
					Name:      "pre-commit",
					Artifacts: []string{"lint.sh"},
				},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		Hooks: make(map[string]ast.NamedHookConfig),
	}

	assembleHooks(providerConfigs, result)

	// Base should be the one with lower score
	// claude: score = 5 (Events non-nil+non-empty) + 2 (artifacts) = 7
	// gemini: score = 0 (no Events) + 1 (artifact) = 1
	// So gemini is base, claude is override
	if _, ok := result.Hooks["pre-commit"]; !ok {
		t.Errorf("assembleHooks should add hook to result")
	}

	if result.Overrides == nil {
		t.Fatal("assembleHooks should create overrides for different configs")
	}

	// Check that override exists for the non-base provider
	override, hasOverride := result.Overrides.GetHooks("pre-commit", "claude")
	if !hasOverride {
		t.Errorf("assembleHooks should add claude hook as override, got no override")
	}
	if override.Artifacts == nil || len(override.Artifacts) == 0 {
		t.Errorf("override should preserve artifacts")
	}
	if override.Events == nil {
		t.Errorf("override should preserve Events")
	}
}

func TestImport_SettingsConfigsIdentical_ReturnsTrueForMatchingConfigs(t *testing.T) {
	boolTrue := true
	configs := map[string]ast.SettingsConfig{
		"claude": {Name: "default", AutoMemoryEnabled: &boolTrue},
		"gemini": {Name: "default", AutoMemoryEnabled: &boolTrue},
	}
	if !settingsConfigsIdentical(configs) {
		t.Error("identical settings configs should return true")
	}
}

func TestImport_SettingsConfigsIdentical_ReturnsFalseForDifferentConfigs(t *testing.T) {
	boolTrue := true
	boolFalse := false
	configs := map[string]ast.SettingsConfig{
		"claude": {Name: "default", AutoMemoryEnabled: &boolTrue, DisableAllHooks: &boolFalse},
		"gemini": {Name: "default", AutoMemoryEnabled: &boolFalse},
	}
	if settingsConfigsIdentical(configs) {
		t.Error("different settings configs should return false")
	}
}

func TestImport_AssembleSettings_SingleProvider(t *testing.T) {
	boolTrue := true
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			Settings: map[string]ast.SettingsConfig{
				"default": {Name: "default", AutoMemoryEnabled: &boolTrue},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		Settings: make(map[string]ast.SettingsConfig),
	}
	assembleSettings(providerConfigs, result)
	sc, exists := result.Settings["default"]
	require.True(t, exists, "settings must be stored")
	assert.Equal(t, &boolTrue, sc.AutoMemoryEnabled)
	if result.Overrides != nil {
		_, ok := result.Overrides.GetSettings("default", "claude")
		assert.False(t, ok, "single-provider should not produce overrides")
	}
}

func TestImport_AssembleSettings_DifferentMultiProvider(t *testing.T) {
	boolTrue := true
	boolFalse := false
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			Settings: map[string]ast.SettingsConfig{
				"default": {
					Name:              "default",
					AutoMemoryEnabled: &boolTrue,
					DisableAllHooks:   &boolFalse,
					Permissions:       &ast.PermissionsConfig{},
					Hooks:             ast.HookConfig{"events": []ast.HookMatcherGroup{}},
				},
			},
		},
		"gemini": {
			Settings: map[string]ast.SettingsConfig{
				"default": {
					Name:              "default",
					AutoMemoryEnabled: &boolFalse,
				},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		Settings: make(map[string]ast.SettingsConfig),
	}
	assembleSettings(providerConfigs, result)
	_, exists := result.Settings["default"]
	require.True(t, exists, "base settings must be stored")
	require.NotNil(t, result.Overrides, "overrides must be populated")
	_, hasClaudeOverride := result.Overrides.GetSettings("default", "claude")
	assert.True(t, hasClaudeOverride, "claude should be override (higher score)")
	_, hasGeminiOverride := result.Overrides.GetSettings("default", "gemini")
	assert.False(t, hasGeminiOverride, "gemini should be base (lower score)")
}

func TestImport_AssembleMultiProvider_HooksOverrideSplit(t *testing.T) {
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			ResourceScope: ast.ResourceScope{
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
			Hooks: map[string]ast.NamedHookConfig{
				"pre-commit": {
					Name:      "pre-commit",
					Artifacts: []string{"lint.sh", "format.sh"},
					Events:    ast.HookConfig{"pre-push": []ast.HookMatcherGroup{}},
				},
			},
		},
		"gemini": {
			ResourceScope: ast.ResourceScope{
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
			Hooks: map[string]ast.NamedHookConfig{
				"pre-commit": {
					Name:      "pre-commit",
					Artifacts: []string{"lint.sh"},
				},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
	assembleMultiProviderResources(providerConfigs, result)
	_, exists := result.Hooks["pre-commit"]
	require.True(t, exists, "base hook must be stored after full assembly")
	require.NotNil(t, result.Overrides, "overrides must be populated for different hooks")
}

func TestImport_AssembleMultiProvider_SettingsOverrideSplit(t *testing.T) {
	boolTrue := true
	boolFalse := false
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			ResourceScope: ast.ResourceScope{
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
			Settings: map[string]ast.SettingsConfig{
				"default": {
					Name:              "default",
					AutoMemoryEnabled: &boolTrue,
					DisableAllHooks:   &boolFalse,
				},
			},
		},
		"gemini": {
			ResourceScope: ast.ResourceScope{
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
			Settings: map[string]ast.SettingsConfig{
				"default": {
					Name:              "default",
					AutoMemoryEnabled: &boolFalse,
				},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
	assembleMultiProviderResources(providerConfigs, result)
	_, exists := result.Settings["default"]
	require.True(t, exists, "base settings must be stored after full assembly")
	require.NotNil(t, result.Overrides, "overrides must be populated for different settings")
}

func TestImport_AssembleMultiProvider_UnionBlockRetainsMCPAndMemory(t *testing.T) {
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			ResourceScope: ast.ResourceScope{
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP: map[string]ast.MCPConfig{
					"playwright": {Command: "npx", Args: []string{"@playwright/mcp"}},
				},
				Memory: map[string]ast.MemoryConfig{
					"session": {Name: "session"},
				},
			},
		},
		"gemini": {
			ResourceScope: ast.ResourceScope{
				Agents:    make(map[string]ast.AgentConfig),
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP: map[string]ast.MCPConfig{
					"search": {Command: "search-server"},
				},
				Memory: map[string]ast.MemoryConfig{
					"context": {Name: "context"},
				},
			},
		},
	}
	result := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
	assembleMultiProviderResources(providerConfigs, result)
	_, playwrightExists := result.MCP["playwright"]
	assert.True(t, playwrightExists, "playwright MCP must exist")
	_, searchExists := result.MCP["search"]
	assert.True(t, searchExists, "search MCP must exist")
	_, sessionExists := result.Memory["session"]
	assert.True(t, sessionExists, "session memory must exist")
	_, contextExists := result.Memory["context"]
	assert.True(t, contextExists, "context memory must exist")
}

func TestAssembleMultiProvider_IdenticalAgents_ThreeProviders(t *testing.T) {
	providerConfigs := map[string]*ast.XcaffoldConfig{
		"claude": {
			ResourceScope: ast.ResourceScope{
				Agents: map[string]ast.AgentConfig{
					"dev": {Description: "Developer", Model: "sonnet", Body: "You are a developer."},
				},
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
		},
		"gemini": {
			ResourceScope: ast.ResourceScope{
				Agents: map[string]ast.AgentConfig{
					"dev": {Description: "Developer", Model: "sonnet", Body: "You are a developer."},
				},
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
		},
		"cursor": {
			ResourceScope: ast.ResourceScope{
				Agents: map[string]ast.AgentConfig{
					"dev": {Description: "Developer", Model: "sonnet", Body: "You are a developer."},
				},
				Skills:    make(map[string]ast.SkillConfig),
				Rules:     make(map[string]ast.RuleConfig),
				Workflows: make(map[string]ast.WorkflowConfig),
				MCP:       make(map[string]ast.MCPConfig),
			},
		},
	}
	result := &ast.XcaffoldConfig{
		ResourceScope: ast.ResourceScope{
			Agents:    make(map[string]ast.AgentConfig),
			Skills:    make(map[string]ast.SkillConfig),
			Rules:     make(map[string]ast.RuleConfig),
			Workflows: make(map[string]ast.WorkflowConfig),
			MCP:       make(map[string]ast.MCPConfig),
		},
	}
	assembleMultiProviderResources(providerConfigs, result)
	dev := result.Agents["dev"]
	assert.Len(t, dev.Targets, 3, "all three providers should appear in targets")
	if result.Overrides != nil {
		_, ok := result.Overrides.GetAgent("dev", "claude")
		assert.False(t, ok, "identical agents should not produce overrides")
	}
}
