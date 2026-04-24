// Package output defines the Output type shared across compiler and renderer
// packages. It is kept in a standalone package to prevent import cycles between
// internal/compiler and internal/renderer.
package output

// Output holds the in-memory result of a compilation pass.
type Output struct {
	// Files maps a clean, relative output path to its rendered content.
	// Keys are guaranteed to be cleaned with filepath.Clean before insertion.
	Files map[string]string
	// RootFiles maps a clean, relative project-root path to its rendered content.
	// These files bypass the path-safety directory confinement.
	RootFiles map[string]string
}
