package generator

import (
	"strings"
	"testing"
	"time"

	"github.com/jvalentini/tabgen/internal/types"
)

func TestCountItems(t *testing.T) {
	tool := &types.Tool{
		Name: "test",
		GlobalFlags: []types.Flag{
			{Name: "--verbose"},
			{Name: "--help"},
		},
		Subcommands: []types.Command{
			{
				Name: "cmd1",
				Flags: []types.Flag{
					{Name: "--flag1"},
				},
			},
			{
				Name: "cmd2",
				Subcommands: []types.Command{
					{Name: "subcmd1"},
				},
			},
		},
	}

	subs, flags := countItems(tool)
	if subs != 3 { // cmd1, cmd2, subcmd1
		t.Errorf("expected 3 subcommands, got %d", subs)
	}
	if flags != 3 { // 2 global + 1 cmd1 flag
		t.Errorf("expected 3 flags, got %d", flags)
	}
}

func TestTruncateTool(t *testing.T) {
	// Create a tool with many subcommands
	cmds := make([]types.Command, MaxSubcommands+100)
	for i := range cmds {
		cmds[i] = types.Command{Name: "cmd" + string(rune('a'+i%26))}
	}

	tool := &types.Tool{
		Name:        "test",
		Subcommands: cmds,
	}

	truncated, warnings := truncateTool(tool)

	if len(truncated.Subcommands) != MaxSubcommands {
		t.Errorf("expected %d subcommands after truncation, got %d", MaxSubcommands, len(truncated.Subcommands))
	}

	if len(warnings) == 0 {
		t.Error("expected warnings about truncation")
	}

	foundWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "truncated subcommands") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("expected warning about truncated subcommands, got: %v", warnings)
	}
}

func TestTruncateFlags(t *testing.T) {
	// Create a tool with many flags
	flags := make([]types.Flag, MaxFlags+50)
	for i := range flags {
		flags[i] = types.Flag{Name: "--flag-" + string(rune('a'+i%26))}
	}

	tool := &types.Tool{
		Name:        "test",
		GlobalFlags: flags,
	}

	truncated, warnings := truncateTool(tool)

	if len(truncated.GlobalFlags) != MaxFlags {
		t.Errorf("expected %d global flags after truncation, got %d", MaxFlags, len(truncated.GlobalFlags))
	}

	if len(warnings) == 0 {
		t.Error("expected warnings about truncation")
	}
}

func TestNoTruncationNeeded(t *testing.T) {
	tool := &types.Tool{
		Name: "test",
		GlobalFlags: []types.Flag{
			{Name: "--verbose"},
		},
		Subcommands: []types.Command{
			{Name: "cmd1"},
		},
	}

	truncated, warnings := truncateTool(tool)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got: %v", warnings)
	}

	if truncated != tool {
		t.Error("expected original tool to be returned when no truncation needed")
	}
}

func TestCheckOutputSize(t *testing.T) {
	// Small script should pass through unchanged
	small := "small script"
	result, warnings := checkOutputSize(small, "test")
	if result != small {
		t.Error("small script should not be modified")
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for small script, got: %v", warnings)
	}

	// Large script should be truncated
	large := strings.Repeat("x", MaxOutputSize+1000)
	result, warnings = checkOutputSize(large, "test")
	if len(result) >= len(large) {
		t.Error("large script should be truncated")
	}
	if len(warnings) == 0 {
		t.Error("expected warning for large script")
	}
	if !strings.Contains(result, "WARNING: Script truncated") {
		t.Error("truncated script should contain truncation warning")
	}
}

func TestGenerateWithLimits(t *testing.T) {
	// Create a normal-sized tool
	tool := &types.Tool{
		Name:     "mytool",
		Path:     "/usr/bin/mytool",
		ParsedAt: time.Now(),
		Source:   "help",
		GlobalFlags: []types.Flag{
			{Name: "--verbose", Short: "-v", Description: "Verbose output"},
		},
		Subcommands: []types.Command{
			{Name: "start", Description: "Start the service"},
			{Name: "stop", Description: "Stop the service"},
		},
	}

	bashGen := NewBash()
	result := bashGen.GenerateWithLimits(tool)

	if result.Script == "" {
		t.Error("expected non-empty script")
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings for normal tool, got: %v", result.Warnings)
	}

	zshGen := NewZsh()
	zshResult := zshGen.GenerateWithLimits(tool)

	if zshResult.Script == "" {
		t.Error("expected non-empty zsh script")
	}
	if len(zshResult.Warnings) != 0 {
		t.Errorf("expected no warnings for normal tool, got: %v", zshResult.Warnings)
	}
}
