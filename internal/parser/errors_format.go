package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RelPathFrom returns path relative to projectRoot, or the original path if rel fails.
func RelPathFrom(projectRoot, absPath string) string {
	if projectRoot == "" || absPath == "" {
		return absPath
	}
	rel, err := filepath.Rel(projectRoot, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return absPath
	}
	return filepath.ToSlash(rel)
}

// UnwrapParseFileError strips nested "error in \"path\":" wrappers from ParseFileExact.
func UnwrapParseFileError(err error) string {
	msg := err.Error()
	for {
		const prefix = "error in \""
		if !strings.HasPrefix(msg, prefix) {
			break
		}
		rest := msg[len(prefix):]
		idx := strings.Index(rest, "\": ")
		if idx < 0 {
			break
		}
		msg = rest[idx+3:]
	}
	return msg
}

// FormatParseFileError formats a per-file parse failure as "<rel-path>: <message>".
func FormatParseFileError(projectRoot, file string, err error) error {
	if err == nil {
		return nil
	}
	rel := RelPathFrom(projectRoot, file)
	return fmt.Errorf("%s: %s", rel, UnwrapParseFileError(err))
}

// FormatValidationError normalizes validation error text for xcaffold validate output.
func FormatValidationError(projectRoot string, err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, prefix := range []string{
		"validation failed for project configuration: ",
		"failed to load variables: ",
	} {
		msg = strings.TrimPrefix(msg, prefix)
	}
	if strings.HasPrefix(msg, "failed to merge config files") {
		if idx := strings.Index(msg, ": "); idx >= 0 {
			msg = msg[idx+2:]
		}
	}
	return rewriteEmbeddedParseErrors(projectRoot, msg)
}

func rewriteEmbeddedParseErrors(projectRoot, msg string) string {
	const prefix = "error in \""
	for {
		i := strings.Index(msg, prefix)
		if i < 0 {
			return msg
		}
		rest := msg[i+len(prefix):]
		end := strings.Index(rest, "\": ")
		if end < 0 {
			return msg
		}
		absPath := rest[:end]
		inner := rest[end+3:]
		rel := RelPathFrom(projectRoot, absPath)
		msg = msg[:i] + rel + ": " + inner
	}
}

// FormatDiagnosticLine returns "<file>: <message>" when FilePath is set.
func FormatDiagnosticLine(d Diagnostic) string {
	if d.FilePath == "" {
		return d.Message
	}
	return d.FilePath + ": " + d.Message
}
