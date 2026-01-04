package parser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/justin/tabgen/internal/types"
)

// Parser extracts command structure from --help and man pages
type Parser struct{}

// New creates a new Parser
func New() *Parser {
	return &Parser{}
}

// MaxSubcommandDepth limits how deep we recurse into subcommands
const MaxSubcommandDepth = 2

// Parse extracts command structure from a tool
func (p *Parser) Parse(name, path string) (*types.Tool, error) {
	tool := &types.Tool{
		Name:     name,
		Path:     path,
		ParsedAt: time.Now(),
	}

	// Detect version
	tool.Version = DetectVersion(path)

	// Try --help first
	helpOutput, helpErr := p.runHelp(path)
	if helpErr != nil {
		// Distinguish permission errors from "no help available"
		if isPermissionError(helpErr) {
			return nil, fmt.Errorf("cannot run %s --help: %w", path, helpErr)
		}
		// Other errors (e.g., tool has no help) are acceptable, continue
	}

	// Try man page as fallback or supplement
	manOutput, manErr := p.getManPage(name)
	if manErr != nil {
		// Permission errors on man page are less critical but worth noting
		if isPermissionError(manErr) {
			// Log but don't fail - man pages are optional
			tool.Source = "help-only"
		}
		// Other errors (no man page) are acceptable
	}

	// Parse what we got
	if helpOutput != "" {
		tool.Source = "help"
		p.parseHelpOutput(tool, helpOutput)
	}

	if manOutput != "" {
		if tool.Source == "" {
			tool.Source = "man"
		} else {
			tool.Source = "both"
		}
		p.parseManPage(tool, manOutput)
	}

	if tool.Source == "" {
		tool.Source = "none"
	}

	// Parse nested subcommands (depth-limited)
	p.parseNestedSubcommands(path, tool.Subcommands, 1)

	return tool, nil
}

// parseNestedSubcommands recursively parses subcommand help
func (p *Parser) parseNestedSubcommands(basePath string, commands []types.Command, depth int) {
	if depth >= MaxSubcommandDepth {
		return
	}

	for i := range commands {
		cmd := &commands[i]

		// Try to get help for this subcommand
		output := p.runSubcommandHelp(basePath, cmd.Name)
		if output == "" {
			continue
		}

		// Parse flags and nested subcommands from output
		p.parseSubcommandOutput(cmd, output)

		// Recurse into nested subcommands
		if len(cmd.Subcommands) > 0 {
			// For nested commands, we need to pass the full command path
			p.parseNestedSubcommands(basePath+" "+cmd.Name, cmd.Subcommands, depth+1)
		}
	}
}

// runSubcommandHelp runs "tool subcommand --help"
func (p *Parser) runSubcommandHelp(basePath, subcommand string) string {
	// Split base path in case it contains spaces (nested commands)
	parts := strings.Fields(basePath)
	args := append(parts[1:], subcommand, "--help")

	cmd := exec.Command(parts[0], args...)
	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		// Try without --help (some tools use "help subcommand")
		args = append(parts[1:], "help", subcommand)
		cmd = exec.Command(parts[0], args...)
		output, _ = cmd.CombinedOutput()
	}
	return string(output)
}

// parseSubcommandOutput extracts flags and nested subcommands from help output
func (p *Parser) parseSubcommandOutput(cmd *types.Command, output string) {
	lines := strings.Split(output, "\n")

	inCommands := false
	inOptions := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		// Detect section headers
		if strings.HasPrefix(lower, "commands:") ||
			strings.HasPrefix(lower, "available commands:") ||
			strings.HasPrefix(lower, "subcommands:") {
			inCommands = true
			inOptions = false
			continue
		}

		if strings.HasPrefix(lower, "options:") ||
			strings.HasPrefix(lower, "flags:") {
			inCommands = false
			inOptions = true
			continue
		}

		if trimmed == "" {
			continue
		}

		// Parse nested subcommands
		if inCommands {
			if subcmd := p.parseCommandLine(line); subcmd != nil {
				cmd.Subcommands = append(cmd.Subcommands, *subcmd)
			}
		}

		// Parse flags
		if inOptions || strings.HasPrefix(trimmed, "-") {
			if flag := p.parseFlagLine(line); flag != nil {
				// Avoid duplicates
				exists := false
				for _, f := range cmd.Flags {
					if f.Name == flag.Name {
						exists = true
						break
					}
				}
				if !exists {
					cmd.Flags = append(cmd.Flags, *flag)
				}
			}
		}

		// Look for indented commands (git-style)
		if !inCommands && !inOptions && len(line) > 3 && (line[0] == ' ' || line[0] == '\t') {
			if subcmd := p.parseIndentedCommand(line); subcmd != nil {
				exists := false
				for _, c := range cmd.Subcommands {
					if c.Name == subcmd.Name {
						exists = true
						break
					}
				}
				if !exists {
					cmd.Subcommands = append(cmd.Subcommands, *subcmd)
				}
			}
		}
	}
}

