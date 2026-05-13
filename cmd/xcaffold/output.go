package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// colorEnabled returns true when ANSI color output is appropriate:
// stdout is a real TTY, --no-color is not set, and NO_COLOR is unset.
func colorEnabled() bool {
	if noColorFlag {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func applyColor(code, s string) string {
	if !colorEnabled() {
		return s
	}
	return code + s + "\033[0m"
}

func colorGreen(s string) string  { return applyColor("\033[32m", s) }
func colorYellow(s string) string { return applyColor("\033[33m", s) }
func colorRed(s string) string    { return applyColor("\033[31m", s) }
func bold(s string) string        { return applyColor("\033[1m", s) }
func dim(s string) string         { return applyColor("\033[2m", s) }

// Glyph helpers — UTF-8 in TTY, ASCII fallback otherwise.
func glyphOK() string    { return pickGlyph("✓", "ok") }
func glyphErr() string   { return pickGlyph("✗", "!!") }
func glyphSrc() string   { return pickGlyph("△", "**") }
func glyphNever() string { return pickGlyph("–", "--") }
func glyphArrow() string { return pickGlyph("→", "->") }
func glyphDot() string   { return pickGlyph("·", ".") }

func pickGlyph(unicode, ascii string) string {
	if colorEnabled() {
		return unicode
	}
	return ascii
}

// headerInfo bundles parameters for formatHeader to reduce function arity.
type headerInfo struct {
	project     string
	blueprint   string
	isGlobal    bool
	provider    string
	lastApplied string
}

// formatHeader builds the breadcrumb header line.
// Format: <project>  ·  [blueprint: <name>  ·  ][global scope  ·  ][<provider>  ·  ]<context>
func formatHeader(info headerInfo) string {
	sep := "  " + glyphDot() + "  "
	var parts []string

	if info.isGlobal {
		parts = append(parts, "~")
		parts = append(parts, "global scope")
	} else {
		parts = append(parts, info.project)
	}

	if info.blueprint != "" {
		parts = append(parts, "blueprint: "+info.blueprint)
	}

	if info.provider != "" {
		parts = append(parts, info.provider)
	}

	var context string
	if info.lastApplied == "" {
		context = "never applied"
	} else {
		elapsed := formatElapsed(info.lastApplied)
		if info.provider != "" {
			context = "applied " + elapsed
		} else {
			context = "last applied " + elapsed
		}
	}
	parts = append(parts, context)

	return strings.Join(parts, sep)
}

// formatElapsed returns a human-readable elapsed time string covering all time buckets.
func formatElapsed(lastApplied string) string {
	t, err := time.Parse(time.RFC3339, lastApplied)
	if err != nil {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	case d < 28*24*time.Hour:
		return fmt.Sprintf("%d weeks ago", int(d.Hours()/(7*24)))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%d months ago", int(d.Hours()/(30*24)))
	default:
		return fmt.Sprintf("%d years ago", int(d.Hours()/(365*24)))
	}
}

// formatArtifactPath strips the internal "root:" prefix and returns (displayPath, isRoot).
func formatArtifactPath(path string) (display string, isRoot bool) {
	const rootPrefix = "root:"
	if strings.HasPrefix(path, rootPrefix) {
		return strings.TrimPrefix(path, rootPrefix), true
	}
	return path, false
}

// plural returns sing when n==1, plur otherwise.
func plural(n int, sing, plur string) string {
	if n == 1 {
		return sing
	}
	return plur
}
