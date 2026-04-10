package policy

import (
	"fmt"
	"regexp"
	"strings"
)

// EvalRequire checks a single PolicyRequire rule against a resource's field values.
// Returns any violations found. resourceKind and resourceID are used for messages.
func EvalRequire(resourceKind, resourceID string, req PolicyRequire, fields map[string]string) []Violation {
	val, _ := fields[req.Field]
	var viols []Violation

	if req.IsPresent != nil && *req.IsPresent {
		if val == "" {
			viols = append(viols, Violation{
				Target:   resourceKind,
				Resource: resourceID,
				Message:  fmt.Sprintf("field %q must be present and non-empty", req.Field),
			})
			return viols // further checks on an absent field are meaningless
		}
	}

	if req.MinLength != nil && len(val) < *req.MinLength {
		viols = append(viols, Violation{
			Target:   resourceKind,
			Resource: resourceID,
			Message:  fmt.Sprintf("field %q must have min_length %d (current: %d)", req.Field, *req.MinLength, len(val)),
		})
	}

	if req.MaxCount != nil {
		// MaxCount applies to list-type fields. The caller encodes the slice
		// length as a decimal string in fields[req.Field] so EvalRequire can
		// remain purely string-based. The engine sets this in evalAgentPolicy
		// for fields like "tools_count".
		if lenStr, ok := fields[req.Field+"_count"]; ok {
			var count int
			if _, err := fmt.Sscan(lenStr, &count); err == nil && count > *req.MaxCount {
				viols = append(viols, Violation{
					Target:   resourceKind,
					Resource: resourceID,
					Message:  fmt.Sprintf("field %q has %d items, max allowed is %d", req.Field, count, *req.MaxCount),
				})
			}
		}
	}

	if len(req.OneOf) > 0 {
		found := false
		for _, allowed := range req.OneOf {
			if val == allowed {
				found = true
				break
			}
		}
		if !found {
			viols = append(viols, Violation{
				Target:   resourceKind,
				Resource: resourceID,
				Message:  fmt.Sprintf("field %q value %q is not in approved list %v", req.Field, val, req.OneOf),
			})
		}
	}

	return viols
}

// EvalDeny checks a single PolicyDeny rule against the full compiled output file map.
// Returns any violations found.
func EvalDeny(policyName string, sev Severity, deny PolicyDeny, files map[string]string) []Violation {
	var viols []Violation

	for path, content := range files {
		// path_contains check
		if deny.PathContains != "" && strings.Contains(path, deny.PathContains) {
			viols = append(viols, Violation{
				Policy:   policyName,
				Severity: sev,
				Path:     path,
				Message:  fmt.Sprintf("output path contains forbidden string %q", deny.PathContains),
			})
		}

		// content_contains check (case-insensitive)
		lower := strings.ToLower(content)
		for _, forbidden := range deny.ContentContains {
			if strings.Contains(lower, strings.ToLower(forbidden)) {
				viols = append(viols, Violation{
					Policy:   policyName,
					Severity: sev,
					Path:     path,
					Message:  fmt.Sprintf("output content contains forbidden string %q", forbidden),
				})
			}
		}

		// content_matches regex check
		if deny.ContentMatches != "" {
			re, err := regexp.Compile(deny.ContentMatches)
			if err == nil && re.MatchString(content) {
				viols = append(viols, Violation{
					Policy:   policyName,
					Severity: sev,
					Path:     path,
					Message:  fmt.Sprintf("output content matches forbidden pattern %q", deny.ContentMatches),
				})
			}
		}
	}

	return viols
}
