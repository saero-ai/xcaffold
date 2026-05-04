---
title: "xcaffold help"
description: "Display general help or schema documentation for xcaffold resource kinds."
---

# xcaffold help

Display general CLI help or schema documentation for xcaffold resource kinds.

The `help` command provides general usage information for the CLI, lists available commands, and can dynamically generate and display `.xcf` schema reference material and file templates.

**Usage:**

```
xcaffold help [command]
xcaffold help --xcf <kind> [--out [path]]
xcaffold --xcf <kind> [--out [path]]
```

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--xcf <kind>` | — | `string` | `""` | Display schema for a resource kind (e.g., `agent`, `skill`, `rule`). |
| `--out [path]` | — | `string` | `"."` | Generate a template `.xcf` file for the requested kind. Use with `--xcf`. |
| `--config <path>` | — | `string` | `""` | Path to project.xcf (inherited from root). |
| `--global` | `-g` | `bool` | `false` | Operate on user-wide global config (inherited from root). |
| `--no-color` | — | `bool` | `false` | Disable color output (inherited from root). |

## Behavior

### General Help

Running `xcaffold help` (or `xcaffold -h`) without flags prints the root help message, including the version, general usage, a list of available commands, and core global flags. 

Running `xcaffold help <command>` (or `xcaffold <command> --help`) prints detailed help for that specific subcommand, including its description, usage pattern, and unique flags.

### Schema Documentation (`--xcf`)

When the `--xcf <kind>` flag is provided, the CLI displays the field-level schema for the specified resource kind. This data is pulled directly from the internal Go AST and includes:

- **Identity**: Kind name, schema version, and file format (`frontmatter+body` or `pure-yaml`).
- **Field Groups**: Fields organized into logical categories (e.g., `Identity`, `Model & Execution`).
- **Field Attributes**:
    - **YAML Key**: The kebab-case key used in the `.xcf` file.
    - **Type**: The Xcaffold type (e.g., `string`, `integer`, `[]string`).
    - **Requirement**: Whether the field is `required` or `optional`.
    - **Description**: Human-readable explanation of the field's purpose.
- **Constraints**: 
    - **Pattern**: Regex validation pattern.
    - **Examples**: Sample values.
    - **Values**: Allowed enumeration values.
    - **Default**: The default value if omitted.
    - **Providers**: Specific provider support (e.g., `Claude(required) Gemini`).

### Template Generation (`--out`)

The `--out` flag (used in conjunction with `--xcf`) generates a skeleton `.xcf` file populated with all valid fields for that kind.

- **Destination**:
    - If no path is provided (just `--out`), it writes `<kind>.xcf` to the current directory.
    - If a path to a directory is provided, it writes `<kind>.xcf` inside that directory.
    - If a path ending in `.xcf` is provided, it writes to that exact file.
- **Format**:
    - Includes frontmatter delimiters (`---`) for relevant kinds.
    - Fields are grouped with decorative headers (e.g., `# ── Identity ────────`).
    - Every field includes a comment describing its type and requirement.
    - Includes `# +xcf:` markers for AI-assisted authoring (containing constraints like patterns and enums).
    - Uses sensible placeholders (e.g., `""`, `[]`, `false`).

## Examples

**Show general help:**
```bash
xcaffold help
```

Output:
```
xcaffold . deterministic agent configuration compiler

  Usage:  xcaffold [command]

  Commands:
    apply       Compile .xcf resources into provider-native agent files
    graph       Visualize the resource dependency graph
    import      Import existing provider config into project.xcf
    init        Bootstrap a new project.xcf configuration
    list        List discovered resources and blueprints
    status      Show compilation state and check for drift across all providers
    validate    Check .xcf syntax, cross-references, and structural invariants

  Flags:
    --config <path>   Path to project.xcf (default: ./project.xcf)
    --no-color        Disable color output
    -h, --help        Show this help
    -v, --version     Show version

-> Run 'xcaffold [command] --help' for details on any command.
```

**Show help for a specific command:**
```bash
xcaffold help apply
```

