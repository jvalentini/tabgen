package generator

import (
	"fmt"

	"github.com/jvalentini/tabgen/internal/types"
)

// Limits for generated completion scripts
const (
	// MaxOutputSize is the maximum size of a generated script in bytes (1MB)
	MaxOutputSize = 1 * 1024 * 1024

	// MaxSubcommands is the maximum number of subcommands to include
	MaxSubcommands = 500

	// MaxFlags is the maximum number of flags per command
	MaxFlags = 200

	// MaxTotalItems is the maximum total items (subcommands + flags) in a tool
	MaxTotalItems = 2000
)

// GenerateResult holds the generated script and any warnings
type GenerateResult struct {
	Script   string   // The generated completion script
	Warnings []string // Any truncation or limit warnings
}

// countItems recursively counts all subcommands and flags in a tool
func countItems(tool *types.Tool) (subcommands int, flags int) {
	flags = len(tool.GlobalFlags)
	for _, cmd := range tool.Subcommands {
		s, f := countCommandItems(cmd)
		subcommands += 1 + s // Count this command plus nested
		flags += f
	}
	return
}

// countCommandItems recursively counts subcommands and flags in a command
func countCommandItems(cmd types.Command) (subcommands int, flags int) {
	flags = len(cmd.Flags)
	for _, sub := range cmd.Subcommands {
		s, f := countCommandItems(sub)
		subcommands += 1 + s
		flags += f
	}
	return
}

// truncateTool creates a copy of the tool with truncated subcommands/flags
// Returns the truncated tool and any warnings generated
func truncateTool(tool *types.Tool) (*types.Tool, []string) {
	var warnings []string

	// Count original items
	origSubs, origFlags := countItems(tool)
	totalItems := origSubs + origFlags

	// Check if truncation is needed
	needsTruncation := origSubs > MaxSubcommands ||
		len(tool.GlobalFlags) > MaxFlags ||
		totalItems > MaxTotalItems

	if !needsTruncation {
		return tool, nil
	}

	// Create truncated copy
	truncated := &types.Tool{
		Name:        tool.Name,
		Path:        tool.Path,
		Version:     tool.Version,
		ParsedAt:    tool.ParsedAt,
		Source:      tool.Source,
		GlobalFlags: tool.GlobalFlags,
		Subcommands: tool.Subcommands,
	}

	// Truncate global flags if needed
	if len(truncated.GlobalFlags) > MaxFlags {
		warnings = append(warnings, fmt.Sprintf(
			"truncated global flags from %d to %d",
			len(truncated.GlobalFlags), MaxFlags))
		truncated.GlobalFlags = truncated.GlobalFlags[:MaxFlags]
	}

	// Truncate subcommands if needed
	if len(truncated.Subcommands) > MaxSubcommands {
		warnings = append(warnings, fmt.Sprintf(
			"truncated subcommands from %d to %d",
			len(truncated.Subcommands), MaxSubcommands))
		truncated.Subcommands = truncated.Subcommands[:MaxSubcommands]
	}

	// Truncate flags within each subcommand
	truncated.Subcommands = truncateSubcommandFlags(truncated.Subcommands, &warnings)

	// Final count check
	finalSubs, finalFlags := countItems(truncated)
	if finalSubs+finalFlags > MaxTotalItems {
		warnings = append(warnings, fmt.Sprintf(
			"tool still has %d items after truncation (max %d)",
			finalSubs+finalFlags, MaxTotalItems))
	}

	return truncated, warnings
}

// truncateSubcommandFlags truncates flags in subcommands recursively
func truncateSubcommandFlags(cmds []types.Command, warnings *[]string) []types.Command {
	result := make([]types.Command, len(cmds))
	for i, cmd := range cmds {
		result[i] = cmd
		if len(cmd.Flags) > MaxFlags {
			*warnings = append(*warnings, fmt.Sprintf(
				"truncated flags for '%s' from %d to %d",
				cmd.Name, len(cmd.Flags), MaxFlags))
			result[i].Flags = cmd.Flags[:MaxFlags]
		}
		if len(cmd.Subcommands) > 0 {
			result[i].Subcommands = truncateSubcommandFlags(cmd.Subcommands, warnings)
		}
	}
	return result
}

// checkOutputSize checks if the generated script exceeds size limits
func checkOutputSize(script string, toolName string) (string, []string) {
	var warnings []string

	if len(script) > MaxOutputSize {
		warnings = append(warnings, fmt.Sprintf(
			"generated script for '%s' exceeds %d bytes (%d bytes), truncating",
			toolName, MaxOutputSize, len(script)))
		// Truncate to max size, trying to end at a newline
		truncated := script[:MaxOutputSize]
		if lastNL := lastNewline(truncated); lastNL > MaxOutputSize/2 {
			truncated = truncated[:lastNL+1]
		}
		// Add truncation comment
		truncated += "\n# WARNING: Script truncated due to size limits\n"
		return truncated, warnings
	}

	return script, warnings
}

// lastNewline finds the last newline index in a string
func lastNewline(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '\n' {
			return i
		}
	}
	return -1
}
