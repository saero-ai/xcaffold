---
title: "Shell Completion"
description: "Enable tab completion for xcaffold commands, flags, and resource names in Bash, Zsh, Fish, and PowerShell."
---

# Shell Completion

`xcaffold` is built on [Cobra](https://github.com/spf13/cobra), which ships a `completion` subcommand for Bash, Zsh, Fish, and PowerShell. Completion covers the seven public commands (`init`, `apply`, `import`, `validate`, `status`, `graph`, `list`) plus flags such as `--target`, `--config`, and per-kind filters on `import` and `list`.

Generate scripts with:

```bash
xcaffold completion bash
xcaffold completion zsh
xcaffold completion fish
xcaffold completion powershell
```

## Bash

### Current session only

```bash
source <(xcaffold completion bash)
```

### Persistent setup

Add to `~/.bashrc` (or `~/.bash_profile` on macOS):

```bash
source <(xcaffold completion bash)
```

Reload: `source ~/.bashrc`

On Linux you can also install system-wide:

```bash
xcaffold completion bash | sudo tee /etc/bash_completion.d/xcaffold > /dev/null
```

## Zsh

### Current session only

```bash
source <(xcaffold completion zsh)
```

### Persistent setup

**macOS** (Homebrew Zsh): save the script and reference it from your config:

```bash
xcaffold completion zsh > "${HOME}/.zfunc/_xcaffold"
```

Add to `~/.zshrc`:

```zsh
fpath=(~/.zfunc $fpath)
autoload -Uz compinit && compinit
```

**Linux** (system site-functions):

```bash
xcaffold completion zsh | sudo tee /usr/local/share/zsh/site-functions/_xcaffold > /dev/null
```

Reload: `exec zsh` or open a new terminal.

## Fish

### Current session only

```fish
xcaffold completion fish | source
```

### Persistent setup

```fish
xcaffold completion fish > ~/.config/fish/completions/xcaffold.fish
```

Fish loads completions from `~/.config/fish/completions/` automatically on the next shell start.

## PowerShell

### Current session only

```powershell
xcaffold completion powershell | Out-String | Invoke-Expression
```

### Persistent setup

Add to your PowerShell profile (`$PROFILE`):

```powershell
xcaffold completion powershell | Out-String | Invoke-Expression
```

To find your profile path: `echo $PROFILE`

## Verify

After enabling completion, start typing and press `Tab`:

```bash
xcaffold <Tab>          # lists commands
xcaffold apply --<Tab>  # lists apply flags
xcaffold import --<Tab> # lists import flags and kind filters
```

If nothing appears, confirm `xcaffold` is on your `PATH` and that you reloaded the shell config (or sourced the one-liner for the current session).

## See also

- [CLI Reference](../reference/commands/index.md)
- [Cobra completion documentation](https://github.com/spf13/cobra/blob/main/site/content/completions/_index.md)
