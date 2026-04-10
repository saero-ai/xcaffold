package policy

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ScanDir walks dir and returns paths to all .xcf files with kind: policy.
// This is the inverse of the config parser's isConfigFile() — it specifically
// looks for kind: policy files and ignores everything else.
func ScanDir(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(d.Name(), ".xcf") {
			return nil
		}
		if isPolicyFile(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// isPolicyFile reads the kind: field from an .xcf file and returns true if
// it is exactly "policy". Files with read/parse errors are silently skipped.
func isPolicyFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var header struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &header); err != nil {
		return false
	}
	return header.Kind == "policy"
}
