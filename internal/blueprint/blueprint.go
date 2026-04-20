package blueprint

import (
	"fmt"

	"github.com/saero-ai/xcaffold/internal/ast"
)

const maxExtendsDepth = 5

// ResolveBlueprintExtends resolves the extends chain for all blueprints.
// Each blueprint's ref-lists are merged with its parent's resolved ref-lists
// using set-union semantics: parent entries appear first, then child entries
// not already present in the parent. The merge is applied to all seven
// ref-list fields (agents, skills, rules, workflows, mcp, policies, memory).
//
// Modifies the blueprints map in place. Errors on circular extends,
// a missing parent, or a chain length exceeding maxExtendsDepth (5).
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
			p.Agents = unionStrings(parent.Agents, p.Agents)
			p.Skills = unionStrings(parent.Skills, p.Skills)
			p.Rules = unionStrings(parent.Rules, p.Rules)
			p.Workflows = unionStrings(parent.Workflows, p.Workflows)
			p.MCP = unionStrings(parent.MCP, p.MCP)
			p.Policies = unionStrings(parent.Policies, p.Policies)
			p.Memory = unionStrings(parent.Memory, p.Memory)
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
// dependency graph from each selected agent. Only ref-list types that are
// empty in the blueprint are expanded — a non-empty list is treated as an
// explicit override and left unchanged.
//
// Agent → auto-includes referenced Skills, Rules, and MCP when the
// corresponding blueprint list is empty. Agents with no entry in scope
// are silently skipped. Duplicate entries across multiple agents are
// deduplicated (first occurrence wins).
func ResolveTransitiveDeps(p *ast.BlueprintConfig, scope *ast.ResourceScope) {
	if len(p.Agents) == 0 || scope == nil {
		return
	}

	autoSkills := len(p.Skills) == 0
	autoRules := len(p.Rules) == 0
	autoMCP := len(p.MCP) == 0

	if !autoSkills && !autoRules && !autoMCP {
		return
	}

	seen := make(map[string]bool)

	for _, agentName := range p.Agents {
		agent, ok := scope.Agents[agentName]
		if !ok {
			continue
		}
		if autoSkills {
			for _, s := range agent.Skills {
				if !seen["skill:"+s] {
					seen["skill:"+s] = true
					p.Skills = append(p.Skills, s)
				}
			}
		}
		if autoRules {
			for _, r := range agent.Rules {
				if !seen["rule:"+r] {
					seen["rule:"+r] = true
					p.Rules = append(p.Rules, r)
				}
			}
		}
		if autoMCP {
			for _, m := range agent.MCP {
				if !seen["mcp:"+m] {
					seen["mcp:"+m] = true
					p.MCP = append(p.MCP, m)
				}
			}
		}
	}
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
