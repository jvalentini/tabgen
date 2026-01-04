package types

import "testing"

func TestContentHash_EmptyTool(t *testing.T) {
	tool := &Tool{Name: "mytool", Path: "/usr/bin/mytool"}
	hash := tool.ContentHash()

	if hash == "" {
		t.Error("expected non-empty hash for empty tool")
	}

	// Same empty tool should produce same hash
	tool2 := &Tool{Name: "different", Path: "/other/path"}
	hash2 := tool2.ContentHash()

	if hash != hash2 {
		t.Errorf("empty tools should have same hash, got %s vs %s", hash, hash2)
	}
}

func TestContentHash_Deterministic(t *testing.T) {
	tool := &Tool{
		Name: "mytool",
		Path: "/usr/bin/mytool",
		Subcommands: []Command{
			{Name: "build", Description: "Build the project"},
			{Name: "test", Description: "Run tests"},
		},
		GlobalFlags: []Flag{
			{Name: "--verbose", Short: "-v", Description: "Enable verbose output"},
		},
	}

	hash1 := tool.ContentHash()
	hash2 := tool.ContentHash()

	if hash1 != hash2 {
		t.Errorf("same tool should produce same hash, got %s vs %s", hash1, hash2)
	}
}

func TestContentHash_DifferentContent(t *testing.T) {
	tool1 := &Tool{
		Name: "mytool",
		Subcommands: []Command{
			{Name: "build", Description: "Build the project"},
		},
	}

	tool2 := &Tool{
		Name: "mytool",
		Subcommands: []Command{
			{Name: "build", Description: "Build the project"},
			{Name: "test", Description: "Run tests"},
		},
	}

	hash1 := tool1.ContentHash()
	hash2 := tool2.ContentHash()

	if hash1 == hash2 {
		t.Error("different content should produce different hashes")
	}
}

func TestContentHash_IgnoresMetadata(t *testing.T) {
	// Version, Name, Path, Source should not affect the hash
	tool1 := &Tool{
		Name:    "mytool",
		Path:    "/usr/bin/mytool",
		Version: "1.0.0",
		Source:  "help",
		Subcommands: []Command{
			{Name: "build", Description: "Build the project"},
		},
	}

	tool2 := &Tool{
		Name:    "different_name",
		Path:    "/different/path",
		Version: "2.0.0",
		Source:  "man",
		Subcommands: []Command{
			{Name: "build", Description: "Build the project"},
		},
	}

	hash1 := tool1.ContentHash()
	hash2 := tool2.ContentHash()

	if hash1 != hash2 {
		t.Errorf("metadata changes should not affect hash, got %s vs %s", hash1, hash2)
	}
}

func TestContentHash_FlagChanges(t *testing.T) {
	tool1 := &Tool{
		Name: "mytool",
		GlobalFlags: []Flag{
			{Name: "--help", Short: "-h", Description: "Show help"},
		},
	}

	tool2 := &Tool{
		Name: "mytool",
		GlobalFlags: []Flag{
			{Name: "--help", Short: "-h", Description: "Show this help message"},
		},
	}

	hash1 := tool1.ContentHash()
	hash2 := tool2.ContentHash()

	if hash1 == hash2 {
		t.Error("different flag descriptions should produce different hashes")
	}
}

func TestContentHash_NestedSubcommands(t *testing.T) {
	tool1 := &Tool{
		Name: "mytool",
		Subcommands: []Command{
			{
				Name:        "config",
				Description: "Manage config",
				Subcommands: []Command{
					{Name: "get", Description: "Get config value"},
				},
			},
		},
	}

	tool2 := &Tool{
		Name: "mytool",
		Subcommands: []Command{
			{
				Name:        "config",
				Description: "Manage config",
				Subcommands: []Command{
					{Name: "get", Description: "Get config value"},
					{Name: "set", Description: "Set config value"},
				},
			},
		},
	}

	hash1 := tool1.ContentHash()
	hash2 := tool2.ContentHash()

	if hash1 == hash2 {
		t.Error("different nested subcommands should produce different hashes")
	}
}
