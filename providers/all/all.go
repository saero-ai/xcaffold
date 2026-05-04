// Package all imports every provider package for their registration side effects.
package all

import (
	_ "github.com/saero-ai/xcaffold/providers/antigravity"
	_ "github.com/saero-ai/xcaffold/providers/claude"
	_ "github.com/saero-ai/xcaffold/providers/copilot"
	_ "github.com/saero-ai/xcaffold/providers/cursor"
	_ "github.com/saero-ai/xcaffold/providers/gemini"
)
