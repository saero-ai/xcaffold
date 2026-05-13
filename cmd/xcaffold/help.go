package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/pkg/schema"
	"github.com/spf13/cobra"
)

const requiredLabel = "required"

func runHelpXcaf(cmd *cobra.Command, kind string, outPath string, outChanged bool) error {
	ks, ok := schema.LookupKind(kind)
	if !ok {
		return fmt.Errorf("unknown kind: %s. Available: %s", kind, strings.Join(schema.KindNames(), ", "))
	}

	if outChanged {
		return generateTemplate(cmd, ks, kind, outPath)
	}

	displayKindSchema(cmd, ks)
	return nil
}

func displayKindSchema(cmd *cobra.Command, ks schema.KindSchema) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "%s %s version %s %s format: %s\n",
		bold("kind: "+ks.Kind), glyphDot(), ks.Version, glyphDot(), ks.Format)
	fmt.Fprintln(w)

	groups := groupFields(ks.Fields)
	for _, g := range groups {
		fmt.Fprintf(w, "  %s\n", dim(g.name))
		for _, f := range g.fields {
			req := "optional"
			if !f.Optional {
				req = requiredLabel
			}
			fmt.Fprintf(w, "    %-26s%-16s%-10s%s\n", f.YAMLKey, f.XCAFType, req, f.Description)
			printFieldConstraints(w, f)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "%s Run 'xcaffold help --xcaf %s --out' to generate a template.\n", glyphArrow(), ks.Kind)
}

func printFieldConstraints(w io.Writer, f schema.Field) {
	indent := "                                                        "

	// Print provider support annotations if available
	if len(f.Provider) > 0 {
		providers := formatProviderSupport(f.Provider)
		if providers != "" {
			fmt.Fprintf(w, "%sProviders: %s\n", indent, providers)
		}
	}

	if f.Pattern != "" {
		fmt.Fprintf(w, "%sPattern: %s\n", indent, f.Pattern)
	}
	if f.Example != "" {
		fmt.Fprintf(w, "%sExamples: %s\n", indent, f.Example)
	}
	if len(f.Enum) > 0 {
		fmt.Fprintf(w, "%sValues: %s\n", indent, strings.Join(f.Enum, ", "))
	}
	if f.Default != "" {
		fmt.Fprintf(w, "%sDefault: %s\n", indent, f.Default)
	}
}

// formatProviderSupport converts a provider map into a human-readable string.
// Returns empty string if all providers are "unsupported".
func formatProviderSupport(providers map[string]string) string {
	// Collect supported providers in alphabetical order
	var order []string
	for name := range providers {
		support := providers[name]
		if support != "unsupported" && support != "xcaffold-only" {
			order = append(order, name)
		}
	}
	// Sort alphabetically for stable output
	sort.Strings(order)

	var parts []string
	for _, name := range order {
		support := providers[name]
		// Capitalize first letter of provider name
		capitalized := strings.ToUpper(name[:1]) + name[1:]

		if support == requiredLabel {
			parts = append(parts, capitalized+"("+requiredLabel+")")
		} else {
			// "optional" is the default, so just show the name
			parts = append(parts, capitalized)
		}
	}

	return strings.Join(parts, " ")
}

type fieldGroup struct {
	name   string
	fields []schema.Field
}

func groupFields(fields []schema.Field) []fieldGroup {
	seen := map[string]int{}
	var groups []fieldGroup
	for _, f := range fields {
		g := f.Group
		if g == "" {
			g = "Other"
		}
		if idx, ok := seen[g]; ok {
			groups[idx].fields = append(groups[idx].fields, f)
		} else {
			seen[g] = len(groups)
			groups = append(groups, fieldGroup{name: g, fields: []schema.Field{f}})
		}
	}
	return groups
}

func generateTemplate(cmd *cobra.Command, ks schema.KindSchema, kind, outPath string) error {
	dest, err := resolveOutPath(kind, outPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(dest)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	if _, err := os.Stat(dest); err == nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "overwriting: %s\n", dest)
	}

	content := buildTemplateContent(ks)
	if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
		return fmt.Errorf("could not write template: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", dest)
	return nil
}

func resolveOutPath(kind, outPath string) (string, error) {
	if outPath == "" || outPath == "." {
		return filepath.Abs(kind + ".xcaf")
	}

	info, err := os.Stat(outPath)
	if err == nil && info.IsDir() {
		return filepath.Join(outPath, kind+".xcaf"), nil
	}

	if !strings.HasSuffix(outPath, ".xcaf") {
		return "", fmt.Errorf("output path must end in .xcaf: %s", outPath)
	}

	abs, err := filepath.Abs(outPath)
	if err != nil {
		return "", fmt.Errorf("could not resolve path: %w", err)
	}
	return abs, nil
}

func buildTemplateContent(ks schema.KindSchema) string {
	var sb strings.Builder
	isFrontmatter := ks.Format == "frontmatter+body"

	if isFrontmatter {
		sb.WriteString("---\n")
	}
	sb.WriteString(fmt.Sprintf("kind: %s\n", ks.Kind))
	sb.WriteString("version: \"1.0\"\n")
	sb.WriteString("\n")

	groups := groupFields(ks.Fields)
	for _, g := range groups {
		writeGroupHeader(&sb, g.name)
		writeGroupFields(&sb, g.fields)
	}

	if isFrontmatter {
		sb.WriteString("---\n")
		sb.WriteString("# Instructions go here.\n")
	}
	return sb.String()
}

func writeGroupHeader(sb *strings.Builder, name string) {
	header := fmt.Sprintf("# ── %s ", name)
	pad := 60 - len(header)
	if pad < 3 {
		pad = 3
	}
	sb.WriteString(header + strings.Repeat("─", pad) + "\n")
}

func writeGroupFields(sb *strings.Builder, fields []schema.Field) {
	for _, f := range fields {
		req := "optional"
		if !f.Optional {
			req = requiredLabel
		}
		sb.WriteString(fmt.Sprintf("# %s (%s, %s): %s\n", f.YAMLKey, f.XCAFType, req, f.Description))
		sb.WriteString(buildMarkerComment(f))
		sb.WriteString(fmt.Sprintf("%s: %s\n", f.YAMLKey, fieldPlaceholder(f)))
		sb.WriteString("\n")
	}
}

func fieldPlaceholder(f schema.Field) string {
	if !f.Optional && f.XCAFType == "string" {
		return "\"my-" + f.YAMLKey + "\""
	}
	return emptyValue(f.XCAFType)
}

func buildMarkerComment(f schema.Field) string {
	var parts []string
	if f.Optional {
		parts = append(parts, "+xcaf:optional")
	} else {
		parts = append(parts, "+xcaf:required")
	}
	if f.Pattern != "" {
		parts = append(parts, "+xcaf:pattern="+f.Pattern)
	}
	if len(f.Enum) > 0 {
		parts = append(parts, "+xcaf:enum="+strings.Join(f.Enum, ","))
	}
	if f.Example != "" {
		parts = append(parts, "+xcaf:example="+f.Example)
	}
	return "# " + strings.Join(parts, " ") + "\n"
}

func emptyValue(xcafType string) string {
	switch {
	case strings.HasPrefix(xcafType, "[]"):
		return "[]"
	case strings.HasPrefix(xcafType, "map") || strings.HasSuffix(xcafType, "Config"):
		return "{}"
	case xcafType == "boolean":
		return "false"
	case xcafType == "integer" || xcafType == "int":
		return "0"
	case xcafType == "string":
		return "\"\""
	default:
		return "\"\""
	}
}
