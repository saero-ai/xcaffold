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
	"github.com/saero-ai/xcaffold/providers"
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

// ParseDirOption configures behaviour of ParseDirectory.
type ParseDirOption func(*parseDirConfig)

type parseDirConfig struct {
	skipGlobal bool
}

// WithSkipGlobal prevents ParseDirectory from loading the implicit global
// configuration (~/.xcaffold/). Use this for project-scoped validation.
func WithSkipGlobal() ParseDirOption {
	return func(c *parseDirConfig) { c.skipGlobal = true }
}

func resolveParseDirOptions(opts []ParseDirOption) parseDirConfig {
	var cfg parseDirConfig
	for _, fn := range opts {
		fn(&cfg)
	}
	return cfg
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

	// Check if filename is a canonical kind filename (<kind>.xcf or <kind>.<provider>.xcf)
	filename := parts[len(parts)-1]
	baseFilename := strings.TrimSuffix(filename, ".xcf")
	// Strip provider suffix: "rule.claude" → "rule"
	if dotIdx := strings.LastIndex(baseFilename, "."); dotIdx >= 0 {
		baseFilename = baseFilename[:dotIdx]
	}

	if baseFilename == kind {
		// Canonical convention: name from segments between kind-dir and filename
		nameSegments := parts[xcfIdx+2 : len(parts)-1]
		if len(nameSegments) == 0 {
			return "", ""
		}
		name = strings.Join(nameSegments, "/")
		return kind, name
	}

	// Legacy: name is parts[xcfIdx+2] with .xcf stripped
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
func ParseDirectory(dir string, opts ...ParseDirOption) (*ast.XcaffoldConfig, error) {
	dirOpts := resolveParseDirOptions(opts)
	merged, err := parseDirectoryUnvalidated(dir, dirOpts)
	if err != nil {
		return nil, err
	}

	if err := validateMerged(merged); err != nil {
		return nil, fmt.Errorf("validation failed for project configuration: %w", err)
	}

	return merged, nil
}

// ParseDirectoryWithCrossRefWarnings parses a directory and returns the config plus
// any cross-reference validation issues separately. Structural errors still return
// as errors. Cross-reference issues are returned as a separate list for caller handling.
func ParseDirectoryWithCrossRefWarnings(dir string, opts ...ParseDirOption) (*ast.XcaffoldConfig, []CrossReferenceIssue, error) {
	dirOpts := resolveParseDirOptions(opts)
	merged, err := parseDirectoryUnvalidated(dir, dirOpts)
	if err != nil {
		return nil, nil, err
	}

	// Validate structural rules (base + permissions), but not cross-references
	if err := validateMergedStructural(merged); err != nil {
		return nil, nil, fmt.Errorf("validation failed for project configuration: %w", err)
	}

	// Collect cross-reference issues separately
	issues := validateCrossReferencesAsList(merged)

	return merged, issues, nil
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

func parseDirectoryUnvalidated(dir string, dirOpts parseDirConfig) (*ast.XcaffoldConfig, error) {
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
				if !providers.IsRegistered(provider) {
					return fmt.Errorf("override file %s: unknown provider %q; valid providers: %s", d.Name(), provider, strings.Join(providers.RegisteredNames(), ", "))
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

	var globalConfig *ast.XcaffoldConfig
	if dirOpts.skipGlobal {
		globalConfig = &ast.XcaffoldConfig{}
	} else {
		var loadErr error
		globalConfig, loadErr = loadGlobalBase()
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load implicit global configuration: %w", loadErr)
		}
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
				if !providers.IsRegistered(provider) {
					return fmt.Errorf("override file %s: unknown provider %q; valid providers: %s", d.Name(), provider, strings.Join(providers.RegisteredNames(), ", "))
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
