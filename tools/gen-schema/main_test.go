package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/saero-ai/xcaffold/internal/schema"
)

// TestParseMarkers_Optional verifies that +xcf:optional marker sets Optional=true
func TestParseMarkers_Optional(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// Some description"},
			{Text: "// +xcf:optional"},
		},
	}

	markers := parseMarkers(comment)

	if !markers.Optional {
		t.Errorf("Expected Optional=true, got %v", markers.Optional)
	}
}

// TestParseMarkers_Required verifies that +xcf:required marker sets Required=true
func TestParseMarkers_Required(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// Some description"},
			{Text: "// +xcf:required"},
		},
	}

	markers := parseMarkers(comment)

	if !markers.Required {
		t.Errorf("Expected Required=true, got %v", markers.Required)
	}
}

// TestParseMarkers_Group verifies that +xcf:group=X marker sets Group correctly
func TestParseMarkers_Group(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// Configuration group"},
			{Text: "// +xcf:group=Identity"},
		},
	}

	markers := parseMarkers(comment)

	if markers.Group != "Identity" {
		t.Errorf("Expected Group='Identity', got '%s'", markers.Group)
	}
}

// TestParseMarkers_Enum verifies that +xcf:enum=a,b,c parses enum values
func TestParseMarkers_Enum(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// Model identifier"},
			{Text: "// +xcf:enum=sonnet,opus,haiku"},
		},
	}

	markers := parseMarkers(comment)

	expected := []string{"sonnet", "opus", "haiku"}
	if !reflect.DeepEqual(markers.Enum, expected) {
		t.Errorf("Expected Enum=%v, got %v", expected, markers.Enum)
	}
}

// TestParseMarkers_Provider verifies that +xcf:provider=provider:behavior parses provider map
func TestParseMarkers_Provider(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// Provider behavior"},
			{Text: "// +xcf:provider=cursor:ignored,gemini:pass-through"},
		},
	}

	markers := parseMarkers(comment)

	expected := map[string]string{
		"cursor": "ignored",
		"gemini": "pass-through",
	}
	if !reflect.DeepEqual(markers.Provider, expected) {
		t.Errorf("Expected Provider=%v, got %v", expected, markers.Provider)
	}
}

// TestParseMarkers_Combined verifies that multiple markers on separate lines are all parsed
func TestParseMarkers_Combined(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// Field description"},
			{Text: "// +xcf:optional"},
			{Text: "// +xcf:group=Configuration"},
			{Text: "// +xcf:enum=value1,value2"},
			{Text: "// +xcf:provider=claude:ignored"},
		},
	}

	markers := parseMarkers(comment)

	if !markers.Optional {
		t.Errorf("Expected Optional=true, got %v", markers.Optional)
	}
	if markers.Group != "Configuration" {
		t.Errorf("Expected Group='Configuration', got '%s'", markers.Group)
	}
	if len(markers.Enum) != 2 || markers.Enum[0] != "value1" {
		t.Errorf("Expected Enum=['value1', 'value2'], got %v", markers.Enum)
	}
	if markers.Provider["claude"] != "ignored" {
		t.Errorf("Expected Provider[claude]='ignored', got '%s'", markers.Provider["claude"])
	}
}

// TestExtractDescription verifies description extraction without marker lines
func TestExtractDescription(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// This is a description"},
			{Text: "// +xcf:optional"},
			{Text: "// More description"},
		},
	}

	desc := extractDescription(comment)

	expected := "This is a description More description"
	if desc != expected {
		t.Errorf("Expected description='%s', got '%s'", expected, desc)
	}
}

// TestExtractDescription_MultiLine verifies multi-line descriptions are joined with spaces
func TestExtractDescription_MultiLine(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// First line"},
			{Text: "// Second line"},
			{Text: "// Third line"},
		},
	}

	desc := extractDescription(comment)

	expected := "First line Second line Third line"
	if desc != expected {
		t.Errorf("Expected description='%s', got '%s'", expected, desc)
	}
}

