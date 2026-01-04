package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

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
