package translator_test

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/saero-ai/xcaffold/internal/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUnit is a helper that builds a SemanticUnit with the given body and
// runs intent detection so each test starts from realistic input.
func newUnit(id, body string) *bir.SemanticUnit {
	return &bir.SemanticUnit{
		ID:             id,
		SourceKind:     bir.SourceWorkflow,
		SourcePlatform: "gemini",
		SourcePath:     "/tmp/" + id + ".md",
		ResolvedBody:   body,
		Intents:        bir.DetectIntents(body),
	}
}

// --- Intent routing ---

func TestTranslate_PureProcedure_EmitsSkillOnly(t *testing.T) {
	body := "## Steps\n1. Clone the repo\n2. Run tests"
	unit := newUnit("deploy", body)

	result := translator.Translate(unit, "claude")

	require.Len(t, result.Primitives, 1)
	p := result.Primitives[0]
	assert.Equal(t, "skill", p.Kind)
	assert.Equal(t, "deploy", p.ID)
}

func TestTranslate_ProcedureAndConstraint_EmitsSkillAndRule(t *testing.T) {
	body := "## Steps\n1. Deploy the service\n\nYou MUST validate inputs before deploying."
	unit := newUnit("deploy-safe", body)

	result := translator.Translate(unit, "claude")

	require.Len(t, result.Primitives, 2)

	kinds := make(map[string]translator.TargetPrimitive)
	for _, p := range result.Primitives {
		kinds[p.Kind] = p
	}

	skill, hasSkill := kinds["skill"]
	rule, hasRule := kinds["rule"]

	assert.True(t, hasSkill, "expected a skill primitive")
	assert.True(t, hasRule, "expected a rule primitive")
	assert.Equal(t, "deploy-safe", skill.ID)
	assert.Equal(t, "deploy-safe-constraints", rule.ID)
}

func TestTranslate_WithAutomation_IncludesPermission(t *testing.T) {
	body := "## Steps\n1. Run pipeline\n// turbo"
	unit := newUnit("pipeline", body)

	result := translator.Translate(unit, "claude")

	kinds := make(map[string]bool)
	for _, p := range result.Primitives {
		kinds[p.Kind] = true
	}

	assert.True(t, kinds["skill"], "expected skill primitive")
	assert.True(t, kinds["permission"], "expected permission primitive")
}

func TestTranslate_AllThreeIntents_EmitsThreePrimitives(t *testing.T) {
	body := "## Steps\n1. Build image\n\nYou MUST not push without approval.\n// turbo"
	unit := newUnit("full", body)

	result := translator.Translate(unit, "claude")

	require.Len(t, result.Primitives, 3, "expected skill, rule, and permission")

	kinds := make(map[string]bool)
	for _, p := range result.Primitives {
		kinds[p.Kind] = true
	}
	assert.True(t, kinds["skill"])
	assert.True(t, kinds["rule"])
	assert.True(t, kinds["permission"])
}

// --- Body content ---

func TestTranslate_SkillBody_IsFullResolvedBody(t *testing.T) {
	body := "## Steps\n1. Step one\n2. Step two"
	unit := newUnit("myjob", body)

	result := translator.Translate(unit, "claude")

	require.Len(t, result.Primitives, 1)
	assert.Equal(t, body, result.Primitives[0].Body)
}

func TestTranslate_RuleBody_ContainsConstraintLines(t *testing.T) {
	body := "## Steps\n1. Do work\n\nYou MUST sanitize inputs. You NEVER skip tests."
	unit := newUnit("constrained", body)

	result := translator.Translate(unit, "claude")

	var ruleBody string
	for _, p := range result.Primitives {
		if p.Kind == kindRuleStr {
			ruleBody = p.Body
		}
	}

	require.NotEmpty(t, ruleBody, "expected a rule body")
	assert.True(t, strings.Contains(ruleBody, "MUST") || strings.Contains(ruleBody, "NEVER"),
		"rule body should contain constraint lines, got: %s", ruleBody)
}

// --- Fallback ---

func TestTranslate_NoIntents_FallbackToSkillWithFullBody(t *testing.T) {
	body := "This is a plain description with no steps and no directives."
	unit := newUnit("plain", body)
	// Ensure no intents were detected.
	require.Empty(t, unit.Intents, "test requires a unit with no intents")

	result := translator.Translate(unit, "claude")

	require.Len(t, result.Primitives, 1)
	p := result.Primitives[0]
	assert.Equal(t, "skill", p.Kind)
	assert.Equal(t, "plain", p.ID)
	assert.Equal(t, body, p.Body)
}

// --- ID conventions ---

func TestTranslate_RuleIDHasConstraintsSuffix(t *testing.T) {
	body := "You MUST always commit with a message."
	unit := newUnit("commit-guide", body)

	result := translator.Translate(unit, "claude")

	for _, p := range result.Primitives {
		if p.Kind == kindRuleStr {
			assert.Equal(t, "commit-guide-constraints", p.ID)
			return
		}
	}
	t.Fatal("no rule primitive found")
}

func TestTranslate_PermissionIDHasPermissionsSuffix(t *testing.T) {
	body := "## Steps\n1. Deploy\n// turbo"
	unit := newUnit("auto-deploy", body)

	result := translator.Translate(unit, "claude")

	for _, p := range result.Primitives {
		if p.Kind == "permission" {
			assert.Equal(t, "auto-deploy-permissions", p.ID)
			return
		}
	}
	t.Fatal("no permission primitive found")
}
