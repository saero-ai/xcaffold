package main

import (
	"bytes"
	"sort"
	"strings"

	"github.com/saero-ai/xcaffold/internal/ast"
	"gopkg.in/yaml.v3"
)

// configDocument is a minimal struct for serializing the kind: config document.
// It deliberately excludes resource maps (agents, skills, rules, workflows, mcp)
// so those are emitted as separate kind documents instead of nested under config.
type configDocument struct {
	Kind     string             `yaml:"kind"`
	Version  string             `yaml:"version"`
	Extends  string             `yaml:"extends,omitempty"`
	Project  *ast.ProjectConfig `yaml:"project,omitempty"`
	Settings ast.SettingsConfig `yaml:"settings,omitempty"`
	Hooks    ast.HookConfig     `yaml:"hooks,omitempty"`
}

// agentDoc is the serialization envelope for a kind: agent document.
type agentDoc struct {
	Kind            string `yaml:"kind"`
	Version         string `yaml:"version"`
	ast.AgentConfig `yaml:",inline"`
}

// skillDoc is the serialization envelope for a kind: skill document.
type skillDoc struct {
	Kind            string `yaml:"kind"`
	Version         string `yaml:"version"`
	ast.SkillConfig `yaml:",inline"`
}

// ruleDoc is the serialization envelope for a kind: rule document.
type ruleDoc struct {
	Kind           string `yaml:"kind"`
	Version        string `yaml:"version"`
	ast.RuleConfig `yaml:",inline"`
}

// workflowDoc is the serialization envelope for a kind: workflow document.
type workflowDoc struct {
	Kind               string `yaml:"kind"`
	Version            string `yaml:"version"`
	ast.WorkflowConfig `yaml:",inline"`
}

// mcpDoc is the serialization envelope for a kind: mcp document.
type mcpDoc struct {
	Kind          string `yaml:"kind"`
	Version       string `yaml:"version"`
	ast.MCPConfig `yaml:",inline"`
}

// MarshalMultiKind serializes an XcaffoldConfig as multi-kind YAML documents
// separated by "---". The first document is always kind: config, followed by
// individual kind: agent, kind: skill, kind: rule, kind: workflow, and kind: mcp
// documents in alphabetical key order (deterministic output).
//
// If header is non-empty it is prepended to the output as a comment block.
func MarshalMultiKind(config *ast.XcaffoldConfig, header string) ([]byte, error) {
	version := config.Version
	if version == "" {
		version = "1.0"
	}

	var docs [][]byte

	// ── kind: config document ────────────────────────────────────────────────
	cfgDoc := configDocument{
		Kind:     "config",
		Version:  version,
		Extends:  config.Extends,
		Project:  config.Project,
		Settings: config.Settings,
		Hooks:    config.Hooks,
	}
	b, err := yaml.Marshal(cfgDoc)
	if err != nil {
		return nil, err
	}
	docs = append(docs, bytes.TrimRight(b, "\n"))

	// ── kind: agent documents (sorted) ───────────────────────────────────────
	if len(config.Agents) > 0 {
		keys := make([]string, 0, len(config.Agents))
		for k := range config.Agents {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			agent := config.Agents[k]
			if agent.Name == "" {
				agent.Name = k
			}
			doc := agentDoc{
				Kind:        "agent",
				Version:     version,
				AgentConfig: agent,
			}
			b, err := yaml.Marshal(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── kind: skill documents (sorted) ───────────────────────────────────────
	if len(config.Skills) > 0 {
		keys := make([]string, 0, len(config.Skills))
		for k := range config.Skills {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			skill := config.Skills[k]
			if skill.Name == "" {
				skill.Name = k
			}
			doc := skillDoc{
				Kind:        "skill",
				Version:     version,
				SkillConfig: skill,
			}
			b, err := yaml.Marshal(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── kind: rule documents (sorted) ────────────────────────────────────────
	if len(config.Rules) > 0 {
		keys := make([]string, 0, len(config.Rules))
		for k := range config.Rules {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rule := config.Rules[k]
			if rule.Name == "" {
				rule.Name = k
			}
			doc := ruleDoc{
				Kind:       "rule",
				Version:    version,
				RuleConfig: rule,
			}
			b, err := yaml.Marshal(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── kind: workflow documents (sorted) ─────────────────────────────────────
	if len(config.Workflows) > 0 {
		keys := make([]string, 0, len(config.Workflows))
		for k := range config.Workflows {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			wf := config.Workflows[k]
			if wf.Name == "" {
				wf.Name = k
			}
			doc := workflowDoc{
				Kind:           "workflow",
				Version:        version,
				WorkflowConfig: wf,
			}
			b, err := yaml.Marshal(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── kind: mcp documents (sorted) ─────────────────────────────────────────
	if len(config.MCP) > 0 {
		keys := make([]string, 0, len(config.MCP))
		for k := range config.MCP {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			mcp := config.MCP[k]
			if mcp.Name == "" {
				mcp.Name = k
			}
			doc := mcpDoc{
				Kind:      "mcp",
				Version:   version,
				MCPConfig: mcp,
			}
			b, err := yaml.Marshal(doc)
			if err != nil {
				return nil, err
			}
			docs = append(docs, bytes.TrimRight(b, "\n"))
		}
	}

	// ── Assemble output ──────────────────────────────────────────────────────
	strs := make([]string, len(docs))
	for i, d := range docs {
		strs[i] = string(d)
	}
	joined := strings.Join(strs, "\n---\n")

	var out strings.Builder
	if header != "" {
		out.WriteString(header)
		out.WriteString("\n\n")
	}
	out.WriteString(joined)
	out.WriteString("\n")

	return []byte(out.String()), nil
}
