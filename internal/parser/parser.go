package parser

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// parseOption controls parsing behaviour per invocation.
type parseOption struct {
	globalScope bool
	sourcePath  string
}

// parseOptionFunc configures a parseOption.
type parseOptionFunc func(*parseOption)

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

// reservedDirToKind maps xcf directory names to their corresponding resource kind.
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
// Pattern: xcf/<resource-kind>/<resource-id>/... returns (kind, name).
// Returns ("", "") if the path does not match the pattern.
func inferKindAndName(filePath string) (kind, name string) {
	parts := strings.Split(filepath.ToSlash(filePath), "/")
	xcfIdx := -1
	for i, p := range parts {
		if p == "xcf" {
			xcfIdx = i
			break
		}
	}
	if xcfIdx < 0 || xcfIdx+2 >= len(parts) {
		return "", ""
	}
	kindDir := parts[xcfIdx+1]
	kind, ok := reservedDirToKind[kindDir]
	if !ok {
		return "", ""
	}
	name = parts[xcfIdx+2]
	name = strings.TrimSuffix(name, ".xcf")
	return kind, name
}

// Parse reads a .xcf YAML configuration from the given reader and returns a
// validated XcaffoldConfig. It treats the configuration as a complete, standalone file.
func Parse(r io.Reader) (*ast.XcaffoldConfig, error) {
	config, err := parsePartial(r)
	if err != nil {
		return nil, err
	}
	if err := validateMerged(config); err != nil {
		return nil, fmt.Errorf("invalid .xcf configuration: %w", err)
	}
	return config, nil
}

// legacyKeyAliases maps pre-migration camelCase/snake_case xcf YAML keys to their
// canonical kebab-case equivalents. This rewrite is applied per-document (after
// splitting on "---") to provide a deprecation-period grace window.
//
// IMPORTANT: Keys inside "kind: settings" documents are provider wire-format
// pass-throughs and are intentionally excluded from aliasing.
//
// TODO: Remove this map and the rewriting logic once the deprecation window closes
// (target: next major version after migration lands).
var legacyKeyAliases = map[string]string{
	// AgentConfig — camelCase (pre-migration convention for agent fields)
	"maxTurns:":               "max-turns:",
	"disallowedTools:":        "disallowed-tools:",
	"permissionMode:":         "permission-mode:",
	"disableModelInvocation:": "disable-model-invocation:",
	"userInvocable:":          "user-invocable:",
	"initialPrompt:":          "initial-prompt:",
	// NOTE: "mcpServers:" aliasing for non-settings documents is handled below in
	// the SettingsConfig block. The "kind: settings" document exemption in
	// isSettingsDocument protects standalone settings files (provider wire-format
	// pass-throughs). The alias only fires for non-settings xcf documents (e.g.
	// kind: global, kind: project) where the settings: sub-block uses the xcf
	// schema key "mcp-servers:" going forward.
	// HookHandler — camelCase Claude-settings mirror (pre-migration convention)
	"statusMessage:":  "status-message:",
	"allowedEnvVars:": "allowed-env-vars:",
	// AgentConfig / SkillConfig / RuleConfig / WorkflowConfig
	"instructions_file:": "instructions-file:",
	// RuleConfig
	"alwaysApply:": "always-apply:",
	// TargetOverride
	"suppress_fidelity_warnings:": "suppress-fidelity-warnings:",
	"skip_synthesis:":             "skip-synthesis:",
	"instructions_override:":      "instructions-override:",
	// ProjectConfig
	"backup_dir:": "backup-dir:",
	// TestConfig
	"cli_path:":    "cli-path:",
	"claude_path:": "claude-path:",
	"judge_model:": "judge-model:",
	// PolicyMatch
	"has_tool:":        "has-tool:",
	"has_field:":       "has-field:",
	"name_matches:":    "name-matches:",
	"target_includes:": "target-includes:",
	// PolicyRequire
	"is_present:": "is-present:",
	"min_length:": "min-length:",
	"max_count:":  "max-count:",
	"one_of:":     "one-of:",
	// PolicyDeny
	"content_contains:": "content-contains:",
	"content_matches:":  "content-matches:",
	"path_contains:":    "path-contains:",
	// MCPConfig — camelCase (pre-migration)
	"authProviderType:": "auth-provider-type:",
	"disabledTools:":    "disabled-tools:",
	// PermissionsConfig — camelCase (pre-migration)
	"defaultMode:":                  "default-mode:",
	"additionalDirectories:":        "additional-directories:",
	"disableBypassPermissionsMode:": "disable-bypass-permissions-mode:",
	// SandboxConfig — camelCase (pre-migration)
	"autoAllowBashIfSandboxed:": "auto-allow-bash-if-sandboxed:",
	"failIfUnavailable:":        "fail-if-unavailable:",
	"allowUnsandboxedCommands:": "allow-unsandboxed-commands:",
	"excludedCommands:":         "excluded-commands:",
	// SandboxFilesystem — camelCase (pre-migration)
	"allowWrite:": "allow-write:",
	"denyWrite:":  "deny-write:",
	"allowRead:":  "allow-read:",
	"denyRead:":   "deny-read:",
	// SandboxNetwork — camelCase (pre-migration)
	"httpProxyPort:":           "http-proxy-port:",
	"socksProxyPort:":          "socks-proxy-port:",
	"allowManagedDomainsOnly:": "allow-managed-domains-only:",
	"allowUnixSockets:":        "allow-unix-sockets:",
	"allowLocalBinding:":       "allow-local-binding:",
	"allowedDomains:":          "allowed-domains:",
	// SettingsConfig — camelCase (pre-migration)
	"autoMode:":                          "auto-mode:",
	"cleanupPeriodDays:":                 "cleanup-period-days:",
	"includeGitInstructions:":            "include-git-instructions:",
	"skipDangerousModePermissionPrompt:": "skip-dangerous-mode-permission-prompt:",
	"autoMemoryEnabled:":                 "auto-memory-enabled:",
	"disableAllHooks:":                   "disable-all-hooks:",
	"mcpServers:":                        "mcp-servers:",
	"statusLine:":                        "status-line:",
	"respectGitignore:":                  "respect-gitignore:",
	"enabledPlugins:":                    "enabled-plugins:",
	"disableSkillShellExecution:":        "disable-skill-shell-execution:",
	"alwaysThinkingEnabled:":             "always-thinking-enabled:",
	"effortLevel:":                       "effort-level:",
	"defaultShell:":                      "default-shell:",
	"outputStyle:":                       "output-style:",
	"plansDirectory:":                    "plans-directory:",
	"otelHeadersHelper:":                 "otel-headers-helper:",
	"autoMemoryDirectory:":               "auto-memory-directory:",
	"availableModels:":                   "available-models:",
	"claudeMdExcludes:":                  "claude-md-excludes:",
}

// rewriteLegacyKeys rewrites pre-migration xcf YAML keys to kebab-case equivalents
// on a per-document basis. The "kind: settings" document type is exempt — settings
// fields are provider-native pass-throughs that must not be mangled.
//
// Detection is line-oriented: the rewriter scans for "key:" at the start of a
// non-indented line (scalar key), which is sufficient for all affected fields.
// Indented values and YAML strings are not affected.
func rewriteLegacyKeys(data []byte) []byte {
	// Split into per-document segments on "---" boundaries so each document
	// can be checked for "kind: settings" independently.
	type segment struct {
		sep  []byte // leading "---\n" or nil for first doc
		body []byte
	}

	var segments []segment
	rest := data

	// First segment: content before the first "---"
	if idx := bytes.Index(rest, []byte("\n---")); idx >= 0 {
		segments = append(segments, segment{nil, rest[:idx+1]})
		rest = rest[idx+1:]
	} else {
		segments = append(segments, segment{nil, rest})
		rest = nil
	}

	// Remaining segments: split on "\n---\n" or "\n---" at EOF
	for len(rest) > 0 {
		markerEnd := 4 // len("---\n")
		if len(rest) < 4 || !bytes.HasPrefix(rest, []byte("---")) {
			// Shouldn't happen; append as-is.
			segments = append(segments, segment{nil, rest})
			break
		}
		// Find next "---"
		next := bytes.Index(rest[3:], []byte("\n---"))
		if next < 0 {
			segments = append(segments, segment{[]byte("---\n"), rest[markerEnd:]})
			break
		}
		cutAt := 3 + next + 1 // position of "\n" before next "---"
		segments = append(segments, segment{[]byte("---\n"), rest[markerEnd:cutAt]})
		rest = rest[cutAt:]
	}

	var out bytes.Buffer
	for _, seg := range segments {
		out.Write(seg.sep)
		// Check if this document is "kind: settings" — exempt from aliasing.
		if isSettingsDocument(seg.body) {
			out.Write(seg.body)
			continue
		}
		out.Write(rewriteDocumentKeys(seg.body))
	}
	return out.Bytes()
}

