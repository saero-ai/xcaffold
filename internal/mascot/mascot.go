package mascot

import (
	"fmt"
	"time"
)

// ANSI Color constants
const (
	lime     = "\033[38;2;198;243;1m"
	charcoal = "\033[38;2;7;10;6m"
	reset    = "\033[0m"
	block    = "█"
	half     = "▄"
	hide     = "\033[?25l"
	show     = "\033[?25h"
)

// Frames for the mascot drop-in animation
var frames = [][]string{
	// Drop
	{
		"            " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "            ",
		"        " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "        ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"            " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "            ",
		"",
	},
	// Squat
	{
		"",
		"",
		"        " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "        ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "                " + lime + "██" + reset + "" + lime + "██" + reset + "    ",
	},
	// Blink
	{
		"        " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "        ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"      " + lime + "██" + reset + "" + lime + "██" + reset + "            " + lime + "██" + reset + "" + lime + "██" + reset + "      ",
		"      " + lime + "██" + reset + "" + lime + "██" + reset + "            " + lime + "██" + reset + "" + lime + "██" + reset + "      ",
	},
	// Think (surprise eye wide open)
	{
		"        " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "        ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"      " + lime + "██" + reset + "" + lime + "██" + reset + "            " + lime + "██" + reset + "" + lime + "██" + reset + "      ",
		"      " + lime + "██" + reset + "" + lime + "██" + reset + "            " + lime + "██" + reset + "" + lime + "██" + reset + "      ",
	},
	// Idle
	{
		"        " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "        ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + charcoal + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"  " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "  ",
		"    " + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "" + lime + "██" + reset + "    ",
		"      " + lime + "██" + reset + "" + lime + "██" + reset + "            " + lime + "██" + reset + "" + lime + "██" + reset + "      ",
		"      " + lime + "██" + reset + "" + lime + "██" + reset + "            " + lime + "██" + reset + "" + lime + "██" + reset + "      ",
	},
}

// AnimateWelcome plays a bouncy repeating mascot animation in the terminal.
func AnimateWelcome() {
	// Print a blank 11-line canvas to establish spacing
	fmt.Print("\n\n\n\n\n\n\n\n\n\n\n")

	fmt.Print(hide) // Hide cursor
	defer fmt.Print(show)

	sequence := []struct {
		frame int
		dur   int
	}{
		{0, 100}, // Drop
		{1, 150}, // Squat
		{3, 150}, // Normal
		{2, 120}, // Blink
		{3, 150}, // Normal
		{1, 150}, // Squat bounce
		{3, 150}, // Normal
		{2, 120}, // Blink
		{3, 150}, // Normal
		{1, 150}, // Squat bounce
		{3, 150}, // Normal
		{2, 120}, // Blink
		{4, 0},   // Idle final
	}

	for _, step := range sequence {
		// Move cursor UP 11 lines to overwrite
		fmt.Print("\033[11A")

		for _, line := range frames[step.frame] {
			fmt.Printf("%s\033[K\n", line) // \033[K clears rest of line
		}

		if step.dur > 0 {
			time.Sleep(time.Duration(step.dur) * time.Millisecond)
		}
	}

	fmt.Println()
}

// PrintBanner prints the static final banner layout, placing the mascot visually
// to the left of the text block.
func PrintBanner() {
	// Move cursor up to align the banner with the middle rows of Xaff (up 8 lines)
	fmt.Print("\033[8A")

	lines := []string{
		fmt.Sprintf("          %s", "xcaffold init"),
		fmt.Sprintf("          %s", "The deterministic agent configuration compiler."),
		fmt.Sprintf("          %s", "──────────────────────────────────────────────────"),
		"",
		fmt.Sprintf("          %s", "Welcome! Let's scaffold your agents."),
	}

	for _, line := range lines {
		// Advance cursor forward horizontally by 34 columns to clear Xaff silhouette
		fmt.Printf("\033[34C%s\n", line)
	}
	// Move down past the rest of the 11-line Xaff (8 up -> 5 down -> means we need 3 more to clear bottom)
	fmt.Print("\n\n\n\n")
}
