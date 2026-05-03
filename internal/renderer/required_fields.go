package renderer

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/schema"
)

// CheckRequiredFields reads per-provider field requirements from the schema
// registry and returns error-level fidelity notes for any required fields
// that are missing from the resource.
func CheckRequiredFields(target, kind, resourceName string, presentFields map[string]string) []FidelityNote {
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
		if providerReq != "required" {
			continue
		}
		if _, present := presentFields[f.YAMLKey]; present {
			continue
		}
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
	return notes
}
