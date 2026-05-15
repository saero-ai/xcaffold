package blueprint

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/saero-ai/xcaffold/internal/ast"
)

const maxExtendsDepth = 10

// mergeClearableList merges parent and child ref-lists during blueprint inheritance.
// If the child has Cleared=true, the parent's values are discarded (child overrides).
// If the child has no values and is not cleared, the parent's values are inherited.
// Otherwise, the union of parent and child values is returned.
func mergeClearableList(parent, child ast.ClearableList) ast.ClearableList {
	if child.Cleared {
		return child
	}
	if len(child.Values) == 0 {
		return parent
	}
	return ast.ClearableList{Values: unionStrings(parent.Values, child.Values)}
}

// ResolveBlueprintExtends resolves the extends chain for all blueprints.
// Each blueprint's ref-lists are merged with its parent's resolved ref-lists
// using set-union semantics: parent entries appear first, then child entries
// not already present in the parent. The merge is applied to all eight
// ref-list fields (agents, skills, rules, workflows, mcp, policies, memory, contexts).
//
// Modifies the blueprints map in place. Errors on circular extends,
// a missing parent, or a chain length exceeding maxExtendsDepth (10).
func ResolveBlueprintExtends(blueprints map[string]ast.BlueprintConfig) error {
	// Pass 1: validate every chain for cycles, missing parents, and max depth.
	// This is done independently of resolution order so that chains of any
	// length trigger the depth error regardless of map iteration order.
	for name := range blueprints {
		if err := validateChain(blueprints, name); err != nil {
			return err
		}
	}

	// Pass 2: resolve in dependency order using memoized DFS.
	resolved := make(map[string]bool)

	var resolve func(name string) error
	resolve = func(name string) error {
		if resolved[name] {
			return nil
		}

		p := blueprints[name]

		if p.Extends != "" {
			if err := resolve(p.Extends); err != nil {
				return err
			}
			parent := blueprints[p.Extends]
			p.Agents = mergeClearableList(parent.Agents, p.Agents)
			p.Skills = mergeClearableList(parent.Skills, p.Skills)
			p.Rules = mergeClearableList(parent.Rules, p.Rules)
			p.Workflows = mergeClearableList(parent.Workflows, p.Workflows)
			p.MCP = mergeClearableList(parent.MCP, p.MCP)
			p.Policies = mergeClearableList(parent.Policies, p.Policies)
			p.Memory = mergeClearableList(parent.Memory, p.Memory)
			p.Contexts = mergeClearableList(parent.Contexts, p.Contexts)
			blueprints[name] = p
		}

		resolved[name] = true
		return nil
	}

	for name := range blueprints {
		if err := resolve(name); err != nil {
			return err
		}
	}
	return nil
}

// validateChain walks the extends chain rooted at name and enforces:
//   - no circular reference
//   - no missing parent
//   - chain length <= maxExtendsDepth
//
// It does not use memoization so that each chain is validated independently
// of map iteration order.
func validateChain(blueprints map[string]ast.BlueprintConfig, start string) error {
	visiting := make(map[string]bool)
	current := start
	depth := 0

	for {
		if depth >= maxExtendsDepth {
			return fmt.Errorf("blueprint extends chain exceeds maximum depth of %d at %q", maxExtendsDepth, current)
		}

		p, ok := blueprints[current]
		if !ok {
			// Only an error when we've followed an extends link; the initial
			// lookup is guaranteed by the caller iterating blueprints.
			return fmt.Errorf("blueprint %q not found", current)
		}

		if p.Extends == "" {
			return nil // reached the root of the chain
		}

		if visiting[current] {
			return fmt.Errorf("circular blueprint extends detected involving %q", current)
		}
		visiting[current] = true

		if _, exists := blueprints[p.Extends]; !exists {
			return fmt.Errorf("blueprint %q extends %q which does not exist", current, p.Extends)
		}

		current = p.Extends
		depth++
	}
}

