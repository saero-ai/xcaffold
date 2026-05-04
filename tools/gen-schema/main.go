package main

import (
	"flag"
	"fmt"
	"go/ast"
	goformat "go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	outputPath     = flag.String("output", "", "output file path (default: stdout)")
	presenceOutput = flag.String("presence-output", "", "output file path for presence extractors")
	packageName    = flag.String("package", "schema", "generated package name")
	validateOnly   = flag.Bool("validate-only", false, "validate markers only, do not generate")
)

var kindStructMap = map[string]string{
	"agent":     "AgentConfig",
	"skill":     "SkillConfig",
	"rule":      "RuleConfig",
	"workflow":  "WorkflowConfig",
	"mcp":       "MCPConfig",
	"policy":    "PolicyConfig",
	"blueprint": "BlueprintConfig",
	"memory":    "MemoryConfig",
	"context":   "ContextConfig",
	"settings":  "SettingsConfig",
	"hooks":     "NamedHookConfig",
	"template":  "TemplateConfig",
}

var kindFormatMap = map[string]string{
	"agent":     "frontmatter+body",
	"skill":     "frontmatter+body",
	"rule":      "frontmatter+body",
	"workflow":  "frontmatter+body",
	"mcp":       "pure-yaml",
	"policy":    "pure-yaml",
	"blueprint": "pure-yaml",
	"memory":    "frontmatter+body",
	"context":   "frontmatter+body",
	"settings":  "pure-yaml",
	"hooks":     "pure-yaml",
	"template":  "frontmatter+body",
}

type MarkerSet struct {
	Optional  bool
	Required  bool
	Group     string
	Enum      []string
	Provider  map[string]string
	Key       string
	FieldType string
	Pattern   string
	Example   string
	Default   string
}

type FieldInfo struct {
	Name        string
	YAMLKey     string
	GoType      string
	Description string
	Markers     MarkerSet
}

// FieldsYAML represents a parsed fields.yaml file for a single provider.
type FieldsYAML struct {
	Provider string                          `yaml:"provider"`
	Version  string                          `yaml:"version"`
	Kinds    map[string]map[string]FieldDecl `yaml:"kinds"`
}

// FieldDecl represents a single field declaration inside fields.yaml.
type FieldDecl struct {
	Support string `yaml:"support"`
}

// readFieldsYAML globs fields.yaml files from both internal/renderer/*/fields.yaml
// and providers/*/fields.yaml under rootDir, parses each file, and returns a map
// keyed by provider name. Consolidated providers live under providers/; legacy
// providers remain under internal/renderer/ until fully migrated.
func readFieldsYAML(rootDir string) (map[string]FieldsYAML, error) {
	patterns := []string{
		filepath.Join(rootDir, "internal", "renderer", "*", "fields.yaml"),
		filepath.Join(rootDir, "providers", "*", "fields.yaml"),
	}

	var allMatches []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob fields.yaml: %w", err)
		}
		allMatches = append(allMatches, matches...)
	}

	result := make(map[string]FieldsYAML, len(allMatches))
	for _, path := range allMatches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		var fy FieldsYAML
		if err := yaml.Unmarshal(data, &fy); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		if fy.Provider == "" {
			return nil, fmt.Errorf("%s: missing provider field", path)
		}
		result[fy.Provider] = fy
	}

	return result, nil
}

