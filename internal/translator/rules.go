package translator

// TargetPrimitive is one output artifact from translation.
type TargetPrimitive struct {
	Kind string // "skill", "rule", "permission"
	ID   string
	Body string
}

// TranslationResult holds the output of translating a single SemanticUnit.
type TranslationResult struct {
	Primitives []TargetPrimitive
}
