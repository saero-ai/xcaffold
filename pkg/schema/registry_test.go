package schema

import (
	"reflect"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/ast"
)

// kindTypeMap maps kind names to their Go struct type for reflection.
var kindTypeMap = map[string]reflect.Type{
	"agent":     reflect.TypeOf(ast.AgentConfig{}),
	"skill":     reflect.TypeOf(ast.SkillConfig{}),
	"rule":      reflect.TypeOf(ast.RuleConfig{}),
	"workflow":  reflect.TypeOf(ast.WorkflowConfig{}),
	"mcp":       reflect.TypeOf(ast.MCPConfig{}),
	"policy":    reflect.TypeOf(ast.PolicyConfig{}),
	"blueprint": reflect.TypeOf(ast.BlueprintConfig{}),
	"memory":    reflect.TypeOf(ast.MemoryConfig{}),
	"context":   reflect.TypeOf(ast.ContextConfig{}),
	"settings":  reflect.TypeOf(ast.SettingsConfig{}),
	"hooks":     reflect.TypeOf(ast.NamedHookConfig{}),
	"template":  reflect.TypeOf(ast.TemplateConfig{}),
}

func TestRegistryMatchesStructs(t *testing.T) {
	for kind, ks := range Registry {
		t.Run(kind, func(t *testing.T) {
			structType, ok := kindTypeMap[kind]
			if !ok {
				t.Fatalf("no struct type mapped for kind %q", kind)
			}

			// Build a map of YAML keys from the struct's tags
			structFields := make(map[string]reflect.StructField)
			for i := 0; i < structType.NumField(); i++ {
				f := structType.Field(i)
				tag := f.Tag.Get("yaml")
				if tag == "" || tag == "-" {
					continue
				}
				key := strings.Split(tag, ",")[0]
				if key == "" {
					continue
				}
				structFields[key] = f
			}

			// Verify every registry field exists in the struct
			for _, regField := range ks.Fields {
				sf, ok := structFields[regField.YAMLKey]
				if !ok {
					t.Errorf("registry has field %q (yaml:%q) but struct %s does not",
						regField.Name, regField.YAMLKey, structType.Name())
					continue
				}

				// Verify Go type matches
				goType := typeString(sf.Type)
				if goType != regField.GoType {
					t.Errorf("field %q: Go type mismatch: struct=%q registry=%q",
						regField.YAMLKey, goType, regField.GoType)
				}
			}

			// Verify no struct fields are missing from the registry
			registryKeys := make(map[string]bool)
			for _, f := range ks.Fields {
				registryKeys[f.YAMLKey] = true
			}
			for yamlKey := range structFields {
				if !registryKeys[yamlKey] {
					t.Errorf("struct %s has field yaml:%q but registry does not",
						structType.Name(), yamlKey)
				}
			}
		})
	}
}

// typeString produces a Go type string matching the extractor's output format.
// It handles both built-in and named types (including type aliases like FlexStringSlice).
// Note: interface{} is normalized to "any" to match the extractor's terminology.
func typeString(t reflect.Type) string {
	// Preserve the named type if it has one (handles type aliases)
	if t.Name() != "" {
		return t.Name()
	}

	switch t.Kind() {
	case reflect.Ptr:
		return "*" + typeString(t.Elem())
	case reflect.Slice:
		return "[]" + typeString(t.Elem())
	case reflect.Map:
		return "map[" + typeString(t.Key()) + "]" + typeString(t.Elem())
	case reflect.Interface:
		// Empty interface{} is represented as "any" in the registry
		return "any"
	default:
		// Fallback for unnamed types
		return t.String()
	}
}
