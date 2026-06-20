package ast

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestContextConfig_PathField_YAMLRoundTrip verifies that the Path field
// marshals and unmarshals correctly in YAML.
func TestContextConfig_PathField_YAMLRoundTrip(t *testing.T) {
	original := ContextConfig{
		Name:        "backend-context",
		Description: "Backend context block",
		Path:        "backend",
		Targets:     []string{"cursor", "claude"},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	// Unmarshal back to verify round-trip
	var unmarshaled ContextConfig
	err = yaml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	// Verify Path field matches
	if unmarshaled.Path != "backend" {
		t.Errorf("expected Path=backend, got %q", unmarshaled.Path)
	}
	if unmarshaled.Name != "backend-context" {
		t.Errorf("expected Name=backend-context, got %q", unmarshaled.Name)
	}
}

// TestContextConfig_PathField_OmitEmpty verifies that empty Path is omitted
// from YAML output via the yaml:"path,omitempty" tag.
func TestContextConfig_PathField_OmitEmpty(t *testing.T) {
	original := ContextConfig{
		Name:        "shared-context",
		Description: "Shared context block",
		Path:        "", // empty path should be omitted
	}

	// Marshal to YAML
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	content := string(data)

	// Verify "path" key is absent from YAML output
	if strings.Contains(content, "path:") {
		t.Errorf("expected 'path' key to be omitted when empty, but found it in output:\n%s", content)
	}

	// Verify name and description are still present
	if !strings.Contains(content, "name:") {
		t.Errorf("expected 'name' key in output, but it's missing")
	}
	if !strings.Contains(content, "description:") {
		t.Errorf("expected 'description' key in output, but it's missing")
	}
}
