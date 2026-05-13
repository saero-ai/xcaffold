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
// Both the resource name and field segments allow hyphens to support
// kebab-case resource identifiers and YAML keys (e.g. require-desc, allowed-tools).
var refPattern = regexp.MustCompile(`\$\{(\w+)\.([\w-]+)\.([\w-]+)\}`)
var varPattern = regexp.MustCompile(`\$\{var\.([a-zA-Z][_a-zA-Z0-9-]*)\}`)
var envPattern = regexp.MustCompile(`\$\{env\.([A-Z0-9_]+)\}`)

// ExpandVariables replaces ${var.*} and ${env.*} in the byte slice.
// It returns an error if a referenced variable is missing.
func ExpandVariables(data []byte, vars map[string]interface{}, envs map[string]string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	result, err := expandVarTokens(string(data), vars)
	if err != nil {
		return nil, err
	}
	result, err = expandEnvTokens(result, envs)
	if err != nil {
		return nil, err
	}
	return []byte(result), nil
}

// expandVarTokens replaces all ${var.*} tokens in s using the vars map.
func expandVarTokens(s string, vars map[string]interface{}) (string, error) {
	var lastErr error
	result := varPattern.ReplaceAllStringFunc(s, func(match string) string {
		if lastErr != nil {
			return match
		}
		submatches := varPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		varName := submatches[1]
		val, ok := vars[varName]
		if !ok {
			lastErr = fmt.Errorf("unresolved variable: ${var.%s}", varName)
			return match
		}
		replacement, err := varValueToString(varName, val)
		if err != nil {
			lastErr = err
			return match
		}
		return replacement
	})
	return result, lastErr
}

// varValueToString converts a variable value to its string representation.
func varValueToString(varName string, val interface{}) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case []interface{}:
		// Format as inline YAML list [a, b]
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = fmt.Sprintf("%v", item)
		}
		return "[" + strings.Join(items, ", ") + "]", nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal variable: ${var.%s}: %w", varName, err)
		}
		return string(b), nil
	}
}

// expandEnvTokens replaces all ${env.*} tokens in s using the envs map.
// If envs is nil, tokens are left unchanged (pass-1 behaviour).
func expandEnvTokens(s string, envs map[string]string) (string, error) {
	var lastErr error
	result := envPattern.ReplaceAllStringFunc(s, func(match string) string {
		if lastErr != nil {
			return match
		}
		submatches := envPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		envName := submatches[1]
		if envs == nil {
			return match // Skip resolution if envs map not provided (pass 1)
		}
		val, ok := envs[envName]
		if !ok {
			lastErr = fmt.Errorf("unresolved environment variable: ${env.%s}", envName)
			return match
		}
		return val
	})
	return result, lastErr
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
	// visited and inStack track DFS state for cycle detection.
	visited := map[resourceKey]bool{}
	inStack := map[resourceKey]bool{}

	kindIDs := collectKindIDs(config)
	for _, ki := range kindIDs {
		if err := resolveResource(config, ki, visited, inStack); err != nil {
			return err
		}
	}
	return nil
}

// collectKindIDs returns a slice of resourceKeys for every resource in config,
// grouped by kind. The order is deterministic within each kind (map iteration
// order is intentionally not sorted — resolution order within a kind does not
// matter because the DFS handles dependencies).
func collectKindIDs(config *ast.XcaffoldConfig) []resourceKey {
	var keys []resourceKey
	for id := range config.Agents {
		keys = append(keys, resourceKey{"agent", id})
	}
	for id := range config.Skills {
		keys = append(keys, resourceKey{"skill", id})
	}
	for id := range config.Rules {
		keys = append(keys, resourceKey{"rule", id})
	}
	for id := range config.Workflows {
		keys = append(keys, resourceKey{"workflow", id})
	}
	for id := range config.MCP {
		keys = append(keys, resourceKey{"mcp", id})
	}
	for id := range config.Policies {
		keys = append(keys, resourceKey{"policy", id})
	}
	for id := range config.Memory {
		keys = append(keys, resourceKey{"memory", id})
	}
	for id := range config.Contexts {
		keys = append(keys, resourceKey{"context", id})
	}
	for id := range config.Templates {
		keys = append(keys, resourceKey{"template", id})
	}
	for id := range config.Hooks {
		keys = append(keys, resourceKey{"hooks", id})
	}
	for id := range config.Settings {
		keys = append(keys, resourceKey{"settings", id})
	}
	return keys
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
		return resolveAgentFields(config, key.name)
	case "skill":
		return resolveSkillFields(config, key.name)
	case "rule":
		return resolveRuleFields(config, key.name)
	case "workflow":
		return resolveWorkflowFields(config, key.name)
	case "mcp":
		return resolveMCPFields(config, key.name)
	case "policy":
		return resolvePolicyFields(config, key.name)
	case "memory":
		return resolveMemoryFields(config, key.name)
	case "context":
		return resolveContextFields(config, key.name)
	case "template":
		return resolveTemplateFields(config, key.name)
	case "hooks":
		return resolveHookFields(config, key.name)
	case "settings":
		return resolveSettingsFields(config, key.name)
	}
	return nil
}

