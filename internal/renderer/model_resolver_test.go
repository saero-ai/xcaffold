package renderer

import "testing"

type stubResolver struct {
	aliases      map[string]string
	supportsBare bool
}

func (s *stubResolver) ResolveAlias(alias string) (string, bool) {
	id, ok := s.aliases[alias]
	return id, ok
}

func (s *stubResolver) SupportsBareAliases() bool { return s.supportsBare }

func TestModelResolver_ResolveAlias(t *testing.T) {
	r := &stubResolver{aliases: map[string]string{"sonnet": "claude-sonnet-4-6"}}
	id, ok := r.ResolveAlias("sonnet")
	if !ok || id != "claude-sonnet-4-6" {
		t.Errorf("ResolveAlias(sonnet) = %q, %v; want claude-sonnet-4-6, true", id, ok)
	}
	_, ok = r.ResolveAlias("nonexistent")
	if ok {
		t.Error("ResolveAlias(nonexistent) should return false")
	}
}

func TestModelResolver_SupportsBareAliases(t *testing.T) {
	r := &stubResolver{supportsBare: true}
	if !r.SupportsBareAliases() {
		t.Error("SupportsBareAliases() = false, want true")
	}
	r2 := &stubResolver{supportsBare: false}
	if r2.SupportsBareAliases() {
		t.Error("SupportsBareAliases() = true, want false")
	}
}
