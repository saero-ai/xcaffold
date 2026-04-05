package bir_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectIntents_EmptyContent_ReturnsNone(t *testing.T) {
	intents := bir.DetectIntents("")
	assert.Empty(t, intents)
}

func TestDetectIntents_ProcedureFromNumberedSteps(t *testing.T) {
	content := `Do these things:
1. First step here
2. Second step here
3. Third step here`

	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentProcedure, intents[0].Type)
}

func TestDetectIntents_ProcedureFromStepsHeading(t *testing.T) {
	content := `# My Skill

## Steps

Do this first.
Then do that.`

	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentProcedure, intents[0].Type)
}

func TestDetectIntents_ConstraintFromMUST(t *testing.T) {
	content := "You MUST always check the input before processing."
	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentConstraint, intents[0].Type)
}

func TestDetectIntents_ConstraintFromNEVER(t *testing.T) {
	content := "NEVER expose credentials in log output."
	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentConstraint, intents[0].Type)
}

func TestDetectIntents_ConstraintFromALWAYS(t *testing.T) {
	content := "ALWAYS run tests before committing."
	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentConstraint, intents[0].Type)
}

func TestDetectIntents_ConstraintFromDONOT(t *testing.T) {
	content := "DO NOT modify files outside the assigned scope."
	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentConstraint, intents[0].Type)
}

func TestDetectIntents_ConstraintFromMANDATORY(t *testing.T) {
	content := "This is MANDATORY: run go vet before merging."
	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentConstraint, intents[0].Type)
}

func TestDetectIntents_ConstraintFromREQUIRED(t *testing.T) {
	content := "A valid token is REQUIRED to proceed."
	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentConstraint, intents[0].Type)
}

func TestDetectIntents_ConstraintCaseInsensitive(t *testing.T) {
	content := "must check the schema before writing."
	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentConstraint, intents[0].Type)
}

func TestDetectIntents_AutomationFromTurboComment(t *testing.T) {
	content := `# My Workflow
// turbo
Do this automatically.`

	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentAutomation, intents[0].Type)
}

func TestDetectIntents_MultipleIntents_ProcedureAndConstraint(t *testing.T) {
	content := `# Setup Rule

NEVER skip validation.

## Steps

1. Read the config
2. Validate schema
3. Write output`

	intents := bir.DetectIntents(content)
	assert.Len(t, intents, 2)

	types := make(map[bir.IntentType]bool)
	for _, intent := range intents {
		types[intent.Type] = true
	}
	assert.True(t, types[bir.IntentProcedure], "expected procedure intent")
	assert.True(t, types[bir.IntentConstraint], "expected constraint intent")
}

func TestDetectIntents_AllThreeIntents(t *testing.T) {
	content := `// turbo
MUST validate input.
1. Step one
2. Step two`

	intents := bir.DetectIntents(content)
	assert.Len(t, intents, 3)

	types := make(map[bir.IntentType]bool)
	for _, intent := range intents {
		types[intent.Type] = true
	}
	assert.True(t, types[bir.IntentProcedure])
	assert.True(t, types[bir.IntentConstraint])
	assert.True(t, types[bir.IntentAutomation])
}

func TestDetectIntents_PureProcedure_NoConstraint(t *testing.T) {
	content := `1. Run go build
2. Run go test
3. Run go vet`

	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Equal(t, bir.IntentProcedure, intents[0].Type)
}

func TestDetectIntents_ProcedureContent_ContainsNumberedLines(t *testing.T) {
	content := `1. First step
2. Second step
3. Third step`

	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Contains(t, intents[0].Content, "1. First step")
	assert.Contains(t, intents[0].Content, "2. Second step")
	assert.Contains(t, intents[0].Content, "3. Third step")
}

func TestDetectIntents_ConstraintContent_ContainsKeywordLines(t *testing.T) {
	content := `Normal line here.
NEVER expose tokens.
Another normal line.
ALWAYS run tests.`

	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Contains(t, intents[0].Content, "NEVER expose tokens.")
	assert.Contains(t, intents[0].Content, "ALWAYS run tests.")
	assert.NotContains(t, intents[0].Content, "Normal line here.")
	assert.NotContains(t, intents[0].Content, "Another normal line.")
}

func TestDetectIntents_AutomationContent_ContainsTurboLine(t *testing.T) {
	content := `Preamble text.
// turbo
More text.`

	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.Contains(t, intents[0].Content, "// turbo")
}

func TestDetectIntents_SourceField_DescribesPattern(t *testing.T) {
	content := "// turbo"
	intents := bir.DetectIntents(content)
	require.Len(t, intents, 1)
	assert.NotEmpty(t, intents[0].Source)
}

func TestDetectIntents_WordBoundary_MustNotMatchSubstring(t *testing.T) {
	// "mustang" should not trigger constraint detection for "must"
	content := "I ride a mustang."
	intents := bir.DetectIntents(content)
	assert.Empty(t, intents)
}

func TestDetectIntents_PlainProse_NoIntents(t *testing.T) {
	content := "This agent helps users find relevant documentation."
	intents := bir.DetectIntents(content)
	assert.Empty(t, intents)
}

func TestDetectIntents_ProcedurePreservesContextBetweenSteps(t *testing.T) {
	content := `## Steps

1. **Stage files**:
   Add specific files to the staging area.
   ` + "```bash" + `
   git add <file>
   ` + "```" + `

2. **Commit**:
   Use Conventional Commits.`

	intents := bir.DetectIntents(content)
	for _, i := range intents {
		if i.Type == bir.IntentProcedure {
			assert.Contains(t, i.Content, "Add specific files")
			assert.Contains(t, i.Content, "git add <file>")
			assert.Contains(t, i.Content, "Conventional Commits")
			return
		}
	}
	t.Fatal("No procedure intent found")
}

func TestDetectIntents_ProcedureWithStepsHeading_ExtractsFullSection(t *testing.T) {
	content := `# My Skill

Some preamble text here.

## Steps

1. First step
   This is the explanation for step one.

2. Second step
   This is the explanation for step two.

## Another Section

This should not be included.`

	intents := bir.DetectIntents(content)
	var procedureIntent *bir.FunctionalIntent
	for i := range intents {
		if intents[i].Type == bir.IntentProcedure {
			procedureIntent = &intents[i]
			break
		}
	}
	require.NotNil(t, procedureIntent, "expected a procedure intent")
	assert.Contains(t, procedureIntent.Content, "explanation for step one")
	assert.Contains(t, procedureIntent.Content, "explanation for step two")
	assert.NotContains(t, procedureIntent.Content, "This should not be included")
	assert.NotContains(t, procedureIntent.Content, "Some preamble text here")
}

func TestDetectIntents_ProcedureWithNumberedLines_NoStepsHeading_ExtractsFromFirstNumber(t *testing.T) {
	content := `Do these things:

1. First step here
   Explanation of step one.

2. Second step here
   Explanation of step two.`

	intents := bir.DetectIntents(content)
	var procedureIntent *bir.FunctionalIntent
	for i := range intents {
		if intents[i].Type == bir.IntentProcedure {
			procedureIntent = &intents[i]
			break
		}
	}
	require.NotNil(t, procedureIntent, "expected a procedure intent")
	assert.Contains(t, procedureIntent.Content, "Explanation of step one")
	assert.Contains(t, procedureIntent.Content, "Explanation of step two")
	assert.NotContains(t, procedureIntent.Content, "Do these things")
}
