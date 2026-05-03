package optimizer_test

import (
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/optimizer"
	"github.com/saero-ai/xcaffold/internal/renderer"
	_ "github.com/saero-ai/xcaffold/providers/antigravity"
	_ "github.com/saero-ai/xcaffold/providers/claude"
	_ "github.com/saero-ai/xcaffold/providers/copilot"
	_ "github.com/saero-ai/xcaffold/providers/cursor"
	_ "github.com/saero-ai/xcaffold/providers/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// PassOrder / Run ordering tests
// ---------------------------------------------------------------------------

// TestOptimizerRun_RequiredPassesRunFirst verifies that required passes for a
// given target are prepended before any user-requested passes.
func TestOptimizerRun_RequiredPassesRunFirst(t *testing.T) {
	o := optimizer.New("antigravity")
	o.AddPass("dedupe")

	order := o.PassOrder()

	// antigravity requires flatten-scopes then inline-imports first.
	require.GreaterOrEqual(t, len(order), 3, "expected at least 3 passes")
	assert.Equal(t, "flatten-scopes", order[0])
	assert.Equal(t, "inline-imports", order[1])
	assert.Equal(t, "dedupe", order[2])
}

// TestOptimizerRun_ExtractCommonBeforeInlineImports_Reordered verifies that
// when the user adds inline-imports before extract-common, the optimizer
// swaps them so extract-common precedes inline-imports.
func TestOptimizerRun_ExtractCommonBeforeInlineImports_Reordered(t *testing.T) {
	o := optimizer.New("claude") // no required passes for claude
	o.AddPass("inline-imports")
	o.AddPass("extract-common")

	order := o.PassOrder()

	extractIdx := -1
	inlineIdx := -1
	for i, p := range order {
		switch p {
		case "extract-common":
			extractIdx = i
		case "inline-imports":
			inlineIdx = i
		}
	}

	require.NotEqual(t, -1, extractIdx, "extract-common not found in order")
	require.NotEqual(t, -1, inlineIdx, "inline-imports not found in order")
	assert.Less(t, extractIdx, inlineIdx, "extract-common must precede inline-imports")
}

// ---------------------------------------------------------------------------
// ParseBudget tests
// ---------------------------------------------------------------------------

func TestParseBudget_Lines(t *testing.T) {
	b, err := optimizer.ParseBudget("lines:200")
	require.NoError(t, err)
	assert.Equal(t, optimizer.BudgetKindLines, b.Kind)
	assert.Equal(t, 200, b.Value)
}

func TestParseBudget_Bytes(t *testing.T) {
	b, err := optimizer.ParseBudget("bytes:12000")
	require.NoError(t, err)
	assert.Equal(t, optimizer.BudgetKindBytes, b.Kind)
	assert.Equal(t, 12000, b.Value)
}

func TestParseBudget_Items(t *testing.T) {
	b, err := optimizer.ParseBudget("items:50")
	require.NoError(t, err)
	assert.Equal(t, optimizer.BudgetKindItems, b.Kind)
	assert.Equal(t, 50, b.Value)
}

// TestParseBudget_TokensRejected verifies that token-based budgets are rejected
// at parse time with an error containing the word "token".
func TestParseBudget_TokensRejected(t *testing.T) {
	_, err := optimizer.ParseBudget("tokens:4096")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "token"),
		"error message must contain the word 'token', got: %s", err.Error())
}

// ---------------------------------------------------------------------------
// ApplySplitLargeRules tests
// ---------------------------------------------------------------------------

// TestSplitLargeRules_SplitsOnLinesBudget verifies that a rules file exceeding
// the line budget is split into two parts with a LevelInfo note emitted.
func TestSplitLargeRules_SplitsOnLinesBudget(t *testing.T) {
	// Build a file body with 10 lines.
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		sb.WriteString("line content\n")
	}
	body := sb.String()

	files := map[string]string{
		".claude/rules/big-rule.md": body,
	}

	budget := optimizer.Budget{Kind: optimizer.BudgetKindLines, Value: 5}
	out, notes, err := optimizer.ApplySplitLargeRules(files, budget)
	require.NoError(t, err)

	// Original key must be gone; split keys must exist.
	_, original := out[".claude/rules/big-rule.md"]
	assert.False(t, original, "original file should be replaced by split parts")

	_, part1 := out[".claude/rules/big-rule-part-1.md"]
	assert.True(t, part1, "part-1 split file must exist")

	_, part2 := out[".claude/rules/big-rule-part-2.md"]
	assert.True(t, part2, "part-2 split file must exist")

	// At least one LevelInfo note must be emitted.
	hasInfo := false
	for _, n := range notes {
		if n.Level == renderer.LevelInfo {
			hasInfo = true
			break
		}
	}
	assert.True(t, hasInfo, "expected at least one LevelInfo note for split")
}

// ---------------------------------------------------------------------------
// ApplyDedupe tests
// ---------------------------------------------------------------------------

// TestDedupe_RemovesDuplicateBodies verifies that files with identical content
// are deduplicated, keeping the lexicographically earlier key.
func TestDedupe_RemovesDuplicateBodies(t *testing.T) {
	files := map[string]string{
		"a.md": "same content",
		"b.md": "same content",
		"c.md": "unique content",
	}

	out, notes, err := optimizer.ApplyDedupe(files)
	require.NoError(t, err)

	// "a.md" is kept (lexicographically first); "b.md" is dropped.
	_, hasA := out["a.md"]
	_, hasB := out["b.md"]
	_, hasC := out["c.md"]

	assert.True(t, hasA, "a.md should be kept")
	assert.False(t, hasB, "b.md should be deduped away")
	assert.True(t, hasC, "c.md should be kept (unique)")

	// At least one LevelInfo note must be emitted for the dedup.
	hasNote := false
	for _, n := range notes {
		if n.Level == renderer.LevelInfo {
			hasNote = true
			break
		}
	}
	assert.True(t, hasNote, "expected at least one LevelInfo note for dedupe")
}

// ---------------------------------------------------------------------------
// DefaultBudget tests
// ---------------------------------------------------------------------------

// TestDefaultBudget_PerTarget verifies the documented defaults for each target.
func TestDefaultBudget_PerTarget(t *testing.T) {
	cases := []struct {
		target    string
		wantKind  optimizer.BudgetKindType
		wantValue int
	}{
		{"claude", optimizer.BudgetKindLines, 200},
		{"antigravity", optimizer.BudgetKindBytes, 12000},
		{"copilot", optimizer.BudgetKindBytes, 4000},
		{"cursor", optimizer.BudgetKindLines, 500},
		{"gemini", optimizer.BudgetKindNone, 0},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.target, func(t *testing.T) {
			b := optimizer.DefaultBudget(tc.target)
			assert.Equal(t, tc.wantKind, b.Kind, "kind mismatch for target %s", tc.target)
			if tc.wantKind != optimizer.BudgetKindNone {
				assert.Equal(t, tc.wantValue, b.Value, "value mismatch for target %s", tc.target)
			}
		})
	}
}