// ResolveTransitiveDeps expands a blueprint's ref-lists by walking the
// dependency graph from each selected agent. Agent-referenced Skills, Rules,
// and MCP are always collected and merged with the blueprint's explicit lists.
//
// If a resource appears in BOTH the blueprint's explicit list AND an agent's
// dependency list, an error is returned telling the user to remove the
// duplicate from the blueprint — it is already included via the agent.
//
// Agents with no entry in scope are silently skipped. Duplicate entries
// across multiple agents are deduplicated (first occurrence wins).
func ResolveTransitiveDeps(p *ast.BlueprintConfig, scope *ast.ResourceScope) error {
	if len(p.Agents.Values) == 0 || scope == nil {
		return nil
	}

	// Build sets of the blueprint's explicit entries for overlap detection.
	explicitSkills := make(map[string]bool, len(p.Skills.Values))
	for _, s := range p.Skills.Values {
		explicitSkills[s] = true
	}
	explicitRules := make(map[string]bool, len(p.Rules.Values))
	for _, r := range p.Rules.Values {
		explicitRules[r] = true
	}
	explicitMCP := make(map[string]bool, len(p.MCP.Values))
	for _, m := range p.MCP.Values {
		explicitMCP[m] = true
	}

	// Seen sets track de-duplication across multiple agents (transitive pass only).
	seenSkill := make(map[string]bool)
	seenRule := make(map[string]bool)
	seenMCP := make(map[string]bool)

	for _, agentName := range p.Agents.Values {
		agent, ok := scope.Agents[agentName]
		if !ok {
			continue
		}
		for _, s := range agent.Skills.Values {
			if explicitSkills[s] {
				return fmt.Errorf("blueprint %q declares skill %q which is already included via agent %q; remove it from the blueprint", p.Name, s, agentName)
			}
			if !seenSkill[s] {
				seenSkill[s] = true
				p.Skills.Values = append(p.Skills.Values, s)
			}
		}
		for _, r := range agent.Rules.Values {
			if explicitRules[r] {
				return fmt.Errorf("blueprint %q declares rule %q which is already included via agent %q; remove it from the blueprint", p.Name, r, agentName)
			}
			if !seenRule[r] {
				seenRule[r] = true
				p.Rules.Values = append(p.Rules.Values, r)
			}
		}
		for _, m := range agent.MCP.Values {
			if explicitMCP[m] {
				return fmt.Errorf("blueprint %q declares mcp %q which is already included via agent %q; remove it from the blueprint", p.Name, m, agentName)
			}
			if !seenMCP[m] {
				seenMCP[m] = true
				p.MCP.Values = append(p.MCP.Values, m)
			}
		}
	}
	return nil
}

// ValidateBlueprintRefs checks that every resource name in every blueprint's
// ref-lists exists in the merged ResourceScope. Returns all errors (not just
// the first) so the user can fix everything in one pass.
func ValidateBlueprintRefs(blueprints map[string]ast.BlueprintConfig, scope *ast.ResourceScope) []error {
	var errs []error
	for bpName, p := range blueprints {
		checkRefs := func(resType string, names []string, catalog map[string]struct{}) {
			for _, name := range names {
				if _, ok := catalog[name]; !ok {
					errs = append(errs, fmt.Errorf("blueprint %q references %s %q which does not exist", bpName, resType, name))
				}
			}
		}
		var scopeAgents, scopeSkills, scopeRules, scopeWorkflows, scopeMCP, scopePolicies, scopeMemory, scopeContexts map[string]struct{}
		if scope != nil {
			scopeAgents = keysSet(scope.Agents)
			scopeSkills = keysSet(scope.Skills)
			scopeRules = keysSet(scope.Rules)
			scopeWorkflows = keysSet(scope.Workflows)
			scopeMCP = keysSet(scope.MCP)
			scopePolicies = keysSet(scope.Policies)
			scopeMemory = keysSet(scope.Memory)
			scopeContexts = keysSet(scope.Contexts)
		}
		checkRefs("agent", p.Agents.Values, scopeAgents)
		checkRefs("skill", p.Skills.Values, scopeSkills)
		checkRefs("rule", p.Rules.Values, scopeRules)
		checkRefs("workflow", p.Workflows.Values, scopeWorkflows)
		checkRefs("mcp", p.MCP.Values, scopeMCP)
		checkRefs("policy", p.Policies.Values, scopePolicies)
		checkRefs("memory", p.Memory.Values, scopeMemory)
		checkRefs("context", p.Contexts.Values, scopeContexts)
	}
	return errs
}

// keysSet returns a set of the keys in m as a map[string]struct{}.
// A nil map returns an empty set (not a panic).
func keysSet[V any](m map[string]V) map[string]struct{} {
	s := make(map[string]struct{}, len(m))
	for k := range m {
		s[k] = struct{}{}
	}
	return s
}

