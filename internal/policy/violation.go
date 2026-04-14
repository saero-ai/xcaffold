package policy

import (
	"fmt"
	"strings"
)

// Severity levels.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityOff     = "off"
)

// Violation represents a single policy check failure.
type Violation struct {
	PolicyName   string
	Severity     string
	ResourceName string // empty for output/settings targets
	FilePath     string // empty for non-output targets
	Message      string
}

// FormatViolations produces human-readable diagnostic output.
func FormatViolations(violations []Violation) string {
	var buf strings.Builder
	for _, v := range violations {
		fmt.Fprintf(&buf, "POLICY VIOLATION [%s] %s\n", v.Severity, v.PolicyName)
		if v.ResourceName != "" {
			fmt.Fprintf(&buf, "  resource: %s\n", v.ResourceName)
		}
		if v.FilePath != "" {
			fmt.Fprintf(&buf, "  file: %s\n", v.FilePath)
		}
		fmt.Fprintf(&buf, "  %s\n\n", v.Message)
	}
	return buf.String()
}

// FilterBySeverity returns only violations matching the given severity.
func FilterBySeverity(violations []Violation, severity string) []Violation {
	var filtered []Violation
	for _, v := range violations {
		if v.Severity == severity {
			filtered = append(filtered, v)
		}
	}
	return filtered
}
