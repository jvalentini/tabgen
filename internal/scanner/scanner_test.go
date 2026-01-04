package scanner

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	s := New(nil)
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if !s.quickMode {
		t.Error("New() should set quickMode to true")
	}
}

func TestNewFull(t *testing.T) {
	s := NewFull(nil)
	if s == nil {
		t.Fatal("NewFull() returned nil")
	}
	if s.quickMode {
		t.Error("NewFull() should set quickMode to false")
	}
}

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"os.ErrPermission", os.ErrPermission, true},
		{"exec.ErrNotFound", exec.ErrNotFound, false},
		{"permission denied string", errors.New("permission denied"), true},
		{"EACCES string", errors.New("open file: EACCES"), true},
		{"operation not permitted", errors.New("operation not permitted"), true},
		{"generic error", errors.New("something went wrong"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPermissionError(tt.err)
			if got != tt.want {
				t.Errorf("isPermissionError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestScanSingle_RealCommand(t *testing.T) {
	// Test with a real command that exists on most systems
	if _, err := exec.LookPath("ls"); err != nil {
		t.Skip("ls command not found")
	}

	s := New(nil)
	entry, err := s.ScanSingle("ls")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.Name != "ls" {
		t.Errorf("expected name 'ls', got %q", entry.Name)
	}
	if entry.Path == "" {
		t.Error("expected non-empty path")
	}
	if entry.LastScan.IsZero() {
		t.Error("expected LastScan to be set")
	}
}

func TestScanSingle_NonExistentCommand(t *testing.T) {
	s := New(nil)
	_, err := s.ScanSingle("thisdoesnotexist123456789")
	if err == nil {
		t.Error("expected error for non-existent command")
	}
}

func TestScan_PathPrecedence(t *testing.T) {
	// Create two temp directories
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	homeDir := t.TempDir()

	// Create same-named executable in both
	for _, dir := range []string{dir1, dir2} {
		path := filepath.Join(dir, "duped")
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho test"), 0755); err != nil {
			t.Fatalf("failed to create duped: %v", err)
		}
	}

	// Create history file that includes duped
	histPath := filepath.Join(homeDir, ".bash_history")
	if err := os.WriteFile(histPath, []byte("duped\n"), 0644); err != nil {
		t.Fatalf("failed to write history: %v", err)
	}

	origPath := os.Getenv("PATH")
	origHome := os.Getenv("HOME")
	os.Setenv("PATH", dir1+string(os.PathListSeparator)+dir2)
	os.Setenv("HOME", homeDir)
	defer func() {
		os.Setenv("PATH", origPath)
		os.Setenv("HOME", origHome)
	}()

	s := New(nil)
	catalog, err := s.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have one entry, from the first directory
	entry, ok := catalog.Tools["duped"]
	if !ok {
		t.Fatal("duped not found in catalog")
	}

	if entry.Path != filepath.Join(dir1, "duped") {
		t.Errorf("expected path from first dir, got %s", entry.Path)
	}
}

func TestScan_UnreadableDirectory(t *testing.T) {
	origPath := os.Getenv("PATH")
	origHome := os.Getenv("HOME")
	// Include a non-existent directory in PATH
	homeDir := t.TempDir()
	os.Setenv("PATH", "/nonexistent/path/that/does/not/exist")
	os.Setenv("HOME", homeDir)
	defer func() {
		os.Setenv("PATH", origPath)
		os.Setenv("HOME", origHome)
	}()

	s := New(nil)
	catalog, err := s.Scan()
	// Should not error, just skip unreadable directories
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(catalog.Tools) != 0 {
		t.Errorf("expected empty catalog, got %d tools", len(catalog.Tools))
	}
}

func TestIsExcluded_InvalidPattern(t *testing.T) {
	s := New([]string{"["})
	_, err := s.isExcluded("anything")
	if err == nil {
		t.Error("expected error for invalid glob pattern")
	}
}

func TestCatalog_LastScanSet(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	script := filepath.Join(tmpDir, "tool")
	if err := os.WriteFile(script, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatalf("failed to create script: %v", err)
	}

	histPath := filepath.Join(homeDir, ".bash_history")
	if err := os.WriteFile(histPath, []byte("tool\n"), 0644); err != nil {
		t.Fatalf("failed to write history: %v", err)
	}

	origPath := os.Getenv("PATH")
	origHome := os.Getenv("HOME")
	os.Setenv("PATH", tmpDir)
	os.Setenv("HOME", homeDir)
	defer func() {
		os.Setenv("PATH", origPath)
		os.Setenv("HOME", origHome)
	}()

	s := New(nil)
	catalog, err := s.Scan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if catalog.LastScan.IsZero() {
		t.Error("expected catalog LastScan to be set")
	}

	entry := catalog.Tools["tool"]
	if entry.LastScan.IsZero() {
		t.Error("expected entry LastScan to be set")
	}
}

func TestScanner_WithHistoryFiltering(t *testing.T) {
	origHome := os.Getenv("HOME")
	origPath := os.Getenv("PATH")

	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	homeDir := filepath.Join(tempDir, "home")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin dir: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("Failed to create home dir: %v", err)
	}

	os.Setenv("PATH", binDir)
	os.Setenv("HOME", homeDir)
	defer func() {
		os.Setenv("PATH", origPath)
		os.Setenv("HOME", origHome)
	}()

	createExecutable := func(name string) {
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho test"), 0755); err != nil {
			t.Fatalf("Failed to create executable %s: %v", name, err)
		}
	}

	createExecutable("git")
	createExecutable("docker")
	createExecutable("unused-tool")
	createExecutable("another-unused")

	histContent := `git status
docker ps
npm install
`
	histPath := filepath.Join(homeDir, ".bash_history")
	if err := os.WriteFile(histPath, []byte(histContent), 0644); err != nil {
		t.Fatalf("Failed to write history: %v", err)
	}

	scanner := New(nil)
	catalog, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if catalog.Tools["git"].Name != "git" {
		t.Error("Expected 'git' to be in catalog")
	}
	if catalog.Tools["docker"].Name != "docker" {
		t.Error("Expected 'docker' to be in catalog")
	}

	if _, exists := catalog.Tools["unused-tool"]; exists {
		t.Error("'unused-tool' should not be in catalog (not in history)")
	}
	if _, exists := catalog.Tools["another-unused"]; exists {
		t.Error("'another-unused' should not be in catalog (not in history)")
	}

	if _, exists := catalog.Tools["npm"]; exists {
		t.Error("'npm' should not be in catalog (not in PATH)")
	}
}

