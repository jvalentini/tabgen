package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Flag represents a command-line flag/option
type Flag struct {
	Name           string   `json:"name"`                     // Long form, e.g., "--output"
	Short          string   `json:"short,omitempty"`          // Short form, e.g., "-o"
	Arg            string   `json:"arg,omitempty"`            // Argument name, e.g., "format"
	ArgumentValues []string `json:"argument_values,omitempty"` // Allowed values, e.g., ["json", "yaml"]
	Description    string   `json:"description,omitempty"`    // Help text
	Required       bool     `json:"required,omitempty"`       // Whether the flag is required
}

// Command represents a command or subcommand
type Command struct {
	Name        string    `json:"name"`                  // Command name
	Aliases     []string  `json:"aliases,omitempty"`     // Alternative names (e.g., "br" for "branch")
	Description string    `json:"description,omitempty"` // Help text
	Subcommands []Command `json:"subcommands,omitempty"` // Nested subcommands
	Flags       []Flag    `json:"flags,omitempty"`       // Command-specific flags
}

// Tool represents a parsed CLI tool
type Tool struct {
	Name        string    `json:"name"`                  // Binary name
	Path        string    `json:"path"`                  // Full path to binary
	Version     string    `json:"version,omitempty"`     // Detected version
	ParsedAt    time.Time `json:"parsed_at"`             // When parsing occurred
	Source      string    `json:"source"`                // "help", "man", or "both"
	Subcommands []Command `json:"subcommands,omitempty"` // Top-level subcommands
	GlobalFlags []Flag    `json:"global_flags,omitempty"` // Flags available to all subcommands
}

// ContentHash computes a hash of the tool's parsed content (subcommands and flags).
// This is used to detect when help output changes without a version bump.
func (t *Tool) ContentHash() string {
	// Create a minimal struct with just the content we care about
	// Excludes: Name, Path, Version, ParsedAt, Source (these change or don't affect completions)
	content := struct {
		Subcommands []Command `json:"subcommands,omitempty"`
		GlobalFlags []Flag    `json:"global_flags,omitempty"`
	}{
		Subcommands: t.Subcommands,
		GlobalFlags: t.GlobalFlags,
	}

	data, err := json.Marshal(content)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CatalogEntry represents a discovered tool in the catalog
type CatalogEntry struct {
	Name             string    `json:"name"`                        // Binary name
	Path             string    `json:"path"`                        // Full path to binary
	Version          string    `json:"version,omitempty"`           // Current detected version
	GeneratedVersion string    `json:"generated_version,omitempty"` // Version when completions were generated
	ContentHash      string    `json:"content_hash,omitempty"`      // Hash of parsed tool content (subcommands/flags)
	Generated        bool      `json:"generated"`                   // Whether completions have been generated
	LastScan         time.Time `json:"last_scan"`                   // When this tool was last scanned
	HasHelp          bool      `json:"has_help,omitempty"`          // Whether --help works
	HasManPage       bool      `json:"has_man_page,omitempty"`      // Whether man page exists
}

// Catalog is the full list of discovered tools
type Catalog struct {
	LastScan time.Time               `json:"last_scan"` // When the last full scan occurred
	Tools    map[string]CatalogEntry `json:"tools"`     // Tool name -> entry
}

// Config holds TabGen configuration
type Config struct {
	TabGenDir    string   `json:"tabgen_dir"`    // Base directory (~/.tabgen)
	Excluded     []string `json:"excluded"`      // Tools to skip
	ScanOnStartup bool    `json:"scan_on_startup"` // Whether to scan on shell startup
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		TabGenDir:    "~/.tabgen",
		Excluded:     []string{},
		ScanOnStartup: true,
	}
}
