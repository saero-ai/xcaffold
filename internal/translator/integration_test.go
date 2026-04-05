package translator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/saero-ai/xcaffold/internal/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// commitChangesWorkflow closely mirrors the real .agents/workflows/commit-changes.md.
// It contains all three intent signals:
//   - numbered steps          → IntentProcedure  → skill
//   - MUST / NEVER / MANDATORY → IntentConstraint  → rule
//   - // turbo-all            → IntentAutomation  → permission
const commitChangesWorkflow = `---
description: Committing changes to Git using Conventional Commits
---
# Workflow: Commit Changes

// turbo-all

## Prerequisites

1. **Check Documentation Consistency**: Verify docs are updated.
2. **Lint & Test**: You MUST run go fmt, go vet, go test before committing. NEVER commit without passing tests.

## Steps

1. **Check repository status**: git status
2. **Review changes**: git diff
3. **Group related changes**: Determine logical groups.
4. **Stage files**: git add specific files
5. **Commit**: Use Conventional Commits. MANDATORY ANTI-JARGON RULE: Keep bodies mechanical.
6. **Repeat** for each logical group.
7. **Verify clean tree**: git status
`

// tddWorkflow is intentionally minimal: numbered steps only, no constraint keywords,
// no turbo annotation — so ImportWorkflow should detect exactly one intent (procedure).
const tddWorkflow = `---
description: Test-Driven Development Loop
---
# TDD

1. Write test
2. Run test
3. Implement
`

// writeTemp writes content to a temporary file with the given name and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// indexPrimitives returns a map from Kind → TargetPrimitive for easy lookup.
func indexPrimitives(primitives []translator.TargetPrimitive) map[string]translator.TargetPrimitive {
	m := make(map[string]translator.TargetPrimitive, len(primitives))
	for _, p := range primitives {
		m[p.Kind] = p
	}
	return m
}

// TestEndToEnd_CommitChanges_DecomposesIntoThreePrimitives validates the full
// pipeline:  real workflow file → ImportWorkflow → Translate → 3 CC primitives.
func TestEndToEnd_CommitChanges_DecomposesIntoThreePrimitives(t *testing.T) {
	path := writeTemp(t, "commit-changes.md", commitChangesWorkflow)

	// Step 1: parse via ImportWorkflow.
	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)
	require.NotNil(t, unit)

	// Step 2: verify all three intents detected.
	require.Len(t, unit.Intents, 3, "expected procedure, constraint, and automation intents")

	intentTypes := make(map[bir.IntentType]bool)
	for _, intent := range unit.Intents {
		intentTypes[intent.Type] = true
	}
	assert.True(t, intentTypes[bir.IntentProcedure], "procedure intent not detected")
	assert.True(t, intentTypes[bir.IntentConstraint], "constraint intent not detected")
	assert.True(t, intentTypes[bir.IntentAutomation], "automation intent not detected")

	// Step 3: translate to Claude primitives.
	result := translator.Translate(unit, "claude")

	// Step 4: verify exactly three primitives produced.
	require.Len(t, result.Primitives, 3, "expected skill, rule, and permission primitives")

	idx := indexPrimitives(result.Primitives)

	_, hasSkill := idx["skill"]
	_, hasRule := idx["rule"]
	_, hasPermission := idx["permission"]

	assert.True(t, hasSkill, "skill primitive not produced")
	assert.True(t, hasRule, "rule primitive not produced")
	assert.True(t, hasPermission, "permission primitive not produced")
}

// TestEndToEnd_CommitChanges_SkillBodyContainsFullWorkflow verifies that the skill
// body is the full resolved workflow content (frontmatter stripped).
func TestEndToEnd_CommitChanges_SkillBodyContainsFullWorkflow(t *testing.T) {
	path := writeTemp(t, "commit-changes.md", commitChangesWorkflow)

	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)

	result := translator.Translate(unit, "claude")

	idx := indexPrimitives(result.Primitives)
	skill, ok := idx["skill"]
	require.True(t, ok, "skill primitive not found")

	// The resolved body is the full workflow with frontmatter stripped.
	// Key structural markers that must be present.
	assert.True(t, strings.Contains(skill.Body, "# Workflow: Commit Changes"),
		"skill body should contain the workflow heading")
	assert.True(t, strings.Contains(skill.Body, "## Steps"),
		"skill body should contain the Steps section")
	assert.True(t, strings.Contains(skill.Body, "git status"),
		"skill body should contain step content")
	assert.True(t, strings.Contains(skill.Body, "Conventional Commits"),
		"skill body should contain Conventional Commits reference")

	// Frontmatter must be stripped.
	assert.False(t, strings.Contains(skill.Body, "description:"),
		"skill body must not contain YAML frontmatter")
}

