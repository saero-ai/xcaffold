package optimizer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

// BudgetKindType names the unit of measurement for a size budget.
type BudgetKindType string

const (
	// BudgetKindNone means no budget constraint is in effect.
	BudgetKindNone BudgetKindType = ""

	// BudgetKindLines measures files in newline-delimited lines.
	BudgetKindLines BudgetKindType = "lines"

	// BudgetKindBytes measures files in raw byte count.
	BudgetKindBytes BudgetKindType = "bytes"

	// BudgetKindItems measures files in discrete items (e.g. rule entries).
	BudgetKindItems BudgetKindType = "items"
)

// Budget pairs a measurement kind with a numeric ceiling.
type Budget struct {
	Kind  BudgetKindType
	Value int
}

// ParseBudget parses a budget string of the form "<kind>:<value>".
// Supported kinds: lines, bytes, items.
// Token-based budgets ("tokens:N") are rejected because cross-provider token
// accounting is not deterministic; the returned error always contains the word "token".
func ParseBudget(s string) (Budget, error) {
	if strings.HasPrefix(s, "tokens:") {
		return Budget{}, fmt.Errorf("token-based budgets are not supported: use lines, bytes, or items instead of token limits")
	}

	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return Budget{}, fmt.Errorf("invalid budget %q: expected <kind>:<value>", s)
	}

	kindStr := parts[0]
	var value int
	if _, err := fmt.Sscanf(parts[1], "%d", &value); err != nil {
		return Budget{}, fmt.Errorf("invalid budget value in %q: %w", s, err)
	}
	if value < 0 {
		return Budget{}, fmt.Errorf("invalid budget value in %q: must be non-negative", s)
	}

	var kind BudgetKindType
	switch kindStr {
	case "lines":
		kind = BudgetKindLines
	case "bytes":
		kind = BudgetKindBytes
	case "items":
		kind = BudgetKindItems
	default:
		return Budget{}, fmt.Errorf("unknown budget kind %q in %q: use lines, bytes, or items", kindStr, s)
	}

	return Budget{Kind: kind, Value: value}, nil
}

// DefaultBudget returns the recommended per-target budget hint from the
// provider manifest. Unknown providers return BudgetKindNone.
func DefaultBudget(target string) Budget {
	m, ok := providers.ManifestFor(target)
	if !ok || m.BudgetKind == "" {
		return Budget{Kind: BudgetKindNone}
	}

	var kind BudgetKindType
	switch m.BudgetKind {
	case "lines":
		kind = BudgetKindLines
	case "bytes":
		kind = BudgetKindBytes
	case "items":
		kind = BudgetKindItems
	default:
		return Budget{Kind: BudgetKindNone}
	}

	return Budget{Kind: kind, Value: m.DefaultBudget}
}

// ---------------------------------------------------------------------------
// Pass implementations
// ---------------------------------------------------------------------------

// ApplyFlattenScopes merges files nested under a rules/ subdirectory into a
// single flat file keyed by "<parent>.md". Only groups with 2+ children are
// merged; single-child groups are left at their original path. Files outside
// rules/ or with fewer than 3 path segments are copied unchanged. Content
// within each merged file is joined in sorted key order with "\n\n---\n\n".
func ApplyFlattenScopes(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	out := make(map[string]string, len(files))
	groups := make(map[string][]string) // parent dir → child keys
	var notes []renderer.FidelityNote

	keys := sortedKeys(files)
	for _, k := range keys {
		normalized := filepath.ToSlash(k)
		parts := strings.Split(normalized, "/")
		if len(parts) >= 3 && isRulesPath(k) {
			parent := strings.Join(parts[:len(parts)-1], "/")
			groups[parent] = append(groups[parent], k)
		} else {
			out[k] = files[k]
		}
	}

	// Sort group keys for deterministic output order.
	groupKeys := make([]string, 0, len(groups))
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	sort.Strings(groupKeys)

	for _, parent := range groupKeys {
		children := groups[parent]
		if len(children) == 1 {
			// Single child: leave at its original path, no merge.
			out[children[0]] = files[children[0]]
			continue
		}
		sort.Strings(children)
		parts := make([]string, 0, len(children))
		for _, c := range children {
			parts = append(parts, files[c])
		}
		merged := strings.Join(parts, "\n\n---\n\n")
		flatKey := parent + ".md"
		out[flatKey] = merged
		notes = append(notes, renderer.FidelityNote{
			Level:    renderer.LevelInfo,
			Kind:     "optimizer",
			Resource: flatKey,
			Code:     "OPTIMIZER_FLATTEN_SCOPES",
			Reason:   fmt.Sprintf("merged %d nested files from %s/ into %s", len(children), parent, flatKey),
		})
	}
	return out, notes, nil
}

