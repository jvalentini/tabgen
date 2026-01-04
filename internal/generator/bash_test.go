package generator

import (
	"strings"
	"testing"

	"github.com/justin/tabgen/internal/types"
)

func TestBash_Generate_Basic(t *testing.T) {
	b := NewBash()
	tool := &types.Tool{
		Name: "mytool",
		GlobalFlags: []types.Flag{
			{Name: "--verbose", Short: "-v", Description: "Enable verbose"},
		},
		Subcommands: []types.Command{
			{Name: "init", Description: "Initialize project"},
		},
	}

	output := b.Generate(tool)

	if !strings.Contains(output, "# Bash completion for mytool") {
		t.Error("expected bash completion header")
	}
	if !strings.Contains(output, "_tabgen_mytool") {
		t.Error("expected function name")
	}
	if !strings.Contains(output, "complete -o default -o bashdefault") {
		t.Error("expected complete command")
	}
}

func TestBash_Generate_WithArgumentValues(t *testing.T) {
	b := NewBash()
	tool := &types.Tool{
		Name: "cli",
		GlobalFlags: []types.Flag{
			{
				Name:           "--format",
				Short:          "-f",
				Arg:            "type",
				ArgumentValues: []string{"json", "yaml", "xml"},
				Description:    "Output format",
			},
		},
	}

	output := b.Generate(tool)

	// Check that the completion includes case statement for flag values
	if !strings.Contains(output, "case \"$prev\" in") {
		t.Error("expected case statement for prev")
	}
	if !strings.Contains(output, "json yaml xml") {
		t.Error("expected argument values in bash completion")
	}
}

func TestBash_GenerateFlagValueCompletions(t *testing.T) {
	b := NewBash()
	var sb strings.Builder

	globalFlags := []types.Flag{
		{
			Name:           "--format",
			Short:          "-f",
			ArgumentValues: []string{"json", "yaml"},
		},
		{
			Name: "--verbose", // No argument values
		},
	}

	subcommands := []types.Command{
		{
			Name: "sub",
			Flags: []types.Flag{
				{
					Name:           "--level",
					ArgumentValues: []string{"debug", "info", "warn"},
				},
			},
		},
	}

	b.generateFlagValueCompletions(&sb, globalFlags, subcommands)

	output := sb.String()

	// Should have case statement
	if !strings.Contains(output, "case \"$prev\" in") {
		t.Error("expected case statement")
	}

	// Should include --format and -f values
	if !strings.Contains(output, "json yaml") {
		t.Error("expected json yaml values")
	}

	// Should include --level values
	if !strings.Contains(output, "debug info warn") {
		t.Error("expected debug info warn values")
	}
}

func TestBash_GenerateFlagValueCompletions_Empty(t *testing.T) {
	b := NewBash()
	var sb strings.Builder

	// No flags with argument values
	globalFlags := []types.Flag{
		{Name: "--verbose"},
	}

	b.generateFlagValueCompletions(&sb, globalFlags, nil)

	output := sb.String()

	// Should be empty when no argument values
	if output != "" {
		t.Errorf("expected empty output, got %q", output)
	}
}

func TestBash_CollectFlags(t *testing.T) {
	flags := []types.Flag{
		{Name: "--verbose", Short: "-v"},
		{Name: "--help"},
		{Short: "-x"},
	}

	result := collectFlags(flags)

	expected := []string{"--verbose", "-v", "--help", "-x"}
	if len(result) != len(expected) {
		t.Errorf("expected %d flags, got %d", len(expected), len(result))
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("flag[%d]: got %q, want %q", i, result[i], v)
		}
	}
}

func TestBash_BashFuncName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"mytool", "_tabgen_mytool"},
		{"my-tool", "_tabgen_my_tool"},
		{"my.tool", "_tabgen_my_tool"},
		{"123tool", "_tabgen_123tool"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := bashFuncName(tt.input)
			if got != tt.want {
				t.Errorf("bashFuncName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBash_Generate_SubcommandWithArgumentValues(t *testing.T) {
	b := NewBash()
	tool := &types.Tool{
		Name: "cli",
		Subcommands: []types.Command{
			{
				Name: "config",
				Flags: []types.Flag{
					{
						Name:           "--output",
						ArgumentValues: []string{"json", "yaml"},
					},
				},
			},
		},
	}

	output := b.Generate(tool)

	// Should include the subcommand's flag argument values
	if !strings.Contains(output, "json yaml") {
		t.Error("expected subcommand flag argument values")
	}
}

func TestBash_Generate_NestedSubcommandFlags(t *testing.T) {
	b := NewBash()
	tool := &types.Tool{
		Name: "cli",
		Subcommands: []types.Command{
			{
				Name: "parent",
				Subcommands: []types.Command{
					{
						Name: "child",
						Flags: []types.Flag{
							{
								Name:           "--type",
								ArgumentValues: []string{"a", "b"},
							},
						},
					},
				},
			},
		},
	}

	output := b.Generate(tool)

	// Should collect argument values from nested subcommands
	if !strings.Contains(output, "a b") {
		t.Error("expected nested subcommand flag argument values")
	}
}
