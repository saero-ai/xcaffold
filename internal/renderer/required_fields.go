package renderer

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/pkg/schema"
)

// FieldCheckInput holds parameters for CheckFieldSupport.
type FieldCheckInput struct {
	Target        string
	Kind          string
	ResourceName  string
	PresentFields map[string]string
	Suppressed    bool
}

// CheckFieldSupport reads per-provider field requirements from the schema
// registry and returns error-level fidelity notes for:
//   - required fields that are missing from the resource
//   - unsupported fields that are present in the resource
//
// When input.Suppressed is true, unsupported-field errors are skipped.
// Required-field errors are never suppressed because they indicate the compiled
// output will be rejected by the target provider.
func CheckFieldSupport(input FieldCheckInput) []FidelityNote {
	ks, ok := schema.LookupKind(input.Kind)
	if !ok {
		return nil
	}

	var notes []FidelityNote
	for _, f := range ks.Fields {
		providerReq, exists := f.Provider[input.Target]
		if !exists {
			continue
		}

		switch providerReq {
		case "required":
			if _, present := input.PresentFields[f.YAMLKey]; !present {
				notes = append(notes, FidelityNote{
					Level:    LevelError,
					Target:   input.Target,
					Kind:     input.Kind,
					Resource: input.ResourceName,
					Field:    f.YAMLKey,
					Code:     CodeFieldRequiredForTarget,
					Reason:   fmt.Sprintf("field %q is required by %s but missing from resource %q", f.YAMLKey, input.Target, input.ResourceName),
				})
			}
		case "unsupported":
			if input.Suppressed {
				continue
			}
			if _, present := input.PresentFields[f.YAMLKey]; present {
				if schema.HasRole(input.Kind, f.YAMLKey) {
					continue
				}
				notes = append(notes, FidelityNote{
					Level:    LevelError,
					Target:   input.Target,
					Kind:     input.Kind,
					Resource: input.ResourceName,
					Field:    f.YAMLKey,
					Code:     CodeFieldUnsupported,
					Reason:   fmt.Sprintf("field %q is unsupported by %s; use a %s.%s.xcaf override or remove from base manifest", f.YAMLKey, input.Target, input.Kind, input.Target),
				})
			}
		}
	}
	return notes
}

// isSuppressed returns true when the resource's target override for the given
// provider has suppress-fidelity-warnings set to true.
func isSuppressed(targets map[string]ast.TargetOverride, target string) bool {
	if targets == nil {
		return false
	}
	to, ok := targets[target]
	if !ok {
		return false
	}
	return to.SuppressFidelityWarnings != nil && *to.SuppressFidelityWarnings
}
