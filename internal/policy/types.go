package policy

// PolicyConfig is the root structure of a parsed kind: policy .xcf file.
// It lives in internal/policy (not internal/ast): different
// kind: values own their own schemas and parsers.
type PolicyConfig struct {
	Kind        string          `yaml:"kind"`
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Severity    string          `yaml:"severity"`
	Target      string          `yaml:"target"`
	Match       PolicyMatch     `yaml:"match"`
	Require     []PolicyRequire `yaml:"require,omitempty"`
	Deny        []PolicyDeny    `yaml:"deny,omitempty"`
}

// PolicyMatch filters which resources this policy applies to.
// All conditions are AND-ed. Empty match block means apply to all.
type PolicyMatch struct {
	HasTool        string `yaml:"has_tool,omitempty"`
	HasField       string `yaml:"has_field,omitempty"`
	NameMatches    string `yaml:"name_matches,omitempty"`
	TargetIncludes string `yaml:"target_includes,omitempty"`
}

// PolicyRequire defines a field requirement that matched resources must satisfy.
type PolicyRequire struct {
	Field     string   `yaml:"field"`
	IsPresent *bool    `yaml:"is_present,omitempty"`
	MinLength *int     `yaml:"min_length,omitempty"`
	MaxCount  *int     `yaml:"max_count,omitempty"`
	OneOf     []string `yaml:"one_of,omitempty"`
}

// PolicyDeny defines content patterns that must NOT appear in compiled output.
type PolicyDeny struct {
	ContentMatches  string   `yaml:"content_matches,omitempty"`
	PathContains    string   `yaml:"path_contains,omitempty"`
	ContentContains []string `yaml:"content_contains,omitempty"`
}

// Severity represents the enforcement level of a policy violation.
type Severity int

const (
	// SeverityWarning reports a violation without blocking apply.
	SeverityWarning Severity = iota
	// SeverityError blocks apply when violations are found.
	SeverityError
)
