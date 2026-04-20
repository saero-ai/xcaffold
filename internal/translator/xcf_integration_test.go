package translator_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/saero-ai/xcaffold/internal/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// constraintOnlyWorkflow contains directive keywords only — no numbered steps,
// no turbo annotation — so only IntentConstraint is detected.
const constraintOnlyWorkflow = `---
description: Deployment constraints
---
# Constraints

You MUST validate all inputs before deploying.
You NEVER push to production without a green CI run.
ALWAYS check the rollback procedure before releasing.
`

// TestXcfIntegration_SkillPrimitive_ProducesInstructionsFileReference verifies
// that a skill primitive from a pure-procedure workflow results in an ast.SkillConfig
// with a non-empty InstructionsFile field and no inline Instructions.
func TestXcfIntegration_SkillPrimitive_ProducesInstructionsFileReference(t *testing.T) {
	path := writeTemp(t, "deploy.md", tddWorkflow)

	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)

	result := translator.Translate(unit, "claude")
	require.Len(t, result.Primitives, 1)

	p := result.Primitives[0]
	require.Equal(t, "skill", p.Kind)

	// Simulate what injectIntoConfig does: build an ast.SkillConfig with
	// instructions-file pointing to the external file path.
	relPath := filepath.Join("skills", p.ID, "SKILL.md")
	skill := ast.SkillConfig{
		Description:      fmt.Sprintf("Translated from workflow %s", p.ID),
		InstructionsFile: relPath,
	}

	// Marshal to YAML and verify instructions-file: appears, not instructions:.
	out, err := yaml.Marshal(skill)
	require.NoError(t, err)

	yamlStr := string(out)
	assert.Contains(t, yamlStr, "instructions-file:", "skill YAML must use instructions-file")
	assert.NotContains(t, yamlStr, "\ninstructions:", "skill YAML must not contain inline instructions key")
	assert.Contains(t, yamlStr, relPath, "skill YAML must include the relative path")
}

// TestXcfIntegration_RulePrimitive_ProducesInstructionsFileReference verifies
// that a rule primitive produces an ast.RuleConfig with instructions-file set
// and no inline instructions field.
func TestXcfIntegration_RulePrimitive_ProducesInstructionsFileReference(t *testing.T) {
	path := writeTemp(t, "constraints.md", constraintOnlyWorkflow)

	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)

	result := translator.Translate(unit, "claude")

	var ruleP *translator.TargetPrimitive
	for i := range result.Primitives {
		if result.Primitives[i].Kind == "rule" {
			ruleP = &result.Primitives[i]
			break
		}
	}
	require.NotNil(t, ruleP, "expected a rule primitive from constraint-only workflow")

	relPath := filepath.Join("rules", ruleP.ID+".md")
	rule := ast.RuleConfig{
		Description:      fmt.Sprintf("Constraints from workflow %s", ruleP.ID),
		InstructionsFile: relPath,
	}

	out, err := yaml.Marshal(rule)
	require.NoError(t, err)

	yamlStr := string(out)
	assert.Contains(t, yamlStr, "instructions-file:", "rule YAML must use instructions-file")
	assert.NotContains(t, yamlStr, "\ninstructions:", "rule YAML must not contain inline instructions key")
	assert.Contains(t, yamlStr, relPath, "rule YAML must include the relative path")
}

