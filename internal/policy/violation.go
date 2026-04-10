package policy

import "fmt"

// Violation records a single policy check failure.
type Violation struct {
	Policy   string
	Target   string
	Resource string // resource ID (agent name, skill name, etc.)
	Path     string // output file path (for deny checks)
	Message  string
	Severity Severity
}

// IsError returns true if this violation has error severity.
func (v Violation) IsError() bool {
	return v.Severity == SeverityError
}

// Format renders the violation as a human-readable string for stderr output.
func (v Violation) Format() string {
	sev := "warning"
	if v.Severity == SeverityError {
		sev = "error"
	}
	if v.Resource != "" {
		return fmt.Sprintf("POLICY VIOLATION [%s] %s\n  %s: %s\n  %s\n", sev, v.Policy, v.Target, v.Resource, v.Message)
	}
	if v.Path != "" {
		return fmt.Sprintf("POLICY VIOLATION [%s] %s\n  file: %s\n  %s\n", sev, v.Policy, v.Path, v.Message)
	}
	return fmt.Sprintf("POLICY VIOLATION [%s] %s\n  %s\n", sev, v.Policy, v.Message)
}