// resolveAgentFields resolves attribute references in a single agent.
func resolveAgentFields(config *ast.XcaffoldConfig, name string) error {
	agent, ok := config.Agents[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&agent).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Agents[name] = agent
	return nil
}

// resolveSkillFields resolves attribute references in a single skill.
func resolveSkillFields(config *ast.XcaffoldConfig, name string) error {
	skill, ok := config.Skills[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&skill).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Skills[name] = skill
	return nil
}

// resolveRuleFields resolves attribute references in a single rule.
func resolveRuleFields(config *ast.XcaffoldConfig, name string) error {
	rule, ok := config.Rules[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&rule).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Rules[name] = rule
	return nil
}

// resolveWorkflowFields resolves attribute references in a single workflow.
func resolveWorkflowFields(config *ast.XcaffoldConfig, name string) error {
	wf, ok := config.Workflows[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&wf).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Workflows[name] = wf
	return nil
}

// resolveMCPFields resolves attribute references in a single MCP config.
func resolveMCPFields(config *ast.XcaffoldConfig, name string) error {
	mcp, ok := config.MCP[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&mcp).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.MCP[name] = mcp
	return nil
}

// resolvePolicyFields resolves attribute references in a single policy.
func resolvePolicyFields(config *ast.XcaffoldConfig, name string) error {
	p, ok := config.Policies[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&p).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Policies[name] = p
	return nil
}

// resolveMemoryFields resolves attribute references in a single memory entry.
func resolveMemoryFields(config *ast.XcaffoldConfig, name string) error {
	m, ok := config.Memory[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&m).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Memory[name] = m
	return nil
}

// resolveContextFields resolves attribute references in a single context.
func resolveContextFields(config *ast.XcaffoldConfig, name string) error {
	ctx, ok := config.Contexts[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&ctx).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Contexts[name] = ctx
	return nil
}

// resolveTemplateFields resolves attribute references in a single template.
func resolveTemplateFields(config *ast.XcaffoldConfig, name string) error {
	tmpl, ok := config.Templates[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&tmpl).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Templates[name] = tmpl
	return nil
}

// resolveHookFields resolves attribute references in a single hook config.
func resolveHookFields(config *ast.XcaffoldConfig, name string) error {
	h, ok := config.Hooks[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&h).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Hooks[name] = h
	return nil
}

// resolveSettingsFields resolves attribute references in a single settings config.
func resolveSettingsFields(config *ast.XcaffoldConfig, name string) error {
	s, ok := config.Settings[name]
	if !ok {
		return nil
	}
	v := reflect.ValueOf(&s).Elem()
	if err := resolveStructFields(config, v); err != nil {
		return err
	}
	config.Settings[name] = s
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
	case "agent", "skill", "rule", "workflow", "mcp", "policy":
		return getCoreResourceValue(config, key)
	case "memory", "context", "template", "hooks", "settings":
		return getExtendedResourceValue(config, key)
	default:
		return reflect.Value{}, fmt.Errorf("attribute reference: unknown resource type %q", key.kind)
	}
}

// getCoreResourceValue resolves agent, skill, rule, workflow, mcp, and policy kinds.
func getCoreResourceValue(config *ast.XcaffoldConfig, key resourceKey) (reflect.Value, error) {
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
	default: // "policy"
		v, ok := config.Policies[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: policy %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	}
}

// getExtendedResourceValue resolves memory, context, template, hooks, and settings kinds.
func getExtendedResourceValue(config *ast.XcaffoldConfig, key resourceKey) (reflect.Value, error) {
	switch key.kind {
	case "memory":
		v, ok := config.Memory[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: memory %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	case "context":
		v, ok := config.Contexts[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: context %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	case "template":
		v, ok := config.Templates[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: template %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	case "hooks":
		v, ok := config.Hooks[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: hooks %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
	default: // "settings"
		v, ok := config.Settings[key.name]
		if !ok {
			return reflect.Value{}, fmt.Errorf("attribute reference: settings %q not found", key.name)
		}
		return reflect.ValueOf(v), nil
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