// runHelp executes tool --help and captures output
func (p *Parser) runHelp(path string) (string, error) {
	cmd := exec.Command(path, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Many tools return non-zero for --help, still use output
		if len(output) > 0 {
			return string(output), nil
		}
		// Try -h as fallback
		cmd = exec.Command(path, "-h")
		output, _ = cmd.CombinedOutput()
	}
	return string(output), nil
}

// getManPage retrieves the man page content
func (p *Parser) getManPage(name string) (string, error) {
	cmd := exec.Command("man", name)
	cmd.Env = []string{"MANWIDTH=120", "LC_ALL=C"}
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// parseHelpOutput extracts structure from --help output
func (p *Parser) parseHelpOutput(tool *types.Tool, output string) {
	lines := strings.Split(output, "\n")

	inCommands := false
	inOptions := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		// Detect section headers
		if strings.HasPrefix(lower, "commands:") ||
			strings.HasPrefix(lower, "available commands:") ||
			strings.HasPrefix(lower, "subcommands:") ||
			lower == "commands" {
			inCommands = true
			inOptions = false
			continue
		}

		if strings.HasPrefix(lower, "options:") ||
			strings.HasPrefix(lower, "flags:") ||
			strings.HasPrefix(lower, "global options:") ||
			strings.HasPrefix(lower, "global flags:") ||
			lower == "options" || lower == "flags" {
			inCommands = false
			inOptions = true
			continue
		}

		// Empty line might end a section
		if trimmed == "" {
			continue
		}

		// Parse commands
		if inCommands {
			if cmd := p.parseCommandLine(line); cmd != nil {
				tool.Subcommands = append(tool.Subcommands, *cmd)
			}
		}

		// Parse options/flags
		if inOptions {
			if flag := p.parseFlagLine(line); flag != nil {
				tool.GlobalFlags = append(tool.GlobalFlags, *flag)
			}
		}

		// Also look for inline flags anywhere (lines starting with -)
		if !inOptions && strings.HasPrefix(strings.TrimSpace(line), "-") {
			if flag := p.parseFlagLine(line); flag != nil {
				// Avoid duplicates
				exists := false
				for _, f := range tool.GlobalFlags {
					if f.Name == flag.Name {
						exists = true
						break
					}
				}
				if !exists {
					tool.GlobalFlags = append(tool.GlobalFlags, *flag)
				}
			}
		}

		// Look for git-style indented commands (3+ spaces, then word, then description)
		// Pattern: "   clone     Clone a repository..."
		if !inCommands && !inOptions && len(line) > 3 && (line[0] == ' ' || line[0] == '\t') {
			if cmd := p.parseIndentedCommand(line); cmd != nil {
				// Avoid duplicates
				exists := false
				for _, c := range tool.Subcommands {
					if c.Name == cmd.Name {
						exists = true
						break
					}
				}
				if !exists {
					tool.Subcommands = append(tool.Subcommands, *cmd)
				}
			}
		}
	}
}

// parseIndentedCommand parses git-style indented command lines
// e.g., "   clone     Clone a repository into a new directory"
func (p *Parser) parseIndentedCommand(line string) *types.Command {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	// Skip if it looks like a flag or continuation
	if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "(") {
		return nil
	}

	// Look for pattern: word + multiple spaces + description
	// Use regex-like split on 2+ spaces
	parts := strings.SplitN(trimmed, "  ", 2)
	if len(parts) < 2 {
		return nil
	}

	cmdName := strings.TrimSpace(parts[0])
	desc := strings.TrimSpace(parts[1])

	// Validate command name: lowercase letters, numbers, hyphens
	if !isValidCommandName(cmdName) {
		return nil
	}

	// Description should start with uppercase (sentence) to filter out false positives
	if len(desc) == 0 {
		return nil
	}

	return &types.Command{
		Name:        cmdName,
		Description: desc,
	}
}