// validateFieldsYAML checks completeness: every canonical field must appear
// in each provider's YAML (missing = error), and no unknown fields may appear
// (unknown = error). Also verifies that every provider declares all required
// kinds (agent, skill, rule).
func validateFieldsYAML(
	yamlData map[string]FieldsYAML,
	canonicalFields map[string][]FieldInfo,
) error {
	var errs []string

	requiredKinds := []string{"agent", "rule", "skill"}

	providers := sortedMapKeys(yamlData)
	for _, provName := range providers {
		fy := yamlData[provName]

		for _, kind := range requiredKinds {
			if _, ok := fy.Kinds[kind]; !ok {
				if _, canonical := canonicalFields[kind]; canonical {
					errs = append(errs, fmt.Sprintf(
						"provider %s missing kind %s entirely",
						provName, kind))
				}
			}
		}

		kinds := sortedMapKeys(fy.Kinds)
		for _, kind := range kinds {
			canonical, ok := canonicalFields[kind]
			if !ok {
				continue
			}
			errs = append(errs,
				validateKindFields(provName, kind, canonical, fy.Kinds[kind])...)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("fields.yaml validation errors:\n  %s",
			strings.Join(errs, "\n  "))
	}
	return nil
}

// validateKindFields checks field-level completeness for one provider+kind.
func validateKindFields(provName, kind string, canonical []FieldInfo, yamlFields map[string]FieldDecl) []string {
	var errs []string
	canonicalKeys := buildYAMLKeySet(canonical)

	for _, key := range sortedCanonicalKeys(canonical) {
		if _, found := yamlFields[key]; !found {
			errs = append(errs, fmt.Sprintf(
				"provider %s kind %s missing field %s", provName, kind, key))
		}
	}
	for _, key := range sortedMapKeys(yamlFields) {
		if !canonicalKeys[key] {
			errs = append(errs, fmt.Sprintf(
				"provider %s kind %s unknown field %s", provName, kind, key))
		}
	}
	return errs
}

// mergeProviderData merges YAML provider data into the fields map.
// YAML values take precedence over +xcf:provider= marker values.
func mergeProviderData(
	fields map[string][]FieldInfo,
	yamlData map[string]FieldsYAML,
) {
	for provName, fy := range yamlData {
		for kind, yamlFields := range fy.Kinds {
			fieldSlice, ok := fields[kind]
			if !ok {
				continue
			}
			for i := range fieldSlice {
				decl, found := yamlFields[fieldSlice[i].YAMLKey]
				if !found {
					continue
				}
				if fieldSlice[i].Markers.Provider == nil {
					fieldSlice[i].Markers.Provider = make(map[string]string)
				}
				fieldSlice[i].Markers.Provider[provName] = decl.Support
			}
		}
	}
}

// buildYAMLKeySet builds a set of YAML keys from canonical fields.
func buildYAMLKeySet(fields []FieldInfo) map[string]bool {
	set := make(map[string]bool, len(fields))
	for _, f := range fields {
		set[f.YAMLKey] = true
	}
	return set
}

// sortedCanonicalKeys returns YAML keys from fields in sorted order.
func sortedCanonicalKeys(fields []FieldInfo) []string {
	keys := make([]string, 0, len(fields))
	for _, f := range fields {
		keys = append(keys, f.YAMLKey)
	}
	sort.Strings(keys)
	return keys
}

// sortedMapKeys returns sorted keys from a string-keyed map.
func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func main() {
	flag.Parse()

	astDir := "internal/ast"
	if _, err := os.Stat(astDir); os.IsNotExist(err) {
		log.Fatalf("internal/ast directory not found: %v", err)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, astDir, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("failed to parse ast package: %v", err)
	}

	pkg, ok := pkgs["ast"]
	if !ok {
		log.Fatalf("ast package not found in %s", astDir)
	}

	if *validateOnly {
		violations := validateMarkers(pkg)
		if len(violations) > 0 {
			for _, v := range violations {
				fmt.Fprintf(os.Stderr, "%s\n", v)
			}
			os.Exit(1)
		}
		fmt.Println("Valid markers on all fields")
		return
	}

	fields := extractFields(pkg)

	yamlData, err := readFieldsYAML(".")
	if err != nil {
		log.Fatalf("failed to read fields.yaml: %v", err)
	}
	if len(yamlData) == 0 {
		log.Fatalf("no fields.yaml files found under internal/renderer/*/fields.yaml or providers/*/fields.yaml")
	}
	if err := validateFieldsYAML(yamlData, fields); err != nil {
		log.Fatalf("fields.yaml validation failed:\n%v", err)
	}
	mergeProviderData(fields, yamlData)

	raw := generateGo(*packageName, fields)
	formatted, err := goformat.Source([]byte(raw))
	if err != nil {
		log.Fatalf("gofmt generated code: %v", err)
	}

	if *outputPath != "" {
		if err := os.WriteFile(*outputPath, formatted, 0644); err != nil {
			log.Fatalf("failed to write output: %v", err)
		}
	} else {
		fmt.Print(string(formatted))
	}

	if *presenceOutput != "" {
		presenceRaw := generatePresenceExtractors(fields)
		presenceFormatted, err := goformat.Source([]byte(presenceRaw))
		if err != nil {
			log.Fatalf("gofmt presence extractors: %v", err)
		}
		if err := os.WriteFile(*presenceOutput, presenceFormatted, 0644); err != nil {
			log.Fatalf("failed to write presence output: %v", err)
		}
	}
}

func validateMarkers(pkg *ast.Package) []string {
	var violations []string

	filenames := sortedFileNames(pkg)
	for _, fname := range filenames {
		file := pkg.Files[fname]
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				kind := lookupKindForStruct(typeSpec.Name.Name)
				if kind == "" {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				for _, f := range structType.Fields.List {
					ymlTag := extractYAMLTag(f)
					if ymlTag == "-" {
						continue
					}

					if f.Doc == nil || len(f.Doc.List) == 0 {
						fieldName := getFieldName(f)
						violations = append(violations, fmt.Sprintf(
							"%s.%s: missing description", kind, fieldName))
						continue
					}

					markers := parseMarkers(f.Doc)
					if !markers.Optional && !markers.Required {
						fieldName := getFieldName(f)
						violations = append(violations, fmt.Sprintf(
							"%s.%s: missing +xcf:optional or +xcf:required", kind, fieldName))
					}

					if markers.Group == "" {
						fieldName := getFieldName(f)
						violations = append(violations, fmt.Sprintf(
							"%s.%s: missing +xcf:group=...", kind, fieldName))
					}
				}
			}
		}
	}

	return violations
}

