package bir

import (
	"regexp"
	"strings"
)

// constraintPattern matches lines containing directive keywords at word boundaries.
// Case-insensitive. Matches: MUST, NEVER, ALWAYS, DO NOT, MANDATORY, REQUIRED.
var constraintPattern = regexp.MustCompile(`(?i)\b(MUST|NEVER|ALWAYS|DO\s+NOT|MANDATORY|REQUIRED)\b`)

// numberedStepPattern matches lines that begin a numbered list item (e.g. "1. ", "12. ").
var numberedStepPattern = regexp.MustCompile(`^\d+\.\s`)

// stepsHeadingPattern matches a markdown "## Steps" heading (case-insensitive).
var stepsHeadingPattern = regexp.MustCompile(`(?i)^##\s+Steps\s*$`)

// turboPattern matches a line containing the // turbo annotation.
var turboPattern = regexp.MustCompile(`//\s*turbo`)

// nextHeadingPattern matches any markdown level-2 heading line.
var nextHeadingPattern = regexp.MustCompile(`^##\s+`)

// DetectIntents performs static, deterministic analysis of content and returns
// zero or more FunctionalIntent values describing the semantic patterns found.
// No LLM calls are made — all detection is regex/heuristic based.
func DetectIntents(content string) []FunctionalIntent {
	if content == "" {
		return nil
	}

	var intents []FunctionalIntent

	if intent, ok := detectProcedure(content); ok {
		intents = append(intents, intent)
	}
	if intent, ok := detectConstraint(content); ok {
		intents = append(intents, intent)
	}
	if intent, ok := detectAutomation(content); ok {
		intents = append(intents, intent)
	}

	return intents
}

// detectProcedure returns a procedure intent if the content contains numbered
// steps (e.g. "1. ", "2. ") or a "## Steps" heading. The full contiguous
// section is preserved in Content — not just the numbered lines — so that
// explanatory text, code blocks, and context between steps survive.
func detectProcedure(content string) (FunctionalIntent, bool) {
	lines := splitLines(content)

	hasStepsHeading := false
	stepsHeadingIdx := -1
	firstNumberedIdx := -1

	for i, line := range lines {
		if stepsHeadingIdx == -1 && stepsHeadingPattern.MatchString(line) {
			hasStepsHeading = true
			stepsHeadingIdx = i
		}
		if firstNumberedIdx == -1 && numberedStepPattern.MatchString(line) {
			firstNumberedIdx = i
		}
	}

	if !hasStepsHeading && firstNumberedIdx == -1 {
		return FunctionalIntent{}, false
	}

	// Determine where the section starts: prefer the ## Steps heading when
	// present so the heading itself is included in Content.
	startIdx := firstNumberedIdx
	if stepsHeadingIdx >= 0 {
		startIdx = stepsHeadingIdx
	}

	// Find the end: the next ## heading after startIdx, or end of content.
	endIdx := len(lines)
	for i := startIdx + 1; i < len(lines); i++ {
		if nextHeadingPattern.MatchString(lines[i]) {
			endIdx = i
			break
		}
	}

	extracted := strings.TrimSpace(strings.Join(lines[startIdx:endIdx], "\n"))

	source := "numbered steps"
	if hasStepsHeading {
		source = "## Steps section"
	}

	return FunctionalIntent{
		Type:    IntentProcedure,
		Content: extracted,
		Source:  source,
	}, true
}

// detectConstraint returns a constraint intent if any line contains a directive
// keyword (MUST, NEVER, ALWAYS, DO NOT, MANDATORY, REQUIRED) at a word boundary.
func detectConstraint(content string) (FunctionalIntent, bool) {
	lines := splitLines(content)

	var matched []string
	for _, line := range lines {
		if constraintPattern.MatchString(line) {
			matched = append(matched, line)
		}
	}

	if len(matched) == 0 {
		return FunctionalIntent{}, false
	}

	return FunctionalIntent{
		Type:    IntentConstraint,
		Content: strings.Join(matched, "\n"),
		Source:  "constraint keyword",
	}, true
}

// detectAutomation returns an automation intent if the content contains a
// // turbo annotation on any line.
func detectAutomation(content string) (FunctionalIntent, bool) {
	lines := splitLines(content)

	var matched []string
	for _, line := range lines {
		if turboPattern.MatchString(line) {
			matched = append(matched, line)
		}
	}

	if len(matched) == 0 {
		return FunctionalIntent{}, false
	}

	return FunctionalIntent{
		Type:    IntentAutomation,
		Content: strings.Join(matched, "\n"),
		Source:  "// turbo annotation",
	}, true
}

// splitLines splits content on newlines, normalizing CRLF to LF first.
func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}