// TestExtractYAMLTag_Simple verifies YAML tag extraction from struct field
func TestExtractYAMLTag_Simple(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "Simple tag with omitempty",
			tag:      "`yaml:\"name,omitempty\"`",
			expected: "name",
		},
		{
			name:     "Tag without omitempty",
			tag:      "`yaml:\"name\"`",
			expected: "name",
		},
		{
			name:     "Dash tag",
			tag:      "`yaml:\"-\"`",
			expected: "-",
		},
		{
			name:     "Inline tag",
			tag:      "`yaml:\",inline\"`",
			expected: "",
		},
		{
			name:     "Complex field with multiple tags",
			tag:      "`json:\"field\" yaml:\"field-name,omitempty\" xml:\"field\"`",
			expected: "field-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &ast.Field{
				Tag: &ast.BasicLit{
					Value: tt.tag,
				},
			}

			result := extractYAMLTag(field)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestTypeString_Ident verifies simple type conversion
func TestTypeString_Ident(t *testing.T) {
	expr := &ast.Ident{Name: "string"}
	result := typeString(expr)

	if result != "string" {
		t.Errorf("Expected 'string', got '%s'", result)
	}
}

// TestTypeString_Pointer verifies pointer type conversion
func TestTypeString_Pointer(t *testing.T) {
	expr := &ast.StarExpr{
		X: &ast.Ident{Name: "bool"},
	}
	result := typeString(expr)

	if result != "*bool" {
		t.Errorf("Expected '*bool', got '%s'", result)
	}
}

// TestTypeString_Slice verifies slice type conversion
func TestTypeString_Slice(t *testing.T) {
	expr := &ast.ArrayType{
		Elt: &ast.Ident{Name: "string"},
	}
	result := typeString(expr)

	if result != "[]string" {
		t.Errorf("Expected '[]string', got '%s'", result)
	}
}

// TestTypeString_Map verifies map type conversion
func TestTypeString_Map(t *testing.T) {
	expr := &ast.MapType{
		Key:   &ast.Ident{Name: "string"},
		Value: &ast.Ident{Name: "MCPConfig"},
	}
	result := typeString(expr)

	if result != "map[string]MCPConfig" {
		t.Errorf("Expected 'map[string]MCPConfig', got '%s'", result)
	}
}

// TestTypeString_Nested verifies nested type conversion (e.g., *[]string)
func TestTypeString_Nested(t *testing.T) {
	expr := &ast.StarExpr{
		X: &ast.ArrayType{
			Elt: &ast.Ident{Name: "string"},
		},
	}
	result := typeString(expr)

	if result != "*[]string" {
		t.Errorf("Expected '*[]string', got '%s'", result)
	}
}

