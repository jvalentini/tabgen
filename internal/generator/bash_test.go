package generator

import (
	"strings"
	"testing"

	"github.com/justin/tabgen/internal/types"
)

func TestEscapeShellString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with space", "with space"},
		{"$var", `\$var`},
		{`"quoted"`, `\"quoted\"`},
		{`back\slash`, `back\\slash`},
		{"`backtick`", "\\`backtick\\`"},
		{"$HOME/path", `\$HOME/path`},
		{`mixed"$\`, `mixed\"\$\\`},
	}

	for _, tt := range tests {
		result := escapeShellString(tt.input)
		if result != tt.expected {
			t.Errorf("escapeShellString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestEscapeCasePattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with*glob", `with\*glob`},
		{"question?mark", `question\?mark`},
		{"bracket[x]", `bracket\[x\]`},
		{"pipe|alt", `pipe\|alt`},
		{"paren)", `paren\)`},
		{`back\slash`, `back\\slash`},
		{"*?[]|)", `\*\?\[\]\|\)`},
	}

	for _, tt := range tests {
		result := escapeCasePattern(tt.input)
		if result != tt.expected {
			t.Errorf("escapeCasePattern(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestBashGenerateWithSpecialChars(t *testing.T) {
	gen := NewBash()

	tool := &types.Tool{
		Name: "my$tool",
		Subcommands: []types.Command{
			{Name: "cmd*with*glob"},
			{Name: "normal"},
		},
		GlobalFlags: []types.Flag{
			{Name: "--flag$var"},
		},
	}

	output := gen.Generate(tool)

	// Verify escaping was applied
	if strings.Contains(output, `"my$tool"`) {
		t.Error("tool name should be escaped in strings")
	}
	if strings.Contains(output, "cmd*with*glob)") {
		t.Error("subcommand case pattern should be escaped")
	}
	if strings.Contains(output, "--flag$var") && !strings.Contains(output, `--flag\$var`) {
		t.Error("flag with $ should be escaped")
	}

	// Verify it contains the escaped versions
	if !strings.Contains(output, `my\$tool`) {
		t.Error("output should contain escaped tool name")
	}
	if !strings.Contains(output, `cmd\*with\*glob)`) {
		t.Error("output should contain escaped case pattern")
	}
}
