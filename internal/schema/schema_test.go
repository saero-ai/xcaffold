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

func TestHasRole_ExistingField_WithRole(t *testing.T) {
	// skills is a composition field with role
	got := HasRole("agent", "skills")
	if !got {
		t.Errorf("HasRole(%q, %q) = %v; want true",
			"agent", "skills", got)
	}
}

func TestHasRole_ExistingField_WithRole_Tools(t *testing.T) {
	// tools is a rendering field with role
	got := HasRole("agent", "tools")
	if !got {
		t.Errorf("HasRole(%q, %q) = %v; want true",
			"agent", "tools", got)
	}
}

func TestHasRole_ExistingField_WithRole_AllowedTools(t *testing.T) {
	// allowed-tools on skill is a rendering field with role
	got := HasRole("skill", "allowed-tools")
	if !got {
		t.Errorf("HasRole(%q, %q) = %v; want true",
			"skill", "allowed-tools", got)
	}
}

func TestHasRole_ExistingField_WithRole_RulePaths(t *testing.T) {
	// paths on rule is a rendering field with role
	got := HasRole("rule", "paths")
	if !got {
		t.Errorf("HasRole(%q, %q) = %v; want true",
			"rule", "paths", got)
	}
}

func TestHasRole_NonexistentField(t *testing.T) {
	// nonexistent-field should not have role
	got := HasRole("agent", "nonexistent-field")
	if got {
		t.Errorf("HasRole(%q, %q) = %v; want false",
			"agent", "nonexistent-field", got)
	}
}

func TestHasRole_NonexistentKind(t *testing.T) {
	// nonexistent-kind should not exist
	got := HasRole("nonexistent-kind", "name")
	if got {
		t.Errorf("HasRole(%q, %q) = %v; want false",
			"nonexistent-kind", "name", got)
	}
}

func TestRegistryGen_NoXcaffoldOnly(t *testing.T) {
	for kindName, ks := range Registry {
		for _, f := range ks.Fields {
			for prov, support := range f.Provider {
				if support == "xcaffold-only" {
					t.Errorf("kind=%s field=%s provider=%s still has xcaffold-only", kindName, f.YAMLKey, prov)
				}
			}
		}
	}
}
