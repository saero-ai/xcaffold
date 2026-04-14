package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestCmd returns a minimal cobra.Command whose output is captured in buf.
func newTestCmd(buf *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	return cmd
}

func TestReviewXCF_AllResourceSections(t *testing.T) {
	xcf := []byte(`---
kind: project
version: "1.0"
name: testproject
---
kind: global
version: "1.0"
agents:
  ceo:
    model: claude-opus-4-5
    description: "Chief executive"
    tools:
      - Read
      - Bash

skills:
  research:
    name: Research Skill
    tools:
      - WebSearch

rules:
  style-guide:
    alwaysApply: true

hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: echo pre

mcp:
  github:
    type: stdio
    command: mcp-github

workflows:
  deploy:
    name: Deploy Workflow
`)

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	err := reviewXCF(cmd, xcf)
	if err != nil {
		t.Fatalf("reviewXCF returned unexpected error: %v", err)
	}

	out := buf.String()

	// Agents section
	if !strings.Contains(out, "-- AGENTS --") {
		t.Error("expected '-- AGENTS --' section header")
	}
	if !strings.Contains(out, "ceo") {
		t.Error("expected agent 'ceo' in output")
	}
	if !strings.Contains(out, "claude-opus-4-5") {
		t.Error("expected agent model 'claude-opus-4-5' in output")
	}

	// No emoji in output
	if strings.Contains(out, "\U0001f916") { // robot emoji
		t.Error("output must not contain robot emoji")
	}

	// Skills section
	if !strings.Contains(out, "-- SKILLS --") {
		t.Error("expected '-- SKILLS --' section header")
	}
	if !strings.Contains(out, "Research Skill") {
		t.Error("expected skill name 'Research Skill' in output")
	}
	if !strings.Contains(out, "WebSearch") {
		t.Error("expected skill tool 'WebSearch' in output")
	}

	// Rules section
	if !strings.Contains(out, "-- RULES --") {
		t.Error("expected '-- RULES --' section header")
	}
	if !strings.Contains(out, "style-guide") {
		t.Error("expected rule 'style-guide' in output")
	}
	if !strings.Contains(out, "(always-apply)") {
		t.Error("expected '(always-apply)' suffix for alwaysApply rule")
	}

	// Hooks section
	if !strings.Contains(out, "-- HOOKS --") {
		t.Error("expected '-- HOOKS --' section header")
	}
	if !strings.Contains(out, "PreToolUse") {
		t.Error("expected hook event 'PreToolUse' in output")
	}

	// MCP section
	if !strings.Contains(out, "-- MCP SERVERS --") {
		t.Error("expected '-- MCP SERVERS --' section header")
	}
	if !strings.Contains(out, "github") {
		t.Error("expected MCP server 'github' in output")
	}

	// Workflows section
	if !strings.Contains(out, "-- WORKFLOWS --") {
		t.Error("expected '-- WORKFLOWS --' section header")
	}
	if !strings.Contains(out, "Deploy Workflow") {
		t.Error("expected workflow name 'Deploy Workflow' in output")
	}
}

func TestReviewXCF_NoEmoji(t *testing.T) {
	xcf := []byte(`kind: global
version: "1.0"
agents:
  my-agent:
    model: claude-sonnet-4-5
`)

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	if err := reviewXCF(cmd, xcf); err != nil {
		t.Fatalf("reviewXCF returned unexpected error: %v", err)
	}

	out := buf.String()
	// Robot emoji codepoint U+1F916
	if strings.ContainsRune(out, '\U0001F916') {
		t.Error("CLI output must not contain robot emoji (violates open-source standards)")
	}
}

func TestReviewXCF_EmptySectionsOmitted(t *testing.T) {
	xcf := []byte(`kind: global
version: "1.0"
agents:
  solo:
    model: claude-haiku-4-5
`)

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	if err := reviewXCF(cmd, xcf); err != nil {
		t.Fatalf("reviewXCF returned unexpected error: %v", err)
	}

	out := buf.String()

	for _, section := range []string{"-- SKILLS --", "-- RULES --", "-- HOOKS --", "-- MCP SERVERS --", "-- WORKFLOWS --"} {
		if strings.Contains(out, section) {
			t.Errorf("section %q should not appear when no entries exist", section)
		}
	}
}
