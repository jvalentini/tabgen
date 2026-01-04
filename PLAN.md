# TabGen Design

> Automatically generate tab completions by analyzing CLI tools

## Architecture

```
~/.tabgen/
├── config.json              # Settings (scan frequency, excluded tools)
├── catalog.json             # Discovered tools + metadata
├── tools/
│   └── <tool>.json          # Parsed data per tool (subcommands, flags, descriptions)
└── completions/
    ├── bash/
    │   └── <tool>           # Generated bash completions
    └── zsh/
        └── _<tool>          # Generated zsh completions

Symlinks:
~/.local/share/bash-completion/completions/tabgen → ~/.tabgen/completions/bash/*
~/.zfunc/_tabgen_* → ~/.tabgen/completions/zsh/*
```

## Commands

| Command | Purpose |
|---------|---------|
| `tabgen scan` | Walk $PATH, discover executables, update catalog |
| `tabgen generate [tool]` | Parse --help/man, generate completions (all if no arg) |
| `tabgen list` | Show discovered tools + generation status |
| `tabgen install` | Set up symlinks, systemd timer/cron, shell hooks |
| `tabgen uninstall` | Clean removal of all TabGen artifacts |
| `tabgen status` | Health check, show installed completions |

## Parsing Pipeline

```
Tool binary
    │
    ├──→ <tool> --help  ──→ Regex parser ──→ Structured JSON
    │
    └──→ man <tool>     ──→ Man parser (fallback) ──→ Structured JSON
```

### Output Schema (`tools/<tool>.json`)

```json
{
  "name": "kubectl",
  "version": "1.28.0",
  "parsed_at": "2024-01-15T...",
  "source": "help",
  "subcommands": [
    {
      "name": "get",
      "description": "Display resources",
      "subcommands": [],
      "flags": [
        {"name": "--output", "short": "-o", "arg": "format", "description": "Output format"}
      ]
    }
  ],
  "global_flags": []
}
```

## Side-by-Side Strategy

- **Bash**: Use `complete -o default -o bashdefault` so TabGen completions only supplement
- **Zsh**: Place TabGen's fpath *after* system paths; existing completions take precedence
- TabGen completions named distinctly to avoid overwriting

## Scan Triggers

1. **Daily**: Systemd timer (preferred) or cron
2. **Shell startup**: Lightweight check in `.bashrc`/`.zshrc` (only scans if $PATH changed or >24h since last)

---

## Roadmap

### MVP (Phase 1)
Core parsing and generation working end-to-end:

- [ ] `tabgen scan` - Walk $PATH, build catalog.json
- [ ] `--help` regex parser - Handle common formats (GNU, subcommand-style, etc.)
- [ ] `man` parser - Extract OPTIONS section, parse flag definitions
- [ ] `tools/<tool>.json` schema and writer
- [ ] Bash completion generator - Produce working completion script from JSON
- [ ] Zsh completion generator - Produce working `_<tool>` script from JSON
- [ ] `tabgen generate <tool>` - Single tool generation
- [ ] `tabgen generate` - Batch generation for all cataloged tools
- [ ] `tabgen list` - Show catalog with status

**Exit criteria**: Can run `tabgen scan && tabgen generate kubectl` and get working completions for both shells.

### Phase 2
Installation and automation:

- [ ] `tabgen install` - Create symlinks, detect systemd vs cron
- [ ] Systemd user timer for daily scan
- [ ] Shell startup hook (bash/zsh) for lightweight rescan
- [ ] `tabgen uninstall` - Clean removal
- [ ] `tabgen status` - Health check
- [ ] Exclusion list in config (ignore certain tools)

### Phase 3
Polish and robustness:

- [ ] Version detection - Store tool version, detect changes
- [ ] Incremental regeneration - Only regenerate when tool updated
- [ ] Better --help heuristics - Handle more edge cases
- [ ] Nested subcommand depth (e.g., `docker container ls`)
- [ ] Flag argument completions (e.g., `--format=<TAB>` suggests json/yaml)

### Future Ideas

- LLM-assisted parsing fallback for weird --help formats
- Import existing completion files to seed the database
- Community sharing of tool definitions
- Fish shell support
