package parser

import (
	"strings"
	"testing"

	"github.com/justin/tabgen/internal/types"
)

func TestParseHelpOutput_GNUStyle(t *testing.T) {
	helpOutput := `Usage: mytool [OPTIONS] COMMAND [ARGS]

A sample tool for testing.

Commands:
  init        Initialize the project
  build       Build the project
  test        Run tests

Options:
  -h, --help            Show this help message
  -v, --verbose         Enable verbose output
  --config <file>       Configuration file path
  --output=FILE         Output file
`

	p := New()
	tool := &types.Tool{Name: "mytool"}
	p.parseHelpOutput(tool, helpOutput)

	// Check subcommands
	if len(tool.Subcommands) != 3 {
		t.Errorf("expected 3 subcommands, got %d", len(tool.Subcommands))
	}

	expectedCmds := map[string]string{
		"init":  "Initialize the project",
		"build": "Build the project",
		"test":  "Run tests",
	}

	for _, cmd := range tool.Subcommands {
		if expected, ok := expectedCmds[cmd.Name]; ok {
			if cmd.Description != expected {
				t.Errorf("command %s: expected description %q, got %q", cmd.Name, expected, cmd.Description)
			}
		} else {
			t.Errorf("unexpected command: %s", cmd.Name)
		}
	}

	// Check flags
	if len(tool.GlobalFlags) < 3 {
		t.Errorf("expected at least 3 global flags, got %d", len(tool.GlobalFlags))
	}
}

func TestParseHelpOutput_GitStyleIndented(t *testing.T) {
	helpOutput := `usage: git [--version] [--help] [-C <path>] <command> [<args>]

These are common Git commands used in various situations:

   clone      Clone a repository into a new directory
   init       Create an empty Git repository
   add        Add file contents to the index
   commit     Record changes to the repository
`

	p := New()
	tool := &types.Tool{Name: "git"}
	p.parseHelpOutput(tool, helpOutput)

	if len(tool.Subcommands) < 4 {
		t.Errorf("expected at least 4 subcommands, got %d", len(tool.Subcommands))
	}

	// Verify clone is detected
	found := false
	for _, cmd := range tool.Subcommands {
		if cmd.Name == "clone" {
			found = true
			if cmd.Description != "Clone a repository into a new directory" {
				t.Errorf("clone description mismatch: got %q", cmd.Description)
			}
			break
		}
	}
	if !found {
		t.Error("clone command not found")
	}
}

func TestParseCommandLine(t *testing.T) {
	tests := []struct {
		line     string
		wantName string
		wantDesc string
		wantNil  bool
	}{
		{
			line:     "  build       Build the project",
			wantName: "build",
			wantDesc: "Build the project",
		},
		{
			line:     "  test, t     Run tests",
			wantName: "test",
			wantDesc: "Run tests",
		},
		{
			line:    "  --flag      Not a command",
			wantNil: true,
		},
		{
			line:    "",
			wantNil: true,
		},
		{
			line:     "deploy        Deploy to production",
			wantName: "deploy",
			wantDesc: "Deploy to production",
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			cmd := p.parseCommandLine(tt.line)
			if tt.wantNil {
				if cmd != nil {
					t.Errorf("expected nil, got %+v", cmd)
				}
				return
			}

			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Name != tt.wantName {
				t.Errorf("name: got %q, want %q", cmd.Name, tt.wantName)
			}
			if cmd.Description != tt.wantDesc {
				t.Errorf("description: got %q, want %q", cmd.Description, tt.wantDesc)
			}
		})
	}
}

func TestParseFlagLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantName  string
		wantShort string
		wantArg   string
		wantDesc  string
		wantNil   bool
	}{
		{
			name:      "long and short flag",
			line:      "  -v, --verbose         Enable verbose output",
			wantName:  "--verbose",
			wantShort: "-v",
			wantDesc:  "Enable verbose output",
		},
		{
			name:     "long flag only",
			line:     "      --debug           Debug mode",
			wantName: "--debug",
			wantDesc: "Debug mode",
		},
		{
			name:      "short flag only",
			line:      "  -h                    Show help",
			wantName:  "-h",
			wantShort: "",
			wantDesc:  "Show help",
		},
		{
			name:     "flag with equals value",
			line:     "      --output=FILE     Output file",
			wantName: "--output",
			wantArg:  "FILE",
			wantDesc: "Output file",
		},
		{
			name:      "flag with angle bracket arg",
			line:      "  -o, --output <file>   Output file path",
			wantName:  "--output",
			wantShort: "-o",
			wantArg:   "file",
			wantDesc:  "Output file path",
		},
		{
			name:    "not a flag",
			line:    "  command     Do something",
			wantNil: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantNil: true,
		},
		{
			name:    "whitespace only",
			line:    "   ",
			wantNil: true,
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := p.parseFlagLine(tt.line)
			if tt.wantNil {
				if flag != nil {
					t.Errorf("expected nil, got %+v", flag)
				}
				return
			}

			if flag == nil {
				t.Fatal("expected flag, got nil")
			}
			if flag.Name != tt.wantName {
				t.Errorf("name: got %q, want %q", flag.Name, tt.wantName)
			}
			if flag.Short != tt.wantShort {
				t.Errorf("short: got %q, want %q", flag.Short, tt.wantShort)
			}
			if flag.Arg != tt.wantArg {
				t.Errorf("arg: got %q, want %q", flag.Arg, tt.wantArg)
			}
			if flag.Description != tt.wantDesc {
				t.Errorf("desc: got %q, want %q", flag.Description, tt.wantDesc)
			}
		})
	}
}

func TestParseIndentedCommand(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantName string
		wantDesc string
		wantNil  bool
	}{
		{
			name:     "git-style indented",
			line:     "   clone      Clone a repository into a new directory",
			wantName: "clone",
			wantDesc: "Clone a repository into a new directory",
		},
		{
			name:     "tab indented",
			line:     "\tinit       Create empty repository",
			wantName: "init",
			wantDesc: "Create empty repository",
		},
		{
			name:    "flag line, not command",
			line:    "   --verbose  Be verbose",
			wantNil: true,
		},
		{
			name:    "parenthetical, not command",
			line:    "   (see more options below)",
			wantNil: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantNil: true,
		},
		{
			name:    "no separator",
			line:    "   singlewordinvalid",
			wantNil: true,
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := p.parseIndentedCommand(tt.line)
			if tt.wantNil {
				if cmd != nil {
					t.Errorf("expected nil, got %+v", cmd)
				}
				return
			}

			if cmd == nil {
				t.Fatal("expected command, got nil")
			}
			if cmd.Name != tt.wantName {
				t.Errorf("name: got %q, want %q", cmd.Name, tt.wantName)
			}
			if cmd.Description != tt.wantDesc {
				t.Errorf("description: got %q, want %q", cmd.Description, tt.wantDesc)
			}
		})
	}
}