Output:
```
Deterministically compiles .xcf resources into provider-native agent files.

  - Strict one-way generation (YAML -> provider-native markdown/JSON)
  - Generates a SHA-256 state manifest for drift detection (.xcaffold/)
  - Automatically purges orphaned target files

Any manually edited files inside the target directory will be overwritten.

Usage:
  xcaffold apply [flags]

Examples:
  $ xcaffold apply
  $ xcaffold apply --dry-run
  $ xcaffold apply --target cursor

Flags:
      --backup             Backup existing target directory before overwriting
      --blueprint string   Compile a specific blueprint (default: all resources)
      --dry-run            Preview changes without writing to disk
      --force              Overwrite customized local files and bypass drift safeguard
  -h, --help               help for apply
      --project string     Apply to an external project registered in the global registry
      --target string      compilation target platform (claude, cursor, antigravity, copilot, gemini; default: claude) (default "claude")

Global Flags:
      --config string      Path to project.xcf (default: ./project.xcf). Use for monorepo sub-directories.
      --no-color           disable color output
      --out string[="."]   Generate template .xcf file (use with --xcf)
      --xcf string         Display schema for a resource kind
```

**Display schema for the 'agent' kind:**
```bash
xcaffold help --xcf agent
```

Output:
```
kind: agent . version 1.0 . format: frontmatter+body

  Identity
    name                      string          required  Unique identifier for this agent within the project.
                                                        Pattern: ^[a-z0-9-]+$
    description               string          optional  Human-readable purpose of this agent.
                                                        Providers: Claude(required) Gemini(required) Copilot(required) Cursor Antigravity

  Model & Execution
    model                     string          optional  LLM model identifier or alias resolved at compile time.
                                                        Providers: Claude Gemini Copilot Cursor Antigravity
                                                        Examples: sonnet
    effort                    string          optional  Reasoning effort level hint for the model provider.
                                                        Providers: Claude Cursor
    max-turns                 int             optional  Maximum conversation turns before the agent exits.
                                                        Providers: Claude Gemini

  Tool Access
    tools                     []string        optional  Ordered list of tools this agent may invoke.
                                                        Providers: Claude Gemini Copilot
    disallowed-tools          []string        optional  Tools explicitly denied to this agent.
                                                        Providers: Claude
    readonly                  boolean         optional  When true, restricts the agent to read-only tool access.
                                                        Providers: Claude Cursor

  Permissions & Invocation
    permission-mode           string          optional  Security mode controlling tool authorization behavior.
                                                        Providers: Claude
    disable-model-invocation  boolean         optional  Prevents the agent from spawning sub-agents.
                                                        Providers: Claude
    user-invocable            boolean         optional  Whether users can invoke this agent directly via slash command.
                                                        Providers: Claude

  Lifecycle
    background                boolean         optional  Runs the agent in background mode without interactive prompts.
                                                        Providers: Claude Cursor
    isolation                 string          optional  Process isolation level for the agent session.
                                                        Providers: Claude

  Memory & Context
    memory                    []string        optional  Named memory banks attached to this agent.
                                                        Providers: Claude
    color                     string          optional  Display color for terminal output differentiation.
    initial-prompt            string          optional  System prompt prepended to every conversation.
                                                        Providers: Claude

  Composition
    skills                    []string        optional  Skill resource IDs attached to this agent.
                                                        Providers: Claude
    rules                     []string        optional  Rule resource IDs governing this agent.
    mcp                       []string        optional  MCP server resource IDs available to this agent.
    assertions                []string        optional  Policy assertion IDs evaluated post-compilation.

  Inline Composition
    mcp-servers               map             optional  Inline MCP server definitions keyed by server name.
                                                        Providers: Claude Gemini Copilot
    hooks                     HookConfig      optional  Inline lifecycle hook definitions for this agent.
                                                        Providers: Claude

  Multi-Target
    targets                   map             optional  Per-provider override configuration keyed by provider name.

-> Run 'xcaffold help --xcf agent --out' to generate a template.
```

**Generate a 'skill' template in the current directory:**
```bash
xcaffold help --xcf skill --out
```

**Generate an 'agent' template at a specific path:**
```bash
xcaffold help --xcf agent --out ./xcf/agents/my-agent.xcf
```

## Notes

- Kinds are validated against the internal taxonomy. Use `xcaffold help --xcf` with an invalid kind to see the list of available kinds.
- Generated templates are syntactically valid and pass `xcaffold validate` immediately.
- Global flags like `--no-color` are honored when displaying schema documentation.
