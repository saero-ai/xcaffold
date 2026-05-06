package resolver

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// refPattern matches ${resource_type.resource_name.field} tokens.
// The field segment allows hyphens to support kebab-case YAML keys (e.g. allowed-tools).
var refPattern = regexp.MustCompile(`\$\{(\w+)\.(\w+)\.([\w-]+)\}`)
var varPattern = regexp.MustCompile(`\$\{var\.([a-zA-Z][_a-zA-Z0-9-]*)\}`)
var envPattern = regexp.MustCompile(`\$\{env\.([A-Z0-9_]+)\}`)

// ExpandVariables replaces ${var.*} and ${env.*} in the byte slice.
// It returns an error if a referenced variable is missing.
func ExpandVariables(data []byte, vars map[string]interface{}, envs map[string]string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	result := string(data)
	var err error

	// Process varPattern matches
	result = varPattern.ReplaceAllStringFunc(result, func(s string) string {
		if err != nil {
			return s
		}
		submatches := varPattern.FindStringSubmatch(s)
		if len(submatches) < 2 {
			return s
		}
		varName := submatches[1]

		val, ok := vars[varName]
		if !ok {
			err = fmt.Errorf("unresolved variable: ${var.%s}", varName)
			return s
		}

		var replacement string
		switch v := val.(type) {
		case string:
			replacement = v
		case int:
			replacement = strconv.Itoa(v)
		case bool:
			replacement = strconv.FormatBool(v)
		case []interface{}:
			// Format as inline YAML list [a, b]
			var items []string
			for _, item := range v {
				items = append(items, fmt.Sprintf("%v", item))
			}
			replacement = "[" + strings.Join(items, ", ") + "]"
		default:
			b, marshalErr := json.Marshal(v)
			if marshalErr != nil {
				err = fmt.Errorf("failed to marshal variable: ${var.%s}: %w", varName, marshalErr)
				return s
			}
			replacement = string(b)
		}
		return replacement
	})

	if err != nil {
		return nil, err
	}

	// Process envPattern matches
	result = envPattern.ReplaceAllStringFunc(result, func(s string) string {
		if err != nil {
			return s
		}
		submatches := envPattern.FindStringSubmatch(s)
		if len(submatches) < 2 {
			return s
		}
		envName := submatches[1]

		val, ok := envs[envName]
		if !ok {
			err = fmt.Errorf("unresolved environment variable: ${env.%s}", envName)
			return s
		}
		return val
	})

	if err != nil {
		return nil, err
	}

	return []byte(result), nil
}

// resourceKey identifies a specific resource in the config.
type resourceKey struct {
	kind string
	name string
}

// ResolveAttributes walks all string and []string fields across all resource maps
// in the config, resolves ${resource_type.resource_name.field} tokens, detects
// cycles, and mutates the config in place.
func ResolveAttributes(config *ast.XcaffoldConfig) error {
	// Collect all resource refs to build a dependency graph for cycle detection.
	// visited and inStack track DFS state.
	visited := map[resourceKey]bool{}
	inStack := map[resourceKey]bool{}

	// Resolve all agents.
	for id := range config.Agents {
		key := resourceKey{"agent", id}
		if err := resolveResource(config, key, visited, inStack); err != nil {
			return err
		}
	}
	// Resolve all skills.
	for id := range config.Skills {
		key := resourceKey{"skill", id}
		if err := resolveResource(config, key, visited, inStack); err != nil {
			return err
		}
	}
	// Resolve all rules.
	for id := range config.Rules {
		key := resourceKey{"rule", id}
		if err := resolveResource(config, key, visited, inStack); err != nil {
			return err
		}
	}
	// Resolve all workflows.
	for id := range config.Workflows {
		key := resourceKey{"workflow", id}
		if err := resolveResource(config, key, visited, inStack); err != nil {
			return err
		}
	}
	// Resolve all MCP configs.
	for id := range config.MCP {
		key := resourceKey{"mcp", id}
		if err := resolveResource(config, key, visited, inStack); err != nil {
			return err
		}
	}

	return nil
}

