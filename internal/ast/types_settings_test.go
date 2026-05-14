package ast

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestActivation_Unmarshal_Always tests unmarshaling a scalar "always".
func TestActivation_Unmarshal_Always(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`"always"`), &a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Mode != "always" {
		t.Errorf("expected Mode=always, got %q", a.Mode)
	}
	if a.Paths != nil {
		t.Errorf("expected Paths=nil, got %v", a.Paths)
	}
}

// TestActivation_Unmarshal_Paths tests unmarshaling a sequence of path globs.
func TestActivation_Unmarshal_Paths(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`["*.go","*.ts"]`), &a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Mode != "paths" {
		t.Errorf("expected Mode=paths, got %q", a.Mode)
	}
	if len(a.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(a.Paths))
	}
	if a.Paths[0] != "*.go" || a.Paths[1] != "*.ts" {
		t.Errorf("unexpected path values: %v", a.Paths)
	}
}

// TestActivation_Unmarshal_InvalidScalar tests that non-"always" scalars reject.
func TestActivation_Unmarshal_InvalidScalar(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`"manual"`), &a)
	if err == nil {
		t.Fatalf("expected error for invalid scalar, got nil")
	}
	if msg := err.Error(); msg == "" {
		t.Errorf("error message is empty")
	}
}

// TestActivation_Unmarshal_MapReject tests that map types are rejected.
func TestActivation_Unmarshal_MapReject(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`{mode: always}`), &a)
	if err == nil {
		t.Fatalf("expected error for map type, got nil")
	}
}

// TestActivation_Marshal_RoundTrip_Always tests round-trip marshaling "always".
func TestActivation_Marshal_RoundTrip_Always(t *testing.T) {
	a := Activation{Mode: "always"}
	data, err := yaml.Marshal(a)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	// Verify the output contains "always"
	if dataStr := string(data); dataStr == "" {
		t.Errorf("empty marshal output")
	}

	// Unmarshal to verify round-trip
	var a2 Activation
	err = yaml.Unmarshal(data, &a2)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if a2.Mode != "always" || a2.Paths != nil {
		t.Errorf("round-trip failed: expected always mode, got Mode=%q Paths=%v", a2.Mode, a2.Paths)
	}
}

// TestActivation_Marshal_RoundTrip_Paths tests round-trip marshaling paths.
func TestActivation_Marshal_RoundTrip_Paths(t *testing.T) {
	a := Activation{Mode: "paths", Paths: []string{"*.go", "internal/**"}}
	data, err := yaml.Marshal(a)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	// Unmarshal to verify round-trip
	var a2 Activation
	err = yaml.Unmarshal(data, &a2)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if a2.Mode != "paths" {
		t.Errorf("expected Mode=paths, got %q", a2.Mode)
	}
	if len(a2.Paths) != 2 || a2.Paths[0] != "*.go" || a2.Paths[1] != "internal/**" {
		t.Errorf("paths mismatch: expected [*.go internal/**], got %v", a2.Paths)
	}
}

// TestActivation_Unmarshal_EmptyList tests unmarshaling an empty sequence.
func TestActivation_Unmarshal_EmptyList(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`[]`), &a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Mode != "paths" {
		t.Errorf("expected Mode=paths for empty list, got %q", a.Mode)
	}
	if a.Paths == nil {
		t.Errorf("expected Paths to be []string (empty but not nil), got nil")
	}
	if len(a.Paths) != 0 {
		t.Errorf("expected empty Paths, got %v", a.Paths)
	}
}

// TestActivation_Unmarshal_SinglePath tests unmarshaling a single-element list.
func TestActivation_Unmarshal_SinglePath(t *testing.T) {
	var a Activation
	err := yaml.Unmarshal([]byte(`["*.go"]`), &a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Mode != "paths" {
		t.Errorf("expected Mode=paths, got %q", a.Mode)
	}
	if len(a.Paths) != 1 || a.Paths[0] != "*.go" {
		t.Errorf("expected [*.go], got %v", a.Paths)
	}
}
