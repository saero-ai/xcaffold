package importer

// Kind identifies the AST resource kind a file maps to.
type Kind string

const (
	KindAgent      Kind = "agent"
	KindSkill      Kind = "skill"
	KindSkillAsset Kind = "skill-asset"
	KindRule       Kind = "rule"
	KindMCP        Kind = "mcp"
	KindHook       Kind = "hook"
	KindSettings   Kind = "settings"
	KindMemory     Kind = "memory"
	KindWorkflow   Kind = "workflow"
	KindPolicy     Kind = "policy"
	KindUnknown    Kind = ""
)

// Layout describes how a provider stores a particular kind on disk.
type Layout int

const (
	FlatFile          Layout = iota // one file per resource
	DirectoryPerEntry               // one subdirectory per resource with canonical file
	StandaloneJSON                  // single JSON file holding all resources of one kind
	EmbeddedJSONKey                 // key inside a container JSON file
	InlineInParent                  // embedded inside another resource definition
	LayoutUnknown     Layout = -1
)

// KindMapping maps a file pattern to a Kind and Layout for one provider.
type KindMapping struct {
	Pattern   string
	Kind      Kind
	Layout    Layout
	Extension string
	JSONKey   string
}
