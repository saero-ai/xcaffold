#!/bin/bash
# demo-setup.sh — Set up and record the xcaffold demo GIF
#
# This creates a demo project, then records your REAL terminal
# using asciinema + agg. The result looks authentic because it IS
# your actual terminal.
#
# Prerequisites:
#   brew install asciinema agg
#   xcaffold must be in PATH (brew install saero-ai/tap/xcaffold)
#
# Usage:
#   bash demo-setup.sh        # creates project + prints instructions
#   bash demo-setup.sh record # creates project AND starts recording
#   bash demo-setup.sh gif    # converts recording to GIF

set -e

DEMO_DIR="/tmp/xcaffold-demo"
CAST_FILE="$DEMO_DIR/demo.cast"
GIF_FILE="assets/demo.gif"

case "${1:-setup}" in
  setup)
    rm -rf "$DEMO_DIR"
    mkdir -p "$DEMO_DIR/xcaf/agents" "$DEMO_DIR/xcaf/rules" "$DEMO_DIR/xcaf/contexts" "$DEMO_DIR/xcaf/skills"

    cat > "$DEMO_DIR/project.xcaf" << 'EOF'
kind: project
version: "1.0"
name: my-app
description: "Agent harness for the engineering team."
targets: [claude, codex, cursor]
EOF

    cat > "$DEMO_DIR/xcaf/contexts/workspace.xcaf" << 'EOF'
---
kind: context
version: "1.0"
name: workspace
description: "Shared workspace context for all providers."
default: true
targets: [claude, codex, cursor]
---
This is a TypeScript monorepo using pnpm workspaces.
Run all tests with: pnpm test
Run linting with: pnpm lint
EOF

    cat > "$DEMO_DIR/xcaf/agents/developer.xcaf" << 'EOF'
---
kind: agent
version: "1.0"
name: developer
description: "Full-stack developer for the application."
model: sonnet
tools: [Read, Write, Edit, Bash, Glob, Grep]
skills: [code-review]
---
You are a senior full-stack developer.
Write clean, tested TypeScript code.
Follow existing patterns in the codebase.
EOF

    cat > "$DEMO_DIR/xcaf/rules/no-secrets.xcaf" << 'EOF'
---
kind: rule
version: "1.0"
name: no-secrets
description: "Prevent secrets from being committed."
activation: always
always-apply: true
---
Never commit API keys, tokens, or credentials to source control.
Use environment variables for all secrets.
Validate .env files are in .gitignore before any commit.
EOF

    cat > "$DEMO_DIR/xcaf/skills/code-review.xcaf" << 'EOF'
---
kind: skill
version: "1.0"
name: code-review
description: "Structured code review with security and quality checks."
allowed-tools: [Read, Glob, Grep, Bash]
user-invocable: true
argument-hint: "PR number or file path to review"
---
Review code for bugs, security issues, and style violations.
Check test coverage for modified functions.
Flag any secrets or credentials in the diff.
EOF

    echo 'export PS1="%F{green}$ %f"' > "$DEMO_DIR/.zshrc"

    echo ""
    echo "Demo project ready at $DEMO_DIR"
    echo ""
    echo "To record the demo (clean prompt, no personal info):"
    echo "  cd $DEMO_DIR"
    echo "  ZDOTDIR=$DEMO_DIR asciinema rec --cols 100 --rows 30 demo.cast"
    echo ""
    echo "Then type these commands (take your time — natural pace looks best):"
    echo ""
    echo "  1. cat project.xcaf"
    echo "  2. xcaffold graph"
    echo "  3. xcaffold apply --yes"
    echo "  4. xcaffold status"
    echo "  5. echo '# UNAUTHORIZED EDIT' >> .claude/rules/no-secrets.md"
    echo "  6. xcaffold status"
    echo "  7. exit"
    echo ""
    echo "Then convert to GIF:"
    echo "  agg --theme monokai --font-size 16 $CAST_FILE $GIF_FILE"
    echo ""
    echo "Or run: bash demo-setup.sh record"
    ;;

  record)
    # Run setup first
    bash "$0" setup
    echo "Starting asciinema recording..."
    echo "Type the 5 commands shown above, then type 'exit' to stop."
    echo ""
    cd "$DEMO_DIR"
    asciinema rec --cols 100 --rows 30 --overwrite "$CAST_FILE"
    echo ""
    echo "Recording saved to $CAST_FILE"
    echo "Convert to GIF: bash demo-setup.sh gif"
    ;;

  gif)
    if [ ! -f "$CAST_FILE" ]; then
      echo "No recording found at $CAST_FILE"
      echo "Run: bash demo-setup.sh record"
      exit 1
    fi
    echo "Converting to GIF..."
    agg --theme monokai --font-size 16 "$CAST_FILE" "$GIF_FILE"
    echo "GIF saved to $GIF_FILE"
    echo "Preview: open $GIF_FILE"
    ;;

  *)
    echo "Usage: bash demo-setup.sh [setup|record|gif]"
    ;;
esac