// ApplyInlineImports resolves "@import <path>" directives found on their own
// line within any file in the map. The referenced path must be an exact key in
// the input map. On a hit the directive line is replaced by the target file's
// content and a LevelInfo note is emitted. On a miss the line is left unchanged
// and a LevelWarning note is emitted identifying the missing target.
func ApplyInlineImports(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	out := copyFiles(files)
	var notes []renderer.FidelityNote

	for k, body := range out {
		lines := strings.Split(body, "\n")
		changed := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "@import ") {
				continue
			}
			target := strings.TrimSpace(strings.TrimPrefix(trimmed, "@import"))
			if content, ok := files[target]; ok {
				lines[i] = content
				changed = true
				notes = append(notes, renderer.FidelityNote{
					Level:    renderer.LevelInfo,
					Kind:     "optimizer",
					Resource: k,
					Code:     "OPTIMIZER_INLINE_IMPORT",
					Reason:   fmt.Sprintf("inlined @import %s into %s", target, k),
				})
			} else {
				notes = append(notes, renderer.FidelityNote{
					Level:      renderer.LevelWarning,
					Kind:       "optimizer",
					Resource:   k,
					Code:       "OPTIMIZER_INLINE_IMPORT_MISSING",
					Reason:     fmt.Sprintf("@import target %q not found in file map", target),
					Mitigation: "check the import path matches an existing file key",
				})
			}
		}
		if changed {
			out[k] = strings.Join(lines, "\n")
		}
	}

	return out, notes, nil
}

// ApplyDedupe removes files whose content is identical to a lexicographically
// earlier key. One LevelInfo note is emitted for each dropped file.
func ApplyDedupe(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	// Sort keys so determinism is guaranteed across map iteration order.
	keys := sortedKeys(files)

	// Track canonical key for each unique content body.
	seen := make(map[string]string, len(files)) // content → first key
	out := make(map[string]string, len(files))
	var notes []renderer.FidelityNote

	for _, k := range keys {
		body := files[k]
		if canon, dup := seen[body]; dup {
			notes = append(notes, renderer.FidelityNote{
				Level:      renderer.LevelInfo,
				Kind:       "optimizer",
				Resource:   k,
				Code:       "OPTIMIZER_DEDUPE",
				Reason:     fmt.Sprintf("file %q is identical to %q and was removed", k, canon),
				Mitigation: "remove the duplicate source or use a shared import",
			})
			continue
		}
		seen[body] = k
		out[k] = body
	}

	return out, notes, nil
}

// ApplySplitLargeRules splits any rules file (path contains "rules/") whose
// size exceeds the budget into two parts. The original key is replaced by
// "<stem>-part-1.md" and "<stem>-part-2.md". One LevelInfo note is emitted
// per split file. Only BudgetKindLines and BudgetKindBytes are evaluated;
// BudgetKindNone / BudgetKindItems leave files unchanged.
func ApplySplitLargeRules(files map[string]string, budget Budget) (map[string]string, []renderer.FidelityNote, error) {
	if budget.Kind != BudgetKindLines && budget.Kind != BudgetKindBytes {
		return copyFiles(files), nil, nil
	}

	out := make(map[string]string, len(files))
	var notes []renderer.FidelityNote

	for k, body := range files {
		if !isRulesPath(k) || !exceedsBudget(body, budget) {
			out[k] = body
			continue
		}

		part1, part2 := splitBody(body, budget)
		stem := rulesFileStem(k)
		dir := filepath.Dir(k)

		key1 := filepath.Join(dir, stem+"-part-1.md")
		key2 := filepath.Join(dir, stem+"-part-2.md")
		out[key1] = part1
		out[key2] = part2

		notes = append(notes, renderer.FidelityNote{
			Level:      renderer.LevelInfo,
			Kind:       "optimizer",
			Resource:   k,
			Code:       "OPTIMIZER_SPLIT_LARGE_RULE",
			Reason:     fmt.Sprintf("file %q exceeded the %s budget of %d and was split into two parts", k, budget.Kind, budget.Value),
			Mitigation: "reduce the rule body size or increase the budget",
		})
	}

	return out, notes, nil
}

// ApplyExtractCommon is a stub that returns the file map unchanged.
// Full implementation hoists repeated content blocks into a shared file.
func ApplyExtractCommon(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	out := copyFiles(files)
	return out, nil, nil
}

// ApplyPruneUnused is a stub that returns the file map unchanged.
// Full implementation removes compiled files no longer produced by any source resource.
func ApplyPruneUnused(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	out := copyFiles(files)
	return out, nil, nil
}

// ApplyNormalizePaths is a stub that returns the file map unchanged.
// Full implementation rewrites output paths to conform to target conventions.
func ApplyNormalizePaths(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	out := copyFiles(files)
	return out, nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func copyFiles(files map[string]string) map[string]string {
	out := make(map[string]string, len(files))
	for k, v := range files {
		out[k] = v
	}
	return out
}

func sortedKeys(files map[string]string) []string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// isRulesPath reports whether the file path sits inside a rules/ directory.
func isRulesPath(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "rules/")
}

// exceedsBudget reports whether body exceeds the budget limit.
func exceedsBudget(body string, budget Budget) bool {
	switch budget.Kind {
	case BudgetKindLines:
		return strings.Count(body, "\n") > budget.Value
	case BudgetKindBytes:
		return len(body) > budget.Value
	default:
		return false
	}
}

// splitBody divides body roughly in half by the budget unit.
func splitBody(body string, budget Budget) (string, string) {
	switch budget.Kind {
	case BudgetKindLines:
		lines := strings.Split(body, "\n")
		mid := len(lines) / 2
		return strings.Join(lines[:mid], "\n") + "\n",
			strings.Join(lines[mid:], "\n")
	case BudgetKindBytes:
		mid := len(body) / 2
		return body[:mid], body[mid:]
	default:
		return body, ""
	}
}

// rulesFileStem returns the filename without its .md extension.
func rulesFileStem(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
