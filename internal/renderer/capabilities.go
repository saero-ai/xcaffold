package renderer

// SecurityFieldSupport declares which per-agent and global security fields a
// renderer can faithfully emit. Fields set to false will be silently dropped
// at render time; securityFieldReport uses this to produce warnings/errors.
type SecurityFieldSupport struct {
	Permissions     bool
	Sandbox         bool
	PermissionMode  bool
	DisallowedTools bool
	Isolation       bool
	Effort          bool
}

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
	SecurityFields      SecurityFieldSupport
}
