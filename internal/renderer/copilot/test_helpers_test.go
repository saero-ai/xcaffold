package copilot_test

import "github.com/saero-ai/xcaffold/internal/renderer"

// filterNotes returns the subset of notes whose Code matches any of the given codes.
func filterNotes(notes []renderer.FidelityNote, codes ...string) []renderer.FidelityNote {
	set := make(map[string]bool, len(codes))
	for _, c := range codes {
		set[c] = true
	}
	var out []renderer.FidelityNote
	for _, n := range notes {
		if set[n.Code] {
			out = append(out, n)
		}
	}
	return out
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(b bool) *bool { return &b }