// isSettingsDocument returns true if the document declares "kind: settings"
// at the top level. Indented "kind: settings" values inside nested maps do
// not qualify — only a zero-indent top-level kind discriminator matters.
func isSettingsDocument(doc []byte) bool {
	for _, line := range bytes.Split(doc, []byte("\n")) {
		// Only consider top-level lines (no leading whitespace).
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		if line[0] == '#' {
			continue
		}
		trimmed := bytes.TrimSpace(line)
		if bytes.Equal(trimmed, []byte("kind: settings")) {
			return true
		}
		if bytes.HasPrefix(trimmed, []byte("kind:")) {
			// A top-level kind that isn't settings — this document is not exempt.
			return false
		}
	}
	return false
}

// rewriteDocumentKeys applies legacyKeyAliases to a single document body.
// It rewrites any line (at any indentation level) whose key position starts
// with a legacy key — preserving the original leading whitespace and any
// YAML list-item marker ("- "). Comment lines (trimmed prefix "#") are skipped.
//
// This handles all field positions in the .xcf YAML structure: top-level fields
// (e.g. "backup-dir:"), nested fields (e.g. "  instructions-file:", "  max-turns:"),
// and list-item fields (e.g. "  - content-contains:") which are common in
// policy deny/require blocks.
func rewriteDocumentKeys(doc []byte) []byte {
	lines := bytes.Split(doc, []byte("\n"))
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		// Calculate leading whitespace length.
		indent := 0
		for indent < len(line) && (line[indent] == ' ' || line[indent] == '\t') {
			indent++
		}
		// Skip comment lines and lines that are only whitespace.
		if indent >= len(line) || line[indent] == '#' {
			continue
		}
		// Advance past a YAML list-item marker ("- ") if present, so that
		// "  - content_contains:" is treated like a key at deeper indent.
		keyStart := indent
		if keyStart+1 < len(line) && line[keyStart] == '-' && line[keyStart+1] == ' ' {
			keyStart += 2
		}
		keyRegion := line[keyStart:]
		for old, newKey := range legacyKeyAliases {
			if bytes.HasPrefix(keyRegion, []byte(old)) {
				// Reconstruct: prefix (indent + optional "- ") + new_key + remainder.
				remainder := keyRegion[len(old):]
				newLine := make([]byte, 0, keyStart+len(newKey)+len(remainder))
				newLine = append(newLine, line[:keyStart]...)
				newLine = append(newLine, newKey...)
				newLine = append(newLine, remainder...)
				lines[i] = newLine
				break
			}
		}
	}
	return bytes.Join(lines, []byte("\n"))
}

// detectRejectedSnakeCaseKeys scans raw .xcf YAML bytes (before legacy-key
// rewriting) for snake_case keys that are rejected with a targeted diagnostic
// rather than silently aliased to their kebab-case equivalents.
//
// Currently enforced:
//   - instructions_file: in a kind: rule document must not be used. Authors
//     must use instructions-file: (renamed in schema v1.1). The rewriter would
//     silently alias this key, masking the usage. We surface it as an error
//     with a message that includes the canonical name "instructions-file" so
//     tooling and users know the correct spelling.
func detectRejectedSnakeCaseKeys(data []byte) error {
	// Split into per-document segments on "---" boundaries (same logic as
	// rewriteLegacyKeys) so we can check kind per document.
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

// extractFrontmatterAndBody splits .xcf file bytes on the frontmatter `---`
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
//     preserves full backward compatibility with multi-kind .xcf files.
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
					"content before the opening '---' delimiter is not allowed in .xcf files (found: %q). "+
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
			".xcf file opens with '---' but has no closing '---' delimiter: " +
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
// content is a .xcf resource document rather than free-form markdown.
func looksLikeYAMLDocument(data []byte) bool {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	return bytes.HasPrefix(trimmed, []byte("kind:"))
}

func parsePartial(r io.Reader, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
	resolved := resolveParseOptions(opts)
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read .xcf input: %w", err)
	}

	// Split on frontmatter delimiter before any YAML processing.
	// Files that do NOT start with "---\n" are treated as pure YAML (body == nil)
	// and fall through unchanged — full backward compatibility is preserved.
	frontmatter, body, err := extractFrontmatterAndBody(data)
	if err != nil {
		return nil, err
	}

	// Detect pre-migration snake_case keys that are rejected (not silently aliased)
	// for specific kinds. This scan runs before rewriteLegacyKeys so the original
	// key spelling is still visible.
	if err := detectRejectedSnakeCaseKeys(frontmatter); err != nil {
		return nil, err
	}

	// Rewrite deprecated camelCase/snake_case keys to kebab-case before decoding.
	// This provides backward compatibility during the migration period.
	frontmatter = rewriteLegacyKeys(frontmatter)

	config := &ast.XcaffoldConfig{}
	decoder := yaml.NewDecoder(bytes.NewReader(frontmatter))
	docIndex := 0

	// Track the kind and name of the last parsed resource for body assignment.
	var lastKind string
	var lastName string

	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to parse .xcf YAML document %d: %w", docIndex, err)
		}

		// yaml.Decoder wraps each document in a DocumentNode; unwrap it.
		docNode := &node
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			docNode = node.Content[0]
		}

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
			// If kind is now known (either explicit or inferred), we can use the inferred name
			// The name will be applied during resource document parsing if YAML name is empty.
		}

		// Warn if YAML kind differs from inferred kind (when both are present)
		if kind != "" && inferredKind != "" && kind != inferredKind {
			config.ParseWarnings = append(config.ParseWarnings, fmt.Sprintf("%s declares kind: %s but path implies kind: %s", resolved.sourcePath, kind, inferredKind))
		}

		switch kind {
		case "":
			return nil, fmt.Errorf(
				"kind field is required: every .xcf document must declare a kind " +
					"(e.g., kind: project, kind: agent, kind: global). " +
					"See https://xcaffold.com/docs/reference/schema",
			)

		case "config":
			return nil, fmt.Errorf(
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
				return nil, parseErr
			}
			lastKind = kind
			lastName = extractScalarField(docNode, "name")

		case "blueprint":
			if config.Version == "" {
				config.Version = extractVersion(docNode)
			}
			if parseErr := parseBlueprintDocument(docNode, config); parseErr != nil {
				return nil, parseErr
			}
			lastKind = "blueprint"
			lastName = extractScalarField(docNode, "name")

		default:
			return nil, fmt.Errorf("unknown resource kind %q in document %d", kind, docIndex)
		}

		// Reject multi-document .xcf files. Each file must contain exactly one
		// resource document. Split multi-resource files into separate .xcf files.
		if docIndex > 0 {
			return nil, fmt.Errorf(
				"multi-document .xcf files are no longer supported; "+
					"each .xcf file must contain exactly one resource (found document %d, kind: %s); "+
					"split into separate files under xcf/",
				docIndex+1, kind)
		}

		docIndex++
	}

	if docIndex == 0 {
		return nil, fmt.Errorf("failed to parse .xcf YAML: EOF")
	}

	// Assign markdown body to the parsed resource's Body or Content field.
	// Only applies to frontmatter-format files (body != nil) with non-empty body text.
	if body != nil && len(strings.TrimSpace(string(body))) > 0 {
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
			if config.Contexts == nil {
				config.Contexts = make(map[string]ast.ContextConfig)
			}
			config.Contexts["root"] = ast.ContextConfig{
				Name: "root",
				Body: trimmedBody,
			}
		}
	}

	o := resolveParseOptions(opts)
	if err := validatePartial(config, o.globalScope); err != nil {
		return nil, fmt.Errorf("invalid .xcf configuration part: %w", err)
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

// ParsedFile pairs a parsed partial config with its source file path.
type ParsedFile struct {
	Config   *ast.XcaffoldConfig
	FilePath string
}

// ParseDirectory recursively scans the given directory for all *.xcf files,
// parses them, merges them strictly (erroring on duplicate IDs), and then
// resolves 'extends:' chains.
func ParseDirectory(dir string) (*ast.XcaffoldConfig, error) {
	merged, err := parseDirectoryUnvalidated(dir)
	if err != nil {
		return nil, err
	}

	if err := validateMerged(merged); err != nil {
		return nil, fmt.Errorf("validation failed for project configuration: %w", err)
	}

	return merged, nil
}

// parseableKinds lists the kind values accepted by isParseableFile.
// Every .xcf document must declare an explicit kind field.
var parseableKinds = map[string]bool{
	"project":   true,
	"agent":     true,
	"skill":     true,
	"rule":      true,
	"workflow":  true,
	"mcp":       true,
	"hooks":     true,
	"settings":  true,
	"global":    true,
	"policy":    true,
	"blueprint": true,
	"context":   true,
	"memory":    true,
}

// isParseableFile reads the kind: field from an .xcf file to determine if it
// should be parsed by the compiler. Returns true for known resource-kind files.
// Returns false for files with unknown, empty, or removed kinds (such as "config").
func isParseableFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	// Extract only the frontmatter portion. Markdown body after the closing
	// --- may contain YAML-invalid syntax (e.g., tables with |) that would
	// cause yaml.Unmarshal on the full file to fail.
	fm, _, _ := extractFrontmatterAndBody(data)
	if len(fm) == 0 {
		fm = data
	}
	var header struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(fm, &header); err != nil {
		return false
	}
	return parseableKinds[header.Kind]
}

