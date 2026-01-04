package parser

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// VersionInfo contains version detection results
type VersionInfo struct {
	Version   string
	DetectedAt time.Time
}

// DetectVersion attempts to get version info from a tool using default settings
// Deprecated: Use Parser.detectVersion() for configurable version detection
func DetectVersion(path string) string {
	return detectVersionWithConfig(path, DefaultConfig())
}

// detectVersion attempts to get version info from a tool using parser config
func (p *Parser) detectVersion(path string) string {
	return detectVersionWithConfig(path, p.config)
}

// detectVersionWithConfig attempts to get version info using provided config
func detectVersionWithConfig(path string, cfg ParserConfig) string {
	for _, flag := range cfg.VersionCmds {
		version := tryVersionFlagWithTimeout(path, flag, cfg.HelpTimeout)
		if version != "" {
			return version
		}
	}

	return ""
}

// tryVersionFlagWithTimeout runs the tool with a version flag and extracts the version
func tryVersionFlagWithTimeout(path, flag string, timeout time.Duration) string {
	ctx, cancel := ctxWithTimeout(timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, flag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	return extractVersion(string(output))
}

// tryVersionFlag runs the tool with a version flag using default timeout
// Deprecated: Use tryVersionFlagWithTimeout for configurable timeout
func tryVersionFlag(path, flag string) string {
	return tryVersionFlagWithTimeout(path, flag, 2*time.Second)
}

// ctxWithTimeout creates a context with timeout
func ctxWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// extractVersion extracts a version string from command output
func extractVersion(output string) string {
	// Common version patterns
	patterns := []*regexp.Regexp{
		// "version 1.2.3" or "v1.2.3"
		regexp.MustCompile(`(?i)(?:version\s+)?v?(\d+\.\d+(?:\.\d+)?(?:[-+][a-zA-Z0-9.]+)?)`),
		// "1.2.3" at start of line
		regexp.MustCompile(`(?m)^(\d+\.\d+(?:\.\d+)?)`),
	}

	// Take first line for simpler matching
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return ""
	}
	firstLine := lines[0]

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(firstLine); len(matches) > 1 {
			return matches[1]
		}
	}

	// If no version found but output is short, use it as-is (trimmed)
	if len(firstLine) < 50 && len(firstLine) > 0 {
		return strings.TrimSpace(firstLine)
	}

	return ""
}