// TestEndToEnd_CommitChanges_RuleBodyContainsConstraintLines verifies that the rule
// primitive body contains the MUST/NEVER/MANDATORY directive lines.
func TestEndToEnd_CommitChanges_RuleBodyContainsConstraintLines(t *testing.T) {
	path := writeTemp(t, "commit-changes.md", commitChangesWorkflow)

	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)

	result := translator.Translate(unit, "claude")

	idx := indexPrimitives(result.Primitives)
	rule, ok := idx["rule"]
	require.True(t, ok, "rule primitive not found")

	assert.True(t, strings.Contains(rule.Body, "MUST"),
		"rule body should contain MUST constraint, got: %s", rule.Body)
	assert.True(t, strings.Contains(rule.Body, "NEVER"),
		"rule body should contain NEVER constraint, got: %s", rule.Body)
	assert.True(t, strings.Contains(rule.Body, "MANDATORY"),
		"rule body should contain MANDATORY constraint, got: %s", rule.Body)
}

// TestEndToEnd_CommitChanges_PermissionBodyContainsTurboAnnotation verifies that
// the permission primitive body captures the // turbo annotation line.
func TestEndToEnd_CommitChanges_PermissionBodyContainsTurboAnnotation(t *testing.T) {
	path := writeTemp(t, "commit-changes.md", commitChangesWorkflow)

	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)

	result := translator.Translate(unit, "claude")

	idx := indexPrimitives(result.Primitives)
	perm, ok := idx["permission"]
	require.True(t, ok, "permission primitive not found")

	assert.True(t, strings.Contains(perm.Body, "turbo"),
		"permission body should contain turbo annotation, got: %s", perm.Body)
}

// TestEndToEnd_CommitChanges_IDConventions verifies that IDs follow the
// -constraints and -permissions suffix conventions.
func TestEndToEnd_CommitChanges_IDConventions(t *testing.T) {
	path := writeTemp(t, "commit-changes.md", commitChangesWorkflow)

	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)

	result := translator.Translate(unit, "claude")

	for _, p := range result.Primitives {
		switch p.Kind {
		case "skill":
			assert.Equal(t, "commit-changes", p.ID,
				"skill ID should be the bare workflow filename stem")
		case "rule":
			assert.Equal(t, "commit-changes-constraints", p.ID,
				"rule ID should carry -constraints suffix")
		case "permission":
			assert.Equal(t, "commit-changes-permissions", p.ID,
				"permission ID should carry -permissions suffix")
		}
	}
}

// TestEndToEnd_TDD_EmitsSkillOnly validates that a simple procedural-only workflow
// produces exactly one skill primitive and nothing else.
func TestEndToEnd_TDD_EmitsSkillOnly(t *testing.T) {
	path := writeTemp(t, "tdd.md", tddWorkflow)

	unit, err := bir.ImportWorkflow(path, "gemini")
	require.NoError(t, err)

	// Verify intent detection: procedure only.
	require.Len(t, unit.Intents, 1, "TDD workflow should have exactly one intent")
	assert.Equal(t, bir.IntentProcedure, unit.Intents[0].Type,
		"the sole intent should be procedure")

	result := translator.Translate(unit, "claude")

	// Verify exactly one primitive: skill.
	require.Len(t, result.Primitives, 1, "TDD workflow should produce only a skill primitive")
	p := result.Primitives[0]
	assert.Equal(t, "skill", p.Kind)
	assert.Equal(t, "tdd", p.ID)

	// Rule and permission must not be present.
	idx := indexPrimitives(result.Primitives)
	_, hasRule := idx["rule"]
	_, hasPerm := idx["permission"]
	assert.False(t, hasRule, "TDD workflow must not produce a rule primitive")
	assert.False(t, hasPerm, "TDD workflow must not produce a permission primitive")
}