// validProviders is the closed set of allowed provider tokens in override filenames.
var validProviders = map[string]bool{
	"claude":      true,
	"gemini":      true,
	"cursor":      true,
	"antigravity": true,
	"copilot":     true,
}

// canonicalKindFilenames lists the resource kinds that can appear as prefixes in override filenames.
var canonicalKindFilenames = map[string]bool{
	"agent":    true,
	"skill":    true,
	"rule":     true,
	"workflow": true,
	"mcp":      true,
	"hooks":    true,
	"settings": true,
	"policy":   true,
	"template": true,
	"memory":   true,
}

// overrideFileEntry represents a detected override file with its parsed metadata.
type overrideFileEntry struct {
	Path     string
	Kind     string
	Provider string
}

// classifyOverrideFile parses a filename to detect <kind>.<provider>.xcf pattern.
// Returns (kind, provider, isOverride). If not an override file, isOverride is false.
func classifyOverrideFile(filename string) (kind, provider string, isOverride bool) {
	name := strings.TrimSuffix(filename, ".xcf")
	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	if !canonicalKindFilenames[parts[0]] {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// newParseFilter creates a map of directory names to skip during xcf scanning.
func newParseFilter(dir string) map[string]bool {
	ignored := map[string]bool{
		".git":         true,
		".worktrees":   true,
		"node_modules": true,
		"vendor":       true,
		".venv":        true,
		".xcaffold":    true,
		".claude":      true,
		".cursor":      true,
		".gemini":      true,
		".agents":      true,
		"dist":         true,
		"build":        true,
		"coverage":     true,
	}
	if data, err := os.ReadFile(filepath.Join(dir, ".gitignore")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				clean := strings.TrimPrefix(line, "/")
				clean = strings.TrimSuffix(clean, "/")
				if !strings.ContainsAny(clean, "*?[") {
					ignored[clean] = true
				}
			}
		}
	}
	return ignored
}

// parseOverrideFile parses a single override file (.provider.xcf) and stores
// the partial resource config in XcaffoldConfig.Overrides.
func parseOverrideFile(entry overrideFileEntry, config *ast.XcaffoldConfig) error {
	data, err := os.ReadFile(entry.Path)
	if err != nil {
		return fmt.Errorf("read override %s: %w", entry.Path, err)
	}

	frontmatter, body, err := extractFrontmatterAndBody(data)
	if err != nil {
		return fmt.Errorf("parse override %s: %w", entry.Path, err)
	}

	// Infer resource name from directory: xcf/agents/<name>/agent.claude.xcf -> name
	resourceName := filepath.Base(filepath.Dir(entry.Path))
	trimmedBody := strings.TrimSpace(string(body))

	// Initialize Overrides if nil
	if config.Overrides == nil {
		config.Overrides = &ast.ResourceOverrides{}
	}

	switch entry.Kind {
	case "agent":
		var cfg ast.AgentConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode agent override %s: %w", entry.Path, err)
		}
		cfg.Body = trimmedBody
		config.Overrides.AddAgent(resourceName, entry.Provider, cfg)
	case "skill":
		var cfg ast.SkillConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode skill override %s: %w", entry.Path, err)
		}
		cfg.Body = trimmedBody
		config.Overrides.AddSkill(resourceName, entry.Provider, cfg)
	case "rule":
		var cfg ast.RuleConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode rule override %s: %w", entry.Path, err)
		}
		cfg.Body = trimmedBody
		config.Overrides.AddRule(resourceName, entry.Provider, cfg)
	case "workflow":
		var cfg ast.WorkflowConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode workflow override %s: %w", entry.Path, err)
		}
		cfg.Body = trimmedBody
		config.Overrides.AddWorkflow(resourceName, entry.Provider, cfg)
	case "mcp":
		var cfg ast.MCPConfig
		if err := yaml.Unmarshal(frontmatter, &cfg); err != nil {
			return fmt.Errorf("decode mcp override %s: %w", entry.Path, err)
		}
		config.Overrides.AddMCP(resourceName, entry.Provider, cfg)
	default:
		return fmt.Errorf("override file %s: unsupported kind %q for overrides", entry.Path, entry.Kind)
	}
	return nil
}

// validateOverrideBasesExist ensures that every override file has a corresponding
// base resource. Override files without bases cannot be applied.
func validateOverrideBasesExist(config *ast.XcaffoldConfig) error {
	if config.Overrides == nil {
		return nil
	}
	for name := range config.Overrides.Agent {
		if _, ok := config.Agents[name]; !ok {
			return fmt.Errorf("override file for agent %q has no base resource", name)
		}
	}
	for name := range config.Overrides.Skill {
		if _, ok := config.Skills[name]; !ok {
			return fmt.Errorf("override file for skill %q has no base resource", name)
		}
	}
	for name := range config.Overrides.Rule {
		if _, ok := config.Rules[name]; !ok {
			return fmt.Errorf("override file for rule %q has no base resource", name)
		}
	}
	for name := range config.Overrides.Workflow {
		if _, ok := config.Workflows[name]; !ok {
			return fmt.Errorf("override file for workflow %q has no base resource", name)
		}
	}
	for name := range config.Overrides.MCP {
		if _, ok := config.MCP[name]; !ok {
			return fmt.Errorf("override file for mcp %q has no base resource", name)
		}
	}
	return nil
}

func parseDirectoryUnvalidated(dir string) (*ast.XcaffoldConfig, error) {
	var files []string
	var overrideFiles []overrideFileEntry
	ignored := newParseFilter(dir)
	providerDir := filepath.Join("xcf", "provider")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || ignored[name]) {
				return filepath.SkipDir
			}
			if rel, relErr := filepath.Rel(dir, path); relErr == nil && rel == providerDir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".xcf") {
			if kind, provider, ok := classifyOverrideFile(d.Name()); ok {
				if !validProviders[provider] {
					return fmt.Errorf("override file %s: unknown provider %q; valid providers: claude, gemini, cursor, antigravity, copilot", d.Name(), provider)
				}
				overrideFiles = append(overrideFiles, overrideFileEntry{
					Path:     path,
					Kind:     kind,
					Provider: provider,
				})
			} else if isParseableFile(path) {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %q: %w", dir, err)
	}

	projectManifest := filepath.Join(dir, ".xcaffold", "project.xcf")
	if _, statErr := os.Stat(projectManifest); statErr == nil && isParseableFile(projectManifest) {
		files = append(files, projectManifest)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *.xcf files found in directory %q", dir)
	}

	var parsedFiles []ParsedFile
	for _, f := range files {
		cfg, err := ParseFileExact(f)
		if err != nil {
			return nil, err
		}
		parsedFiles = append(parsedFiles, ParsedFile{Config: cfg, FilePath: f})
	}

	globalConfig, err := loadGlobalBase()
	if err != nil {
		return nil, fmt.Errorf("failed to load implicit global configuration: %w", err)
	}

	merged, err := mergeAllStrict(parsedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config files in %q: %w", dir, err)
	}

	if merged.Extends != "" {
		merged, err = resolveExtends(dir, merged)
		if err != nil {
			return nil, err
		}
	}

	// Implicitly overlay the project configuration on top of the global base
	merged = mergeConfigOverride(globalConfig, merged)

	// Parse override files
	for _, of := range overrideFiles {
		if err := parseOverrideFile(of, merged); err != nil {
			return nil, err
		}
	}

	// Validate that every override has a corresponding base
	if err := validateOverrideBasesExist(merged); err != nil {
		return nil, err
	}

	if err := loadExtras(dir, merged); err != nil {
		return nil, fmt.Errorf("failed to load extras: %w", err)
	}

	return merged, nil
}

