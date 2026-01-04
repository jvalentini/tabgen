package parser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jvalentini/tabgen/internal/config"
	"github.com/jvalentini/tabgen/internal/types"
)

// ParserConfig holds parser configuration options
type ParserConfig struct {
	// MaxDepth limits how deep we recurse into subcommands (default: 2)
	MaxDepth int
	// HelpTimeout is the timeout for running help/version commands (default: 5s)
	HelpTimeout time.Duration
	// VersionCmds are the flags to try when detecting version (default: --version, -V, version, -v)
	VersionCmds []string
}

// DefaultConfig returns a ParserConfig with sensible defaults
func DefaultConfig() ParserConfig {
	return ParserConfig{
		MaxDepth:    2,
		HelpTimeout: 5 * time.Second,
		VersionCmds: []string{"--version", "-V", "version", "-v"},
	}
}

// Parser extracts command structure from --help and man pages
type Parser struct {
	config ParserConfig
}

// New creates a new Parser with optional config. If no config provided, uses defaults.
func New(cfg ...ParserConfig) *Parser {
	var parserConfig ParserConfig
	if len(cfg) > 0 {
		parserConfig = cfg[0]
	} else {
		parserConfig = DefaultConfig()
	}
	// Apply defaults for zero values
	if parserConfig.MaxDepth == 0 {
		parserConfig.MaxDepth = 2
	}
	if parserConfig.HelpTimeout == 0 {
		parserConfig.HelpTimeout = 5 * time.Second
	}
	if len(parserConfig.VersionCmds) == 0 {
		parserConfig.VersionCmds = []string{"--version", "-V", "version", "-v"}
	}
	return &Parser{config: parserConfig}
}

// Config returns the parser's current configuration
func (p *Parser) Config() ParserConfig {
	return p.config
}

// UniqueSet provides O(1) duplicate detection for any slice type
type UniqueSet[T any] struct {
	seen  map[string]bool
	items *[]T
	key   func(T) string
}

// NewUniqueSet creates a UniqueSet wrapping an existing slice
func NewUniqueSet[T any](items *[]T, key func(T) string) *UniqueSet[T] {
	seen := make(map[string]bool, len(*items))
	for _, item := range *items {
		seen[key(item)] = true
	}
	return &UniqueSet[T]{seen: seen, items: items, key: key}
}

// Add appends item if not already present (O(1) lookup), returns true if added
func (s *UniqueSet[T]) Add(item T) bool {
	k := s.key(item)
	if s.seen[k] {
		return false
	}
	s.seen[k] = true
	*s.items = append(*s.items, item)
	return true
}

// newFlagSet creates a UniqueSet for flags
func newFlagSet(flags *[]types.Flag) *UniqueSet[types.Flag] {
	return NewUniqueSet(flags, func(f types.Flag) string { return f.Name })
}

// newCommandSet creates a UniqueSet for commands
func newCommandSet(commands *[]types.Command) *UniqueSet[types.Command] {
	return NewUniqueSet(commands, func(c types.Command) string { return c.Name })
}

// MaxSubcommandDepth is kept for backward compatibility
// Deprecated: Use Config().MaxDepth instead
const MaxSubcommandDepth = 2

