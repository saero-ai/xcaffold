// Package optimizer implements the optimization-pass framework for xcaffold's
// translate pipeline. Passes transform a map of output file paths to their
// contents, returning a revised map and a slice of FidelityNotes describing
// any transformations applied.
package optimizer

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/renderer"
	"github.com/saero-ai/xcaffold/providers"
)

// RequiredPasses returns the mandatory leading passes for a target from its manifest.
func RequiredPasses(target string) []string {
	m, ok := providers.ManifestFor(target)
	if !ok {
		return nil
	}
	return m.RequiredPasses
}

// Optimizer holds the target and ordered list of user-requested passes.
type Optimizer struct {
	target     string
	userPasses []string
}

// New creates an Optimizer for the given target.
func New(target string) *Optimizer {
	return &Optimizer{target: target}
}

// AddPass appends a pass name to the user-requested list.
func (o *Optimizer) AddPass(name string) {
	o.userPasses = append(o.userPasses, name)
}

// PassOrder returns the full ordered pass list:
//  1. Required passes for the target (prepended in target order).
//  2. User passes in command-line order, with one swap: if extract-common
//     appears after inline-imports in the user list, they are swapped so that
//     extract-common always precedes inline-imports.
func (o *Optimizer) PassOrder() []string {
	required := RequiredPasses(o.target) // nil == no required passes

	// Copy user passes and apply the extract-common / inline-imports swap.
	user := make([]string, len(o.userPasses))
	copy(user, o.userPasses)

	extractIdx := -1
	inlineIdx := -1
	for i, p := range user {
		switch p {
		case "extract-common":
			extractIdx = i
		case "inline-imports":
			inlineIdx = i
		}
	}

	// Swap only when inline-imports comes before extract-common in the user list.
	if extractIdx != -1 && inlineIdx != -1 && inlineIdx < extractIdx {
		user[inlineIdx], user[extractIdx] = user[extractIdx], user[inlineIdx]
	}

	order := make([]string, 0, len(required)+len(user))
	order = append(order, required...)
	order = append(order, user...)
	return order
}

// Run executes all passes in PassOrder. Notes from all passes are accumulated.
// The first error from any pass aborts execution.
func (o *Optimizer) Run(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
	order := o.PassOrder()

	// Detect whether the extract-common / inline-imports swap was performed so
	// we can emit a single reorder note.
	extractIdx := -1
	inlineIdx := -1
	for i, p := range o.userPasses {
		switch p {
		case "extract-common":
			extractIdx = i
		case "inline-imports":
			inlineIdx = i
		}
	}
	swapped := extractIdx != -1 && inlineIdx != -1 && inlineIdx < extractIdx

	var allNotes []renderer.FidelityNote

	if swapped {
		allNotes = append(allNotes, renderer.FidelityNote{
			Level:      renderer.LevelInfo,
			Target:     o.target,
			Kind:       "optimizer",
			Code:       renderer.CodeOptimizerPassReordered,
			Reason:     "extract-common was reordered to precede inline-imports to satisfy dependency constraints",
			Mitigation: "declare extract-common before inline-imports to suppress this note",
		})
	}

	current := files
	for _, name := range order {
		applyFn, err := lookupPass(name)
		if err != nil {
			return nil, nil, fmt.Errorf("optimizer: unknown pass %q: %w", name, err)
		}
		var notes []renderer.FidelityNote
		current, notes, err = applyFn(current)
		if err != nil {
			return nil, nil, fmt.Errorf("optimizer: pass %q failed: %w", name, err)
		}
		allNotes = append(allNotes, notes...)
	}

	return current, allNotes, nil
}

// lookupPass returns the Apply function for the named pass.
func lookupPass(name string) (func(map[string]string) (map[string]string, []renderer.FidelityNote, error), error) {
	switch name {
	case "flatten-scopes":
		return ApplyFlattenScopes, nil
	case "inline-imports":
		return ApplyInlineImports, nil
	case "dedupe":
		return ApplyDedupe, nil
	case "extract-common":
		return ApplyExtractCommon, nil
	case "prune-unused":
		return ApplyPruneUnused, nil
	case "normalize-paths":
		return ApplyNormalizePaths, nil
	case "split-large-rules":
		// split-large-rules requires a budget; when called via Run it uses the
		// default budget for the target. Callers who need a custom budget should
		// call ApplySplitLargeRules directly.
		return func(files map[string]string) (map[string]string, []renderer.FidelityNote, error) {
			return ApplySplitLargeRules(files, Budget{Kind: BudgetKindNone})
		}, nil
	default:
		return nil, fmt.Errorf("no such pass")
	}
}