func extractFields(pkg *ast.Package) map[string][]FieldInfo {
	result := make(map[string][]FieldInfo)

	filenames := sortedFileNames(pkg)
	for _, fname := range filenames {
		file := pkg.Files[fname]
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				kind := lookupKindForStruct(typeSpec.Name.Name)
				if kind == "" {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				var fields []FieldInfo
				for _, f := range structType.Fields.List {
					for _, name := range f.Names {
						ymlTag := extractYAMLTag(f)
						if ymlTag == "-" {
							continue
						}

						desc := ""
						if f.Doc != nil {
							desc = extractDescription(f.Doc)
						}

						markers := parseMarkers(f.Doc)

						fields = append(fields, FieldInfo{
							Name:        name.Name,
							YAMLKey:     ymlTag,
							GoType:      typeString(f.Type),
							Description: desc,
							Markers:     markers,
						})
					}
				}

				result[kind] = fields
			}
		}
	}

	return result
}

func lookupKindForStruct(structName string) string {
	for kind, struct_ := range kindStructMap {
		if struct_ == structName {
			return kind
		}
	}
	return ""
}

func extractYAMLTag(f *ast.Field) string {
	if f.Tag == nil {
		return ""
	}

	tag := f.Tag.Value
	tag = strings.Trim(tag, "`")

	for _, part := range strings.Split(tag, " ") {
		if strings.HasPrefix(part, "yaml:") {
			val := strings.TrimPrefix(part, "yaml:")
			val = strings.Trim(val, "\"")

			if idx := strings.Index(val, ","); idx != -1 {
				return val[:idx]
			}
			return val
		}
	}
	return ""
}

func extractDescription(comment *ast.CommentGroup) string {
	var lines []string
	for _, c := range comment.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimSpace(text)

		if !strings.HasPrefix(text, "+xcf:") {
			lines = append(lines, text)
		}
	}

	desc := strings.Join(lines, " ")
	return strings.TrimSpace(desc)
}

func parseMarkers(comment *ast.CommentGroup) MarkerSet {
	result := MarkerSet{
		Provider: make(map[string]string),
	}

	if comment == nil {
		return result
	}

	for _, c := range comment.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimSpace(text)

		if !strings.HasPrefix(text, "+xcf:") {
			continue
		}

		marker := strings.TrimPrefix(text, "+xcf:")

		if marker == "optional" {
			result.Optional = true
		} else if marker == "required" {
			result.Required = true
		} else if strings.HasPrefix(marker, "group=") {
			result.Group = strings.TrimPrefix(marker, "group=")
		} else if strings.HasPrefix(marker, "enum=") {
			enumStr := strings.TrimPrefix(marker, "enum=")
			result.Enum = strings.Split(enumStr, ",")
		} else if strings.HasPrefix(marker, "provider=") {
			provStr := strings.TrimPrefix(marker, "provider=")
			for _, kv := range strings.Split(provStr, ",") {
				parts := strings.Split(kv, ":")
				if len(parts) == 2 {
					result.Provider[parts[0]] = parts[1]
				}
			}
		} else if strings.HasPrefix(marker, "type=") {
			result.FieldType = strings.TrimPrefix(marker, "type=")
		} else if strings.HasPrefix(marker, "pattern=") {
			result.Pattern = strings.TrimPrefix(marker, "pattern=")
		} else if strings.HasPrefix(marker, "example=") {
			result.Example = strings.TrimPrefix(marker, "example=")
		} else if strings.HasPrefix(marker, "default=") {
			result.Default = strings.TrimPrefix(marker, "default=")
		}
	}

	return result
}