// Parse extracts command structure from a tool
func (p *Parser) Parse(name, path string) (*types.Tool, error) {
	// Validate inputs
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	if path == "" {
		return nil, errors.New("path cannot be empty")
	}

	// Check path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}
		return nil, fmt.Errorf("cannot access path %s: %w", path, err)
	}

	// Check path is executable
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not an executable: %s", path)
	}
	if info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("path is not executable: %s", path)
	}

	config.LogSection("Parsing " + name)
	config.Logf("Path: %s", path)

	tool := &types.Tool{
		Name:     name,
		Path:     path,
		ParsedAt: time.Now(),
	}

	// Detect version
	tool.Version = p.detectVersion(path)
	if tool.Version != "" {
		config.Logf("Detected version: %s", tool.Version)
	} else {
		config.Logf("No version detected")
	}

	// Try --help first
	config.Logf("Running: %s --help", path)
	helpOutput, helpErr := p.runHelp(path)
	if helpErr != nil {
		config.Logf("--help error: %v", helpErr)
		// Distinguish permission errors from "no help available"
		if isPermissionError(helpErr) {
			return nil, fmt.Errorf("cannot run %s --help: %w", path, helpErr)
		}
		// Other errors (e.g., tool has no help) are acceptable, continue
	}

	if helpOutput != "" {
		config.Logf("--help output: %d bytes", len(helpOutput))
		config.LogSnippet("--help output", helpOutput, 20)
	} else {
		config.Logf("--help returned no output")
	}

	// Try man page as fallback or supplement
	config.Logf("Checking man page for: %s", name)
	manOutput, manErr := p.getManPage(name)
	if manErr != nil {
		config.Logf("man page error: %v", manErr)
		// Permission errors on man page are less critical but worth noting
		if isPermissionError(manErr) {
			// Log but don't fail - man pages are optional
			tool.Source = "help-only"
		}
		// Other errors (no man page) are acceptable
	} else if manOutput != "" {
		config.Logf("man page output: %d bytes", len(manOutput))
	}

	// Parse what we got
	if helpOutput != "" {
		tool.Source = "help"
		config.Logf("Parsing --help output...")
		p.parseHelpOutput(tool, helpOutput)
		config.Logf("Found %d subcommands, %d global flags from --help",
			len(tool.Subcommands), len(tool.GlobalFlags))
	}

	if manOutput != "" {
		if tool.Source == "" {
			tool.Source = "man"
		} else {
			tool.Source = "both"
		}
		config.Logf("Parsing man page...")
		p.parseManPage(tool, manOutput)
		config.Logf("Total flags after man page: %d", len(tool.GlobalFlags))
	}

	if tool.Source == "" {
		tool.Source = "none"
		config.Logf("No help or man page found - tool unparseable")
	}

	// Parse nested subcommands (depth-limited)
	if len(tool.Subcommands) > 0 {
		config.Logf("Parsing nested subcommands (max depth: %d)...", MaxSubcommandDepth)
		p.parseNestedSubcommands(path, tool.Subcommands, 1)
	}

	config.Logf("Parse complete: source=%s, subcommands=%d, flags=%d",
		tool.Source, len(tool.Subcommands), len(tool.GlobalFlags))

	return tool, nil
}

// parseNestedSubcommands recursively parses subcommand help
func (p *Parser) parseNestedSubcommands(basePath string, commands []types.Command, depth int) {
	if depth >= p.config.MaxDepth {
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
	ctx, cancel := context.WithTimeout(context.Background(), p.config.HelpTimeout)
	defer cancel()

	// Split base path in case it contains spaces (nested commands)
	parts := strings.Fields(basePath)
	args := append(parts[1:], subcommand, "--help")

	cmd := exec.CommandContext(ctx, parts[0], args...)
	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		// Try without --help (some tools use "help subcommand")
		args = append(parts[1:], "help", subcommand)
		cmd = exec.CommandContext(ctx, parts[0], args...)
		output, _ = cmd.CombinedOutput()
	}
	return string(output)
}

// parseSubcommandOutput extracts flags and nested subcommands from help output
func (p *Parser) parseSubcommandOutput(cmd *types.Command, output string) {
	lines := strings.Split(output, "\n")

	// Use sets for O(1) duplicate detection
	flagSet := newFlagSet(&cmd.Flags)
	cmdSet := newCommandSet(&cmd.Subcommands)

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
				cmdSet.Add(*subcmd)
			}
		}

		// Parse flags
		if inOptions || strings.HasPrefix(trimmed, "-") {
			if flag := p.parseFlagLine(line); flag != nil {
				flagSet.Add(*flag)
			}
		}

		// Look for indented commands (git-style)
		if !inCommands && !inOptions && len(line) > 3 && (line[0] == ' ' || line[0] == '\t') {
			if subcmd := p.parseIndentedCommand(line); subcmd != nil {
				cmdSet.Add(*subcmd)
			}
		}
	}
}

// runHelp executes tool --help and captures output
func (p *Parser) runHelp(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.HelpTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Many tools return non-zero for --help, still use output
		if len(output) > 0 {
			return string(output), nil
		}
		// Try -h as fallback
		cmd = exec.CommandContext(ctx, path, "-h")
		output, _ = cmd.CombinedOutput()
	}
	return string(output), nil
}

