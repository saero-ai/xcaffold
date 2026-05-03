package renderer

import (
	"testing"
)

func TestCheckFieldSupport_MissingDescription(t *testing.T) {
	fields := map[string]string{
		"name": "test-agent",
	}
	notes := CheckFieldSupport("claude", "agent", "test-agent", fields, false)
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

func TestCheckFieldSupport_AllPresent(t *testing.T) {
	fields := map[string]string{
		"name":        "test-agent",
		"description": "A test agent",
	}
	notes := CheckFieldSupport("claude", "agent", "test-agent", fields, false)
	for _, n := range notes {
		if n.Code == CodeFieldRequiredForTarget {
			t.Errorf("unexpected required-field note for field %q", n.Field)
		}
	}
}

func TestCheckFieldSupport_UnsupportedTarget(t *testing.T) {
	fields := map[string]string{
		"name": "test-agent",
	}
	// Antigravity does not require description on agents (it's unsupported)
	notes := CheckFieldSupport("antigravity", "agent", "test-agent", fields, false)
	for _, n := range notes {
		if n.Code == CodeFieldRequiredForTarget && n.Field == "description" {
			t.Error("antigravity should not require description on agents")
		}
	}
}

func TestCheckFieldSupport_Unsupported_EmitsError(t *testing.T) {
	// "tools" is unsupported for cursor on agent kind
	fields := map[string]string{
		"name":        "my-agent",
		"description": "A test agent",
		"tools":       "set",
	}
	notes := CheckFieldSupport("cursor", "agent", "my-agent", fields, false)
	found := false
	for _, n := range notes {
		if n.Code == CodeFieldUnsupported && n.Field == "tools" {
			found = true
			if n.Level != LevelError {
				t.Errorf("expected LevelError, got %s", n.Level)
			}
			if n.Target != "cursor" {
				t.Errorf("expected target cursor, got %s", n.Target)
			}
		}
	}
	if !found {
		t.Error("expected FIELD_UNSUPPORTED note for tools field on cursor")
	}
}

func TestCheckFieldSupport_Supported_NoNote(t *testing.T) {
	// "tools" is optional for claude on agent kind — no unsupported note
	fields := map[string]string{
		"name":        "my-agent",
		"description": "A test agent",
		"tools":       "set",
	}
	notes := CheckFieldSupport("claude", "agent", "my-agent", fields, false)
	for _, n := range notes {
		if n.Code == CodeFieldUnsupported && n.Field == "tools" {
			t.Error("tools is supported by claude; should not emit FIELD_UNSUPPORTED")
		}
	}
}

func TestCheckFieldSupport_Suppressed_NoNote(t *testing.T) {
	// "tools" is unsupported for cursor, but suppressed=true skips the error
	fields := map[string]string{
		"name":        "my-agent",
		"description": "A test agent",
		"tools":       "set",
	}
	notes := CheckFieldSupport("cursor", "agent", "my-agent", fields, true)
	for _, n := range notes {
		if n.Code == CodeFieldUnsupported && n.Field == "tools" {
			t.Error("suppressed=true should skip FIELD_UNSUPPORTED notes")
		}
	}
}

func TestCheckFieldSupport_Required_Missing_EmitsError(t *testing.T) {
	// "description" is required for claude on agent kind
	fields := map[string]string{
		"name": "my-agent",
	}
	notes := CheckFieldSupport("claude", "agent", "my-agent", fields, false)
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
		t.Error("expected FIELD_REQUIRED_FOR_TARGET note for description on claude")
	}
}

func TestCheckFieldSupport_Suppressed_StillEmitsRequired(t *testing.T) {
	// Even with suppressed=true, required-field errors are NOT suppressed
	fields := map[string]string{
		"name": "my-agent",
	}
	notes := CheckFieldSupport("claude", "agent", "my-agent", fields, true)
	found := false
	for _, n := range notes {
		if n.Code == CodeFieldRequiredForTarget && n.Field == "description" {
			found = true
		}
	}
	if !found {
		t.Error("suppression should NOT suppress required-field errors")
	}
}
