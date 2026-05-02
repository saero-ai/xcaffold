package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"sort"
	"strings"
)

var (
	outputPath   = flag.String("output", "", "output file path (default: stdout)")
	packageName  = flag.String("package", "schema", "generated package name")
	validateOnly = flag.Bool("validate-only", false, "validate markers only, do not generate")
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
	output := generateGo(*packageName, fields)

	if *outputPath != "" {
		if err := os.WriteFile(*outputPath, []byte(output), 0644); err != nil {
			log.Fatalf("failed to write output: %v", err)
		}
	} else {
		fmt.Print(output)
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