func TestIsValidCommandName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"init", true},
		{"build", true},
		{"my-command", true},
		{"my_command", true},
		{"cmd123", true},
		{"UPPERCASE", true},
		{"MixedCase", true},
		{"", false},
		{"with spaces", false},
		{"with.dot", false},
		{"with/slash", false},
		{"verylongcommandnamethatexceedsthirtychars", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isValidCommandName(tt.input)
			if got != tt.want {
				t.Errorf("isValidCommandName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsManSectionHeader(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"OPTIONS", true},
		{"NAME", true},
		{"DESCRIPTION", true},
		{"SEE ALSO", true},
		{"EXIT STATUS", true},
		{"EXAMPLES", true},
		{"OPTIONS ", true},
		{"Some random text", false},
		{"options", false}, // case sensitive
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isManSectionHeader(tt.input)
			if got != tt.want {
				t.Errorf("isManSectionHeader(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseManPage(t *testing.T) {
	manOutput := `NAME
       mytool - a sample tool

SYNOPSIS
       mytool [OPTIONS]

OPTIONS
       -h, --help          Show help
       --verbose           Verbose mode
       --config <file>     Configuration file

EXAMPLES
       mytool --verbose
`

	p := New()
	tool := &types.Tool{Name: "mytool"}
	p.parseManPage(tool, manOutput)

	if len(tool.GlobalFlags) < 2 {
		t.Errorf("expected at least 2 flags from man page, got %d", len(tool.GlobalFlags))
	}

	// Verify --verbose is detected
	found := false
	for _, flag := range tool.GlobalFlags {
		if flag.Name == "--verbose" {
			found = true
			break
		}
	}
	if !found {
		t.Error("--verbose flag not found in man page parse")
	}
}

func TestParseSubcommandOutput(t *testing.T) {
	subcommandHelp := `Usage: tool subcmd [OPTIONS]

Commands:
  nested1     First nested command
  nested2     Second nested command

Options:
  --sub-flag       Subcommand specific flag
  -f, --force      Force operation
`

	p := New()
	cmd := &types.Command{Name: "subcmd"}
	p.parseSubcommandOutput(cmd, subcommandHelp)

	// Check nested commands
	if len(cmd.Subcommands) != 2 {
		t.Errorf("expected 2 nested subcommands, got %d", len(cmd.Subcommands))
	}

	// Check flags
	if len(cmd.Flags) < 1 {
		t.Errorf("expected at least 1 flag, got %d", len(cmd.Flags))
	}
}

func TestParseHelpOutput_AvoidDuplicates(t *testing.T) {
	// Test that indented command detection avoids duplicates
	// The indented dedup logic only runs when NOT in Commands/Options section
	// First indented command detected, then Commands section adds same command
	// Finally another indented detection (after Options section resets inCommands)
	helpOutput := `Usage: tool [OPTIONS]

   build      Build the project (first indented detection)

Commands:
  test        Run tests

Options:
  --verbose   Be verbose

   build      Build the project (indented after Options, should dedup)
`

	p := New()
	tool := &types.Tool{Name: "tool"}
	p.parseHelpOutput(tool, helpOutput)

	// Should have build only once (first indented detection, second is deduplicated)
	buildCount := 0
	for _, cmd := range tool.Subcommands {
		if cmd.Name == "build" {
			buildCount++
		}
	}
	if buildCount != 1 {
		t.Errorf("expected 1 build command (no duplicates), got %d", buildCount)
	}

	// test command from Commands section should be present
	testFound := false
	for _, cmd := range tool.Subcommands {
		if cmd.Name == "test" {
			testFound = true
		}
	}
	if !testFound {
		t.Error("expected test command from Commands section")
	}
}

func TestParseHelpOutput_InlineFlagDedup(t *testing.T) {
	// Inline flag detection (outside Options/Flags sections) should deduplicate
	// This tests flags appearing before any section header
	helpOutput := `Usage: tool [OPTIONS]

  --verbose       First inline verbose

Some description here.

  --verbose       Second inline verbose (should be deduplicated)
`

	p := New()
	tool := &types.Tool{Name: "tool"}
	p.parseHelpOutput(tool, helpOutput)

	// Inline flag detection should deduplicate
	verboseCount := 0
	for _, flag := range tool.GlobalFlags {
		if flag.Name == "--verbose" {
			verboseCount++
		}
	}
	if verboseCount != 1 {
		t.Errorf("expected 1 --verbose flag (inline detection should dedupe), got %d", verboseCount)
	}
}

func TestParseHelpOutput_SectionHeaders(t *testing.T) {
	tests := []struct {
		name          string
		helpOutput    string
		wantCommands  int
		wantFlags     int
	}{
		{
			name: "Available commands header",
			helpOutput: `Usage: tool [command]

Available commands:
  start       Start the service
  stop        Stop the service
`,
			wantCommands: 2,
		},
		{
			name: "Subcommands header",
			helpOutput: `Usage: tool [command]

Subcommands:
  run         Run the app
  test        Test the app
`,
			wantCommands: 2,
		},
		{
			name: "Flags header",
			helpOutput: `Usage: tool [flags]

Flags:
  --debug     Enable debug mode
  --quiet     Quiet mode
`,
			wantFlags: 2,
		},
		{
			name: "Global options header",
			helpOutput: `Usage: tool [options]

Global Options:
  --config    Config file
  --env       Environment
`,
			wantFlags: 2,
		},
		{
			name: "Global flags header",
			helpOutput: `Usage: tool [options]

Global Flags:
  --verbose   Be verbose
`,
			wantFlags: 1,
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &types.Tool{Name: "tool"}
			p.parseHelpOutput(tool, tt.helpOutput)

			if tt.wantCommands > 0 && len(tool.Subcommands) != tt.wantCommands {
				t.Errorf("expected %d commands, got %d", tt.wantCommands, len(tool.Subcommands))
			}
			if tt.wantFlags > 0 && len(tool.GlobalFlags) != tt.wantFlags {
				t.Errorf("expected %d flags, got %d", tt.wantFlags, len(tool.GlobalFlags))
			}
		})
	}
}

func TestNew(t *testing.T) {
	p := New()
	if p == nil {
		t.Error("New() returned nil")
	}
}

func TestMaxSubcommandDepth(t *testing.T) {
	if MaxSubcommandDepth != 2 {
		t.Errorf("MaxSubcommandDepth should be 2, got %d", MaxSubcommandDepth)
	}
}

func TestParse_InputValidation(t *testing.T) {
	p := New()

	tests := []struct {
		name    string
		toolName string
		path    string
		wantErr string
	}{
		{
			name:    "empty name",
			toolName: "",
			path:    "/bin/ls",
			wantErr: "name cannot be empty",
		},
		{
			name:    "empty path",
			toolName: "ls",
			path:    "",
			wantErr: "path cannot be empty",
		},
		{
			name:    "non-existent path",
			toolName: "fake",
			path:    "/nonexistent/path/to/binary",
			wantErr: "path does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(tt.toolName, tt.path)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}


func TestParseFlagLine_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantName string
		wantArg  string
	}{
		{
			name:     "flag with bracket arg",
			line:     "  --format [json|yaml]   Output format",
			wantName: "--format",
			wantArg:  "json|yaml",
		},
		{
			name:     "flag multiple spaces before desc",
			line:     "  --long               Very long description here",
			wantName: "--long",
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := p.parseFlagLine(tt.line)
			if flag == nil {
				t.Fatal("expected flag, got nil")
			}
			if flag.Name != tt.wantName {
				t.Errorf("name: got %q, want %q", flag.Name, tt.wantName)
			}
			if tt.wantArg != "" && flag.Arg != tt.wantArg {
				t.Errorf("arg: got %q, want %q", flag.Arg, tt.wantArg)
			}
		})
	}
}

func TestParseSubcommandOutput_EmptyOutput(t *testing.T) {
	p := New()
	cmd := &types.Command{Name: "subcmd"}
	p.parseSubcommandOutput(cmd, "")

	if len(cmd.Subcommands) != 0 {
		t.Errorf("expected 0 subcommands from empty output, got %d", len(cmd.Subcommands))
	}
}

func TestParseSubcommandOutput_FlagsOutsideSection(t *testing.T) {
	// Flags appearing with - prefix even outside Options section
	output := `Usage: tool subcmd

  --direct-flag     A flag appearing directly

Commands:
  nested     A nested command
`
	p := New()
	cmd := &types.Command{Name: "subcmd"}
	p.parseSubcommandOutput(cmd, output)

	// Should detect the inline flag
	flagFound := false
	for _, f := range cmd.Flags {
		if f.Name == "--direct-flag" {
			flagFound = true
		}
	}
	if !flagFound {
		t.Error("expected --direct-flag to be detected")
	}
}

func TestParseSubcommandOutput_IndentedSubcommands(t *testing.T) {
	// Git-style indented subcommands
	output := `Usage: tool subcmd

Some description.

   action1     First action
   action2     Second action
`
	p := New()
	cmd := &types.Command{Name: "subcmd"}
	p.parseSubcommandOutput(cmd, output)

	if len(cmd.Subcommands) != 2 {
		t.Errorf("expected 2 indented subcommands, got %d", len(cmd.Subcommands))
	}
}

func TestParseSubcommandOutput_DuplicateFlags(t *testing.T) {
	output := `Options:
  --force     Force it

Some text

  --force     Force again (duplicate)
`
	p := New()
	cmd := &types.Command{Name: "subcmd"}
	p.parseSubcommandOutput(cmd, output)

	// Should deduplicate flags
	forceCount := 0
	for _, f := range cmd.Flags {
		if f.Name == "--force" {
			forceCount++
		}
	}
	if forceCount != 1 {
		t.Errorf("expected 1 --force flag (deduplicated), got %d", forceCount)
	}
}

func TestParseSubcommandOutput_DuplicateSubcommands(t *testing.T) {
	output := `Usage: tool subcmd

   action     First detection

   action     Duplicate (should be deduped)
`
	p := New()
	cmd := &types.Command{Name: "subcmd"}
	p.parseSubcommandOutput(cmd, output)

	actionCount := 0
	for _, c := range cmd.Subcommands {
		if c.Name == "action" {
			actionCount++
		}
	}
	if actionCount != 1 {
		t.Errorf("expected 1 action subcommand (deduplicated), got %d", actionCount)
	}
}

func TestParseManPage_EndOfSection(t *testing.T) {
	// Test that parsing stops when a new section starts
	manOutput := `OPTIONS
       --flag1    First flag

EXAMPLES
       Example usage here
       --notaflag    This looks like a flag but is in EXAMPLES section
`
	p := New()
	tool := &types.Tool{Name: "tool"}
	p.parseManPage(tool, manOutput)

	// Should only have flag1, not "notaflag" from EXAMPLES
	if len(tool.GlobalFlags) != 1 {
		t.Errorf("expected 1 flag (from OPTIONS only), got %d", len(tool.GlobalFlags))
	}
}

func TestParseManPage_FlagContinuation(t *testing.T) {
	// Test description continuation for flags
	manOutput := `OPTIONS
       --verbose
              Enable verbose mode with detailed output
`
	p := New()
	tool := &types.Tool{Name: "tool"}
	p.parseManPage(tool, manOutput)

	if len(tool.GlobalFlags) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(tool.GlobalFlags))
	}
	if tool.GlobalFlags[0].Description != "Enable verbose mode with detailed output" {
		t.Errorf("expected description from continuation, got %q", tool.GlobalFlags[0].Description)
	}
}

func TestParseCommandLine_ShortAlias(t *testing.T) {
	// Test "command, c" format with just alias
	p := New()
	cmd := p.parseCommandLine("  b, build       Build it")
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	// The code takes the first part before comma
	if cmd.Name != "b" {
		t.Errorf("expected name 'b', got %q", cmd.Name)
	}
}

func TestParseIndentedCommand_LongName(t *testing.T) {
	// Command name at max length limit
	p := New()
	// 30 chars is the max
	cmd := p.parseIndentedCommand("   aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  Some description")
	if cmd == nil {
		t.Fatal("expected command with 30-char name")
	}

	// 31 chars should fail
	cmd = p.parseIndentedCommand("   aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  Some description")
	if cmd != nil {
		t.Error("expected nil for 31-char command name (over limit)")
	}
}