func parseDirectoryRaw(dir string, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
	var files []string
	var overrideFiles []overrideFileEntry
	ignored := newParseFilter(dir)
	providerDir := filepath.Join("xcf", "provider")

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || ignored[name]) {
				return filepath.SkipDir
			}
			if rel, relErr := filepath.Rel(dir, path); relErr == nil && rel == providerDir {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".xcf") {
			if kind, provider, ok := classifyOverrideFile(d.Name()); ok {
				if !validProviders[provider] {
					return fmt.Errorf("override file %s: unknown provider %q; valid providers: claude, gemini, cursor, antigravity, copilot", d.Name(), provider)
				}
				overrideFiles = append(overrideFiles, overrideFileEntry{
					Path:     path,
					Kind:     kind,
					Provider: provider,
				})
			} else if isParseableFile(path) {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %q: %w", dir, err)
	}

	projectManifest := filepath.Join(dir, ".xcaffold", "project.xcf")
	if _, statErr := os.Stat(projectManifest); statErr == nil && isParseableFile(projectManifest) {
		files = append(files, projectManifest)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no *.xcf files found in directory %q", dir)
	}

	var parsedFiles []ParsedFile
	for _, f := range files {
		cfg, err := ParseFileExact(f, opts...)
		if err != nil {
			return nil, err
		}
		parsedFiles = append(parsedFiles, ParsedFile{Config: cfg, FilePath: f})
	}

	merged, err := mergeAllStrict(parsedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config files in %q: %w", dir, err)
	}

	// Parse override files
	for _, of := range overrideFiles {
		if err := parseOverrideFile(of, merged); err != nil {
			return nil, err
		}
	}

	// Validate that every override has a corresponding base
	if err := validateOverrideBasesExist(merged); err != nil {
		return nil, err
	}

	if err := loadExtras(dir, merged); err != nil {
		return nil, fmt.Errorf("failed to load extras: %w", err)
	}

	return merged, nil
}

func ParseFileExact(path string, opts ...parseOptionFunc) (*ast.XcaffoldConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config %q: %w", path, err)
	}
	defer f.Close()

	// Prepend source path so kind-specific parsers can derive contextual
	// metadata from the file's on-disk location (e.g., xcf/agents/<agentID>/memory/).
	// Caller-supplied opts override this by appearing later in the slice.
	opts = append([]parseOptionFunc{withSourcePath(path)}, opts...)

	config, err := parsePartial(f, opts...)
	if err != nil {
		return nil, fmt.Errorf("error in %q: %w", path, err)
	}
	return config, nil
}

// loadGlobalBase implicitly discovers and loads the global configuration
// from ~/.xcaffold/ (or falls back to legacy ~/.claude/global.xcf).
// It returns an empty config if no global config is found.
// Resources loaded from this base are tagged as Inherited=true during merge.
func loadGlobalBase() (*ast.XcaffoldConfig, error) {
	if os.Getenv("XCAFFOLD_SKIP_GLOBAL") == "true" {
		return &ast.XcaffoldConfig{}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return &ast.XcaffoldConfig{}, nil // ignore errors, just no global
	}

	xcaffoldDir := filepath.Join(home, ".xcaffold")
	if stat, err := os.Stat(xcaffoldDir); err == nil && stat.IsDir() {
		// Parse the dir, but disable global loading to avoid infinite recursion!
		// parseDirectoryRaw natively parses a dir without applying global base.
		cfg, err := parseDirectoryRaw(xcaffoldDir, withGlobalScope())
		if err != nil {
			// TODO: surface global scope parse errors once the schema is finalized.
			return &ast.XcaffoldConfig{}, nil
		}
		// If the global config itself extends something, resolve it!
		if cfg.Extends != "" {
			visited := map[string]bool{xcaffoldDir: true}
			cfg, err = resolveExtendsRecursive(xcaffoldDir, cfg, visited)
			if err != nil {
				// TODO: surface extends resolution errors once global scope ships.
				return &ast.XcaffoldConfig{}, nil
			}
		}
		return cfg, nil
	}

	return &ast.XcaffoldConfig{}, nil
}

// ParseFile reads a .xcf YAML configuration from the given path, resolving
// 'extends:' references recursively. Evaluated as a strict, single file entry point.
func ParseFile(path string) (*ast.XcaffoldConfig, error) {
	globalConfig, err := loadGlobalBase()
	if err != nil {
		return nil, fmt.Errorf("failed to load implicit global configuration: %w", err)
	}

	config, err := ParseFileExact(path)
	if err != nil {
		return nil, err
	}
	if config.Extends != "" {
		config, err = resolveExtends(filepath.Dir(path), config)
		if err != nil {
			return nil, err
		}
	}

	// Implicitly overlay the project configuration on top of the global base
	merged := mergeConfigOverride(globalConfig, config)

	if err := validateMerged(merged); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return merged, nil
}

func resolveExtends(contextDir string, config *ast.XcaffoldConfig) (*ast.XcaffoldConfig, error) {
	visited := make(map[string]bool)
	return resolveExtendsRecursive(contextDir, config, visited)
}

//nolint:gocyclo
func resolveExtendsRecursive(contextDir string, config *ast.XcaffoldConfig, visited map[string]bool) (*ast.XcaffoldConfig, error) {
	if config.Extends == "" {
		return config, nil
	}

	var basePath string
	if config.Extends == "global" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not resolve 'extends: global': %w", err)
		}

		xcaffoldDir := filepath.Join(home, ".xcaffold")
		if stat, err := os.Stat(xcaffoldDir); err == nil && stat.IsDir() {
			if visited[xcaffoldDir] {
				return nil, fmt.Errorf("circular dependency detected: global setup extends itself")
			}
			visited[xcaffoldDir] = true

			baseConfig, err := parseDirectoryRaw(xcaffoldDir, withGlobalScope())
			if err != nil {
				return nil, fmt.Errorf("failed to parse global directory %q: %w", xcaffoldDir, err)
			}
			if baseConfig.Extends != "" {
				baseConfig, err = resolveExtendsRecursive(xcaffoldDir, baseConfig, visited)
				if err != nil {
					return nil, err
				}
			}
			return mergeConfigOverride(baseConfig, config), nil
		}

		legacyPath := filepath.Join(home, ".claude", "global.xcf")
		if _, err := os.Stat(legacyPath); err == nil {
			fmt.Fprintf(os.Stderr, "WARNING: extends: global resolved from legacy path %s -- expected location is %s\n", legacyPath, xcaffoldDir)
			basePath = legacyPath
		} else {
			return nil, fmt.Errorf("could not resolve 'extends: global': no global config found")
		}
	} else if filepath.IsAbs(config.Extends) {
		basePath = config.Extends
	} else {
		basePath = filepath.Join(contextDir, config.Extends)
	}

	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("could not resolve extends path %q: %w", basePath, err)
	}

	if visited[absPath] {
		return nil, fmt.Errorf("circular extends detected: %q", absPath)
	}
	visited[absPath] = true

	parsed, err := ParseFileExact(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load base config %q: %w", config.Extends, err)
	}

	baseConfig, err := resolveExtendsRecursive(filepath.Dir(absPath), parsed, visited)
	if err != nil {
		return nil, err
	}

	return mergeConfigOverride(baseConfig, config), nil
}

