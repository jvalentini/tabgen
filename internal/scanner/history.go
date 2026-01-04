package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// GetUsedCommands extracts command names from shell history files
// Returns a set (map) of command names that the user has actually executed
func GetUsedCommands() (map[string]bool, error) {
	usedCommands := make(map[string]bool)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return usedCommands, err
	}

	historyFiles := []string{
		filepath.Join(homeDir, ".bash_history"),
		filepath.Join(homeDir, ".zsh_history"),
	}

	for _, histFile := range historyFiles {
		if err := parseHistoryFile(histFile, usedCommands); err != nil {
			if !os.IsNotExist(err) {
				return usedCommands, err
			}
		}
	}

	return usedCommands, nil
}

// parseHistoryFile reads a history file and extracts command names
func parseHistoryFile(path string, commands map[string]bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Zsh history format: ": timestamp:duration;command"
		if strings.HasPrefix(line, ":") {
			parts := strings.SplitN(line, ";", 2)
			if len(parts) == 2 {
				line = parts[1]
			}
		}

		cmd := extractCommand(line)
		if cmd != "" {
			commands[cmd] = true
		}
	}

	return scanner.Err()
}

// extractCommand gets the base command from a shell history line
func extractCommand(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}

	for _, prefix := range []string{"sudo ", "doas "} {
		if after, found := strings.CutPrefix(line, prefix); found {
			line = strings.TrimSpace(after)
		}
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}

	cmdIdx := 0
	for cmdIdx < len(fields) && strings.Contains(fields[cmdIdx], "=") {
		cmdIdx++
	}
	if cmdIdx >= len(fields) {
		return ""
	}

	cmd := fields[cmdIdx]

	if strings.Contains(cmd, "/") {
		cmd = filepath.Base(cmd)
	}

	builtins := map[string]bool{
		"cd": true, "echo": true, "exit": true, "export": true,
		"set": true, "unset": true, "source": true, ".": true,
		"[": true, "[[": true, "alias": true, "bg": true,
		"fg": true, "jobs": true, "kill": true, "pwd": true,
		"read": true, "wait": true, "history": true,
	}
	if builtins[cmd] {
		return ""
	}

	if strings.ContainsAny(cmd[:1], "|&<>;(){}") {
		return ""
	}

	return cmd
}
