package resolver

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fileHasProjectKind reads the first bytes of a .xcaf file and returns true
// if its top-level kind field is "project". Uses lightweight string parsing
// rather than YAML unmarshaling to avoid adding a YAML dependency.
func fileHasProjectKind(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n == 0 {
		return false
	}

	for _, line := range strings.Split(string(buf[:n]), "\n") {
		// Only match non-indented kind: to avoid nested YAML structures
		if !strings.HasPrefix(line, "kind:") {
			continue
		}
		val := strings.TrimSpace(line[5:]) // after "kind:"
		val = strings.Trim(val, "\"'")
		return val == "project"
	}
	return false
}

// DirHasProjectManifest checks whether dir contains any top-level .xcaf file
// whose kind field is "project". It checks project.xcaf first as a fast path,
// then scans remaining .xcaf files if needed.
func DirHasProjectManifest(dir string) bool {
	// Fast path: most projects use the conventional name.
	if fileHasProjectKind(filepath.Join(dir, "project.xcaf")) {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".xcaf") || e.Name() == "project.xcaf" {
			continue
		}
		if fileHasProjectKind(filepath.Join(dir, e.Name())) {
			return true
		}
	}
	return false
}

// FindProjectManifestPath returns the path to the first .xcaf file with
// kind:project in dir, or empty string if none found.
func FindProjectManifestPath(dir string) string {
	p := filepath.Join(dir, "project.xcaf")
	if fileHasProjectKind(p) {
		return p
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".xcaf") || e.Name() == "project.xcaf" {
			continue
		}
		p = filepath.Join(dir, e.Name())
		if fileHasProjectKind(p) {
			return p
		}
	}
	return ""
}

// FindProjectRoot walks up from start directory looking for a directory with kind:project.
// Returns the directory containing a project manifest, or an empty string if not found.
func FindProjectRoot(start string) string {
	curr := start
	for {
		if DirHasProjectManifest(curr) {
			return curr
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			return ""
		}
		curr = parent
	}
}

// dirContainsXCAF returns true if the directory contains at least one *.xcaf file
// at the top level (not recursively). Hidden directories are ignored.
func dirContainsXCAF(dir string) bool {
	// Check for project.xcaf at root level
	if _, err := os.Stat(filepath.Join(dir, "project.xcaf")); err == nil {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".xcaf") {
			return true
		}
	}
	return false
}

// FindConfigDir walks up from start looking for the nearest directory that
// contains at least one *.xcaf file. The search stops at (and includes) the
// home boundary. Returns the directory path or an error if none is found.
func FindConfigDir(start, home string) (string, error) {
	curr := start
	for {
		if dirContainsXCAF(curr) {
			return curr, nil
		}
		if curr == home {
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break // filesystem root
		}
		curr = parent
	}
	return "", fmt.Errorf("no *.xcaf files found between %q and %q\n\nHint: run 'xcaffold init' to create one, or use --config to specify a path", start, home)
}

// FindXCAFFiles returns all *.xcaf file paths in the given directory (recursive),
// sorted alphabetically. Hidden directories and node_modules are skipped.
// Nested projects (directories containing kind:project) are treated as boundaries.
func FindXCAFFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != dir && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			if path != dir && DirHasProjectManifest(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".xcaf") && d.Name() != "registry.xcaf" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan %q for xcaf files: %w", dir, err)
	}
	sort.Strings(files)
	return files, nil
}

// FindVariableFiles returns paths to discovered variable files (project.vars,
// project.<target>.vars) in the project's xcaf/ directory.
func FindVariableFiles(baseDir, target, customFile string) []string {
	var files []string
	xcafDir := filepath.Join(baseDir, "xcaf")

	if customFile != "" {
		if filepath.IsAbs(customFile) {
			files = append(files, customFile)
		} else {
			files = append(files, filepath.Join(baseDir, customFile))
		}
	} else {
		p := filepath.Join(xcafDir, "project.vars")
		if _, err := os.Stat(p); err == nil {
			files = append(files, p)
		}
	}

	if target != "" {
		p := filepath.Join(xcafDir, fmt.Sprintf("project.%s.vars", target))
		if _, err := os.Stat(p); err == nil {
			files = append(files, p)
		}
	}

	return files
}
