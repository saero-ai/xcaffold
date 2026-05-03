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
	ModelField        bool
	RuleActivations   []string // e.g., ["always", "path-glob"]
	RuleEncoding      RuleEncodingCapabilities
	SecurityFields    SecurityFieldSupport

	// AgentToolsField declares whether this provider supports a tools: field
	// in agent frontmatter. When false, the tools block is silently dropped.
	// When true, tools are emitted — but Claude-native tool names are validated
	// unless AgentNativeToolsOnly is also true.
	AgentToolsField bool

	// AgentNativeToolsOnly declares that this provider's native tool vocabulary
	// IS the Claude Core tool set. Only Claude should set this true. All other
	// providers that set AgentToolsField: true must validate against Claude-native names.
	AgentNativeToolsOnly bool
}
