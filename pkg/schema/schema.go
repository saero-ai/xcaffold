package schema

import "sort"

// KindSchema holds all field metadata for a single xcaffold resource kind.
type KindSchema struct {
	Kind    string
	Version string
	Format  string // "frontmatter+body" or "pure-yaml"
	Fields  []Field
}

// Field describes a single field on a resource kind.
type Field struct {
	Name          string
	YAMLKey       string
	GoType        string
	XCFType       string // Human-readable type ([]string, map, etc.)
	Optional      bool
	Description   string
	Group         string
	Enum          []string
	Pattern       string
	ExclusiveWith []string
	Provider      map[string]string // provider name → behavior
	Default       string
	Example       string
}

// LookupKind retrieves schema metadata for a given kind name.
// Returns false if the kind is not in the registry.
func LookupKind(name string) (KindSchema, bool) {
	ks, ok := Registry[name]
	return ks, ok
}

// KindNames returns a sorted list of all registered kind names.
func KindNames() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// FieldSupportForTarget returns the provider support value for a specific field
// on a given kind. Returns "" if the kind or field is not found.
// Common return values: "optional", "required", "unsupported", "xcaffold-only".
func FieldSupportForTarget(kind, yamlKey, target string) string {
	ks, ok := Registry[kind]
	if !ok {
		return ""
	}
	for _, f := range ks.Fields {
		if f.YAMLKey == yamlKey {
			return f.Provider[target]
		}
	}
	return ""
}

// Registry is populated by the generated registry_gen.go file.
// It maps kind names to their schema metadata.
var Registry = map[string]KindSchema{}