// getManPage retrieves the man page content
func (p *Parser) getManPage(name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.HelpTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "man", name)
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

	// Use sets for O(1) duplicate detection
	flagSet := newFlagSet(&tool.GlobalFlags)
	cmdSet := newCommandSet(&tool.Subcommands)

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
			config.Logf("Detected COMMANDS section: %q", trimmed)
			inCommands = true
			inOptions = false
			continue
		}

		if strings.HasPrefix(lower, "options:") ||
			strings.HasPrefix(lower, "flags:") ||
			strings.HasPrefix(lower, "global options:") ||
			strings.HasPrefix(lower, "global flags:") ||
			lower == "options" || lower == "flags" {
			config.Logf("Detected OPTIONS section: %q", trimmed)
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
				cmdSet.Add(*cmd)
			}
		}

		// Parse options/flags
		if inOptions {
			if flag := p.parseFlagLine(line); flag != nil {
				flagSet.Add(*flag)
			}
		}

		// Also look for inline flags anywhere (lines starting with -)
		if !inOptions && strings.HasPrefix(strings.TrimSpace(line), "-") {
			if flag := p.parseFlagLine(line); flag != nil {
				flagSet.Add(*flag)
			}
		}

		// Look for git-style indented commands (3+ spaces, then word, then description)
		// Pattern: "   clone     Clone a repository..."
		if !inCommands && !inOptions && len(line) > 3 && (line[0] == ' ' || line[0] == '\t') {
			if cmd := p.parseIndentedCommand(line); cmd != nil {
				cmdSet.Add(*cmd)
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

	// Handle "command, c" or "c, command" format - extract name and aliases
	var primaryName string
	var aliases []string

	if strings.Contains(cmdPart, ",") {
		// Split on comma and collect all valid names
		nameParts := strings.Split(cmdPart, ",")
		var validNames []string
		for _, np := range nameParts {
			np = strings.TrimSpace(np)
			if isValidCommandName(np) {
				validNames = append(validNames, np)
			}
		}
		if len(validNames) == 0 {
			return nil
		}
		// Use the longest name as primary, rest as aliases
		primaryName = validNames[0]
		for _, n := range validNames[1:] {
			if len(n) > len(primaryName) {
				aliases = append(aliases, primaryName)
				primaryName = n
			} else {
				aliases = append(aliases, n)
			}
		}
	} else {
		// Validate: should be a simple word
		if !isValidCommandName(cmdPart) {
			return nil
		}
		primaryName = cmdPart
	}

	cmd := &types.Command{
		Name:    primaryName,
		Aliases: aliases,
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
	//   --format=json|yaml  Description
	//   --format {json,yaml} Description

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
	for token := range strings.FieldsSeq(flagPart) {
		token = strings.TrimSuffix(token, ",")

		if strings.HasPrefix(token, "--") {
			// Long flag
			name := token
			// Handle --flag=VALUE or --flag=val1|val2
			if idx := strings.Index(name, "="); idx > 0 {
				argPart := name[idx+1:]
				name = name[:idx]

				// Check for pipe-separated values (json|yaml|text)
				argPart = strings.Trim(argPart, "<>[](){}")
				if strings.Contains(argPart, "|") {
					values := strings.Split(argPart, "|")
					for i, v := range values {
						values[i] = strings.TrimSpace(v)
					}
					flag.ArgumentValues = values
					if len(values) > 0 {
						flag.Arg = "value"
					}
				} else {
					flag.Arg = argPart
				}
			}
			flag.Name = name
		} else if strings.HasPrefix(token, "-") && len(token) == 2 {
			// Short flag
			flag.Short = token
		} else if strings.HasPrefix(token, "<") || strings.HasPrefix(token, "[") {
			// Argument placeholder, may contain choices
			argContent := strings.Trim(token, "<>[]")
			if strings.Contains(argContent, "|") {
				values := strings.Split(argContent, "|")
				for i, v := range values {
					values[i] = strings.TrimSpace(v)
				}
				flag.ArgumentValues = values
				flag.Arg = "value"
			} else {
				flag.Arg = argContent
			}
		} else if strings.HasPrefix(token, "{") || strings.HasPrefix(token, "(") {
			// Choices in braces: {json,yaml} or (json|yaml)
			content := strings.Trim(token, "{}()")
			var values []string
			if strings.Contains(content, "|") {
				values = strings.Split(content, "|")
			} else if strings.Contains(content, ",") {
				values = strings.Split(content, ",")
			}
			if len(values) > 0 {
				for i, v := range values {
					values[i] = strings.TrimSpace(v)
				}
				flag.ArgumentValues = values
				flag.Arg = "value"
			}
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

	// Use set for O(1) duplicate detection
	flagSet := newFlagSet(&tool.GlobalFlags)

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
				prevLen := len(tool.GlobalFlags)
				flagSet.Add(*flag)
				if len(tool.GlobalFlags) > prevLen {
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