// resolveResource resolves all attribute references in the resource identified by key,
// using DFS to detect circular references.
func resolveResource(config *ast.XcaffoldConfig, key resourceKey, visited, inStack map[resourceKey]bool) error {
	if visited[key] {
		return nil
	}
	if inStack[key] {
		return fmt.Errorf("circular reference detected involving %s.%s", key.kind, key.name)
	}

	inStack[key] = true

	// Get the struct value for this resource.
	val, err := getResourceValue(config, key)
	if err != nil {
		return err
	}

	// Find all references in this resource's string/[]string fields.
	refs, err := collectRefs(val)
	if err != nil {
		return err
	}

	// Resolve each dependency first (DFS).
	for _, dep := range refs {
		depKey := resourceKey{dep[0], dep[1]}
		if err := resolveResource(config, depKey, visited, inStack); err != nil {
			return err
		}
	}

	// Now resolve the fields in this resource.
	if err := resolveFields(config, key); err != nil {
		return err
	}

	delete(inStack, key)
	visited[key] = true
	return nil
}

// collectRefs returns unique (resourceType, resourceName) pairs referenced by the resource.
func collectRefs(val reflect.Value) ([][2]string, error) {
	seen := map[[2]string]bool{}
	var result [][2]string

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, nil
	}

	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		f := val.Field(i)
		switch f.Kind() {
		case reflect.String:
			for _, m := range refPattern.FindAllStringSubmatch(f.String(), -1) {
				k := [2]string{m[1], m[2]}
				if !seen[k] {
					seen[k] = true
					result = append(result, k)
				}
			}
		case reflect.Slice:
			if t.Field(i).Type.Elem().Kind() == reflect.String {
				for j := 0; j < f.Len(); j++ {
					elem := f.Index(j).String()
					for _, m := range refPattern.FindAllStringSubmatch(elem, -1) {
						k := [2]string{m[1], m[2]}
						if !seen[k] {
							seen[k] = true
							result = append(result, k)
						}
					}
				}
			}
		case reflect.Struct:
			if cl, ok := f.Interface().(ast.ClearableList); ok {
				for _, elem := range cl.Values {
					for _, m := range refPattern.FindAllStringSubmatch(elem, -1) {
						k := [2]string{m[1], m[2]}
						if !seen[k] {
							seen[k] = true
							result = append(result, k)
						}
					}
				}
			}
		}
	}
	return result, nil
}

// resolveFields replaces all ${...} tokens in the string/[]string fields of the
// named resource with their resolved values.
func resolveFields(config *ast.XcaffoldConfig, key resourceKey) error {
	// We need to operate on a copy then write back (maps store values, not pointers).
	switch key.kind {
	case "agent":
		agent, ok := config.Agents[key.name]
		if !ok {
			return nil
		}
		v := reflect.ValueOf(&agent).Elem()
		if err := resolveStructFields(config, v); err != nil {
			return err
		}
		config.Agents[key.name] = agent

	case "skill":
		skill, ok := config.Skills[key.name]
		if !ok {
			return nil
		}
		v := reflect.ValueOf(&skill).Elem()
		if err := resolveStructFields(config, v); err != nil {
			return err
		}
		config.Skills[key.name] = skill

	case "rule":
		rule, ok := config.Rules[key.name]
		if !ok {
			return nil
		}
		v := reflect.ValueOf(&rule).Elem()
		if err := resolveStructFields(config, v); err != nil {
			return err
		}
		config.Rules[key.name] = rule

	case "workflow":
		wf, ok := config.Workflows[key.name]
		if !ok {
			return nil
		}
		v := reflect.ValueOf(&wf).Elem()
		if err := resolveStructFields(config, v); err != nil {
			return err
		}
		config.Workflows[key.name] = wf

	case "mcp":
		mcp, ok := config.MCP[key.name]
		if !ok {
			return nil
		}
		v := reflect.ValueOf(&mcp).Elem()
		if err := resolveStructFields(config, v); err != nil {
			return err
		}
		config.MCP[key.name] = mcp
	}
	return nil
}

// resolveStructFields iterates over all string and []string fields in the struct
// pointed to by v and replaces ${...} tokens with resolved values.
func resolveStructFields(config *ast.XcaffoldConfig, v reflect.Value) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}
		switch field.Kind() {
		case reflect.String:
			resolved, err := interpolateString(config, field.String())
			if err != nil {
				return err
			}
			field.SetString(resolved)

		case reflect.Slice:
			if t.Field(i).Type.Elem().Kind() != reflect.String {
				continue
			}
			resolved, err := resolveStringSlice(config, field)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(resolved))
		case reflect.Struct:
			if _, ok := field.Interface().(ast.ClearableList); ok {
				valuesField := field.FieldByName("Values")
				if valuesField.IsValid() && valuesField.CanSet() {
					resolved, err := resolveStringSlice(config, valuesField)
					if err != nil {
						return err
					}
					valuesField.Set(reflect.ValueOf(resolved))
				}
			}
		}
	}
	return nil
}

