// Package prompt provides simple, dependency-free terminal prompt helpers
// for interactive CLI wizards. All input is read from an [io.Reader] so
// that callers can inject a [strings.Reader] in tests.
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
)

// P is a prompt session bound to specific input/output streams.
// Use [New] to create one, or call the package-level helpers which
// default to os.Stdin / os.Stdout.
type P struct {
	in  *bufio.Reader
	out io.Writer
}

// SelectOption represents a single item in a multi-select list.
type SelectOption struct {
	Label    string // Display label (e.g., ".claude — 12 agent(s)")
	Value    string // Machine value (e.g., ".claude")
	Selected bool   // Pre-selected by default?
}

// New creates a prompt session reading from r and writing to w.
// Pass os.Stdin / os.Stdout for normal interactive use, or a
// *strings.Reader / *bytes.Buffer in tests.
func New(r io.Reader, w io.Writer) *P {
	return &P{in: bufio.NewReader(r), out: w}
}

// defaultPrompt is the package-level session wired to the real terminal.
var defaultPrompt = New(os.Stdin, os.Stdout)

// Ask prints a question and reads a line from the user. If the user
// presses Enter without typing anything, defaultVal is returned.
// Typical usage: Ask("Project name", "my-project")
func (p *P) Ask(question, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Fprintf(p.out, "  %s [%s]: ", question, defaultVal)
	} else {
		fmt.Fprintf(p.out, "  %s: ", question)
	}
	line, err := p.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// Confirm prints a yes/no question and returns true if the user
// answers "y" or "yes" (case-insensitive). If the user presses
// Enter without typing, defaultYes controls the result.
//
//	[Y/n] — displayed when defaultYes is true
//	[y/N] — displayed when defaultYes is false
func (p *P) Confirm(question string, defaultYes bool) (bool, error) {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	fmt.Fprintf(p.out, "  %s %s: ", question, hint)
	line, err := p.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	switch line {
	case "":
		return defaultYes, nil
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// MultiSelect presents an interactive checkbox selector using arrow keys
// and space/x to toggle. Returns the selected values. All options with
// Selected=true are pre-checked.
func (p *P) MultiSelect(title string, options []SelectOption) ([]string, error) {
	var huhOpts []huh.Option[string]
	for _, o := range options {
		huhOpts = append(huhOpts, huh.NewOption(o.Label, o.Value))
	}

	// Pre-select defaults
	var selected []string
	for _, o := range options {
		if o.Selected {
			selected = append(selected, o.Value)
		}
	}

	ms := huh.NewMultiSelect[string]().
		Title(title).
		Options(huhOpts...).
		Value(&selected)

	form := huh.NewForm(huh.NewGroup(ms))
	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}

// Ask is a package-level convenience that calls Ask on the default
// (os.Stdin / os.Stdout) session.
func Ask(question, defaultVal string) (string, error) {
	return defaultPrompt.Ask(question, defaultVal)
}

// Confirm is a package-level convenience that calls Confirm on the
// default session.
func Confirm(question string, defaultYes bool) (bool, error) {
	return defaultPrompt.Confirm(question, defaultYes)
}

// MultiSelect is a package-level convenience that calls MultiSelect
// on the default (os.Stdin / os.Stdout) session.
func MultiSelect(title string, options []SelectOption) ([]string, error) {
	return defaultPrompt.MultiSelect(title, options)
}
