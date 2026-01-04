# TabGen

<p align="center">
  <img src="banner.png" alt="TabGen - Automatic Tab Completion Generator" />
</p>

Automatically generate tab completions for CLI tools by analyzing their `--help` output and man pages.

## Overview

TabGen scans your system for executable tools **that you actually use** (based on shell history), parses their help documentation, and generates working tab completion scripts for both Bash and Zsh. It works alongside existing completions without overwriting them, using intelligent caching to avoid unnecessary regeneration.

## Installation

```bash
go install github.com/jvalentini/tabgen@latest
```

## Quick Start

```bash
# 1. Install TabGen
go install github.com/jvalentini/tabgen@latest

# 2. Scan your PATH for executables (filters by shell history)
tabgen scan

# 3. Generate completions for a specific tool
tabgen generate kubectl

# 4. Or generate for all discovered tools (uses concurrent workers)
tabgen generate

# 5. Optional: Force regeneration with verbose output
tabgen -v generate --force

# 6. Install shell hooks and symlinks
tabgen install

# 7. Restart your shell or source your rc file
source ~/.bashrc  # for bash
source ~/.zshrc   # for zsh

# 8. Start using completions!
kubectl get <TAB>              # Shows: pods, services, deployments, etc.
kubectl get pods --<TAB>       # Shows: --all-namespaces, --field-selector, etc.
```

## Commands

| Command | Description |
|---------|-------------|
| `tabgen scan` | Discover executables in `$PATH` that appear in shell history |
| `tabgen generate [tool]` | Generate completions for one or all tools (concurrent by default) |
| `tabgen generate -f\|--force` | Force regeneration even if up-to-date |
| `tabgen generate -w\|--workers N` | Set number of concurrent workers (default: CPU count) |
| `tabgen list` | Show discovered tools with generation status |
| `tabgen list --all` | Show all tools including those without completions |
| `tabgen install` | Set up symlinks, shell hooks, and daily scan timer |
| `tabgen install --skip-timer` | Install without setting up automatic scanning |
| `tabgen uninstall` | Remove all TabGen artifacts |
| `tabgen uninstall --keep-data` | Uninstall but keep generated completions |
| `tabgen status` | Show installation health and statistics |
| `tabgen exclude list` | Show excluded tool patterns |
| `tabgen exclude add <pattern>` | Add a tool or pattern to exclusions |
| `tabgen exclude remove <pattern>` | Remove a pattern from exclusions |
| `tabgen exclude clear` | Clear all exclusions |

**Global Options:**
- `-v, --verbose`: Show detailed parsing and debug output

## How It Works

1. **Scan**: TabGen walks your `$PATH` directories and discovers executable files **that appear in your shell history** (`.bash_history`, `.zsh_history`). This focuses on tools you actually use, avoiding clutter from rarely-used binaries. Metadata is stored in a catalog.

2. **Parse**: For each tool, TabGen runs `--help` (or `-h`) and reads man pages to extract:
   - Subcommands and their descriptions
   - Flags (short and long forms)
   - Flag arguments and allowed values
   - Nested subcommands (up to 2 levels deep)
   - Command aliases

3. **Generate**: The parsed data is transformed into shell-specific completion scripts using **concurrent processing**:
   - Bash: Uses `complete` builtins with fallback behavior
   - Zsh: Generates `_tool` completion functions with `_arguments`
   - Supports parallel generation with configurable workers

4. **Install**: Symlinks are created to standard completion directories, and shell hooks are added to your rc files. Optionally sets up daily automatic scanning via systemd timers or cron.

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

### Tool JSON Schema

Each `tools/<tool>.json` file contains structured data:

```json
{
  "name": "kubectl",
  "version": "1.28.0",
  "parsed_at": "2024-01-15T10:30:00Z",
  "source": "help",
  "subcommands": [
    {
      "name": "get",
      "aliases": ["g"],
      "description": "Display one or many resources",
      "flags": [
        {
          "name": "--output",
          "short": "-o",
          "arg": "format",
          "argument_values": ["json", "yaml", "wide"],
          "description": "Output format"
        }
      ],
      "subcommands": []
    }
  ],
  "global_flags": [
    {
      "name": "--help",
      "short": "-h",
      "description": "Show help"
    }
  ]
}
```

### Catalog JSON Schema

`catalog.json` tracks all discovered tools:

```json
{
  "last_scan": "2024-01-15T10:00:00Z",
  "tools": {
    "kubectl": {
      "name": "kubectl",
      "path": "/usr/local/bin/kubectl",
      "version": "1.28.0",
      "generated_version": "1.28.0",
      "content_hash": "a1b2c3d4...",
      "generated": true,
      "last_scan": "2024-01-15T10:00:00Z",
      "has_help": true,
      "has_man_page": true
    }
  }
}
```

## Features

### Shell History Filtering

Only generates completions for tools you **actually use**. TabGen scans `.bash_history` and `.zsh_history` to identify frequently-used commands, avoiding wasted effort on rarely-used binaries in your `$PATH`.

