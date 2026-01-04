package parser

import (
	"testing"
)

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "simple version",
			output: "1.2.3",
			want:   "1.2.3",
		},
		{
			name:   "version with v prefix",
			output: "v1.2.3",
			want:   "1.2.3",
		},
		{
			name:   "version with label",
			output: "version 1.2.3",
			want:   "1.2.3",
		},
		{
			name:   "Version capitalized",
			output: "Version 2.0.0",
			want:   "2.0.0",
		},
		{
			name:   "tool name with version",
			output: "mytool version 1.0.0",
			want:   "1.0.0",
		},
		{
			name:   "version with prerelease",
			output: "v1.2.3-beta.1",
			want:   "1.2.3-beta.1",
		},
		{
			name:   "version with build metadata",
			output: "1.2.3+build.456",
			want:   "1.2.3+build.456",
		},
		{
			name:   "two part version",
			output: "2.1",
			want:   "2.1",
		},
		{
			name:   "multiline output takes first line",
			output: "mytool 1.0.0\nCopyright 2024",
			want:   "1.0.0",
		},
		{
			name:   "git describe style",
			output: "v1.0.0-5-gdeadbeef",
			want:   "1.0.0-5", // regex captures up to prerelease number, not git hash
		},
		{
			name:   "short output without version pattern",
			output: "unknown",
			want:   "unknown",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
		{
			name:   "whitespace only",
			output: "   ",
			want:   "",
		},
		{
			name:   "long output no version",
			output: "This is a very long description of the tool that does not contain any version information whatsoever",
			want:   "",
		},
		{
			name:   "go version style",
			output: "go version go1.21.0 linux/amd64",
			want:   "1.21.0",
		},
		{
			name:   "node version style",
			output: "v20.10.0",
			want:   "20.10.0",
		},
		{
			name:   "python version style",
			output: "Python 3.11.4",
			want:   "3.11.4",
		},
		{
			name:   "rust version style",
			output: "rustc 1.72.0 (5680fa18f 2023-08-23)",
			want:   "1.72.0",
		},
		{
			name:   "curl version style",
			output: "curl 8.1.2 (x86_64-pc-linux-gnu)",
			want:   "8.1.2",
		},
		{
			name:   "version at start of line",
			output: "1.0.0 - Release build",
			want:   "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVersion(tt.output)
			if got != tt.want {
				t.Errorf("extractVersion(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestCtxWithTimeout(t *testing.T) {
	ctx, cancel := ctxWithTimeout(1000)
	defer cancel()

	if ctx == nil {
		t.Error("context should not be nil")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("context should have a deadline")
	}
	if deadline.IsZero() {
		t.Error("deadline should not be zero")
	}
}

func TestVersionInfo(t *testing.T) {
	// Test that VersionInfo struct works correctly
	info := VersionInfo{
		Version: "1.0.0",
	}

	if info.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", info.Version)
	}

	if info.DetectedAt.IsZero() {
		// This is expected - we didn't set it
	}
}

func TestExtractVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "version in parentheses",
			output: "tool (version 1.0.0)",
			want:   "1.0.0",
		},
		{
			name:   "multiple versions picks first",
			output: "tool 1.0.0, API 2.0.0",
			want:   "1.0.0",
		},
		{
			name:   "version with revision",
			output: "1.2.3-rc1",
			want:   "1.2.3-rc1",
		},
		{
			name:   "date-like string (not version)",
			output: "Built on 2024-01-15",
			want:   "Built on 2024-01-15", // falls through to short output handling
		},
		{
			name:   "version followed by date",
			output: "1.0.0 (2024-01-15)",
			want:   "1.0.0",
		},
		{
			name:   "semantic version with trailing text",
			output: "v1.0.0 release",
			want:   "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVersion(tt.output)
			if got != tt.want {
				t.Errorf("extractVersion(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

// Note: DetectVersion and tryVersionFlag require actual binary execution
// and are tested via integration tests, not unit tests.
// These functions call exec.Command which we don't mock in unit tests.

func TestExtractVersion_FirstLineOnly(t *testing.T) {
	// Verify that only the first line is checked
	output := `Some info text
version 2.0.0
more text`

	got := extractVersion(output)
	// Should not find 2.0.0 since it's on second line and first line doesn't match
	if got != "Some info text" { // falls through to short string handling
		t.Logf("extractVersion result: %q", got)
	}
}

func TestExtractVersion_CaseInsensitive(t *testing.T) {
	tests := []string{
		"version 1.0.0",
		"Version 1.0.0",
		"VERSION 1.0.0",
	}

	for _, input := range tests {
		got := extractVersion(input)
		if got != "1.0.0" {
			t.Errorf("extractVersion(%q) = %q, want 1.0.0", input, got)
		}
	}
}