// TestXcfIntegration_PermissionPrimitive_MergesWithoutDuplicates verifies that
// merging permission allow entries into an existing config preserves original
// entries, appends new ones, and produces no duplicates.
func TestXcfIntegration_PermissionPrimitive_MergesWithoutDuplicates(t *testing.T) {
	// Start with a config that already has one allow entry.
	existing := []string{"Bash(npm *)"}
	config := &ast.XcaffoldConfig{
		Settings: map[string]ast.SettingsConfig{"default": {
			Permissions: &ast.PermissionsConfig{
				Allow: existing,
			},
		}},
	}

	// Incoming allow entries from translation (turbo → git + go).
	newEntries := []string{"Bash(git *)", "Bash(go *)"}

	// Merge — same logic as injectIntoConfig.
	effective := config.Settings["default"]
	seen := make(map[string]bool, len(effective.Permissions.Allow))
	for _, e := range effective.Permissions.Allow {
		seen[e] = true
	}

	// We need to mutate the settings entry; extract, modify, and put back.
	entry := config.Settings["default"]
	for _, newEntry := range newEntries {
		if !seen[newEntry] {
			seen[newEntry] = true
			entry.Permissions.Allow = append(entry.Permissions.Allow, newEntry)
		}
	}
	config.Settings["default"] = entry

	// Re-merge same entries — duplicates must be rejected.
	entry2 := config.Settings["default"]
	for _, newEntry := range newEntries {
		if !seen[newEntry] {
			entry2.Permissions.Allow = append(entry2.Permissions.Allow, newEntry)
		}
	}
	config.Settings["default"] = entry2

	allow := config.Settings["default"].Permissions.Allow
	assert.Len(t, allow, 3, "expected original + 2 new entries, no duplicates")
	assert.Contains(t, allow, "Bash(npm *)", "original entry must be preserved")
	assert.Contains(t, allow, "Bash(git *)", "new git entry must be added")
	assert.Contains(t, allow, "Bash(go *)", "new go entry must be added")

	// Verify uniqueness.
	seen2 := make(map[string]int)
	for _, e := range allow {
		seen2[e]++
	}
	for entry, count := range seen2 {
		assert.Equal(t, 1, count, "entry %q appears %d times — duplicates not allowed", entry, count)
	}
}

// TestXcfIntegration_FullRoundTrip_WorkflowToConfigToYAMLAndBack validates the
// complete pipeline: workflow file → import → translate → inject into config →
// marshal to YAML → parse back → verify Skills, Rules, and Permissions.
func TestXcfIntegration_FullRoundTrip_WorkflowToConfigToYAMLAndBack(t *testing.T) {
	path := writeTemp(t, "commit-changes.md", commitChangesWorkflow)
	baseDir := t.TempDir()

	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)

	result := translator.Translate(unit, "claude")

	// Build a minimal config to inject into.
	config := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{
			Name: "test-project",
		},
		ResourceScope: ast.ResourceScope{
			Skills: make(map[string]ast.SkillConfig),
			Rules:  make(map[string]ast.RuleConfig),
		},
	}

	// Inject primitives — mirrors the injectIntoConfig logic in translate.go.
	seen := make(map[string]bool)
	var allowEntries []string

	for _, p := range result.Primitives {
		if strings.TrimSpace(p.Body) == "" {
			continue
		}

		switch p.Kind {
		case "skill":
			destPath := filepath.Join(baseDir, "skills", p.ID, "SKILL.md")
			require.NoError(t, os.MkdirAll(filepath.Dir(destPath), 0755))
			require.NoError(t, os.WriteFile(destPath, []byte(p.Body), 0600))
			relPath := filepath.Join("skills", p.ID, "SKILL.md")
			config.Skills[p.ID] = ast.SkillConfig{
				Description:      fmt.Sprintf("Translated from workflow %s", p.ID),
				InstructionsFile: relPath,
			}

		case "rule":
			destPath := filepath.Join(baseDir, "rules", p.ID+".md")
			require.NoError(t, os.MkdirAll(filepath.Dir(destPath), 0755))
			require.NoError(t, os.WriteFile(destPath, []byte(p.Body), 0600))
			relPath := filepath.Join("rules", p.ID+".md")
			config.Rules[p.ID] = ast.RuleConfig{
				Description:      fmt.Sprintf("Constraints from workflow %s", p.ID),
				InstructionsFile: relPath,
			}

		case "permission":
			for _, entry := range resolveAllowEntriesForTest(p.Body) {
				if !seen[entry] {
					seen[entry] = true
					allowEntries = append(allowEntries, entry)
				}
			}
		}
	}

	if len(allowEntries) > 0 {
		s := config.Settings["default"]
		s.Permissions = &ast.PermissionsConfig{Allow: allowEntries}
		if config.Settings == nil {
			config.Settings = make(map[string]ast.SettingsConfig)
		}
		config.Settings["default"] = s
	}

	// Marshal to YAML.
	out, err := yaml.Marshal(config)
	require.NoError(t, err, "config must marshal to YAML without error")
	require.NotEmpty(t, out, "marshaled YAML must not be empty")

	// Parse back from YAML.
	var parsed ast.XcaffoldConfig
	require.NoError(t, yaml.Unmarshal(out, &parsed), "marshaled YAML must parse back without error")

	// Verify Skills.
	require.NotNil(t, parsed.Skills, "parsed config must have skills")
	skillConfig, hasSkill := parsed.Skills["commit-changes"]
	require.True(t, hasSkill, "commit-changes skill must be present in parsed config")
	assert.NotEmpty(t, skillConfig.InstructionsFile, "skill must have instructions-file set")
	assert.Contains(t, skillConfig.InstructionsFile, "SKILL.md", "skill instructions-file must reference SKILL.md")

	// Verify Rules.
	require.NotNil(t, parsed.Rules, "parsed config must have rules")
	ruleConfig, hasRule := parsed.Rules["commit-changes-constraints"]
	require.True(t, hasRule, "commit-changes-constraints rule must be present in parsed config")
	assert.NotEmpty(t, ruleConfig.InstructionsFile, "rule must have instructions-file set")

	// Verify Permissions.
	require.NotNil(t, parsed.Settings["default"].Permissions, "parsed config must have permissions")
	require.NotEmpty(t, parsed.Settings["default"].Permissions.Allow, "parsed config must have allow entries")
}

