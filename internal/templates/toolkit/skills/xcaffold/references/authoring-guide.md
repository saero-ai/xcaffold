# xcaffold Manifest Authoring Guide

## Project structure

All xcaffold manifests live under `xcaf/`, organized by resource kind.
Each resource kind has its own directory with a `.xcaf` file.

```
xcaf/
  agents/
    reviewer/
      agent.xcaf            # kind: agent
    xaff/
      agent.xcaf            # base agent
      agent.claude.xcaf     # per-provider override
  skills/
    tdd/
      skill.xcaf            # kind: skill
      references/
        guide.md           # supporting files (optional)
  rules/
    my-rule/
      rule.xcaf             # kind: rule
  workflows/
    setup/
      workflow.xcaf         # kind: workflow
  mcp/
    filesystem/
      mcp.xcaf              # kind: mcp
  policies/
    require-agent-description.xcaf
  hooks/
    pre-tool-use/
      hooks.xcaf            # kind: hooks
  settings.xcaf             # kind: settings (singleton)
```

## Adding a resource

1. Identify the resource kind (agent, skill, rule, workflow, mcp, hooks, settings).

2. Create a subdirectory under `xcaf/<kind>/` with the resource name:

```bash
mkdir -p xcaf/agents/reviewer
mkdir -p xcaf/skills/tdd
mkdir -p xcaf/rules/my-rule
mkdir -p xcaf/workflows/setup
mkdir -p xcaf/mcp/filesystem
```

3. Create the `.xcaf` file with the resource name as the filename:

```yaml
# xcaf/agents/reviewer/agent.xcaf
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

### kind: agent — `xcaf/agents/<name>/agent.xcaf`

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

### kind: skill — `xcaf/skills/<name>/skill.xcaf`

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

### kind: rule — `xcaf/rules/<name>/rule.xcaf`

```yaml
kind: rule
version: \"1.0\"
name: my-rule
description: \"What this rule enforces.\"
activation: always
---
Rule instructions injected into agent context when active.
```

### kind: workflow — `xcaf/workflows/<name>/workflow.xcaf`

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

### kind: mcp — `xcaf/mcp/<name>/mcp.xcaf`

```yaml
kind: mcp
version: \"1.0\"
name: filesystem
type: stdio
command: npx
args: [\"-y\", \"@modelcontextprotocol/server-filesystem\", \".\"]
```

### kind: hooks — `xcaf/hooks/<name>/hooks.xcaf`

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

### kind: memory — `xcaf/agents/<name>/memory/<id>.md`

```yaml
---
kind: memory
version: \"1.0\"
name: project-context
---
Memory content goes here.
```

### kind: settings — singleton, declared in `project.xcaf` (pure YAML)

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

Fields marked 'dropped' remain in `xcaf/` source. xcaffold removes them at compile time.

## Common validation errors

| Error | Cause | Fix |
|---|---|---|
| `require-agent-instructions` | `instructions:` < 10 chars | Add meaningful instructions |
| `require-agent-description` | `description:` missing | Add a description field |
| `duplicate agent ID` | Same `name:` in two `.xcaf` files | Rename one |
| `path traversal` | Relative path with `..` | Use path relative to project root |