func TestScanner_WithExclusions(t *testing.T) {
	origHome := os.Getenv("HOME")
	origPath := os.Getenv("PATH")

	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	homeDir := filepath.Join(tempDir, "home")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin dir: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("Failed to create home dir: %v", err)
	}

	os.Setenv("PATH", binDir)
	os.Setenv("HOME", homeDir)
	defer func() {
		os.Setenv("PATH", origPath)
		os.Setenv("HOME", origHome)
	}()

	createExecutable := func(name string) {
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho test"), 0755); err != nil {
			t.Fatalf("Failed to create executable %s: %v", name, err)
		}
	}

	createExecutable("git")
	createExecutable("docker")
	createExecutable("test.dll")

	histContent := `git status
docker ps
test.dll run
`
	histPath := filepath.Join(homeDir, ".bash_history")
	if err := os.WriteFile(histPath, []byte(histContent), 0644); err != nil {
		t.Fatalf("Failed to write history: %v", err)
	}

	scanner := New([]string{"*.dll"})
	catalog, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if catalog.Tools["git"].Name != "git" {
		t.Error("Expected 'git' to be in catalog")
	}
	if catalog.Tools["docker"].Name != "docker" {
		t.Error("Expected 'docker' to be in catalog")
	}

	if _, exists := catalog.Tools["test.dll"]; exists {
		t.Error("'test.dll' should be excluded by pattern *.dll")
	}
}

func TestScanner_EmptyPath(t *testing.T) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	scanner := New(nil)
	catalog, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan with empty PATH should not fail: %v", err)
	}

	if len(catalog.Tools) != 0 {
		t.Errorf("Expected empty catalog with empty PATH, got %d tools", len(catalog.Tools))
	}
}

func TestScanner_HiddenFilesSkipped(t *testing.T) {
	origHome := os.Getenv("HOME")
	origPath := os.Getenv("PATH")

	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	homeDir := filepath.Join(tempDir, "home")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin dir: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("Failed to create home dir: %v", err)
	}

	os.Setenv("PATH", binDir)
	os.Setenv("HOME", homeDir)
	defer func() {
		os.Setenv("PATH", origPath)
		os.Setenv("HOME", origHome)
	}()

	createExecutable := func(name string) {
		path := filepath.Join(binDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\necho test"), 0755); err != nil {
			t.Fatalf("Failed to create executable %s: %v", name, err)
		}
	}

	createExecutable("git")
	createExecutable(".hidden")

	histContent := `git status
.hidden run
`
	histPath := filepath.Join(homeDir, ".bash_history")
	if err := os.WriteFile(histPath, []byte(histContent), 0644); err != nil {
		t.Fatalf("Failed to write history: %v", err)
	}

	scanner := New(nil)
	catalog, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if catalog.Tools["git"].Name != "git" {
		t.Error("Expected 'git' to be in catalog")
	}

	if _, exists := catalog.Tools[".hidden"]; exists {
		t.Error("Hidden files (starting with .) should be skipped")
	}
}

func TestIsExcluded(t *testing.T) {
	scanner := New([]string{"*.dll", "test-*", "exact"})

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"matches glob pattern", "foo.dll", true},
		{"matches prefix pattern", "test-tool", true},
		{"exact match", "exact", true},
		{"no match", "normal-tool", false},
		{"partial match not enough", "testing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded, err := scanner.isExcluded(tt.input)
			if err != nil {
				t.Fatalf("isExcluded failed: %v", err)
			}
			if excluded != tt.expected {
				t.Errorf("isExcluded(%q) = %v, want %v", tt.input, excluded, tt.expected)
			}
		})
	}
}
