package bir

// SourceKind names the xcf primitive that was analyzed.
type SourceKind string

const (
	SourceAgent    SourceKind = "agent"
	SourceSkill    SourceKind = "skill"
	SourceRule     SourceKind = "rule"
	SourceHook     SourceKind = "hook"
	SourceWorkflow SourceKind = "workflow"
)

// IntentType classifies the functional role of content within a SemanticUnit.
type IntentType string

const (
	// IntentConstraint covers directive language: MUST, NEVER, ALWAYS, etc. Maps to Rule.
	IntentConstraint IntentType = "constraint"
	// IntentProcedure covers sequential instructions: numbered steps or ## Steps heading. Maps to Skill.
	IntentProcedure IntentType = "procedure"
	// IntentAutomation covers auto-execution annotations: // turbo. Maps to Permission.
	IntentAutomation IntentType = "automation"
)

// FunctionalIntent is a detected semantic pattern within a SemanticUnit's body.
type FunctionalIntent struct {
	Type    IntentType
	Content string // extracted content for this intent
	Source  string // pattern that triggered detection
}

// ProjectIR is the root BIR for an entire .xcf compilation.
type ProjectIR struct {
	Units []SemanticUnit
}

// SemanticUnit is the IR for a single .xcf primitive.
// Phase 0: pass-through with ID, kind, and resolved body only.
// Phase 3+: populated with SourcePlatform, SourcePath, and Intents.
type SemanticUnit struct {
	ID             string
	SourceKind     SourceKind
	SourcePlatform string // "claude", "cursor", "gemini"
	SourcePath     string // file path for error messages
	ResolvedBody   string
	Intents        []FunctionalIntent
}