// parseCommandLine extracts a command from a help line
func (p *Parser) parseCommandLine(line string) *types.Command {
	// Common patterns:
	//   command     Description here
	//   command, c  Description here
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	// Skip if it looks like a flag
	if strings.HasPrefix(trimmed, "-") {
		return nil
	}

	// Split on multiple spaces (command name vs description)
	parts := strings.SplitN(trimmed, "  ", 2)
	if len(parts) == 0 {
		return nil
	}

	cmdPart := strings.TrimSpace(parts[0])
	// Handle "command, c" format
	if idx := strings.Index(cmdPart, ","); idx > 0 {
		cmdPart = strings.TrimSpace(cmdPart[:idx])
	}

	// Validate: should be a simple word
	if !isValidCommandName(cmdPart) {
		return nil
	}

	cmd := &types.Command{
		Name: cmdPart,
	}
	if len(parts) > 1 {
		cmd.Description = strings.TrimSpace(parts[1])
	}

	return cmd
}

// parseFlagLine extracts a flag from a help line
func (p *Parser) parseFlagLine(line string) *types.Flag {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	// Common patterns:
	//   -f, --flag          Description
	//   --flag              Description
	//   -f                  Description
	//   --flag=VALUE        Description
	//   --flag <value>      Description

	if !strings.HasPrefix(trimmed, "-") {
		return nil
	}

	flag := &types.Flag{}

	// Split on multiple spaces to separate flag from description
	parts := strings.SplitN(trimmed, "  ", 2)
	flagPart := parts[0]
	if len(parts) > 1 {
		flag.Description = strings.TrimSpace(parts[1])
	}

	// Parse the flag part
	tokens := strings.Fields(flagPart)
	for _, token := range tokens {
		token = strings.TrimSuffix(token, ",")

		if strings.HasPrefix(token, "--") {
			// Long flag
			name := token
			// Handle --flag=VALUE
			if idx := strings.Index(name, "="); idx > 0 {
				flag.Arg = strings.Trim(name[idx+1:], "<>[]")
				name = name[:idx]
			}
			flag.Name = name
		} else if strings.HasPrefix(token, "-") && len(token) == 2 {
			// Short flag
			flag.Short = token
		} else if strings.HasPrefix(token, "<") || strings.HasPrefix(token, "[") {
			// Argument placeholder
			flag.Arg = strings.Trim(token, "<>[]")
		}
	}

	// Need at least a long or short name
	if flag.Name == "" && flag.Short == "" {
		return nil
	}

	// If only short, promote it to name
	if flag.Name == "" {
		flag.Name = flag.Short
		flag.Short = ""
	}

	return flag
}

// parseManPage extracts structure from man page output
func (p *Parser) parseManPage(tool *types.Tool, output string) {
	lines := strings.Split(output, "\n")

	inOptions := false
	var currentFlag *types.Flag

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect OPTIONS section
		if trimmed == "OPTIONS" || strings.HasPrefix(trimmed, "OPTIONS") {
			inOptions = true
			continue
		}

		// Detect end of OPTIONS (next major section)
		if inOptions && len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			if isManSectionHeader(trimmed) {
				inOptions = false
				continue
			}
		}

		if !inOptions {
			continue
		}

		// In OPTIONS section, look for flag definitions
		// Man pages typically have flags at a certain indentation
		if strings.HasPrefix(trimmed, "-") {
			if flag := p.parseFlagLine(line); flag != nil {
				// Check for duplicate
				exists := false
				for _, f := range tool.GlobalFlags {
					if f.Name == flag.Name {
						exists = true
						break
					}
				}
				if !exists {
					tool.GlobalFlags = append(tool.GlobalFlags, *flag)
					currentFlag = &tool.GlobalFlags[len(tool.GlobalFlags)-1]
				}
			}
		} else if currentFlag != nil && trimmed != "" && currentFlag.Description == "" {
			// Continuation of description
			currentFlag.Description = trimmed
		}
	}
}

// isValidCommandName checks if a string looks like a valid command name
func isValidCommandName(s string) bool {
	if s == "" || len(s) > 30 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// isManSectionHeader checks if a line is a man page section header
func isManSectionHeader(s string) bool {
	headers := []string{
		"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "ARGUMENTS",
		"COMMANDS", "EXIT STATUS", "ENVIRONMENT", "FILES",
		"EXAMPLES", "SEE ALSO", "BUGS", "AUTHOR", "AUTHORS",
		"HISTORY", "NOTES", "CAVEATS", "DIAGNOSTICS",
	}
	for _, h := range headers {
		if s == h || strings.HasPrefix(s, h+" ") {
			return true
		}
	}
	return false
}

// isPermissionError checks if an error is a permission-related error
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	// Check for os.ErrPermission
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	// Check for exec errors that indicate permission issues
	if errors.Is(err, exec.ErrNotFound) {
		return false // Not found is not a permission error
	}
	// Check error message for common permission indicators
	errStr := err.Error()
	return strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "EACCES") ||
		strings.Contains(errStr, "operation not permitted")
}
