package renderer

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/pkg/schema"
)

// CheckFieldSupport reads per-provider field requirements from the schema
// registry and returns error-level fidelity notes for:
//   - required fields that are missing from the resource
//   - unsupported fields that are present in the resource
//
// When suppressed is true, unsupported-field errors are skipped. Required-field
// errors are never suppressed because they indicate the compiled output will be
// rejected by the target provider.
func CheckFieldSupport(target, kind, resourceName string, presentFields map[string]string, suppressed bool) []FidelityNote {
	ks, ok := schema.LookupKind(kind)
	if !ok {
		return nil
	}

	var notes []FidelityNote
	for _, f := range ks.Fields {
		providerReq, exists := f.Provider[target]
		if !exists {
			continue
		}

		switch providerReq {
		case "required":
			if _, present := presentFields[f.YAMLKey]; !present {
				notes = append(notes, FidelityNote{
					Level:    LevelError,
					Target:   target,
					Kind:     kind,
					Resource: resourceName,
					Field:    f.YAMLKey,
					Code:     CodeFieldRequiredForTarget,
					Reason:   fmt.Sprintf("field %q is required by %s but missing from resource %q", f.YAMLKey, target, resourceName),
				})
			}
		case "unsupported":
			if suppressed {
				continue
			}
			if _, present := presentFields[f.YAMLKey]; present {
				if schema.HasRole(kind, f.YAMLKey) {
					continue
				}
				notes = append(notes, FidelityNote{
					Level:    LevelError,
					Target:   target,
					Kind:     kind,
					Resource: resourceName,
					Field:    f.YAMLKey,
					Code:     CodeFieldUnsupported,
					Reason:   fmt.Sprintf("field %q is unsupported by %s; use a %s.%s.xcaf override or remove from base manifest", f.YAMLKey, target, kind, target),
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