// ApplyBlueprint filters the config's ResourceScope to only include resources
// named in the given blueprint. Returns a shallow copy with filtered maps.
// The input config is not modified.
//
// If blueprintName is empty, returns config unmodified (no filtering).
// If blueprintName is unknown, returns an error listing available blueprints.
func ApplyBlueprint(config *ast.XcaffoldConfig, blueprintName string) (*ast.XcaffoldConfig, error) {
	if blueprintName == "" {
		return config, nil
	}

	p, ok := config.Blueprints[blueprintName]
	if !ok {
		available := sortedKeys(config.Blueprints)
		return nil, fmt.Errorf("blueprint %q not found; available: %v", blueprintName, available)
	}

	filtered := *config // shallow copy preserves Hooks, Settings, Blueprints, etc.
	filtered.ResourceScope = ast.ResourceScope{
		Agents:    filterMap(config.Agents, p.Agents.Values),
		Skills:    filterMap(config.Skills, p.Skills.Values),
		Rules:     filterMap(config.Rules, p.Rules.Values),
		Workflows: filterMap(config.Workflows, p.Workflows.Values),
		MCP:       filterMap(config.MCP, p.MCP.Values),
		Policies:  filterMapClearable(config.Policies, p.Policies),
		Memory:    filterMap(config.Memory, p.Memory.Values),
		Contexts:  filterMap(config.Contexts, p.Contexts.Values),
	}

	// Named settings selection: if the blueprint specifies a settings key,
	// filter to only that entry. An empty Settings field means "keep all".
	if p.Settings != "" {
		s, ok := config.Settings[p.Settings]
		if !ok {
			return nil, fmt.Errorf("blueprint %q references settings %q which does not exist", blueprintName, p.Settings)
		}
		filtered.Settings = map[string]ast.SettingsConfig{p.Settings: s}
	}

	// Named hooks selection: if the blueprint specifies a hooks key,
	// filter to only that entry. An empty Hooks field means "keep all".
	if p.Hooks != "" {
		h, ok := config.Hooks[p.Hooks]
		if !ok {
			return nil, fmt.Errorf("blueprint %q references hooks %q which does not exist", blueprintName, p.Hooks)
		}
		filtered.Hooks = map[string]ast.NamedHookConfig{p.Hooks: h}
	}

	return &filtered, nil
}

// filterMapClearable is like filterMap but treats an omitted ClearableList
// (Cleared=false, Values=nil) as "keep all" rather than "keep none".
// This is used for policies where omission means "apply all policies".
func filterMapClearable[V any](source map[string]V, selector ast.ClearableList) map[string]V {
	if selector.Cleared {
		return nil
	}
	if len(selector.Values) == 0 {
		return source
	}
	return filterMap(source, selector.Values)
}

// filterMap returns a new map containing only entries whose key appears in allowed.
// If allowed is empty, returns nil (blueprint selects zero of this type).
func filterMap[V any](source map[string]V, allowed []string) map[string]V {
	if len(allowed) == 0 {
		return nil
	}
	result := make(map[string]V, len(allowed))
	for _, name := range allowed {
		if v, ok := source[name]; ok {
			result[name] = v
		}
	}
	return result
}

// sortedKeys returns the sorted keys of a map.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// BlueprintHash computes a SHA-256 hash of a blueprint's resolved resource ref-lists.
// The hash is order-independent (sorts each list before hashing).
// Returns a "sha256:<hex>" prefixed string.
func BlueprintHash(p ast.BlueprintConfig) string {
	h := sha256.New()
	for _, entry := range []struct {
		label string
		refs  []string
	}{
		{"agents", p.Agents.Values},
		{"skills", p.Skills.Values},
		{"rules", p.Rules.Values},
		{"workflows", p.Workflows.Values},
		{"mcp", p.MCP.Values},
		{"policies", p.Policies.Values},
		{"memory", p.Memory.Values},
		{"contexts", p.Contexts.Values},
	} {
		fmt.Fprintf(h, "%s:", entry.label)
		sorted := make([]string, len(entry.refs))
		copy(sorted, entry.refs)
		sort.Strings(sorted)
		for _, name := range sorted {
			fmt.Fprintf(h, "%s,", name)
		}
		fmt.Fprintln(h)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

// unionStrings returns the set union of a and b, preserving order.
// Elements from a appear first, followed by elements from b not already in a.
func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