// interpolateString replaces all ${...} tokens in s with their string values.
func interpolateString(config *ast.XcaffoldConfig, s string) (string, error) {
	var lastErr error
	result := refPattern.ReplaceAllStringFunc(s, func(match string) string {
		if lastErr != nil {
			return match
		}
		parts := refPattern.FindStringSubmatch(match)
		// parts: [full, resourceType, resourceName, fieldName]
		val, err := lookupValue(config, parts[1], parts[2], parts[3])
		if err != nil {
			lastErr = err
			return match
		}
		strVal, ok := val.(string)
		if !ok {
			lastErr = fmt.Errorf("attribute %s.%s.%s is not a string field; cannot interpolate into string", parts[1], parts[2], parts[3])
			return match
		}
		return strVal
	})
	if lastErr != nil {
		return "", lastErr
	}
	return result, nil
}

// resolveStringSlice resolves a []string field. If a single element is a bare
// ${...} reference to a []string field, it is expanded in-place.
func resolveStringSlice(config *ast.XcaffoldConfig, field reflect.Value) ([]string, error) {
	var result []string
	for i := 0; i < field.Len(); i++ {
		elem := field.Index(i).String()
		// Check if the entire element is a single reference token.
		if m := refPattern.FindStringSubmatch(elem); m != nil && m[0] == elem {
			val, err := lookupValue(config, m[1], m[2], m[3])
			if err != nil {
				return nil, err
			}
			switch v := val.(type) {
			case []string:
				result = append(result, v...)
			case string:
				result = append(result, v)
			default:
				return nil, fmt.Errorf("attribute %s.%s.%s has unsupported type for []string expansion", m[1], m[2], m[3])
			}
		} else {
			// Interpolate any embedded tokens as strings.
			resolved, err := interpolateString(config, elem)
			if err != nil {
				return nil, err
			}
			result = append(result, resolved)
		}
	}
	return result, nil
}

// lookupValue returns the value of fieldName on the resource identified by
// resourceType and resourceName, using yaml struct tag matching.
func lookupValue(config *ast.XcaffoldConfig, resourceType, resourceName, fieldName string) (interface{}, error) {
	resource, err := getResourceValue(config, resourceKey{resourceType, resourceName})
	if err != nil {
		return nil, err
	}
	return getFieldByYAMLTag(resource, fieldName)
}

// getResourceValue returns a reflect.Value for the resource identified by key.
func getResourceValue(config *ast.XcaffoldConfig, key resourceKey) (reflect.Value, error) {
	switch key.kind {
	case "agent":
		v, ok := config.Agents[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: agent %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	case "skill":
		v, ok := config.Skills[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: skill %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	case "rule":
		v, ok := config.Rules[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: rule %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	case "workflow":
		v, ok := config.Workflows[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: workflow %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	case "mcp":
		v, ok := config.MCP[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: mcp %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	default:
		return reflect.Value{}, fmt.Errorf("attribute reference: unknown resource type %q", key.kind)
	}
}

// getFieldByYAMLTag returns the value of the struct field whose yaml tag matches tag.
// Only the first segment of the yaml tag (before any comma) is compared.
func getFieldByYAMLTag(v reflect.Value, tag string) (interface{}, error) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, fmt.Errorf("nil pointer when looking up field %q", tag)
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", v.Kind())
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		yamlTag := t.Field(i).Tag.Get("yaml")
		name := strings.SplitN(yamlTag, ",", 2)[0]
		if name == "-" {
			continue
		}
		if name == tag {
			fval := v.Field(i).Interface()
			// Unwrap ClearableList to its Values slice for attribute resolution.
			if cl, ok := fval.(ast.ClearableList); ok {
				return cl.Values, nil
			}
			return fval, nil
		}
	}
	return nil, fmt.Errorf("attribute reference: field %q not found in resource", tag)
}
