package parser

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/resolver"
	"gopkg.in/yaml.v3"
)

// parseOption controls parsing behaviour per invocation.
type parseOption struct {
	globalScope bool
	sourcePath  string
	Vars        map[string]interface{}
	Envs        map[string]string
}

// parseOptionFunc configures a parseOption.
type parseOptionFunc func(*parseOption)

func withVars(vars map[string]interface{}) parseOptionFunc {
	return func(o *parseOption) { o.Vars = vars }
}

func withEnvs(envs map[string]string) parseOptionFunc {
	return func(o *parseOption) { o.Envs = envs }
}

// withGlobalScope marks the parse as global scope, which allows absolute
// instructions-file paths (global configs reference files like ~/.claude/agents/*.md).
func withGlobalScope() parseOptionFunc {
	return func(o *parseOption) { o.globalScope = true }
}

// withSourcePath carries the originating file path into parse-time routines
// that need it.
func withSourcePath(path string) parseOptionFunc {
	return func(o *parseOption) { o.sourcePath = path }
}

func resolveParseOptions(opts []parseOptionFunc) parseOption {
	var o parseOption
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// ParseDirOption configures behaviour of ParseDirectory.
type ParseDirOption func(*parseDirConfig)

type parseDirConfig struct {
	skipGlobal bool
	Target     string
	VarFile    string
}

// WithSkipGlobal prevents ParseDirectory from loading the implicit global
// configuration (~/.xcaffold/). Use this for project-scoped validation.
func WithSkipGlobal() ParseDirOption {
	return func(c *parseDirConfig) { c.skipGlobal = true }
}

// WithVarFile sets a custom variable file to use.
func WithVarFile(path string) ParseDirOption {
	return func(c *parseDirConfig) { c.VarFile = path }
}

func resolveParseDirOptions(opts []ParseDirOption) parseDirConfig {
	var cfg parseDirConfig
	for _, fn := range opts {
		fn(&cfg)
	}
	return cfg
}

// reservedDirToKind maps xcaf directory names to their corresponding resource kind.
// Used for filesystem-as-schema inference when kind: is omitted from YAML.
var reservedDirToKind = map[string]string{
	"agents":     "agent",
	"skills":     "skill",
	"rules":      "rule",
	"workflows":  "workflow",
	"mcp":        "mcp",
	"hooks":      "hooks",
	"settings":   "settings",
	"memory":     "memory",
	"blueprints": "blueprint",
	"context":    "context",
	"policy":     "policy",
	"template":   "template",
}

// inferKindAndName extracts kind and name from a file path when not explicit in YAML.
// Pattern: xcaf/<resource-kind>/<resource-id>/... returns (kind, name).
// Returns ("", "") if the path does not match the pattern.
func inferKindAndName(filePath string) (kind, name string) {
	parts := strings.Split(filepath.ToSlash(filePath), "/")
	xcafIdx := -1
	for i, p := range parts {
		if p == "xcaf" {
			xcafIdx = i
			break
		}
	}
	if xcafIdx < 0 || xcafIdx+2 >= len(parts) {
		return "", ""
	}
	kindDir := parts[xcafIdx+1]
	kind, ok := reservedDirToKind[kindDir]
	if !ok {
		return "", ""
	}

	// Check if filename is a canonical kind filename (<kind>.xcaf or <kind>.<provider>.xcaf)
	filename := parts[len(parts)-1]
	baseFilename := strings.TrimSuffix(filename, ".xcaf")
	// Strip provider suffix: "rule.claude" → "rule"
	if dotIdx := strings.LastIndex(baseFilename, "."); dotIdx >= 0 {
		baseFilename = baseFilename[:dotIdx]
	}

	if baseFilename == kind {
		// Canonical convention: name from segments between kind-dir and filename
		nameSegments := parts[xcafIdx+2 : len(parts)-1]
		if len(nameSegments) == 0 {
			return "", ""
		}
		name = strings.Join(nameSegments, "/")
		return kind, name
	}

	// Legacy: name is parts[xcafIdx+2] with .xcaf stripped
	name = parts[xcafIdx+2]
	name = strings.TrimSuffix(name, ".xcaf")
	return kind, name
}

// Parse reads a .xcaf YAML configuration from the given reader and returns a
// validated XcaffoldConfig. It treats the configuration as a complete, standalone file.
func Parse(r io.Reader) (*ast.XcaffoldConfig, error) {
	config, err := parsePartial(r)
	if err != nil {
		return nil, err
	}
	if err := validateMerged(config); err != nil {
		return nil, fmt.Errorf("invalid .xcaf configuration: %w", err)
	}
	return config, nil
}

// detectRejectedSnakeCaseKeys scans raw .xcaf YAML bytes for snake_case keys
// that must use kebab-case instead. Returns a targeted diagnostic with the
// correct spelling.
//
// Currently enforced:
//   - instructions_file: in a kind: rule document must be instructions-file:.
func detectRejectedSnakeCaseKeys(data []byte) error {
	// Split into per-document segments on "---" boundaries so we can
	// check kind per document.
	type segment struct{ body []byte }
	var segments []segment
	rest := data

	if idx := bytes.Index(rest, []byte("\n---")); idx >= 0 {
		segments = append(segments, segment{rest[:idx+1]})
		rest = rest[idx+1:]
	} else {
		segments = append(segments, segment{rest})
		rest = nil
	}
	for len(rest) > 0 {
		if len(rest) < 4 || !bytes.HasPrefix(rest, []byte("---")) {
			segments = append(segments, segment{rest})
			break
		}
		markerEnd := 4 // len("---\n")
		next := bytes.Index(rest[3:], []byte("\n---"))
		if next < 0 {
			segments = append(segments, segment{rest[markerEnd:]})
			break
		}
		cutAt := 3 + next + 1
		segments = append(segments, segment{rest[markerEnd:cutAt]})
		rest = rest[cutAt:]
	}

	for _, seg := range segments {
		if !isRuleDocument(seg.body) {
			continue
		}
		for _, line := range bytes.Split(seg.body, []byte("\n")) {
			// Strip leading whitespace to handle any indentation level.
			trimmed := bytes.TrimLeft(line, " \t")
			if bytes.HasPrefix(trimmed, []byte("instructions_file:")) {
				return fmt.Errorf(
					"rule document: unknown field \"instructions_file\" — " +
						"use instructions-file: instead (renamed in schema v1.1)",
				)
			}
		}
	}
	return nil
}

// isRuleDocument returns true if the YAML document segment declares "kind: rule"
// at the top level.
func isRuleDocument(doc []byte) bool {
	for _, line := range bytes.Split(doc, []byte("\n")) {
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' || line[0] == '#' {
			continue
		}
		trimmed := bytes.TrimSpace(line)
		if bytes.Equal(trimmed, []byte("kind: rule")) {
			return true
		}
		if bytes.HasPrefix(trimmed, []byte("kind:")) {
			return false
		}
	}
	return false
}

