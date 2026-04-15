package translator

// TargetPrimitive is one output artifact from translation.
type TargetPrimitive struct {
	Kind string // "skill", "rule", "permission", "workflow", "prompt-file", "custom-command"
	ID   string
	Body string // legacy body field; kept for backward compatibility

	// Content holds the full text of the artifact. Callers created by
	// TranslateWorkflow use Content; callers created by Translate use Body.
	// Renderers that write workflow primitives to disk should prefer Content
	// when non-empty, falling back to Body.
	Content string

	// Path is the output file path for primitives that carry their own
	// path (e.g. prompt-file, custom-command). Empty for primitives whose
	// path is determined by the renderer (skill, rule, workflow).
	Path string
}

// TranslationResult holds the output of translating a single SemanticUnit.
type TranslationResult struct {
	Primitives []TargetPrimitive
}