### Smart Regeneration

TabGen uses two mechanisms to avoid unnecessary regeneration:
1. **Version detection**: Tracks tool versions and only regenerates when updated
2. **Content hashing**: Detects changes in help output even when version numbers don't change

Use `--force` to regenerate regardless of these checks.

### Concurrent Processing

Generation uses parallel workers (default: CPU count) for fast processing of large catalogs:

```bash
tabgen generate -w 8  # Use 8 workers
```

### Nested Subcommands

Parses multi-level command structures like `docker container ls` or `kubectl get pods` up to 2 levels deep.

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

## Technical Architecture

### Scanning Pipeline

1. **PATH Discovery**: Walks all directories in `$PATH` environment variable
2. **History Parsing**: Reads `.bash_history` and `.zsh_history` to identify used commands
3. **Filtering**: Applies exclusion patterns and filters for executables in history
4. **Cataloging**: Stores metadata (path, version, timestamps) in `catalog.json`

### Parsing Pipeline

1. **Help Execution**: Runs `<tool> --help` with 5-second timeout
2. **Man Page Fallback**: Reads `man <tool>` if help fails or as supplement
3. **Regex Extraction**: Parses output for:
   - Section headers (Commands:, Options:, Flags:)
   - Flag patterns: `-f, --flag <arg>`, `--flag=value`, etc.
   - Subcommand patterns with descriptions
   - Argument value lists: `{json,yaml}`, `json|yaml`
4. **Structure Building**: Creates nested Command/Flag hierarchy
5. **JSON Storage**: Saves to `tools/<tool>.json`

### Generation Pipeline

1. **Worker Pool**: Creates N workers (default: CPU count)
2. **Version Check**: Compares current version/hash with generated version/hash
3. **Skip Logic**: Skips if unchanged (unless `--force`)
4. **Bash Generation**: Creates completion function using `_init_completion` and `compgen`
5. **Zsh Generation**: Creates completion function using `_arguments` and `_describe`
6. **Catalog Update**: Marks tool as generated with current version/hash

### Completion Loading

**Bash**:
- Symlinks created in `~/.local/share/bash-completion/completions/`
- Shell hook in `~/.bashrc` adds TabGen completion directory to path
- `complete -o default -o bashdefault` allows fallback to system completions

**Zsh**:
- Symlinks created in `~/.zfunc/`
- Shell hook in `~/.zshrc` adds `~/.zfunc` to `$fpath`
- Zsh's completion system loads functions automatically
- TabGen's `fpath` entry placed after system paths for proper precedence

## Supported Shells

- **Bash**: Full completion support with `_init_completion` and `compgen`
- **Zsh**: Full completion support with `_arguments` and `_describe`

## What Gets Parsed

TabGen extracts the following from `--help` output and man pages:

### Commands
- **Primary names**: `clone`, `push`, `pull`
- **Aliases**: `br` for `branch`, `co` for `checkout`
- **Descriptions**: One-line help text
- **Nested subcommands**: Up to 2 levels (e.g., `docker container ls`)

### Flags
- **Long form**: `--output`, `--verbose`
- **Short form**: `-o`, `-v`
- **Arguments**: `<file>`, `<format>`, `VALUE`
- **Allowed values**: `json|yaml`, `{json,yaml,wide}`
- **Descriptions**: Help text for each flag

### Patterns Recognized

**Command sections**:
- `Commands:`
- `Available Commands:`
- `Subcommands:`

**Flag sections**:
- `Options:`
- `Flags:`
- `Global Options:`
- `Global Flags:`

**Flag formats**:
- `-f, --flag` (short and long)
- `--flag=VALUE` (with argument)
- `--flag <value>` (with argument)
- `--format {json,yaml}` (with choices)
- `--format json|yaml` (with choices)

## Performance

TabGen is designed for speed and efficiency:

- **Concurrent generation**: Uses all CPU cores by default (configurable with `-w`)
- **Smart caching**: Only regenerates when tool versions or help output change
- **Quick scanning**: Default scan mode skips slow `--help` checks
- **History filtering**: Only processes tools you actually use

Example performance on a typical system:
- Scan 500 tools from PATH: ~1 second
- Generate 50 completions: ~5 seconds (8-core CPU)
- Incremental regeneration: ~1 second (only changed tools)

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

If a tool isn't generating properly:
1. Check verbose output: `tabgen -v generate <tool>`
2. Verify the tool appears in shell history (required for scan)
3. Add problematic tools to exclusions if they cause issues

### Completions not loading?

1. Ensure shell hooks are installed: `tabgen status`
2. Restart your shell or source your rc file
3. Check that completion directories exist and are readable
4. Verify symlinks are valid: `ls -la ~/.local/share/bash-completion/completions/`

### Performance issues?

If generation is slow:
1. Increase workers: `tabgen generate -w 16`
2. Generate specific tools only: `tabgen generate kubectl docker`
3. Add exclusions for problematic tools that hang on `--help`

## License

MIT