func sortedFileNames(pkg *ast.Package) []string {
	names := make([]string, 0, len(pkg.Files))
	for name := range pkg.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func getFieldName(f *ast.Field) string {
	if len(f.Names) > 0 {
		return f.Names[0].Name
	}
	return ""
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	default:
		return ""
	}
}

func generateGo(pkgName string, fields map[string][]FieldInfo) string {
	var buf strings.Builder

	buf.WriteString("// Code generated by gen-schema; DO NOT EDIT.\n\n")
	buf.WriteString("package " + pkgName + "\n\n")
	buf.WriteString("func init() {\n")

	kinds := make([]string, 0, len(kindStructMap))
	for k := range kindStructMap {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)

	for _, kind := range kinds {
		if _, ok := fields[kind]; !ok {
			continue
		}

		buf.WriteString(fmt.Sprintf("\tRegistry[\"%s\"] = KindSchema{\n", kind))
		buf.WriteString(fmt.Sprintf("\t\tKind: \"%s\",\n", kind))
		buf.WriteString("\t\tVersion: \"1.0\",\n")
		buf.WriteString(fmt.Sprintf("\t\tFormat: \"%s\",\n", kindFormatMap[kind]))
		buf.WriteString("\t\tFields: []Field{\n")

		for _, f := range fields[kind] {
			writeFieldEntry(&buf, f)
		}

		buf.WriteString("\t\t},\n")
		buf.WriteString("\t}\n")
	}

	buf.WriteString("}\n")
	return buf.String()
}

func writeFieldEntry(buf *strings.Builder, f FieldInfo) {
	buf.WriteString("\t\t\t{\n")
	buf.WriteString(fmt.Sprintf("\t\t\t\tName: \"%s\",\n", f.Name))
	buf.WriteString(fmt.Sprintf("\t\t\t\tYAMLKey: \"%s\",\n", f.YAMLKey))
	buf.WriteString(fmt.Sprintf("\t\t\t\tGoType: \"%s\",\n", f.GoType))
	xcfType := f.Markers.FieldType
	if xcfType == "" {
		xcfType = mapToXCFType(f.GoType)
	}
	buf.WriteString(fmt.Sprintf("\t\t\t\tXCFType: \"%s\",\n", xcfType))
	buf.WriteString(fmt.Sprintf("\t\t\t\tOptional: %v,\n", f.Markers.Optional))
	buf.WriteString(fmt.Sprintf("\t\t\t\tDescription: %q,\n", f.Description))
	buf.WriteString(fmt.Sprintf("\t\t\t\tGroup: \"%s\",\n", f.Markers.Group))

	if len(f.Markers.Enum) > 0 {
		buf.WriteString("\t\t\t\tEnum: []string{")
		for i, e := range f.Markers.Enum {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("%q", strings.TrimSpace(e)))
		}
		buf.WriteString("},\n")
	}

	if len(f.Markers.Provider) > 0 {
		provKeys := make([]string, 0, len(f.Markers.Provider))
		for k := range f.Markers.Provider {
			provKeys = append(provKeys, k)
		}
		sort.Strings(provKeys)
		buf.WriteString("\t\t\t\tProvider: map[string]string{\n")
		for _, prov := range provKeys {
			buf.WriteString(fmt.Sprintf("\t\t\t\t\t\"%s\": \"%s\",\n", prov, f.Markers.Provider[prov]))
		}
		buf.WriteString("\t\t\t\t},\n")
	}

	if f.Markers.Pattern != "" {
		buf.WriteString(fmt.Sprintf("\t\t\t\tPattern: %q,\n", f.Markers.Pattern))
	}
	if f.Markers.Default != "" {
		buf.WriteString(fmt.Sprintf("\t\t\t\tDefault: %q,\n", f.Markers.Default))
	}
	if f.Markers.Example != "" {
		buf.WriteString(fmt.Sprintf("\t\t\t\tExample: %q,\n", f.Markers.Example))
	}

	buf.WriteString("\t\t\t},\n")
}