// Merge operations

// mergeAllStrict is used to merge files living in the same directory.
// Duplicate maps (like Agents, Skills, etc.) cause errors.
//
//nolint:gocyclo
func mergeAllStrict(parsedFiles []ParsedFile) (*ast.XcaffoldConfig, error) {
	if len(parsedFiles) == 0 {
		return &ast.XcaffoldConfig{}, nil
	}
	merged := &ast.XcaffoldConfig{}

	agentOrigins := map[string]string{}
	skillOrigins := map[string]string{}
	ruleOrigins := map[string]string{}
	mcpOrigins := map[string]string{}
	workflowOrigins := map[string]string{}
	policyOrigins := map[string]string{}
	blueprintOrigins := map[string]string{}
	contextOrigins := map[string]string{}
	settingsOrigin := ""
	localOrigin := ""

	for _, pf := range parsedFiles {
		p := pf.Config
		f := pf.FilePath
		var err error

		if merged.Version != "" && p.Version != "" && merged.Version != p.Version {
			return nil, fmt.Errorf("conflicting versions declared: %q vs %q", merged.Version, p.Version)
		}
		if p.Version != "" {
			merged.Version = p.Version
		}

		if p.Project != nil && p.Project.Name != "" {
			if merged.Project != nil && merged.Project.Name != "" && merged.Project.Name != p.Project.Name {
				return nil, fmt.Errorf("multiple files declare project.name: %q vs %q", merged.Project.Name, p.Project.Name)
			}
			if merged.Project == nil {
				merged.Project = &ast.ProjectConfig{}
			}
			// Copy scalar metadata fields; Local and ResourceScope are merged separately below.
			if p.Project.Name != "" {
				merged.Project.Name = p.Project.Name
			}
			if p.Project.Description != "" {
				merged.Project.Description = p.Project.Description
			}
			if p.Project.Version != "" {
				merged.Project.Version = p.Project.Version
			}
			if p.Project.Author != "" {
				merged.Project.Author = p.Project.Author
			}
			if p.Project.Homepage != "" {
				merged.Project.Homepage = p.Project.Homepage
			}
			if p.Project.Repository != "" {
				merged.Project.Repository = p.Project.Repository
			}
			if p.Project.License != "" {
				merged.Project.License = p.Project.License
			}
			if p.Project.BackupDir != "" {
				merged.Project.BackupDir = p.Project.BackupDir
			}
			// Propagate targets and ref lists declared by kind: project documents.
			// These fields use yaml:"-" so they are not decoded from YAML
			// directly; only kind: project documents populate them.
			if len(p.Project.Targets) > 0 {
				merged.Project.Targets = p.Project.Targets
			}
			if len(p.Project.AgentRefs) > 0 {
				merged.Project.AgentRefs = p.Project.AgentRefs
			}
			if len(p.Project.SkillRefs) > 0 {
				merged.Project.SkillRefs = p.Project.SkillRefs
			}
			if len(p.Project.RuleRefs) > 0 {
				merged.Project.RuleRefs = p.Project.RuleRefs
			}
			if len(p.Project.WorkflowRefs) > 0 {
				merged.Project.WorkflowRefs = p.Project.WorkflowRefs
			}
			if len(p.Project.MCPRefs) > 0 {
				merged.Project.MCPRefs = p.Project.MCPRefs
			}
			if len(p.Project.PolicyRefs) > 0 {
				merged.Project.PolicyRefs = p.Project.PolicyRefs
			}
			if p.Project.Body != "" {
				merged.Project.Body = p.Project.Body
			}
		}

		if p.Extends != "" {
			if merged.Extends != "" && merged.Extends != p.Extends {
				return nil, fmt.Errorf("multiple files declare extends: %q vs %q", merged.Extends, p.Extends)
			}
			merged.Extends = p.Extends
		}

		merged.Agents, agentOrigins, err = mergeMapStrict(merged.Agents, p.Agents, "agent", agentOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.Skills, skillOrigins, err = mergeMapStrict(merged.Skills, p.Skills, "skill", skillOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.Rules, ruleOrigins, err = mergeMapStrict(merged.Rules, p.Rules, "rule", ruleOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.MCP, mcpOrigins, err = mergeMapStrict(merged.MCP, p.MCP, "mcp", mcpOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.Workflows, workflowOrigins, err = mergeMapStrict(merged.Workflows, p.Workflows, "workflow", workflowOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.Policies, policyOrigins, err = mergeMapStrict(merged.Policies, p.Policies, "policy", policyOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.Blueprints, blueprintOrigins, err = mergeMapStrict(merged.Blueprints, p.Blueprints, "blueprint name", blueprintOrigins, f)
		if err != nil {
			return nil, err
		}

		merged.Contexts, contextOrigins, err = mergeMapStrict(merged.Contexts, p.Contexts, "context", contextOrigins, f)
		if err != nil {
			return nil, err
		}

		// Hooks are additive (merge named hook blocks).
		merged.Hooks = mergeNamedHooksAdditive(merged.Hooks, p.Hooks)

		// Accumulate parse warnings from each individual file parse.
		merged.ParseWarnings = append(merged.ParseWarnings, p.ParseWarnings...)

		// Overwrite test blocks (assuming only one file declares test config).
		// Test now lives in ProjectConfig.
		if p.Project != nil {
			pTest := p.Project.Test
			if pTest.CliPath != "" || pTest.ClaudePath != "" || pTest.JudgeModel != "" {
				if merged.Project == nil {
					merged.Project = &ast.ProjectConfig{}
				}
				merged.Project.Test = pTest
			}
		}

		// Track which file first contributed non-empty settings/local.
		if settingsOrigin == "" && len(p.Settings) > 0 {
			settingsOrigin = f
		}
		if p.Project != nil && localOrigin == "" && !isEmptySettings(p.Project.Local) {
			localOrigin = f
		}

		// Deep merge settings map (conflicting scalar keys within the same named entry → error).
		merged.Settings, err = mergeSettingsMapStrict(merged.Settings, p.Settings, settingsOrigin, f)
		if err != nil {
			return nil, err
		}
		// Deep merge local block (now lives in ProjectConfig).
		if p.Project != nil {
			if merged.Project == nil {
				merged.Project = &ast.ProjectConfig{}
			}
			merged.Project.Local, err = mergeSettingsStrict(merged.Project.Local, p.Project.Local, localOrigin, f)
			if err != nil {
				return nil, err
			}
		}
	}
	return merged, nil
}

func mergeMapStrict[K comparable, V any](base, child map[K]V, kind string, baseOrigins map[K]string, childFile string) (map[K]V, map[K]string, error) {
	if base == nil && child == nil {
		return nil, baseOrigins, nil
	}
	if base == nil {
		origins := make(map[K]string, len(child))
		for k := range child {
			origins[k] = childFile
		}
		return child, origins, nil
	}
	if child == nil {
		return base, baseOrigins, nil
	}
	merged := make(map[K]V, len(base)+len(child))
	origins := make(map[K]string, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
		origins[k] = baseOrigins[k]
	}
	for k, v := range child {
		if _, exists := merged[k]; exists {
			return nil, nil, fmt.Errorf("duplicate %s ID \"%v\" found in %s and %s", kind, k, filepath.Base(origins[k]), filepath.Base(childFile))
		}
		merged[k] = v
		origins[k] = childFile
	}
	return merged, origins, nil
}

func mergeHooksAdditive(base, child ast.HookConfig) ast.HookConfig {
	if base == nil {
		return child
	}
	if child == nil {
		return base
	}
	merged := make(ast.HookConfig)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = append(merged[k], v...)
	}
	return merged
}

// mergeNamedHooksAdditive merges two map[string]NamedHookConfig values additively.
// Events within each named block are appended across base and child.
func mergeNamedHooksAdditive(base, child map[string]ast.NamedHookConfig) map[string]ast.NamedHookConfig {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	merged := make(map[string]ast.NamedHookConfig, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for name, nh := range child {
		if existing, ok := merged[name]; ok {
			existing.Events = mergeHooksAdditive(existing.Events, nh.Events)
			merged[name] = existing
		} else {
			merged[name] = nh
		}
	}
	return merged
}

// mergeSettingsMapStrict merges two map[string]SettingsConfig values from the same
// directory. Named entries are merged per-name using mergeSettingsStrict.
func mergeSettingsMapStrict(base, child map[string]ast.SettingsConfig, baseFile, childFile string) (map[string]ast.SettingsConfig, error) {
	if len(child) == 0 {
		return base, nil
	}
	if len(base) == 0 {
		return child, nil
	}
	merged := make(map[string]ast.SettingsConfig, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for name, cs := range child {
		if bs, ok := merged[name]; ok {
			result, err := mergeSettingsStrict(bs, cs, baseFile, childFile)
			if err != nil {
				return nil, err
			}
			merged[name] = result
		} else {
			merged[name] = cs
		}
	}
	return merged, nil
}

// mergeSettingsMapOverride merges two map[string]SettingsConfig for extends
// resolution. Child entries override base entries with the same name.
func mergeSettingsMapOverride(base, child map[string]ast.SettingsConfig) map[string]ast.SettingsConfig {
	if len(base) == 0 && len(child) == 0 {
		return nil
	}
	merged := make(map[string]ast.SettingsConfig, len(base)+len(child))
	for k, v := range base {
		merged[k] = v
	}
	for name, cs := range child {
		if bs, ok := merged[name]; ok {
			merged[name] = mergeSettingsOverride(bs, cs)
		} else {
			merged[name] = cs
		}
	}
	return merged
}

// mergeConfigOverride is used for extends resolution where the child overrides the base entirely.
// Base resources (those not overridden by the child) are tagged Inherited=true so renderers
// can skip them during project-scope compilation — they are already compiled at global scope.
func mergeConfigOverride(base, child *ast.XcaffoldConfig) *ast.XcaffoldConfig {
	merged := &ast.XcaffoldConfig{
		Version: child.Version, // child overrides version
	}

	if merged.Version == "" {
		merged.Version = base.Version
	}

	if base.Project != nil || child.Project != nil {
		merged.Project = &ast.ProjectConfig{}
		if base.Project != nil {
			*merged.Project = *base.Project
		}
		if child.Project != nil {
			if child.Project.Name != "" {
				merged.Project.Name = child.Project.Name
			}
			if child.Project.Description != "" {
				merged.Project.Description = child.Project.Description
			}
			if child.Project.BackupDir != "" {
				merged.Project.BackupDir = child.Project.BackupDir
			}
			// Propagate targets and ref lists from kind: project documents.
			if len(child.Project.Targets) > 0 {
				merged.Project.Targets = child.Project.Targets
			}
			if len(child.Project.AgentRefs) > 0 {
				merged.Project.AgentRefs = child.Project.AgentRefs
			}
			if len(child.Project.SkillRefs) > 0 {
				merged.Project.SkillRefs = child.Project.SkillRefs
			}
			if len(child.Project.RuleRefs) > 0 {
				merged.Project.RuleRefs = child.Project.RuleRefs
			}
			if len(child.Project.WorkflowRefs) > 0 {
				merged.Project.WorkflowRefs = child.Project.WorkflowRefs
			}
			if len(child.Project.MCPRefs) > 0 {
				merged.Project.MCPRefs = child.Project.MCPRefs
			}
			if len(child.Project.PolicyRefs) > 0 {
				merged.Project.PolicyRefs = child.Project.PolicyRefs
			}
			// Test override
			if child.Project.Test.CliPath != "" {
				merged.Project.Test.CliPath = child.Project.Test.CliPath
			}
			if child.Project.Test.ClaudePath != "" {
				merged.Project.Test.ClaudePath = child.Project.Test.ClaudePath
			}
			if child.Project.Test.JudgeModel != "" {
				merged.Project.Test.JudgeModel = child.Project.Test.JudgeModel
			}
			// Local settings override
			var baseLocal ast.SettingsConfig
			if base.Project != nil {
				baseLocal = base.Project.Local
			}
			merged.Project.Local = mergeSettingsOverride(baseLocal, child.Project.Local)

			// Project instructions fields. A set field on the child wins; an empty
			// field on the child preserves the base value (matches the same
			// convention applied to Name, Description, and other scalar fields above).
			if child.Project.Body != "" {
				merged.Project.Body = child.Project.Body
			}
		}
	}

	merged.Extends = "" // after resolving, extends is empty

	// Tag all base resources as inherited so renderers skip them during project-scope
	// compilation. Resources the child declares (same ID) are child-owned and NOT tagged.
	merged.Agents = mergeAgentsOverrideInherited(base.Agents, child.Agents)
	merged.Skills = mergeSkillsOverrideInherited(base.Skills, child.Skills)
	merged.Rules = mergeRulesOverrideInherited(base.Rules, child.Rules)
	merged.MCP = mergeMCPOverrideInherited(base.MCP, child.MCP)
	merged.Workflows = mergeWorkflowsOverrideInherited(base.Workflows, child.Workflows)
	merged.Policies = mergeMapOverride(base.Policies, child.Policies)
	merged.Blueprints = mergeMapOverride(base.Blueprints, child.Blueprints)
	merged.Contexts = mergeContextsOverrideInherited(base.Contexts, child.Contexts)
	merged.Hooks = mergeNamedHooksAdditive(base.Hooks, child.Hooks)

	merged.Settings = mergeSettingsMapOverride(base.Settings, child.Settings)

	// Preserve parse warnings from the child (project-level); base (global) warnings are discarded.
	merged.ParseWarnings = append(merged.ParseWarnings, child.ParseWarnings...)

	return merged
}

func mergeMapOverride[K comparable, V any](base, child map[K]V) map[K]V {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[K]V)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range child {
		merged[k] = v // child overrides base completely
	}
	return merged
}

// mergeMapOverrideInherited merges two maps where base resources are tagged
// Inherited=true. Child resources (which override base) take precedence and are
// NOT tagged. This is implemented per concrete type because Go generics cannot
// assign to struct fields through a type parameter without reflection.

func mergeAgentsOverrideInherited(base, child map[string]ast.AgentConfig) map[string]ast.AgentConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.AgentConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeContextsOverrideInherited(base, child map[string]ast.ContextConfig) map[string]ast.ContextConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.ContextConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeSkillsOverrideInherited(base, child map[string]ast.SkillConfig) map[string]ast.SkillConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.SkillConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeRulesOverrideInherited(base, child map[string]ast.RuleConfig) map[string]ast.RuleConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.RuleConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeMCPOverrideInherited(base, child map[string]ast.MCPConfig) map[string]ast.MCPConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.MCPConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

func mergeWorkflowsOverrideInherited(base, child map[string]ast.WorkflowConfig) map[string]ast.WorkflowConfig {
	if base == nil && child == nil {
		return nil
	}
	merged := make(map[string]ast.WorkflowConfig, len(base)+len(child))
	for k, v := range base {
		v.Inherited = true
		merged[k] = v
	}
	for k, v := range child {
		v.Inherited = false
		merged[k] = v
	}
	return merged
}

// mergeInstructionsScopesOverrideInherited merges two InstructionsScope slices where
// base scopes are tagged Inherited=true. Child scopes (same path) take precedence
// and are NOT tagged. Scopes unique to the base are tagged Inherited=true and appended.
// When base is empty, child scopes are returned verbatim (existing Inherited tags preserved).

// Validations

func validateID(kind, id string) error {
	if strings.ContainsAny(id, "\\") || strings.Contains(id, "..") {
		return fmt.Errorf("%s id contains invalid characters: %q", kind, id)
	}
	if strings.Contains(id, "/") && kind != "rule" {
		return fmt.Errorf("%s id contains invalid characters: %q", kind, id)
	}
	return nil
}

var knownTools = map[string]bool{
	"Read": true, "Write": true, "Edit": true, "MultiEdit": true,
	"Bash": true, "Glob": true, "Grep": true, "LS": true,
	"WebFetch": true, "WebSearch": true,
	"TodoRead": true, "TodoWrite": true,
	"NotebookRead": true, "NotebookEdit": true,
	"Task": true, "Computer": true, "AskUserQuestion": true,
	"Agent": true, "ExitPlanMode": true, "EnterPlanMode": true,
	"mcp": true,
}

var validHookEvents = map[string]bool{
	"PreToolUse": true, "PostToolUse": true, "PostToolUseFailure": true,
	"PermissionRequest": true, "PermissionDenied": true,
	"SessionStart": true, "SessionEnd": true,
	"UserPromptSubmit": true, "Stop": true, "StopFailure": true,
	"SubagentStart": true, "SubagentStop": true, "TeammateIdle": true,
	"TaskCreated": true, "TaskCompleted": true,
	"PreCompact": true, "PostCompact": true,
	"InstructionsLoaded": true, "ConfigChange": true,
	"CwdChanged": true, "FileChanged": true,
	"WorktreeCreate": true, "WorktreeRemove": true,
	"Elicitation": true, "ElicitationResult": true,
	"Notification": true,
}

// validRuleActivations is the set of accepted activation values for rule kind.
var validRuleActivations = map[string]bool{
	ast.RuleActivationAlways:         true,
	ast.RuleActivationPathGlob:       true,
	ast.RuleActivationModelDecided:   true,
	ast.RuleActivationManualMention:  true,
	ast.RuleActivationExplicitInvoke: true,
}

// pathFreeActivations are rule activations that must have an empty paths list.
var pathFreeActivations = map[string]bool{
	ast.RuleActivationAlways:         true,
	ast.RuleActivationModelDecided:   true,
	ast.RuleActivationManualMention:  true,
	ast.RuleActivationExplicitInvoke: true,
}

// validExcludeAgents is the set of accepted values for the exclude-agents field.
var validExcludeAgents = map[string]bool{
	"code-review": true,
	"cloud-agent": true,
}

// validateRuleActivations enforces activation enum and paths co-constraints
// across all rules in the config. It also validates exclude-agents enum values
// and emits a deprecation warning to stderr when always-apply is used without
// the activation field.
func validateRuleActivations(c *ast.XcaffoldConfig) error {
	for _, rule := range c.Rules {
		if rule.Activation != "" {
			if !validRuleActivations[rule.Activation] {
				return fmt.Errorf(
					"rule %q: activation must be one of: always, path-glob, model-decided, manual-mention, explicit-invoke (got %q)",
					rule.Name, rule.Activation,
				)
			}
			if rule.Activation == ast.RuleActivationPathGlob && len(rule.Paths) == 0 {
				return fmt.Errorf(
					"rule %q: activation %q requires at least one path in paths",
					rule.Name, rule.Activation,
				)
			}
			if pathFreeActivations[rule.Activation] && len(rule.Paths) > 0 {
				return fmt.Errorf(
					"rule %q: paths must be empty when activation is %q",
					rule.Name, rule.Activation,
				)
			}
		}
		for _, agent := range rule.ExcludeAgents {
			if !validExcludeAgents[agent] {
				return fmt.Errorf(
					"rule %q: exclude-agents value %q must be one of: code-review, cloud-agent",
					rule.Name, agent,
				)
			}
		}
		if rule.AlwaysApply != nil && rule.Activation == "" {
			fmt.Fprintf(os.Stderr,
				"DEPRECATION: rule %q uses always-apply without activation; migrate to activation: always\n",
				rule.Name,
			)
		}
	}
	return nil
}

// validLoweringStrategies is the set of accepted lowering-strategy values for
// workflow targets.<provider>.provider["lowering-strategy"].
var validLoweringStrategies = map[string]bool{
	"rule-plus-skill": true,
	"prompt-file":     true,
	"custom-command":  true,
}

// validateWorkflows enforces semantic constraints on all workflow entries:
//   - steps and top-level instructions/instructions-file are mutually exclusive
//   - every step must have a non-empty name
//   - targets.<provider>.provider["lowering-strategy"] must be a known value
//   - api-version, if set, must be "workflow/v1"
//   - step instructions-file paths may not target reserved output directories
func validateWorkflows(c *ast.XcaffoldConfig) error {
	for id, wf := range c.Workflows {
		// api-version validation
		if wf.ApiVersion != "" && wf.ApiVersion != "workflow/v1" {
			return fmt.Errorf("workflow %q: api-version %q is not supported; only \"workflow/v1\" is accepted", id, wf.ApiVersion)
		}

		// steps vs frontmatter+body mutex
		if len(wf.Steps) > 0 && wf.Body != "" {
			return fmt.Errorf("workflow %q: steps and inline body are mutually exclusive; use steps for multi-step workflows or the markdown body for single-body workflows", id)
		}

		// per-step validations
		for i, step := range wf.Steps {
			if step.Name == "" {
				return fmt.Errorf("workflow %q: step[%d] is missing a required name field", id, i)
			}
			// step body is required
			if step.Body == "" {
				return fmt.Errorf("workflow %q step %q: must define step instructions in markdown body under ## %s", id, step.Name, step.Name)
			}
		}

		// lowering-strategy enum validation across all target providers
		for provider, override := range wf.Targets {
			if override.Provider == nil {
				continue
			}
			if raw, ok := override.Provider["lowering-strategy"]; ok {
				strategy, _ := raw.(string)
				if !validLoweringStrategies[strategy] {
					return fmt.Errorf("workflow %q: targets.%s.provider[\"lowering-strategy\"] %q is invalid; must be one of: rule-plus-skill, prompt-file, custom-command", id, provider, strategy)
				}
			}
		}
	}
	return nil
}

func validatePartial(c *ast.XcaffoldConfig, globalScope bool) error {
	if err := validateIDs(c); err != nil {
		return err
	}
	var hookEvents ast.HookConfig
	if dh, ok := c.Hooks["default"]; ok {
		hookEvents = dh.Events
	}
	if err := validateHookEvents(hookEvents); err != nil {
		return err
	}

	if err := validateRuleActivations(c); err != nil {
		return err
	}
	if err := validateWorkflows(c); err != nil {
		return err
	}
	return nil
}

func validateMerged(c *ast.XcaffoldConfig) error {
	if err := validateBase(c); err != nil {
		return err
	}
	if err := validateCrossReferences(c); err != nil {
		return err
	}
	if err := validatePermissions(c); err != nil {
		return err
	}
	return nil
}

// parsePermissionRule parses a permission rule string of the form "ToolName" or
// "ToolName(pattern)". It applies strings.TrimSpace to both the tool name and
// the pattern. Returns (toolName, pattern, nil) on success, or ("", "", err).
func parsePermissionRule(rule string) (toolName, pattern string, err error) {
	idx := strings.Index(rule, "(")
	if idx == -1 {
		// bare tool name
		name := strings.TrimSpace(rule)
		if name == "" {
			return "", "", fmt.Errorf("permissions: empty rule string")
		}
		return name, "", nil
	}
	// has a pattern
	name := strings.TrimSpace(rule[:idx])
	rest := rule[idx+1:]
	if !strings.HasSuffix(rest, ")") {
		return "", "", fmt.Errorf("permissions: malformed rule %q — missing closing parenthesis", rule)
	}
	pat := strings.TrimSpace(rest[:len(rest)-1])
	if pat == "" {
		return "", "", fmt.Errorf("permissions: malformed rule %q — empty pattern", rule)
	}
	return name, pat, nil
}

// validatePermissions validates permission rule strings in settings.permissions
// and checks for agent/settings contradictions.
//
//nolint:gocyclo
func validatePermissions(c *ast.XcaffoldConfig) error {
	settings := c.Settings["default"]
	if settings.Permissions == nil {
		return nil
	}
	p := settings.Permissions

	allowSet := make(map[string]bool)
	denySet := make(map[string]bool)
	askSet := make(map[string]bool)

	for _, rule := range p.Allow {
		name, _, err := parsePermissionRule(rule)
		if err != nil {
			return fmt.Errorf("invalid .xcf configuration: %w", err)
		}
		if !knownTools[name] {
			return fmt.Errorf("permissions: unknown tool %q in allow rule %q", name, rule)
		}
		allowSet[rule] = true
	}
	for _, rule := range p.Deny {
		name, _, err := parsePermissionRule(rule)
		if err != nil {
			return fmt.Errorf("invalid .xcf configuration: %w", err)
		}
		if !knownTools[name] {
			return fmt.Errorf("permissions: unknown tool %q in deny rule %q", name, rule)
		}
		denySet[rule] = true
	}
	for _, rule := range p.Ask {
		name, _, err := parsePermissionRule(rule)
		if err != nil {
			return fmt.Errorf("invalid .xcf configuration: %w", err)
		}
		if !knownTools[name] {
			return fmt.Errorf("permissions: unknown tool %q in ask rule %q", name, rule)
		}
		askSet[rule] = true
	}

	// Contradiction checks
	for rule := range allowSet {
		if denySet[rule] {
			return fmt.Errorf("permissions: rule %q appears in both allow and deny", rule)
		}
		if askSet[rule] {
			return fmt.Errorf("permissions: rule %q appears in both allow and ask", rule)
		}
	}

	// Agent cross-reference checks
	for agentID, agent := range c.Agents {
		// disallowed-tools vs settings.permissions.allow
		for _, tool := range agent.DisallowedTools {
			for rule := range allowSet {
				ruleName, _, _ := parsePermissionRule(rule)
				if ruleName == tool {
					return fmt.Errorf("agent %q: tool %q is in disallowed-tools but also in settings.permissions.allow", agentID, tool)
				}
			}
		}
		// agent.tools vs settings.permissions.deny (bare deny only)
		for _, tool := range agent.Tools {
			if denySet[tool] {
				return fmt.Errorf("agent %q: tool %q is required by agent but is unconditionally denied in settings.permissions.deny", agentID, tool)
			}
		}
	}

	return nil
}

func validateBase(c *ast.XcaffoldConfig) error {
	if c.Version == "" {
		return fmt.Errorf("version is required (e.g. \"1.0\")")
	}

	if c.Extends == "" && c.Project != nil {
		name := strings.TrimSpace(c.Project.Name)
		if name == "" {
			return fmt.Errorf("project.name is required and must not be empty unless extending another config")
		}
	}

	return nil
}

// validateProjectInstructions checks mutual exclusivity, duplicate paths, and
// enum values for ProjectConfig instructions fields.

func validateResourceIDs[T any](resources map[string]T, kind string) error {
	for id := range resources {
		if err := validateID(kind, id); err != nil {
			return err
		}
	}
	return nil
}

func validateIDs(c *ast.XcaffoldConfig) error {
	if err := validateResourceIDs(c.Agents, "agent"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Skills, "skill"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Rules, "rule"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Hooks, "hook-block"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.MCP, "mcp"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Workflows, "workflow"); err != nil {
		return err
	}
	if err := validateResourceIDs(c.Policies, "policy"); err != nil {
		return err
	}
	return nil
}

func validateHookEvents(hooks ast.HookConfig) error {
	for event := range hooks {
		if !validHookEvents[event] {
			return fmt.Errorf("unknown hook event %q; see documentation for supported lifecycle events", event)
		}
	}
	return nil
}

func validateCrossReferences(c *ast.XcaffoldConfig) error {
	for agentID, agent := range c.Agents {
		for _, skillID := range agent.Skills {
			if _, ok := c.Skills[skillID]; !ok {
				return fmt.Errorf("agent %q references undefined skill %q", agentID, skillID)
			}
		}
		for _, ruleID := range agent.Rules {
			if _, ok := c.Rules[ruleID]; !ok {
				return fmt.Errorf("agent %q references undefined rule %q", agentID, ruleID)
			}
		}
		for _, mcpID := range agent.MCP {
			if _, ok := c.MCP[mcpID]; !ok {
				return fmt.Errorf("agent %q references undefined mcp server %q", agentID, mcpID)
			}
		}
	}

	if c.Project != nil {
		for _, policyRef := range c.Project.PolicyRefs {
			if _, ok := c.Policies[policyRef]; !ok {
				return fmt.Errorf("project references policy %q but no policy with that name was found", policyRef)
			}
		}
	}

	return nil
}

// Diagnostic represents a single validation finding returned by ValidateFile.
// Severity is either "error" or "warning". Errors cause non-zero exits in
// xcaffold validate; warnings are informational only.
type Diagnostic struct {
	Severity string // "error" or "warning"
	Message  string
}

// knownPlugins is the hardcoded registry of officially supported plugin IDs.
// Plugin validation produces warnings only — custom plugins are not errors.
var knownPlugins = map[string]bool{
	"commit-commands":   true,
	"security-guidance": true,
	"code-review":       true,
	"pr-review-toolkit": true,
}

// ValidateFile parses the .xcf file at path, runs file-existence checks and
// plugin validation, and returns all diagnostics. ParseFile already runs
// validateCrossReferences internally, so this function does not duplicate it.
func ValidateFile(path string) []Diagnostic {
	config, err := ParseFile(path)
	if err != nil {
		return []Diagnostic{{Severity: "error", Message: err.Error()}}
	}
	var diags []Diagnostic
	diags = append(diags, validateFileRefs(config, filepath.Dir(path))...)
	diags = append(diags, validatePlugins(config)...)
	return diags
}

// validateFileRefs checks that instructions-file paths and skill references
// exist on disk, and detects duplicate IDs across resource types.
//
//nolint:gocyclo
func validateFileRefs(c *ast.XcaffoldConfig, baseDir string) []Diagnostic {
	var diags []Diagnostic

	// Skill subdirectory file sets: warn on missing files for references, scripts, assets, examples
	for id, skill := range c.Skills {
		for _, subdirPaths := range []struct {
			subdir string
			paths  []string
		}{
			{"references", skill.References},
			{"scripts", skill.Scripts},
			{"assets", skill.Assets},
			{"examples", skill.Examples},
		} {
			for _, ref := range subdirPaths.paths {
				if ref == "" {
					continue
				}
				abs := filepath.Join(baseDir, ref)
				if _, err := os.Stat(abs); os.IsNotExist(err) {
					diags = append(diags, Diagnostic{
						Severity: "warning",
						Message:  fmt.Sprintf("skill %q %s file that does not exist: %q", id, subdirPaths.subdir, ref),
					})
				}
			}
		}
	}

	// Duplicate ID check across resource types
	seen := make(map[string][]string) // id -> []resourceType
	for id := range c.Agents {
		seen[id] = append(seen[id], "agent")
	}
	for id := range c.Skills {
		seen[id] = append(seen[id], "skill")
	}
	for id := range c.Rules {
		seen[id] = append(seen[id], "rule")
	}
	for id := range c.Workflows {
		seen[id] = append(seen[id], "workflow")
	}
	for id, types := range seen {
		if len(types) > 1 {
			diags = append(diags, Diagnostic{
				Severity: "warning",
				Message:  fmt.Sprintf("ID %q is used in both %s and %s; this may cause confusion", id, types[0], types[1]),
			})
		}
	}

	return diags
}

// validatePlugins checks settings.enabledPlugins and local.enabledPlugins
// against the knownPlugins registry. Unknown plugins produce warnings only.
func validatePlugins(c *ast.XcaffoldConfig) []Diagnostic {
	var diags []Diagnostic
	check := func(plugins map[string]bool, block string) {
		for id := range plugins {
			if !knownPlugins[id] {
				diags = append(diags, Diagnostic{
					Severity: "warning",
					Message: fmt.Sprintf(
						"%s.enabledPlugins: unknown plugin %q; known plugins: commit-commands, security-guidance, code-review, pr-review-toolkit",
						block, id,
					),
				})
			}
		}
	}
	check(c.Settings["default"].EnabledPlugins, "settings")
	if c.Project != nil {
		check(c.Project.Local.EnabledPlugins, "local")
	}
	return diags
}

// reservedOutputPrefixes are compiler output directories and well-known agent
// config paths. instructions-file paths starting with these prefixes create
// circular dependencies where the compiler reads its own output, or reference
// files managed by other providers outside the project tree.
var reservedOutputPrefixes = []string{
	"~/.claude/",
	"~/.gemini/",
	".agents/",
	".antigravity/",
	".claude/",
	".cursor/",
	".cursorrules",
	".gemini/",
}

// reservedOutputFilenames are root-level files written directly by the compiler.
// Pointing instructions-file at one of these creates a circular read dependency.
var reservedOutputFilenames = []string{
	"CLAUDE.md",
	"AGENTS.md",
	"GEMINI.md",
}

// reservedOutputPaths are specific files and directories written by the compiler.
// Exact-match and prefix-match are both applied (directory entries end with /).
var reservedOutputPaths = []string{
	".github/copilot-instructions.md",
	".github/instructions/",
	".github/prompts/",
}
