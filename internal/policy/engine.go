package policy

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"github.com/saero-ai/xcaffold/internal/output"
)

//go:embed builtin/*.xcf
var builtinFS embed.FS

// Engine loads and evaluates policies against a parsed config and compiled output.
type Engine struct {
	// policies is the effective policy set after override resolution.
	// LoadDir() called after LoadBuiltin() overrides by name.
	policies map[string]PolicyConfig
}

// NewEngine returns an empty policy engine.
func NewEngine() *Engine {
	return &Engine{policies: make(map[string]PolicyConfig)}
}

// PolicyCount returns the number of currently loaded policies.
func (e *Engine) PolicyCount() int {
	return len(e.policies)
}

// LoadBuiltin loads the four //go:embed built-in policies.
func (e *Engine) LoadBuiltin() error {
	return fs.WalkDir(builtinFS, "builtin", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".xcf") {
			return nil
		}
		data, err := builtinFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("policy: builtin read %q: %w", path, err)
		}
		cfg, err := parseBytes(data, path)
		if err != nil {
			return err
		}
		e.policies[cfg.Name] = *cfg
		return nil
	})
}

// LoadDir loads all kind: policy files from dir. Policies with the same name
// as an already-loaded policy override it (including severity: off to disable).
func (e *Engine) LoadDir(dir string) error {
	paths, err := ScanDir(dir)
	if err != nil {
		return fmt.Errorf("policy: scan %q: %w", dir, err)
	}
	for _, p := range paths {
		cfg, err := ParseFile(p)
		if err != nil {
			return err
		}
		e.policies[cfg.Name] = *cfg // override by name
	}
	return nil
}

// Evaluate runs all active (non-off) policies against the config snapshot and
// compiled output. Returns all violations found.
func (e *Engine) Evaluate(config *ast.XcaffoldConfig, out *output.Output) []Violation {
	var viols []Violation

	for _, pol := range e.policies {
		if pol.Severity == "off" {
			continue
		}
		sev := resolveSeverity(pol.Severity)

		switch pol.Target {
		case "agent":
			viols = append(viols, e.evalAgentPolicy(pol, sev, config)...)
		case "skill":
			viols = append(viols, e.evalSkillPolicy(pol, sev, config)...)
		case "output":
			viols = append(viols, e.evalOutputPolicy(pol, sev, out.Files)...)
		}
	}
	return viols
}

// EvaluateAgents runs agent-target policies against a simplified agent field map.
// Used by tests to verify override behaviour without a full ast.XcaffoldConfig.
func (e *Engine) EvaluateAgents(agents map[string]map[string]string) []Violation {
	var viols []Violation
	for _, pol := range e.policies {
		if pol.Severity == "off" || pol.Target != "agent" {
			continue
		}
		sev := resolveSeverity(pol.Severity)
		for agentID, fields := range agents {
			props := make(map[string]any, len(fields))
			for k, v := range fields {
				props[k] = v
			}
			if !MatchAgent(pol.Match, props) || !MatchName(pol.Match, agentID) {
				continue
			}
			for _, req := range pol.Require {
				vs := EvalRequire("agent", agentID, req, fields)
				for i := range vs {
					vs[i].Policy = pol.Name
					vs[i].Severity = sev
				}
				viols = append(viols, vs...)
			}
		}
	}
	return viols
}

func (e *Engine) evalAgentPolicy(pol PolicyConfig, sev Severity, config *ast.XcaffoldConfig) []Violation {
	var viols []Violation
	for agentID, agent := range config.Agents {
		props := map[string]any{
			"description": agent.Description,
			"model":       agent.Model,
			"tools":       agent.Tools,
			"memory":      agent.Memory,
		}
		if !MatchAgent(pol.Match, props) || !MatchName(pol.Match, agentID) {
			continue
		}
		fields := map[string]string{
			"description": agent.Description,
			"model":       agent.Model,
			"memory":      agent.Memory,
			"tools_count": fmt.Sprintf("%d", len(agent.Tools)),
		}
		for _, req := range pol.Require {
			vs := EvalRequire("agent", agentID, req, fields)
			for i := range vs {
				vs[i].Policy = pol.Name
				vs[i].Severity = sev
			}
			viols = append(viols, vs...)
		}
		for _, deny := range pol.Deny {
			vs := EvalDeny(pol.Name, sev, deny, map[string]string{agentID: agent.Instructions})
			viols = append(viols, vs...)
		}
	}
	return viols
}

func (e *Engine) evalSkillPolicy(pol PolicyConfig, sev Severity, config *ast.XcaffoldConfig) []Violation {
	var viols []Violation
	for skillID, skill := range config.Skills {
		fields := map[string]string{
			"instructions": skill.Instructions,
			"description":  skill.Description,
		}
		for _, req := range pol.Require {
			vs := EvalRequire("skill", skillID, req, fields)
			for i := range vs {
				vs[i].Policy = pol.Name
				vs[i].Severity = sev
			}
			viols = append(viols, vs...)
		}
	}
	return viols
}

func (e *Engine) evalOutputPolicy(pol PolicyConfig, sev Severity, files map[string]string) []Violation {
	var viols []Violation
	for _, deny := range pol.Deny {
		viols = append(viols, EvalDeny(pol.Name, sev, deny, files)...)
	}
	return viols
}

func resolveSeverity(s string) Severity {
	if s == "error" {
		return SeverityError
	}
	return SeverityWarning
}
