# TabGen

Automatically generate tab completions for CLI tools by analyzing their `--help` output and man pages.

## Overview

TabGen scans your system for executable tools, parses their help documentation, and generates working tab completion scripts for both Bash and Zsh. It works alongside existing completions without overwriting them.

## Installation

```bash
go install github.com/justin/tabgen@latest
```

## Quick Start

```bash
# 1. Scan your PATH for executables
tabgen scan

# 2. Generate completions for a specific tool
tabgen generate kubectl

# 3. Or generate for all discovered tools
tabgen generate

# 4. Install shell hooks and symlinks
tabgen install

# 5. Restart your shell or source your rc file
source ~/.bashrc  # for bash
source ~/.zshrc   # for zsh
```

## Commands

| Command | Description |
|---------|-------------|
| `tabgen scan` | Discover executables in `$PATH` and update catalog |
| `tabgen generate [tool]` | Generate completions for one or all tools |
| `tabgen generate -f` | Force regeneration even if up-to-date |
| `tabgen list` | Show discovered tools with generation status |
| `tabgen list --all` | Show all tools including skipped ones |
| `tabgen install` | Set up symlinks, shell hooks, and daily scan timer |
| `tabgen install --skip-timer` | Install without setting up automatic scanning |
| `tabgen uninstall` | Remove all TabGen artifacts |
| `tabgen uninstall --keep-data` | Uninstall but keep generated completions |
| `tabgen status` | Show installation health and statistics |
| `tabgen exclude list` | Show excluded tool patterns |
| `tabgen exclude add <pattern>` | Add a tool or pattern to exclusions |
| `tabgen exclude remove <pattern>` | Remove a pattern from exclusions |
| `tabgen exclude clear` | Clear all exclusions |

## How It Works

1. **Scan**: TabGen walks your `$PATH` directories and discovers executable files, storing metadata in a catalog.

2. **Parse**: For each tool, TabGen runs `--help` (or `-h`) and reads man pages to extract:
   - Subcommands and their descriptions
   - Flags (short and long forms)
   - Flag arguments
   - Nested subcommands (up to 2 levels deep)

3. **Generate**: The parsed data is transformed into shell-specific completion scripts:
   - Bash: Uses `complete` builtins
   - Zsh: Generates `_tool` completion functions

4. **Install**: Symlinks are created to standard completion directories, and shell hooks are added to your rc files.

## Data Layout

TabGen stores all data in `~/.tabgen/`:

```
~/.tabgen/
├── config.json              # Settings (exclusions, scan options)
├── catalog.json             # Discovered tools and metadata
├── tools/
│   └── <tool>.json          # Parsed structure per tool
└── completions/
    ├── bash/
    │   └── <tool>           # Generated bash completions
    └── zsh/
        └── _<tool>          # Generated zsh completions
```

## Features

### Version-Aware Regeneration

TabGen detects tool versions and only regenerates completions when a tool is updated. Use `--force` to regenerate regardless.

### Nested Subcommands

Parses multi-level command structures like `docker container ls` or `kubectl get pods`.

### Exclusion Lists

Skip tools that don't need completions or cause issues:

```bash
tabgen exclude add python2.7
tabgen exclude add "*.dll"
```

### Automatic Scanning

The `install` command sets up either:
- **Systemd timer** (Linux with systemd): Daily scan via user timer
- **Cron job**: Daily scan at 4am as fallback

### Non-Destructive

TabGen completions work alongside existing system completions:
- Bash: Uses `complete -o default -o bashdefault` to supplement
- Zsh: Places TabGen's fpath after system paths

## Configuration

The `~/.tabgen/config.json` file contains:

```json
{
  "tabgen_dir": "~/.tabgen",
  "excluded": ["python2.7", "*.dll"],
  "scan_on_startup": true
}
```

## Supported Shells

- **Bash**: Full completion support
- **Zsh**: Full completion support with compdef integration

## Troubleshooting

### Check installation status
```bash
tabgen status
```

### Tool not generating completions?

Some tools have non-standard help formats. TabGen works best with tools that follow common conventions:
- GNU-style flags (`--option`, `-o`)
- Section headers like "Commands:", "Options:", "Flags:"
- Man pages with OPTIONS sections

### Completions not loading?

1. Ensure shell hooks are installed: `tabgen status`
2. Restart your shell or source your rc file
3. Check that completion directories exist and are readable

## License

MIT
