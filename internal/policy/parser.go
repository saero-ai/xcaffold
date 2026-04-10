package policy

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseFile reads and strictly parses a policy .xcf file.
// Uses KnownFields(true) to fail closed on unknown properties.
func ParseFile(path string) (*PolicyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("policy: read err: %w", err)
	}

	cfg := &PolicyConfig{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true) // Fail-closed on unknown schema fields

	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("policy: parse error in %q: %w", path, err)
	}

	if cfg.Kind != "policy" {
		return nil, fmt.Errorf("policy: %q has kind %q, expected \"policy\"", path, cfg.Kind)
	}

	return cfg, nil
}
