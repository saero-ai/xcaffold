package renderer

import (
	"testing"
)

func TestCheckRequiredFields_MissingDescription(t *testing.T) {
	fields := map[string]string{
		"name": "test-agent",
	}
	notes := CheckRequiredFields("claude", "agent", "test-agent", fields)
	if len(notes) == 0 {
		t.Fatal("expected fidelity notes for missing required fields")
	}
	found := false
	for _, n := range notes {
		if n.Code == CodeFieldRequiredForTarget && n.Field == "description" {
			found = true
			if n.Level != LevelError {
				t.Errorf("expected LevelError, got %s", n.Level)
			}
		}
	}
	if !found {
		t.Error("expected FIELD_REQUIRED_FOR_TARGET note for description field")
	}
}

func TestCheckRequiredFields_AllPresent(t *testing.T) {
	fields := map[string]string{
		"name":        "test-agent",
		"description": "A test agent",
	}
	notes := CheckRequiredFields("claude", "agent", "test-agent", fields)
	for _, n := range notes {
		if n.Code == CodeFieldRequiredForTarget {
			t.Errorf("unexpected required-field note for field %q", n.Field)
		}
	}
}

func TestCheckRequiredFields_UnsupportedTarget(t *testing.T) {
	fields := map[string]string{
		"name": "test-agent",
	}
	// Antigravity does not require description on agents (it's unsupported)
	notes := CheckRequiredFields("antigravity", "agent", "test-agent", fields)
	for _, n := range notes {
		if n.Code == CodeFieldRequiredForTarget && n.Field == "description" {
			t.Error("antigravity should not require description on agents")
		}
	}
}