// extractFrontmatterAndBody splits .xcaf file bytes on the frontmatter `---`
// delimiter. The format is:
//
//	---
//	<yaml frontmatter>
//	---
//	<markdown body>
//
// Rules:
//   - If data does NOT start with "---\n", it is treated as pure YAML (no body
//     extraction). frontmatter == data, body == nil. Existing multi-document
//     YAML files and legacy single-document files continue to work unchanged.
//   - If data starts with "---\n" and the region following the closing "---\n"
//     starts with another YAML kind document (begins with "kind:"), the file is
//     treated as a multi-document YAML stream (no frontmatter split). This
//     preserves full backward compatibility with multi-kind .xcaf files.
//   - If data starts with "---\n" with no closing "---\n", an error is
//     returned.
//   - body is the raw bytes after the closing "---\n". Callers must TrimSpace
//     before use; an empty or whitespace-only body is treated as no body.
func extractFrontmatterAndBody(data []byte) (frontmatter []byte, body []byte, err error) {
	const delim = "---\n"
	if !bytes.HasPrefix(data, []byte(delim)) {
		// Detect frontmatter file with preamble content before ---
		// Only trigger if:
		// 1. There's a --- delimiter somewhere in the file
		// 2. The preamble (text before ---) is NOT valid YAML
		// 3. The content after --- IS valid YAML
		// This avoids false positives on multi-document YAML (kind: global ... --- kind: agent ...)
		if idx := bytes.Index(data, []byte(delim)); idx > 0 {
			preamble := data[:idx]
			rest := data[idx+len(delim):]
			if !looksLikeYAMLDocument(preamble) && looksLikeYAMLDocument(rest) {
				preambleStr := strings.TrimSpace(string(preamble))
				firstLine := preambleStr
				if nl := strings.IndexByte(preambleStr, '\n'); nl > 0 {
					firstLine = preambleStr[:nl]
				}
				if len(firstLine) > 60 {
					firstLine = firstLine[:57] + "..."
				}
				return nil, nil, fmt.Errorf(
					"content before the opening '---' delimiter is not allowed in .xcaf files (found: %q). "+
						"Remove any text or comments before the first '---' line",
					firstLine)
			}
		}
		// Pure YAML mode — no frontmatter/body split.
		return data, nil, nil
	}
	rest := data[len(delim):]
	idx := bytes.Index(rest, []byte(delim))
	if idx == -1 {
		// Starts with "---\n" but no closing delimiter found.
		// If the remainder looks like a multi-document YAML stream or a single
		// YAML doc (has "kind:"), treat the whole file as pure YAML to stay
		// compatible with single-document files that begin with "---\n".
		if looksLikeYAMLDocument(rest) {
			return data, nil, nil
		}
		return nil, nil, fmt.Errorf(
			".xcaf file opens with '---' but has no closing '---' delimiter: " +
				"every frontmatter block must be closed with a line containing only '---'",
		)
	}
	candidate := rest[idx+len(delim):]
	// If the body after the closing "---\n" starts another YAML document
	// (detected by a top-level "kind:" key), the entire file is a multi-document
	// YAML stream — fall back to pure YAML mode for full backward compatibility.
	if looksLikeYAMLDocument(candidate) {
		return data, nil, nil
	}
	frontmatter = rest[:idx]
	body = candidate
	return frontmatter, body, nil
}

