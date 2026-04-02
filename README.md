# xcaffold

The enterprise fleet management layer for Anthropic Claude Code.
`xcaffold` enforces one-way compilation for agent configuration compiler, translating declarative `.xcf` YAML configurations into deterministically generated, token-analyzed `.claude/` markdown files.

## Example Usage

```yaml
version: "1.0"
project:
  name: "acme-web-platform"

agents:
  developer:
    description: "Expert React developer."
    model: claude-3-5-sonnet-20241022
    tools: [Read, Write, Bash, Glob]
    instructions: |
      You are a frontend developer specializing in standard React.
    assertions:
      - "The agent must not write files outside the project directory."
      - "The agent must run tests before marking a task complete."

test:
  claude_path: ""                           # Defaults to 'claude' on $PATH
  judge_model: "claude-3-5-haiku-20241022"  # Used by xcaffold test --judge
```

To install or compile this infrastructure locally:

**Option 1: Install globally (Recommended)**
```bash
$ go install github.com/saero-ai/xcaffold/cmd/xcaffold@latest
# Now you can run the commands directly anywhere
$ xcaffold init   # Bootstraps the initial scaffold.xcf file
$ xcaffold plan   # Runs static token-bloat analysis
$ xcaffold apply  # Compiles to .claude/ and writes scaffold.lock
$ xcaffold diff   # Detects shadow AI edits made directly to markdown files
$ xcaffold test   # Runs a sandboxed local simulation of a Claude agent
```

**Option 2: Build and run the local executable**
```bash
$ go build -o xcaffold ./cmd/xcaffold/...
$ ./xcaffold init
```

## Argument Reference

The following commands are supported:

* `init` - Scaffolds a new `scaffold.xcf` base configuration in the current working directory.
* `plan` - (Dry Run) Performs static token-bloat analysis on the Abstract Syntax Tree (AST) to evaluate budget compliance without writing files.
* `apply` - Translates the configuration into Claude Code native structures (`.claude/agents/*.md`) and generates a cryptographic tracker manifest.
* `diff` - Compares the `scaffold.lock` SHA-256 hashes against actual files on disk to flag manual configuration drift within the `.claude/` directory.
* `test` - Runs a sandboxed local simulation of a Claude agent using a transport-layer HTTP intercept proxy. Tool calls are mocked and logged to a trace file. Accepts an optional `--judge` flag for LLM-as-a-Judge evaluation.

### `xcaffold test` Flags

| Flag | Default | Description |
|---|---|---|
| `--agent`, `-a` | _(required)_ | Agent ID from `scaffold.xcf` to simulate. |
| `--judge` | `false` | Run LLM-as-a-Judge evaluation after simulation. |
| `--output`, `-o` | `trace.jsonl` | Path to write the execution trace. |
| `--claude-path` | `""` | Path to the `claude` binary. Overrides `test.claude_path` in `scaffold.xcf`. |
| `--judge-model` | `""` | Anthropic model for the judge. Overrides `test.judge_model` in `scaffold.xcf`. |

The `scaffold.xcf` file supports the following top-level blocks:

* `project` - (Required) Object. Defines project identity containing a `name` string.
* `agents` - (Optional) Map. Defines Claude agents with `description`, `instructions`, `model`, `effort`, `tools`, `blocked_tools`, `skills`, `rules`, `mode`, and `assertions`.
* `skills` - (Optional) Map. Defines prompt packages with `description`, `instructions`, `paths`, and `tools`.
* `rules` - (Optional) Map. Defines path-gated formatting guidelines with `paths` and `instructions`.
* `hooks` - (Optional) Map. Lifecycle events mapped to shell scripts (`event`, `match`, `run`).
* `mcp` - (Optional) Map. Local MCP server contexts via `command`, `args`, and `env`.
* `test` - (Optional) Object. Configures the local simulator with `claude_path` and `judge_model`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported/generated on disk:

* `.claude/agents/*.md` - Native markdown persona definitions.
* `scaffold.lock` - A SHA-256 state manifest tracking generated artifacts for drift detection.
* `trace.jsonl` - Newline-delimited JSON execution trace, written by `xcaffold test`.

## Import/Compatibility

`xcaffold` enforces **One-Way Compilation**. It does not currently support pulling down existing `.claude/` markdown files. It will overwrite any files in the `.claude/` directory that it manages. We strongly recommend adding `.claude/` to your `.gitignore` and only committing `scaffold.xcf` and `scaffold.lock`.
