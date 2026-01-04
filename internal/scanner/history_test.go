package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractCommand(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"simple command", "git status", "git"},
		{"command with args", "docker ps -a", "docker"},
		{"with sudo", "sudo apt update", "apt"},
		{"with doas", "doas pkg install", "pkg"},
		{"environment variable", "VAR=value command", "command"},
		{"multiple env vars", "A=1 B=2 command", "command"},
		{"comment line", "# this is a comment", ""},
		{"empty line", "", ""},
		{"whitespace only", "   ", ""},
		{"full path", "/usr/bin/git status", "git"},
		{"builtin cd", "cd /home/user", ""},
		{"builtin echo", "echo hello", ""},
		{"builtin pwd", "pwd", ""},
		{"sudo with env", "sudo VAR=value apt install", "apt"},
		{"double sudo", "sudo sudo command", "sudo"},
		{"trailing spaces", "git status  ", "git"},
		{"leading spaces", "  docker ps", "docker"},
		{"pipe character", "|grep foo", ""},
		{"ampersand", "&wait", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommand(tt.line)
			if got != tt.expected {
				t.Errorf("extractCommand(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

func TestParseHistoryFile(t *testing.T) {
	tempDir := t.TempDir()
	histFile := filepath.Join(tempDir, "test_history")

	content := `git status
docker ps
sudo apt update
npm install
# comment line

VAR=value make build
cd /tmp
echo hello
`

	if err := os.WriteFile(histFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test history file: %v", err)
	}

	commands := make(map[string]bool)
	if err := parseHistoryFile(histFile, commands); err != nil {
		t.Fatalf("parseHistoryFile failed: %v", err)
	}

	expectedCommands := []string{"git", "docker", "apt", "npm", "make"}
	for _, cmd := range expectedCommands {
		if !commands[cmd] {
			t.Errorf("Expected command %q not found in parsed history", cmd)
		}
	}

	unexpectedCommands := []string{"cd", "echo", "VAR=value", "comment"}
	for _, cmd := range unexpectedCommands {
		if commands[cmd] {
			t.Errorf("Unexpected command %q found in parsed history", cmd)
		}
	}
}

func TestParseHistoryFile_ZshFormat(t *testing.T) {
	tempDir := t.TempDir()
	histFile := filepath.Join(tempDir, "test_zsh_history")

	content := `: 1609459200:0;git commit -m "test"
: 1609459201:0;docker build .
: 1609459202:0;npm test
: 1609459203:0;sudo apt upgrade
`

	if err := os.WriteFile(histFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test zsh history file: %v", err)
	}

	commands := make(map[string]bool)
	if err := parseHistoryFile(histFile, commands); err != nil {
		t.Fatalf("parseHistoryFile failed: %v", err)
	}

	expectedCommands := []string{"git", "docker", "npm", "apt"}
	for _, cmd := range expectedCommands {
		if !commands[cmd] {
			t.Errorf("Expected command %q not found in parsed zsh history", cmd)
		}
	}
}

func TestParseHistoryFile_MissingFile(t *testing.T) {
	commands := make(map[string]bool)
	err := parseHistoryFile("/nonexistent/file", commands)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Expected os.IsNotExist error, got: %v", err)
	}
}

func TestExtractCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"only env vars", "A=1 B=2", ""},
		{"command after many env vars", "A=1 B=2 C=3 D=4 command", "command"},
		{"path with spaces", "/usr/local/bin/my tool", "my"},
		{"equals in command", "command=value arg", "arg"},
		{"nested sudo", "sudo sudo sudo command", "sudo"},
		{"doas and sudo", "doas sudo command", "sudo"},
		{"sudo doas", "sudo doas command", "command"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommand(tt.line)
			if got != tt.expected {
				t.Errorf("extractCommand(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

func TestGetUsedCommands_Integration(t *testing.T) {
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	bashHistContent := `git status
docker ps
npm install
`
	bashHistPath := filepath.Join(tempDir, ".bash_history")
	if err := os.WriteFile(bashHistPath, []byte(bashHistContent), 0644); err != nil {
		t.Fatalf("Failed to write bash history: %v", err)
	}

	zshHistContent := `: 1609459200:0;kubectl get pods
: 1609459201:0;make build
`
	zshHistPath := filepath.Join(tempDir, ".zsh_history")
	if err := os.WriteFile(zshHistPath, []byte(zshHistContent), 0644); err != nil {
		t.Fatalf("Failed to write zsh history: %v", err)
	}

	commands, err := GetUsedCommands()
	if err != nil {
		t.Fatalf("GetUsedCommands failed: %v", err)
	}

	expectedCommands := []string{"git", "docker", "npm", "kubectl", "make"}
	for _, cmd := range expectedCommands {
		if !commands[cmd] {
			t.Errorf("Expected command %q not found", cmd)
		}
	}
}

func TestGetUsedCommands_NoHistoryFiles(t *testing.T) {
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	commands, err := GetUsedCommands()
	if err != nil {
		t.Fatalf("Expected no error when history files don't exist, got: %v", err)
	}

	if len(commands) != 0 {
		t.Errorf("Expected empty command map, got %d commands", len(commands))
	}
}
