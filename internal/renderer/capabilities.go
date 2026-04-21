package renderer

// CapabilitySet declares which resource kinds a renderer supports.
// The compiler orchestrator uses this to auto-emit RENDERER_KIND_UNSUPPORTED
// fidelity notes for unsupported kinds, eliminating silent drops.
type CapabilitySet struct {
	Agents              bool
	Skills              bool
	Rules               bool
	Workflows           bool
	Hooks               bool
	Settings            bool
	MCP                 bool
	Memory              bool
	ProjectInstructions bool
	SkillSubdirs        []string // e.g., ["references", "scripts", "assets"]
	ModelField          bool
	RuleActivations     []string // e.g., ["always", "path-glob"]
}