// looksLikeYAMLDocument returns true when data begins with a line of the form
// "kind: <value>" at the top level (no leading whitespace), indicating the
// content is a .xcaf resource document rather than free-form markdown.
func looksLikeYAMLDocument(data []byte) bool {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	return bytes.HasPrefix(trimmed, []byte("kind:"))
}

// extractKindAndInferredName reads kind and inferred name from a document node.
// It handles kind inference from the file path and warns on kind mismatches.
func extractKindAndInferredName(docNode *yaml.Node, resolved parseOption, config *ast.XcaffoldConfig) (string, string) {
	kind := extractKind(docNode)

	// Infer kind and name from file path if not explicit in YAML.
	// If kind is empty, infer both kind and name.
	// If kind is provided but name is not, infer only the name.
	var inferredKind, inferredName string
	if resolved.sourcePath != "" {
		inferredKind, inferredName = inferKindAndName(resolved.sourcePath)
		// If kind was empty in YAML, use the inferred kind
		if kind == "" && inferredKind != "" {
			kind = inferredKind
		}
	}

	// Warn if YAML kind differs from inferred kind (when both are present)
	if kind != "" && inferredKind != "" && kind != inferredKind {
		config.ParseWarnings = append(config.ParseWarnings, fmt.Sprintf("%s declares kind: %s but path implies kind: %s", resolved.sourcePath, kind, inferredKind))
	}

	return kind, inferredName
}

// routeDocument routes a single document to the appropriate parser based on its kind.
// It updates config and returns the lastKind and lastName for body assignment.
func routeDocument(docNode *yaml.Node, kind string, inferredName string, config *ast.XcaffoldConfig, resolved parseOption, docIndex int) (string, string, error) {
	switch kind {
	case "":
		return "", "", fmt.Errorf(
			"kind field is required: every .xcaf document must declare a kind " +
				"(e.g., kind: project, kind: agent, kind: global). " +
				"See https://xcaffold.com/docs/reference/schema",
		)

	case "config":
		return "", "", fmt.Errorf(
			"kind \"config\" has been removed: migrate to kind: project with " +
				"individual resource documents (kind: agent, kind: skill, etc.). " +
				"For global config, use kind: global. " +
				"See https://xcaffold.com/docs/migration/config-removal",
		)

	case "agent", "skill", "rule", "workflow", "mcp", "project", "hooks", "settings", "global", "policy", "context", "memory":
		// Resource-kind document: route to the kind-aware parser.
		// Propagate the resource version to config.Version if not already set.
		if config.Version == "" {
			config.Version = extractVersion(docNode)
		}
		if parseErr := parseResourceDocument(docNode, kind, config, resolved.sourcePath, inferredName); parseErr != nil {
			return "", "", parseErr
		}
		return kind, extractScalarField(docNode, "name"), nil

	case "blueprint":
		if config.Version == "" {
			config.Version = extractVersion(docNode)
		}
		if parseErr := parseBlueprintDocumentFromNode(docNode, config); parseErr != nil {
			return "", "", parseErr
		}
		return "blueprint", extractScalarField(docNode, "name"), nil

	default:
		return "", "", fmt.Errorf("unknown resource kind %q in document %d", kind, docIndex)
	}
}

