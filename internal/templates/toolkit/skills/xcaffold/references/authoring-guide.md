# xcaffold Manifest Authoring Guide

## Project structure

All xcaffold manifests live under `xcf/`, organized by resource kind.
Each resource kind has its own directory with a `.xcf` file.

```
xcf/
  agents/
    reviewer/
      agent.xcf            # kind: agent
    xaff/
      agent.xcf            # base agent
      agent.claude.xcf     # per-provider override
  skills/
    tdd/
      skill.xcf            # kind: skill
      references/
        guide.md           # supporting files (optional)
  rules/
    my-rule/
      rule.xcf             # kind: rule
  workflows/
    setup/
      workflow.xcf         # kind: workflow
  mcp/
    filesystem/
      mcp.xcf              # kind: mcp
  policies/
    require-agent-description.xcf
  hooks/
    pre-tool-use/
      hooks.xcf            # kind: hooks
  settings.xcf             # kind: settings (singleton)
```

## Adding a resource

1. Identify the resource kind (agent, skill, rule, workflow, mcp, hooks, settings).

2. Create a subdirectory under `xcf/<kind>/` with the resource name:

```bash
mkdir -p xcf/agents/reviewer
mkdir -p xcf/skills/tdd
mkdir -p xcf/rules/my-rule
mkdir -p xcf/workflows/setup
mkdir -p xcf/mcp/filesystem
```

3. Create the `.xcf` file with the resource name as the filename:

```yaml
# xcf/agents/reviewer/agent.xcf
---
kind: agent
version: \"1.0\"
name: reviewer
description: \"Code reviewer agent.\"
tools: [Read, Glob, Grep]
---
You are a thorough code reviewer.
Provide actionable feedback on code quality, correctness, and security.
```

4. Validate and apply:

```bash
xcaffold validate
xcaffold apply --target claude
```

## Schema quick reference

### kind: agent — `xcf/agents/<name>/agent.xcf`

```yaml
kind: agent
version: \"1.0\"
name: reviewer
description: \"Short description (required).\"
model: \"claude-sonnet-4-6\"     # claude, gemini, antigravity only
effort: \"high\"                 # claude only
tools: [Read, Write, Edit, Bash]
skills: [tdd, reviewing]
rules: [my-rule]
memory: [project-context]
---
Your agent instructions here. Minimum 10 characters.
```

### kind: skill — `xcf/skills/<name>/skill.xcf`

```yaml
kind: skill
version: \"1.0\"
name: tdd
description: \"Guides test-driven development.\"
allowed-tools: [Read, Write, Edit, Bash]
references: [references/guide.md]
---
Write a failing test first. Write minimal code to pass. Refactor.
```

### kind: rule — `xcf/rules/<name>/rule.xcf`

```yaml
kind: rule
version: \"1.0\"
name: my-rule
description: \"What this rule enforces.\"
activation: always
---
Rule instructions injected into agent context when active.
```

### kind: workflow — `xcf/workflows/<name>/workflow.xcf`

```yaml
kind: workflow
version: \"1.0\"
name: setup
steps:
  - name: install
    description: \"Install dependencies.\"
  - name: configure
    description: \"Configure application.\"
```

### kind: mcp — `xcf/mcp/<name>/mcp.xcf`

```yaml
kind: mcp
version: \"1.0\"
name: filesystem
type: stdio
command: npx
args: [\"-y\", \"@modelcontextprotocol/server-filesystem\", \".\"]
```

### kind: hooks — `xcf/hooks/<name>/hooks.xcf`

```yaml
kind: hooks
version: \"1.0\"
events:
  PreToolUse:
    - matcher: Bash
      hooks:
        - type: command
          command: \"validate.sh\"
```

### kind: memory — `xcf/agents/<name>/memory/<id>.md`

```yaml
---
kind: memory
version: \"1.0\"
name: project-context
---
Memory content goes here.
```

### kind: settings — `xcf/settings.xcf` (singleton, pure YAML)

```yaml
kind: settings
version: \"1.0\"
mcp-servers: []
permissions: {}
```

## Field support matrix (cross-provider)

| Field | claude | cursor | gemini | copilot | antigravity |
|---|---|---|---|---|---|
| `description` | YES | YES | YES | YES | YES |
| `instructions` | YES | YES | YES | YES | YES |
| `allowed-tools` | YES | YES | YES | YES | YES |
| `model` | YES | dropped | YES | dropped | YES |
| `effort` | YES | dropped | dropped | dropped | dropped |
| `permission-mode` | YES | dropped | dropped | dropped | dropped |
| `hooks` | YES | dropped | dropped | dropped | dropped |
| `memory` | YES | dropped | dropped | dropped | dropped |

Fields marked 'dropped' remain in `xcf/` source. xcaffold removes them at compile time.

## Common validation errors

| Error | Cause | Fix |
|---|---|---|
| `require-agent-instructions` | `instructions:` < 10 chars | Add meaningful instructions |
| `require-agent-description` | `description:` missing | Add a description field |
| `duplicate agent ID` | Same `name:` in two `.xcf` files | Rename one |
| `path traversal` | Relative path with `..` | Use path relative to project root |