// presenceKinds lists the kinds that get presence extractors.
// Only frontmatter kinds used by the orchestrator.
var presenceKinds = []struct {
	kind       string
	structName string
	paramName  string
}{
	{"agent", "AgentConfig", "a"},
	{"skill", "SkillConfig", "s"},
	{"rule", "RuleConfig", "r"},
}

// generatePresenceExtractors produces Go source for type-safe presence
// extractor functions that replace the manual implementations in
// required_fields.go. Each function checks every schema-registered field
// for a non-zero value and populates a map[string]string.
func generatePresenceExtractors(fields map[string][]FieldInfo) string {
	var buf strings.Builder

	buf.WriteString("// Code generated by gen-schema; DO NOT EDIT.\n\n")
	buf.WriteString("package renderer\n\n")
	buf.WriteString("import \"github.com/saero-ai/xcaffold/internal/ast\"\n\n")

	for _, pk := range presenceKinds {
		kindFields, ok := fields[pk.kind]
		if !ok {
			continue
		}
		writePresenceFunc(&buf, pk, kindFields)
	}

	return buf.String()
}

// writePresenceFunc emits a single ExtractXxxPresentFields function.
func writePresenceFunc(buf *strings.Builder, pk struct {
	kind       string
	structName string
	paramName  string
}, fields []FieldInfo) {
	title := strings.ToUpper(pk.kind[:1]) + pk.kind[1:]
	funcName := "Extract" + title + "PresentFields"

	buf.WriteString(fmt.Sprintf(
		"func %s(%s ast.%s) map[string]string {\n",
		funcName, pk.paramName, pk.structName,
	))
	buf.WriteString("\tm := make(map[string]string)\n")

	for _, f := range fields {
		writePresenceCheck(buf, pk.paramName, f)
	}

	// Body has yaml:"-" so it's not in the schema registry.
	// Add it as a special case since all 3 frontmatter kinds have it.
	buf.WriteString(fmt.Sprintf(
		"\tif %s.Body != \"\" {\n\t\tm[\"body\"] = \"set\"\n\t}\n",
		pk.paramName,
	))

	buf.WriteString("\treturn m\n}\n\n")
}

// writePresenceCheck emits the zero-value check for a single field.
func writePresenceCheck(buf *strings.Builder, param string, f FieldInfo) {
	accessor := param + "." + f.Name
	switch {
	case f.GoType == "string":
		buf.WriteString(fmt.Sprintf(
			"\tif %s != \"\" {\n\t\tm[\"%s\"] = %s\n\t}\n",
			accessor, f.YAMLKey, accessor,
		))
	case f.GoType == "*bool" || f.GoType == "*int":
		buf.WriteString(fmt.Sprintf(
			"\tif %s != nil {\n\t\tm[\"%s\"] = \"set\"\n\t}\n",
			accessor, f.YAMLKey,
		))
	case f.GoType == "int":
		buf.WriteString(fmt.Sprintf(
			"\tif %s != 0 {\n\t\tm[\"%s\"] = \"set\"\n\t}\n",
			accessor, f.YAMLKey,
		))
	case strings.HasPrefix(f.GoType, "[]"),
		strings.HasPrefix(f.GoType, "map["),
		f.GoType == "FlexStringSlice",
		f.GoType == "HookConfig":
		buf.WriteString(fmt.Sprintf(
			"\tif len(%s) > 0 {\n\t\tm[\"%s\"] = \"set\"\n\t}\n",
			accessor, f.YAMLKey,
		))
	case f.GoType == "ClearableList":
		buf.WriteString(fmt.Sprintf(
			"\tif len(%s.Values) > 0 {\n\t\tm[\"%s\"] = \"set\"\n\t}\n",
			accessor, f.YAMLKey,
		))
	}
}

func mapToXCFType(goType string) string {
	goType = strings.TrimSpace(goType)

	if strings.HasPrefix(goType, "*bool") {
		return "boolean"
	}
	if strings.HasPrefix(goType, "*int") {
		return "integer"
	}
	if goType == "string" {
		return "string"
	}
	if strings.HasPrefix(goType, "[]") {
		return goType
	}
	if strings.HasPrefix(goType, "map") {
		return "map"
	}

	return goType
}