// assignBodyToResource assigns markdown body text to the last parsed resource.
// Returns an error if the body assignment is invalid (e.g., project with body).
func assignBodyToResource(config *ast.XcaffoldConfig, lastKind, lastName string, body []byte) error {
	if body == nil || len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}

	trimmedBody := strings.TrimSpace(string(body))
	switch lastKind {
	case "agent":
		if a, ok := config.Agents[lastName]; ok {
			a.Body = trimmedBody
			config.Agents[lastName] = a
		}
	case "skill":
		if s, ok := config.Skills[lastName]; ok {
			s.Body = trimmedBody
			config.Skills[lastName] = s
		}
	case "rule":
		if r, ok := config.Rules[lastName]; ok {
			r.Body = trimmedBody
			config.Rules[lastName] = r
		}
	case "workflow":
		if w, ok := config.Workflows[lastName]; ok {
			if len(w.Steps) > 0 {
				assignWorkflowStepBodies(&w, trimmedBody)
			} else {
				w.Body = trimmedBody
			}
			config.Workflows[lastName] = w
		}
	case "context":
		if ctx, ok := config.Contexts[lastName]; ok {
			ctx.Body = trimmedBody
			config.Contexts[lastName] = ctx
		}
	case "project":
		return fmt.Errorf("invalid project document: kind: project does not support a markdown body — use kind: context for workspace-level instructions")
	}
	return nil
}

// processYAMLDocuments decodes YAML documents and routes them to appropriate parsers.
// Returns the kind and name of the last processed document.
func processYAMLDocuments(frontmatter []byte, config *ast.XcaffoldConfig, resolved parseOption) (string, string, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(frontmatter))
	docIndex := 0
	var lastKind, lastName string

	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return "", "", fmt.Errorf("failed to parse .xcaf YAML document %d: %w", docIndex, err)
		}

		// yaml.Decoder wraps each document in a DocumentNode; unwrap it.
		docNode := &node
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			docNode = node.Content[0]
		}

		// Extract kind and inferred name with warnings.
		kind, inferredName := extractKindAndInferredName(docNode, resolved, config)

		// Route the document to the appropriate parser.
		var routeErr error
		lastKind, lastName, routeErr = routeDocument(docNode, kind, inferredName, config, resolved, docIndex)
		if routeErr != nil {
			return "", "", routeErr
		}

		// Reject multi-document .xcaf files.
		if docIndex > 0 {
			return "", "", fmt.Errorf(
				"multi-document .xcaf files are no longer supported; "+
					"each .xcaf file must contain exactly one resource (found document %d, kind: %s); "+
					"split into separate files under xcaf/",
				docIndex+1, kind)
		}

		docIndex++
	}

	if docIndex == 0 {
		return "", "", fmt.Errorf("failed to parse .xcaf YAML: EOF")
	}

	return lastKind, lastName, nil
}

func parsePartial(r io.Reader, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
	resolved := resolveParseOptions(opts)
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read .xcaf input: %w", err)
	}

	if len(resolved.Vars) > 0 || len(resolved.Envs) > 0 {
		data, err = resolver.ExpandVariables(data, resolved.Vars, resolved.Envs)
		if err != nil {
			return nil, err
		}
	}

	frontmatter, body, err := extractFrontmatterAndBody(data)
	if err != nil {
		return nil, err
	}

	if err := detectRejectedSnakeCaseKeys(frontmatter); err != nil {
		return nil, err
	}

	config := &ast.XcaffoldConfig{}
	lastKind, lastName, err := processYAMLDocuments(frontmatter, config, resolved)
	if err != nil {
		return nil, err
	}

	if err := assignBodyToResource(config, lastKind, lastName, body); err != nil {
		return nil, err
	}

	o := resolveParseOptions(opts)
	if err := validatePartial(config, o.globalScope); err != nil {
		return nil, fmt.Errorf("invalid .xcaf configuration part: %w", err)
	}
	return config, nil
}

var workflowStepHeading = regexp.MustCompile(`(?m)^##\s+([a-zA-Z0-9_-]+)\s*$`)

// assignWorkflowStepBodies processes the raw markdown workflow body, slicing it into step-specific
// blocks based on ## <step-name> headings and assigning those blocks to their respective WorkflowStep.
func assignWorkflowStepBodies(w *ast.WorkflowConfig, body string) {
	if body == "" {
		return
	}

	matches := workflowStepHeading.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return
	}

	stepMap := make(map[string]string)
	for i, match := range matches {
		nameStart := match[2]
		nameEnd := match[3]
		name := body[nameStart:nameEnd]

		bodyStart := match[1]
		bodyEnd := len(body)
		if i+1 < len(matches) {
			bodyEnd = matches[i+1][0]
		}
		stepMap[name] = strings.TrimSpace(body[bodyStart:bodyEnd])
	}

	for i, step := range w.Steps {
		if content, ok := stepMap[step.Name]; ok {
			w.Steps[i].Body = content
		}
	}
}
