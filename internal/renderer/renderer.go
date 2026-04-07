// Package renderer defines the TargetRenderer interface and the Registry that
// maps target names to their renderer implementations. This is the extension
// point for adding new output targets (e.g. cursor, antigravity) without modifying
// the compiler core.
package renderer

import (
	"fmt"
	"sort"

	"github.com/saero-ai/xcaffold/internal/output"
)

// TargetRenderer renders a compiled file map for a specific target environment.
type TargetRenderer interface {
	// Target returns the canonical name of this renderer (e.g. "claude").
	Target() string

	// OutputDir returns the base output directory for this target
	// (e.g. ".claude", ".cursor/rules").
	OutputDir() string

	// Render takes a map of relative path → content and returns a compiler
	// Output containing the files as they should be written to disk.
	Render(files map[string]string) *output.Output
}

// Registry stores TargetRenderers keyed by their target name.
type Registry struct {
	renderers map[string]TargetRenderer
}

// NewRegistry returns an empty Registry ready for use.
func NewRegistry() *Registry {
	return &Registry{
		renderers: make(map[string]TargetRenderer),
	}
}

// Register adds tr to the registry. If a renderer with the same target name is
// already registered it is silently replaced.
func (r *Registry) Register(tr TargetRenderer) {
	r.renderers[tr.Target()] = tr
}

// Get returns the TargetRenderer registered for the given target name.
// It returns an error if no renderer has been registered for that target.
func (r *Registry) Get(target string) (TargetRenderer, error) {
	tr, ok := r.renderers[target]
	if !ok {
		return nil, fmt.Errorf("renderer: no renderer registered for target %q", target)
	}
	return tr, nil
}

// Targets returns a sorted list of all registered target names.
func (r *Registry) Targets() []string {
	names := make([]string, 0, len(r.renderers))
	for name := range r.renderers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