// TestMapToXCFType uses table-driven tests for Go-to-XCF type mapping
func TestMapToXCFType(t *testing.T) {
	tests := []struct {
		goType   string
		expected string
	}{
		{goType: "*bool", expected: "boolean"},
		{goType: "*int", expected: "integer"},
		{goType: "string", expected: "string"},
		{goType: "[]string", expected: "[]string"},
		{goType: "[]int", expected: "[]int"},
		{goType: "map[string]string", expected: "map"},
		{goType: "map[x]y", expected: "map"},
		{goType: "FlexStringSlice", expected: "FlexStringSlice"},
		{goType: "*int64", expected: "integer"},
		{goType: "  *bool  ", expected: "boolean"},
		{goType: "*string", expected: "*string"},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			result := mapToXCFType(tt.goType)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestValidateMarkers_MissingDescription verifies violation when field lacks doc comment
func TestValidateMarkers_MissingDescription(t *testing.T) {
	src := `
package ast

type AgentConfig struct {
	Name string
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}

	pkg := &ast.Package{
		Name:  "ast",
		Files: map[string]*ast.File{"test.go": f},
	}

	violations := validateMarkers(pkg)

	if len(violations) == 0 {
		t.Error("Expected violations for missing description, got none")
	}

	foundMissing := false
	for _, v := range violations {
		if strings.Contains(v, "missing description") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Errorf("Expected violation about missing description, got: %v", violations)
	}
}

// TestValidateMarkers_MissingOptionalRequired verifies violation when field lacks +xcf:optional/required
func TestValidateMarkers_MissingOptionalRequired(t *testing.T) {
	src := `
package ast

type SkillConfig struct {
	// Description here but no marker
	Name string
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}

	pkg := &ast.Package{
		Name:  "ast",
		Files: map[string]*ast.File{"test.go": f},
	}

	violations := validateMarkers(pkg)

	foundViolation := false
	for _, v := range violations {
		if strings.Contains(v, "missing +xcf:optional or +xcf:required") {
			foundViolation = true
			break
		}
	}
	if !foundViolation {
		t.Errorf("Expected violation about missing +xcf:optional or +xcf:required, got: %v", violations)
	}
}

// TestValidateMarkers_MissingGroup verifies violation when field lacks +xcf:group
func TestValidateMarkers_MissingGroup(t *testing.T) {
	src := `
package ast

type RuleConfig struct {
	// A description field
	// +xcf:optional
	Name string
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}

	pkg := &ast.Package{
		Name:  "ast",
		Files: map[string]*ast.File{"test.go": f},
	}

	violations := validateMarkers(pkg)

	foundViolation := false
	for _, v := range violations {
		if strings.Contains(v, "missing +xcf:group") {
			foundViolation = true
			break
		}
	}
	if !foundViolation {
		t.Errorf("Expected violation about missing +xcf:group, got: %v", violations)
	}
}

// TestValidateMarkers_AllPresent verifies no violations when all required markers are present
func TestValidateMarkers_AllPresent(t *testing.T) {
	src := `
package ast

type WorkflowConfig struct {
	// Identifies the workflow name
	// +xcf:optional
	// +xcf:group=Metadata
	Name string
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}

	pkg := &ast.Package{
		Name:  "ast",
		Files: map[string]*ast.File{"test.go": f},
	}

	violations := validateMarkers(pkg)

	if len(violations) > 0 {
		t.Errorf("Expected no violations, got: %v", violations)
	}
}

// TestValidateMarkers_UnknownStruct verifies that unknown struct types are skipped
func TestValidateMarkers_UnknownStruct(t *testing.T) {
	src := `
package ast

type UnknownConfig struct {
	Name string
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}

	pkg := &ast.Package{
		Name:  "ast",
		Files: map[string]*ast.File{"test.go": f},
	}

	violations := validateMarkers(pkg)

	// Unknown structs should not generate violations
	if len(violations) > 0 {
		t.Errorf("Expected no violations for unknown struct, got: %v", violations)
	}
}

// TestExtractFields_SingleKind verifies field extraction for a single kind
func TestExtractFields_SingleKind(t *testing.T) {
	src := `
package ast

type MCPConfig struct {
	// The MCP server name
	// +xcf:required
	// +xcf:group=Metadata
	Name string ` + "`yaml:\"name\"`" + `
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}

	pkg := &ast.Package{
		Name:  "ast",
		Files: map[string]*ast.File{"test.go": f},
	}

	result := extractFields(pkg)

	fields, ok := result["mcp"]
	if !ok {
		t.Error("Expected 'mcp' key in result")
		return
	}

	if len(fields) == 0 {
		t.Error("Expected at least one field")
		return
	}

	field := fields[0]
	if field.Name != "Name" {
		t.Errorf("Expected Name='Name', got '%s'", field.Name)
	}
	if field.YAMLKey != "name" {
		t.Errorf("Expected YAMLKey='name', got '%s'", field.YAMLKey)
	}
	if field.GoType != "string" {
		t.Errorf("Expected GoType='string', got '%s'", field.GoType)
	}
	if !field.Markers.Required {
		t.Error("Expected Required=true")
	}
	if field.Markers.Group != "Metadata" {
		t.Errorf("Expected Group='Metadata', got '%s'", field.Markers.Group)
	}
}

// TestExtractFields_SkipDashTag verifies fields with yaml:"-" tag are skipped
func TestExtractFields_SkipDashTag(t *testing.T) {
	src := `
package ast

type PolicyConfig struct {
	// Should be included
	// +xcf:optional
	// +xcf:group=Core
	Name string ` + "`yaml:\"name\"`" + `

	// Should be skipped
	// +xcf:optional
	// +xcf:group=Internal
	Internal string ` + "`yaml:\"-\"`" + `
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test source: %v", err)
	}

	pkg := &ast.Package{
		Name:  "ast",
		Files: map[string]*ast.File{"test.go": f},
	}

	result := extractFields(pkg)

	fields, ok := result["policy"]
	if !ok {
		t.Error("Expected 'policy' key in result")
		return
	}

	if len(fields) != 1 {
		t.Errorf("Expected 1 field (dash-tag should be skipped), got %d", len(fields))
		return
	}

	if fields[0].Name != "Name" {
		t.Errorf("Expected first field to be 'Name', got '%s'", fields[0].Name)
	}
}

// TestExtractYAMLTag_NoTag verifies empty string is returned when field has no tag
func TestExtractYAMLTag_NoTag(t *testing.T) {
	field := &ast.Field{
		Tag: nil,
	}

	result := extractYAMLTag(field)

	if result != "" {
		t.Errorf("Expected empty string for no tag, got '%s'", result)
	}
}

// TestParseMarkers_NilComment verifies nil comment group returns empty MarkerSet
func TestParseMarkers_NilComment(t *testing.T) {
	markers := parseMarkers(nil)

	if markers.Optional || markers.Required {
		t.Error("Expected empty MarkerSet for nil comment")
	}
	if markers.Group != "" {
		t.Errorf("Expected empty Group, got '%s'", markers.Group)
	}
	if len(markers.Enum) > 0 {
		t.Errorf("Expected empty Enum, got %v", markers.Enum)
	}
	if len(markers.Provider) > 0 {
		t.Errorf("Expected empty Provider, got %v", markers.Provider)
	}
}

// TestParseMarkers_FieldType verifies +xcf:type=CustomType marker parsing
func TestParseMarkers_FieldType(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// Custom field type"},
			{Text: "// +xcf:type=FlexStringSlice"},
		},
	}

	markers := parseMarkers(comment)

	if markers.FieldType != "FlexStringSlice" {
		t.Errorf("Expected FieldType='FlexStringSlice', got '%s'", markers.FieldType)
	}
}

// TestExtractDescription_EmptyComment verifies empty description handling
func TestExtractDescription_EmptyComment(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// +xcf:optional"},
			{Text: "// +xcf:group=Test"},
		},
	}

	desc := extractDescription(comment)

	if desc != "" {
		t.Errorf("Expected empty description, got '%s'", desc)
	}
}

// TestTypeString_Complex verifies complex nested type conversion
func TestTypeString_Complex(t *testing.T) {
	// map[string]*int
	expr := &ast.MapType{
		Key: &ast.Ident{Name: "string"},
		Value: &ast.StarExpr{
			X: &ast.Ident{Name: "int"},
		},
	}

	result := typeString(expr)

	if result != "map[string]*int" {
		t.Errorf("Expected 'map[string]*int', got '%s'", result)
	}
}

// TestLookupKindForStruct verifies struct name to kind mapping
func TestLookupKindForStruct(t *testing.T) {
	tests := []struct {
		structName string
		expected   string
	}{
		{"AgentConfig", "agent"},
		{"SkillConfig", "skill"},
		{"RuleConfig", "rule"},
		{"WorkflowConfig", "workflow"},
		{"MCPConfig", "mcp"},
		{"PolicyConfig", "policy"},
		{"BlueprintConfig", "blueprint"},
		{"MemoryConfig", "memory"},
		{"ContextConfig", "context"},
		{"SettingsConfig", "settings"},
		{"NamedHookConfig", "hooks"},
		{"UnknownConfig", ""},
	}

	for _, tt := range tests {
		t.Run(tt.structName, func(t *testing.T) {
			result := lookupKindForStruct(tt.structName)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestParseMarkers_WhitespaceHandling verifies markers with extra whitespace are handled correctly
func TestParseMarkers_WhitespaceHandling(t *testing.T) {
	comment := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "//   Description with spaces   "},
			{Text: "//   +xcf:optional   "},
			{Text: "// +xcf:group=   Test   "},
		},
	}

	markers := parseMarkers(comment)

	if !markers.Optional {
		t.Error("Expected Optional=true even with whitespace")
	}
	// Group value includes the whitespace as provided
	if !strings.Contains(markers.Group, "Test") {
		t.Errorf("Expected Group to contain 'Test', got '%s'", markers.Group)
	}
}

// TestAgentProviderMarkers_DescriptionRequired verifies that description field
// has the correct provider markers in the schema registry.
func TestAgentProviderMarkers_DescriptionRequired(t *testing.T) {
	ks, ok := schema.LookupKind("agent")
	if !ok {
		t.Fatal("agent kind not in registry")
	}
	var descField schema.Field
	for _, f := range ks.Fields {
		if f.YAMLKey == "description" {
			descField = f
			break
		}
	}
	if descField.Provider == nil {
		t.Fatal("description field has no provider markers")
	}
	if descField.Provider["claude"] != "required" {
		t.Errorf("description: claude provider = %q, want %q", descField.Provider["claude"], "required")
	}
	if descField.Provider["gemini"] != "required" {
		t.Errorf("description: gemini provider = %q, want %q", descField.Provider["gemini"], "required")
	}
	if descField.Provider["copilot"] != "required" {
		t.Errorf("description: copilot provider = %q, want %q", descField.Provider["copilot"], "required")
	}
	if descField.Provider["cursor"] != "optional" {
		t.Errorf("description: cursor provider = %q, want %q", descField.Provider["cursor"], "optional")
	}
}

// TestSkillProviderMarkers_AllowedToolsProviders verifies that allowed-tools field
// has the correct provider markers in the schema registry.
func TestSkillProviderMarkers_AllowedToolsProviders(t *testing.T) {
	ks, ok := schema.LookupKind("skill")
	if !ok {
		t.Fatal("skill kind not in registry")
	}
	for _, f := range ks.Fields {
		if f.YAMLKey == "allowed-tools" {
			if f.Provider == nil {
				t.Fatal("allowed-tools has no provider markers")
			}
			if f.Provider["claude"] != "optional" {
				t.Errorf("allowed-tools: claude = %q, want %q", f.Provider["claude"], "optional")
			}
			if f.Provider["copilot"] != "optional" {
				t.Errorf("allowed-tools: copilot = %q, want %q", f.Provider["copilot"], "optional")
			}
			return
		}
	}
	t.Fatal("allowed-tools field not found in skill schema")
}

// TestRuleProviderMarkers_DescriptionProviders verifies that description field
// has the correct provider markers in the schema registry.
func TestRuleProviderMarkers_DescriptionProviders(t *testing.T) {
	ks, ok := schema.LookupKind("rule")
	if !ok {
		t.Fatal("rule kind not in registry")
	}
	for _, f := range ks.Fields {
		if f.YAMLKey == "description" {
			if f.Provider == nil {
				t.Fatal("description has no provider markers")
			}
			if f.Provider["claude"] != "optional" {
				t.Errorf("description: claude = %q, want %q", f.Provider["claude"], "optional")
			}
			if f.Provider["cursor"] != "optional" {
				t.Errorf("description: cursor = %q, want %q", f.Provider["cursor"], "optional")
			}
			return
		}
	}
	t.Fatal("description field not found in rule schema")
}

// TestMCPProviderMarkers verifies that MCP server payload fields have correct provider markers.
func TestMCPProviderMarkers(t *testing.T) {
	ks, ok := schema.LookupKind("mcp")
	if !ok {
		t.Fatal("mcp kind not in registry")
	}
	for _, f := range ks.Fields {
		if f.YAMLKey == "command" {
			if f.Provider == nil {
				t.Fatal("mcp command field has no provider markers")
			}
			if f.Provider["claude"] != "optional" {
				t.Errorf("command: claude = %q, want %q", f.Provider["claude"], "optional")
			}
			if f.Provider["gemini"] != "optional" {
				t.Errorf("command: gemini = %q, want %q", f.Provider["gemini"], "optional")
			}
			if f.Provider["copilot"] != "optional" {
				t.Errorf("command: copilot = %q, want %q", f.Provider["copilot"], "optional")
			}
			return
		}
	}
	t.Fatal("command field not found in mcp schema")
}

// TestWorkflowProviderMarkers verifies that workflow payload fields have correct provider markers.
func TestWorkflowProviderMarkers(t *testing.T) {
	ks, ok := schema.LookupKind("workflow")
	if !ok {
		t.Fatal("workflow kind not in registry")
	}
	for _, f := range ks.Fields {
		if f.YAMLKey == "description" {
			if f.Provider == nil {
				t.Fatal("workflow description field has no provider markers")
			}
			if f.Provider["antigravity"] != "optional" {
				t.Errorf("description: antigravity = %q, want %q", f.Provider["antigravity"], "optional")
			}
			return
		}
	}
	t.Fatal("description field not found in workflow schema")
}

// TestContextProviderMarkers verifies that context payload fields have correct provider markers.
func TestContextProviderMarkers(t *testing.T) {
	ks, ok := schema.LookupKind("context")
	if !ok {
		t.Fatal("context kind not in registry")
	}
	for _, f := range ks.Fields {
		if f.YAMLKey == "description" {
			if f.Provider == nil {
				t.Fatal("context description field has no provider markers")
			}
			if f.Provider["cursor"] != "optional" {
				t.Errorf("description: cursor = %q, want %q", f.Provider["cursor"], "optional")
			}
			return
		}
	}
	t.Fatal("description field not found in context schema")
}

// TestHooksProviderMarkers verifies that hooks events field has correct provider markers.
func TestHooksProviderMarkers(t *testing.T) {
	ks, ok := schema.LookupKind("hooks")
	if !ok {
		t.Fatal("hooks kind not in registry")
	}
	for _, f := range ks.Fields {
		if f.YAMLKey == "events" {
			if f.Provider == nil {
				t.Fatal("hooks events field has no provider markers")
			}
			if f.Provider["claude"] != "optional" {
				t.Errorf("events: claude = %q, want %q", f.Provider["claude"], "optional")
			}
			return
		}
	}
	t.Fatal("events field not found in hooks schema")
}

// TestGenSchema_ReadsFieldsYAML verifies that readFieldsYAML correctly parses
// mock fields.yaml files from a temporary directory structure.
func TestGenSchema_ReadsFieldsYAML(t *testing.T) {
	dir := t.TempDir()

	// Create mock renderer directories with fields.yaml
	claudeDir := dir + "/internal/renderer/claude"
	cursorDir := dir + "/internal/renderer/cursor"
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		t.Fatal(err)
	}

	claudeYAML := `provider: claude
version: "1.0"
kinds:
  agent:
    name: { support: xcaffold-only }
    description: { support: required }
  skill:
    name: { support: xcaffold-only }
    description: { support: optional }
`
	cursorYAML := `provider: cursor
version: "1.0"
kinds:
  agent:
    name: { support: xcaffold-only }
    description: { support: optional }
  skill:
    name: { support: xcaffold-only }
    description: { support: optional }
`
	if err := os.WriteFile(claudeDir+"/fields.yaml", []byte(claudeYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cursorDir+"/fields.yaml", []byte(cursorYAML), 0644); err != nil {
		t.Fatal(err)
	}

	data, err := readFieldsYAML(dir)
	if err != nil {
		t.Fatalf("readFieldsYAML failed: %v", err)
	}

	if len(data) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(data))
	}

	claude, ok := data["claude"]
	if !ok {
		t.Fatal("expected claude in result")
	}
	if claude.Provider != "claude" {
		t.Errorf("expected Provider='claude', got '%s'", claude.Provider)
	}
	if claude.Kinds["agent"]["description"].Support != "required" {
		t.Errorf("expected claude agent description=required, got '%s'",
			claude.Kinds["agent"]["description"].Support)
	}

	cursor, ok := data["cursor"]
	if !ok {
		t.Fatal("expected cursor in result")
	}
	if cursor.Kinds["agent"]["description"].Support != "optional" {
		t.Errorf("expected cursor agent description=optional, got '%s'",
			cursor.Kinds["agent"]["description"].Support)
	}
}

// TestGenSchema_CompletenessGate_MissingField verifies that validateFieldsYAML
// returns an error when a provider's fields.yaml is missing a canonical field.
func TestGenSchema_CompletenessGate_MissingField(t *testing.T) {
	canonicalFields := map[string][]FieldInfo{
		"agent": {
			{Name: "Name", YAMLKey: "name"},
			{Name: "Description", YAMLKey: "description"},
			{Name: "Model", YAMLKey: "model"},
		},
	}

	yamlData := map[string]FieldsYAML{
		"claude": {
			Provider: "claude",
			Version:  "1.0",
			Kinds: map[string]map[string]FieldDecl{
				"agent": {
					"name":        {Support: "xcaffold-only"},
					"description": {Support: "required"},
					// "model" is missing
				},
			},
		},
	}

	err := validateFieldsYAML(yamlData, canonicalFields)
	if err == nil {
		t.Fatal("expected error for missing field, got nil")
	}
	if !strings.Contains(err.Error(), "missing field") {
		t.Errorf("expected 'missing field' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "model") {
		t.Errorf("expected 'model' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("expected 'claude' in error, got: %v", err)
	}
}

// TestGenSchema_CompletenessGate_UnknownField verifies that validateFieldsYAML
// returns an error when a provider's fields.yaml contains a field not in the AST.
func TestGenSchema_CompletenessGate_UnknownField(t *testing.T) {
	canonicalFields := map[string][]FieldInfo{
		"agent": {
			{Name: "Name", YAMLKey: "name"},
			{Name: "Description", YAMLKey: "description"},
		},
	}

	yamlData := map[string]FieldsYAML{
		"gemini": {
			Provider: "gemini",
			Version:  "1.0",
			Kinds: map[string]map[string]FieldDecl{
				"agent": {
					"name":        {Support: "xcaffold-only"},
					"description": {Support: "required"},
					"bogus-field": {Support: "optional"},
				},
			},
		},
	}

	err := validateFieldsYAML(yamlData, canonicalFields)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("expected 'unknown field' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "bogus-field") {
		t.Errorf("expected 'bogus-field' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "gemini") {
		t.Errorf("expected 'gemini' in error, got: %v", err)
	}
}

// TestGenSchema_MergeProviderData verifies that mergeProviderData correctly
// merges YAML provider data into the fields map, with YAML taking precedence.
func TestGenSchema_MergeProviderData(t *testing.T) {
	fields := map[string][]FieldInfo{
		"agent": {
			{
				Name:    "Description",
				YAMLKey: "description",
				Markers: MarkerSet{
					Provider: map[string]string{
						"claude": "optional", // marker says optional
					},
				},
			},
		},
	}

	yamlData := map[string]FieldsYAML{
		"claude": {
			Provider: "claude",
			Kinds: map[string]map[string]FieldDecl{
				"agent": {
					"description": {Support: "required"}, // YAML says required
				},
			},
		},
		"cursor": {
			Provider: "cursor",
			Kinds: map[string]map[string]FieldDecl{
				"agent": {
					"description": {Support: "unsupported"},
				},
			},
		},
	}

	mergeProviderData(fields, yamlData)

	desc := fields["agent"][0]
	if desc.Markers.Provider["claude"] != "required" {
		t.Errorf("expected claude=required (YAML precedence), got '%s'",
			desc.Markers.Provider["claude"])
	}
	if desc.Markers.Provider["cursor"] != "unsupported" {
		t.Errorf("expected cursor=unsupported (new from YAML), got '%s'",
			desc.Markers.Provider["cursor"])
	}
}

// TestGenSchema_ValidateFieldsYAML_ValidInput verifies that valid fields.yaml
// data passes validation without errors.
func TestGenSchema_ValidateFieldsYAML_ValidInput(t *testing.T) {
	canonicalFields := map[string][]FieldInfo{
		"agent": {
			{Name: "Name", YAMLKey: "name"},
			{Name: "Description", YAMLKey: "description"},
		},
		"skill": {
			{Name: "Name", YAMLKey: "name"},
		},
	}

	yamlData := map[string]FieldsYAML{
		"claude": {
			Provider: "claude",
			Version:  "1.0",
			Kinds: map[string]map[string]FieldDecl{
				"agent": {
					"name":        {Support: "xcaffold-only"},
					"description": {Support: "required"},
				},
				"skill": {
					"name": {Support: "xcaffold-only"},
				},
			},
		},
	}

	err := validateFieldsYAML(yamlData, canonicalFields)
	if err != nil {
		t.Errorf("expected no error for valid input, got: %v", err)
	}
}

// TestGenSchema_ValidateFieldsYAML_SkipsNonOverlappingKinds verifies that
// kinds only present in canonical or only in YAML (but not both) are skipped.
func TestGenSchema_ValidateFieldsYAML_SkipsNonOverlappingKinds(t *testing.T) {
	canonicalFields := map[string][]FieldInfo{
		"agent": {{Name: "Name", YAMLKey: "name"}},
		"mcp":   {{Name: "Name", YAMLKey: "name"}}, // not in YAML
	}

	yamlData := map[string]FieldsYAML{
		"claude": {
			Provider: "claude",
			Kinds: map[string]map[string]FieldDecl{
				"agent": {
					"name": {Support: "xcaffold-only"},
				},
				// mcp not present in YAML — should not cause error
			},
		},
	}

	err := validateFieldsYAML(yamlData, canonicalFields)
	if err != nil {
		t.Errorf("expected no error when kinds don't overlap, got: %v", err)
	}
}
