package renderer

import "testing"

type stubResolver struct {
	aliases map[string]string
}

func (s *stubResolver) ResolveAlias(alias string) (string, bool) {
	id, ok := s.aliases[alias]
	return id, ok
}

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
