---
title: "Configuration Examples"
description: "Standalone .xcf snippet files demonstrating different features and topology patterns"
---

# Configuration Examples

This directory provides standalone snippet files demonstrating different `.xcf` features and topology patterns. Each file contains exactly one resource document, matching the format the parser requires.

## Project

- [minimal.xcf](minimal.xcf) — A bare-bones `kind: project` starting point
- [multi-agent.xcf](multi-agent.xcf) — A `kind: project` referencing multiple agents, skills, and rules
- [mcp.xcf](mcp.xcf) — A `kind: project` referencing MCP server resources
- [test-assertions.xcf](test-assertions.xcf) — A `kind: project` with LLM-as-a-Judge test configuration

## Agents

Body-bearing kinds (`agent`, `skill`, `rule`) use frontmatter format: YAML between the opening and closing `---`, followed by the instruction body as plain text.

- [minimal-agent.xcf](minimal-agent.xcf) — A bare-bones agent
- [agent-developer.xcf](agent-developer.xcf) — Developer agent with tools and a linked skill
- [agent-reviewer.xcf](agent-reviewer.xcf) — Code reviewer agent with read-only tools
- [agent-frontend.xcf](agent-frontend.xcf) — Frontend agent example
- [agent-backend.xcf](agent-backend.xcf) — Backend agent example
- [agent-tester.xcf](agent-tester.xcf) — Agent with LLM-as-a-Judge assertions

## Skills

- [skill-tdd.xcf](skill-tdd.xcf) — TDD workflow skill with allowed-tools
- [skill-react.xcf](skill-react.xcf) — React development skill

## Rules

- [rule-testing-framework.xcf](rule-testing-framework.xcf) — Always-active testing convention rule
- [rule-secure-coding.xcf](rule-secure-coding.xcf) — Secure coding rule

## Settings, Hooks, and MCP

Non-body kinds (`project`, `settings`, `hooks`, `policy`, `mcp`, `global`) use pure YAML format — no frontmatter delimiters.

- [settings.xcf](settings.xcf) — Settings with an MCP filesystem server
- [hooks.xcf](hooks.xcf) — Pre-tool-use hook definition
- [global.xcf](global.xcf) — Global shared configuration
- [mcp-local-db.xcf](mcp-local-db.xcf) — stdio MCP server (SQLite)
- [mcp-remote-api.xcf](mcp-remote-api.xcf) — SSE MCP server with authorization header

## Policies

- [policy-require.xcf](policy-require.xcf) — Require agents to use an approved model identifier
- [policy-deny.xcf](policy-deny.xcf) — Block compiled output containing TODO or FIXME markers
- [policy-override.xcf](policy-override.xcf) — Silence a built-in policy check during a migration window

## CI

- [xcaffold-diff.yml](xcaffold-diff.yml) — GitHub Actions workflow for CI diff checks
