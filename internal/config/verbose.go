package config

import (
	"fmt"
	"os"
)

// Verbose controls debug output globally
var Verbose bool

// Logf prints a formatted message if verbose mode is enabled
func Logf(format string, args ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
	}
}

// LogSection prints a section header if verbose mode is enabled
func LogSection(name string) {
	if Verbose {
		fmt.Fprintf(os.Stderr, "\n[verbose] === %s ===\n", name)
	}
}

// LogSnippet prints a snippet of text (first N lines) if verbose mode is enabled
func LogSnippet(label string, text string, maxLines int) {
	if !Verbose || text == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "[verbose] %s:\n", label)

	lines := 0
	start := 0
	for i, c := range text {
		if c == '\n' {
			lines++
			if lines <= maxLines {
				fmt.Fprintf(os.Stderr, "  | %s\n", text[start:i])
			}
			start = i + 1
		}
	}
	// Handle last line without newline
	if start < len(text) && lines < maxLines {
		fmt.Fprintf(os.Stderr, "  | %s\n", text[start:])
		lines++
	}
	if lines > maxLines {
		fmt.Fprintf(os.Stderr, "  | ... (%d more lines)\n", lines-maxLines)
	}
}
