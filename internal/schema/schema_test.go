package schema

import "testing"

func TestFieldSupportForTarget_Found(t *testing.T) {
	got := FieldSupportForTarget("agent", "model", "claude")
	if got != "optional" {
		t.Errorf("FieldSupportForTarget(%q, %q, %q) = %q; want %q",
			"agent", "model", "claude", got, "optional")
	}
}

func TestFieldSupportForTarget_NotFound_UnknownTarget(t *testing.T) {
	got := FieldSupportForTarget("agent", "model", "nonexistent")
	if got != "" {
		t.Errorf("FieldSupportForTarget(%q, %q, %q) = %q; want empty string",
			"agent", "model", "nonexistent", got)
	}
}

func TestFieldSupportForTarget_NotFound_UnknownKind(t *testing.T) {
	got := FieldSupportForTarget("nonexistent-kind", "model", "claude")
	if got != "" {
		t.Errorf("FieldSupportForTarget(%q, %q, %q) = %q; want empty string",
			"nonexistent-kind", "model", "claude", got)
	}
}

func TestFieldSupportForTarget_NotFound_UnknownField(t *testing.T) {
	got := FieldSupportForTarget("agent", "no-such-field", "claude")
	if got != "" {
		t.Errorf("FieldSupportForTarget(%q, %q, %q) = %q; want empty string",
			"agent", "no-such-field", "claude", got)
	}
}

func TestFieldSupportForTarget_Unsupported(t *testing.T) {
	// effort is unsupported on gemini
	got := FieldSupportForTarget("agent", "effort", "gemini")
	if got != "unsupported" {
		t.Errorf("FieldSupportForTarget(%q, %q, %q) = %q; want %q",
			"agent", "effort", "gemini", got, "unsupported")
	}
}