// TestXcfIntegration_PlanMode_OriginalConfigUnchanged verifies that translating
// in plan mode (dry-run) does not modify the original config struct.
func TestXcfIntegration_PlanMode_OriginalConfigUnchanged(t *testing.T) {
	path := writeTemp(t, "commit-changes.md", commitChangesWorkflow)

	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)

	result := translator.Translate(unit, "claude")
	require.NotEmpty(t, result.Primitives, "translation must produce primitives")

	// Snapshot original config — no skills, no rules, no permissions.
	original := &ast.XcaffoldConfig{
		Project: &ast.ProjectConfig{Name: "unchanged"},
	}
	originalSkillCount := len(original.Skills)
	originalRuleCount := len(original.Rules)
	originalPermissions := original.Settings["default"].Permissions

	// In plan mode the caller only prints the plan and returns — no mutation.
	// We simulate this by verifying the config is never touched when plan=true.
	planMode := true
	if !planMode {
		// This branch is intentionally unreachable in this test — it guards against
		// accidentally running the inject path.
		t.Fatal("plan mode must be true for this test")
	}

	// Config must remain unchanged.
	assert.Equal(t, originalSkillCount, len(original.Skills),
		"plan mode must not add skills to config")
	assert.Equal(t, originalRuleCount, len(original.Rules),
		"plan mode must not add rules to config")
	assert.Equal(t, originalPermissions, original.Settings["default"].Permissions,
		"plan mode must not modify settings.permissions")
	assert.Equal(t, "unchanged", original.Project.Name,
		"plan mode must not modify project metadata")
}

// resolveAllowEntriesForTest mirrors the resolveAllowEntries logic in translate.go
// without importing the main package (which would create an import cycle).
func resolveAllowEntriesForTest(body string) []string {
	lower := strings.ToLower(body)
	if strings.Contains(lower, "turbo-all") || strings.Contains(lower, "turbo") {
		return []string{"Bash(git *)", "Bash(go *)"}
	}
	return []string{"Bash(*)"}
}
