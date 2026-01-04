package generator

import (
	"strings"
	"testing"

	"github.com/justin/tabgen/internal/types"
)

func TestZsh_Generate_Basic(t *testing.T) {
	z := NewZsh()
	tool := &types.Tool{
		Name: "mytool",
		GlobalFlags: []types.Flag{
			{Name: "--verbose", Short: "-v", Description: "Enable verbose"},
		},
		Subcommands: []types.Command{
			{Name: "init", Description: "Initialize project"},
		},
	}

	output := z.Generate(tool)

	if !strings.Contains(output, "#compdef mytool") {
		t.Error("expected #compdef header")
	}
	if !strings.Contains(output, "_tabgen_mytool") {
		t.Error("expected function name")
	}
	if !strings.Contains(output, "init") {
		t.Error("expected init subcommand")
	}
}

func TestZsh_FormatFlagSpec_WithArgumentValues(t *testing.T) {
	z := NewZsh()

	tests := []struct {
		name     string
		flag     types.Flag
		wantPart string
	}{
		{
			name: "flag with argument values",
			flag: types.Flag{
				Name:           "--format",
				Short:          "-f",
				Arg:            "type",
				ArgumentValues: []string{"json", "yaml", "xml"},
				Description:    "Output format",
			},
			wantPart: ":type:(json yaml xml)",
		},
		{
			name: "long-only with values",
			flag: types.Flag{
				Name:           "--output",
				Arg:            "format",
				ArgumentValues: []string{"text", "binary"},
				Description:    "Output type",
			},
			wantPart: ":format:(text binary)",
		},
		{
			name: "flag without argument values",
			flag: types.Flag{
				Name:        "--config",
				Arg:         "file",
				Description: "Config file",
			},
			wantPart: ":file:'",
		},
		{
			name: "boolean flag",
			flag: types.Flag{
				Name:        "--verbose",
				Description: "Be verbose",
			},
			wantPart: "--verbose[Be verbose]'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := z.formatFlagSpec(tt.flag)
			if !strings.Contains(spec, tt.wantPart) {
				t.Errorf("expected spec to contain %q, got %q", tt.wantPart, spec)
			}
		})
	}
}

func TestZsh_FormatArgCompletion(t *testing.T) {
	z := NewZsh()

	tests := []struct {
		name string
		flag types.Flag
		want string
	}{
		{
			name: "with values",
			flag: types.Flag{Arg: "format", ArgumentValues: []string{"json", "yaml"}},
			want: ":format:(json yaml)'",
		},
		{
			name: "with values no arg name",
			flag: types.Flag{ArgumentValues: []string{"a", "b", "c"}},
			want: ":value:(a b c)'",
		},
		{
			name: "arg only",
			flag: types.Flag{Arg: "file"},
			want: ":file:'",
		},
		{
			name: "empty",
			flag: types.Flag{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := z.formatArgCompletion(tt.flag)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestZsh_Generate_WithArgumentValues(t *testing.T) {
	z := NewZsh()
	tool := &types.Tool{
		Name: "cli",
		GlobalFlags: []types.Flag{
			{
				Name:           "--format",
				Short:          "-f",
				Arg:            "type",
				ArgumentValues: []string{"json", "yaml"},
				Description:    "Output format",
			},
		},
	}

	output := z.Generate(tool)

	// Check that the completion spec includes the argument values
	if !strings.Contains(output, "(json yaml)") {
		t.Error("expected argument values in zsh completion")
	}
}

func TestZsh_ZshFuncName(t *testing.T) {
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
			got := zshFuncName(tt.input)
			if got != tt.want {
				t.Errorf("zshFuncName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
