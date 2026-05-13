package renderer

import (
	"testing"
)

func TestCheckFieldSupport_MissingDescription(t *testing.T) {
	fields := map[string]string{
		"name": "test-agent",
	}
	notes := CheckFieldSupport(FieldCheckInput{Target: "claude", Kind: "agent", ResourceName: "test-agent", PresentFields: fields, Suppressed: false})
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
	notes := CheckFieldSupport(FieldCheckInput{Target: "claude", Kind: "agent", ResourceName: "test-agent", PresentFields: fields, Suppressed: false})
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
	notes := CheckFieldSupport(FieldCheckInput{Target: "antigravity", Kind: "agent", ResourceName: "test-agent", PresentFields: fields, Suppressed: false})
	for _, n := range notes {
		if n.Code == CodeFieldRequiredForTarget && n.Field == "description" {
			t.Error("antigravity should not require description on agents")
		}
	}
}

func TestCheckFieldSupport_Unsupported_WithRole_NoError(t *testing.T) {
	// "tools" is unsupported for cursor on agent kind, but has Role:["rendering"].
	// Two-layer fidelity check: unsupported + has xcaf role = silently skipped.
	// The field is an intentional xcaffold extension compiled by the renderer,
	// not a user authoring mistake, so no error is warranted.
	fields := map[string]string{
		"name":        "my-agent",
		"description": "A test agent",
		"tools":       "set",
	}
	notes := CheckFieldSupport(FieldCheckInput{Target: "cursor", Kind: "agent", ResourceName: "my-agent", PresentFields: fields, Suppressed: false})
	for _, n := range notes {
		if n.Code == CodeFieldUnsupported && n.Field == "tools" {
			t.Errorf("tools has an xcaf role; should be silently skipped for cursor, got: %s", n.Reason)
		}
	}
}

func TestCheckFieldSupport_Supported_NoNote(t *testing.T) {
	// "tools" is optional for claude on agent kind — no unsupported note
	fields := map[string]string{
		"name":        "my-agent",
		"description": "A test agent",
		"tools":       "set",
	}
	notes := CheckFieldSupport(FieldCheckInput{Target: "claude", Kind: "agent", ResourceName: "my-agent", PresentFields: fields, Suppressed: false})
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
	notes := CheckFieldSupport(FieldCheckInput{Target: "cursor", Kind: "agent", ResourceName: "my-agent", PresentFields: fields, Suppressed: true})
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
	notes := CheckFieldSupport(FieldCheckInput{Target: "claude", Kind: "agent", ResourceName: "my-agent", PresentFields: fields, Suppressed: false})
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
	notes := CheckFieldSupport(FieldCheckInput{Target: "claude", Kind: "agent", ResourceName: "my-agent", PresentFields: fields, Suppressed: true})
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

func TestCheckFieldSupport_Unsupported_WithRole_Silent(t *testing.T) {
	// "skills" is unsupported by gemini but has +xcaf:role=composition,rendering.
	// Fields with an xcaf role should be silently skipped rather than emitting an
	// error — they are intentional xcaffold extensions, not user mistakes.
	fields := map[string]string{
		"name":        "my-agent",
		"description": "Agent with skills",
		"skills":      "set",
	}
	notes := CheckFieldSupport(FieldCheckInput{Target: "gemini", Kind: "agent", ResourceName: "my-agent", PresentFields: fields, Suppressed: false})
	for _, n := range notes {
		if n.Field == "skills" && n.Code == CodeFieldUnsupported {
			t.Errorf("skills field should be silently skipped for gemini (has xcaf role), got error: %s", n.Reason)
		}
	}
}

func TestCheckFieldSupport_TwoLayer_SkillsOnGemini(t *testing.T) {
	// Two-layer fidelity integration test: an agent with `skills:` targeting gemini
	// should not produce a fidelity error (the original bug fix).
	// "skills" is unsupported by gemini but has xcaf role, so it should be silently skipped.
	fields := map[string]string{
		"name":        "my-agent",
		"description": "Agent with skills",
		"skills":      "set",
	}
	notes := CheckFieldSupport(FieldCheckInput{Target: "gemini", Kind: "agent", ResourceName: "my-agent", PresentFields: fields, Suppressed: false})
	for _, n := range notes {
		if n.Field == "skills" && n.Code == CodeFieldUnsupported {
			t.Errorf("skills field should be silently skipped for gemini (has xcaf role), got error: %s", n.Reason)
		}
	}
}

func TestCheckFieldSupport_TwoLayer_CleanAgentZeroNotes(t *testing.T) {
	// Two-layer fidelity integration test: a clean agent on gemini with all required
	// fields should produce zero notes.
	fields := map[string]string{
		"name":        "clean-agent",
		"description": "Fully specified",
	}
	notes := CheckFieldSupport(FieldCheckInput{Target: "gemini", Kind: "agent", ResourceName: "clean-agent", PresentFields: fields, Suppressed: false})
	if len(notes) != 0 {
		t.Errorf("expected zero notes for clean agent on gemini, got %d: %v", len(notes), notes)
	}
}
