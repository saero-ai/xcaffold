package translator

import (
	"github.com/saero-ai/xcaffold/internal/bir"
)

// Translate decomposes a SemanticUnit into target platform primitives.
// Each detected intent maps to a distinct output artifact:
//   - IntentProcedure  → skill   (full resolved body)
//   - IntentConstraint → rule    (constraint lines, ID suffixed with "-constraints")
//   - IntentAutomation → permission (automation lines, ID suffixed with "-permissions")
//
// If the unit carries no intents, the entire body is emitted as a skill fallback.
// The target argument is reserved for future renderer selection and is unused here.
func Translate(unit *bir.SemanticUnit, target string) TranslationResult {
	if len(unit.Intents) == 0 {
		return TranslationResult{
			Primitives: []TargetPrimitive{
				{Kind: "skill", ID: unit.ID, Body: unit.ResolvedBody},
			},
		}
	}

	var primitives []TargetPrimitive

	for _, intent := range unit.Intents {
		switch intent.Type {
		case bir.IntentProcedure:
			primitives = append(primitives, TargetPrimitive{
				Kind: "skill",
				ID:   unit.ID,
				Body: unit.ResolvedBody,
			})

		case bir.IntentConstraint:
			primitives = append(primitives, TargetPrimitive{
				Kind: "rule",
				ID:   unit.ID + "-constraints",
				Body: intent.Content,
			})

		case bir.IntentAutomation:
			primitives = append(primitives, TargetPrimitive{
				Kind: "permission",
				ID:   unit.ID + "-permissions",
				Body: intent.Content,
			})
		}
	}

	return TranslationResult{Primitives: primitives}
}
