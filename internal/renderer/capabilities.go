package renderer

// RuleEncodingCapabilities declares how the renderer encodes rule metadata.
type RuleEncodingCapabilities struct {
	// Description declares how the description field is encoded.
	//   "frontmatter" → inside YAML block
	//   "prose"       → first plain paragraph of body
	//   "omit"        → dropped entirely
	Description string

	// Activation declares how the rule applicability conditions are encoded.
	//   "frontmatter" → inside YAML block
	//   "omit"        → not encodable; emitting fidelity notes
	Activation string
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
	// SkillArtifactDirs maps canonical artifact names to provider output subdirectory
	// names. An empty string value means the artifact files are flattened to the
	// skill root directory alongside SKILL.md (no subdirectory created).
	SkillArtifactDirs map[string]string // canonical name → provider output subdir ("" = flatten to root)
	RuleActivations   []string          // e.g., ["always", "path-glob"]
	RuleEncoding      RuleEncodingCapabilities

	// AgentNativeToolsOnly declares that this provider's native tool vocabulary
	// IS the Claude Core tool set. Only Claude should set this true. All other
	// providers must validate against Claude-native names via the schema registry.
	AgentNativeToolsOnly bool
}
