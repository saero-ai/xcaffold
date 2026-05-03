package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RewriteHookCommandPath replaces source-relative script paths in a hook
// command string with destination-relative paths. If the command does not
// reference srcBase, it is returned unchanged.
func RewriteHookCommandPath(command, srcBase, dstBase string) string {
	if !strings.Contains(command, srcBase) {
		return command
	}
	for _, part := range strings.Fields(command) {
		if strings.HasPrefix(part, srcBase) {
			filename := filepath.Base(part)
			newPath := filepath.Join(dstBase, filename)
			command = strings.Replace(command, part, newPath, 1)
		}
	}
	return command
}

// CompileHookArtifacts copies files from hook artifact subdirectories to the
// provider output directory. Returns a map of output path → file content.
func CompileHookArtifacts(hookName string, artifacts []string, srcDir, dstDir string) (map[string]string, error) {
	files := make(map[string]string)
	for _, artifactDir := range artifacts {
		srcPath := filepath.Join(srcDir, artifactDir)
		entries, err := os.ReadDir(srcPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read hook artifact dir %s: %w", srcPath, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(srcPath, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("read hook artifact %s: %w", entry.Name(), err)
			}
			outPath := filepath.Join(dstDir, entry.Name())
			files[outPath] = string(content)
		}
	}
	return files, nil
}
