package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/justin/tabgen/internal/types"
)

// Storage handles reading and writing TabGen data files
type Storage struct {
	baseDir string
}

// New creates a new Storage instance
func New(baseDir string) (*Storage, error) {
	// Expand ~ to home directory
	if baseDir == "" || baseDir == "~/.tabgen" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		baseDir = filepath.Join(home, ".tabgen")
	}

	// Ensure directories exist
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "tools"),
		filepath.Join(baseDir, "completions", "bash"),
		filepath.Join(baseDir, "completions", "zsh"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	return &Storage{baseDir: baseDir}, nil
}

// BaseDir returns the base directory path
func (s *Storage) BaseDir() string {
	return s.baseDir
}

// LoadCatalog loads the catalog from disk
func (s *Storage) LoadCatalog() (*types.Catalog, error) {
	path := filepath.Join(s.baseDir, "catalog.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &types.Catalog{Tools: make(map[string]types.CatalogEntry)}, nil
		}
		return nil, err
	}

	var catalog types.Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	if catalog.Tools == nil {
		catalog.Tools = make(map[string]types.CatalogEntry)
	}
	return &catalog, nil
}

// SaveCatalog saves the catalog to disk
func (s *Storage) SaveCatalog(catalog *types.Catalog) error {
	path := filepath.Join(s.baseDir, "catalog.json")
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadTool loads a parsed tool from disk
func (s *Storage) LoadTool(name string) (*types.Tool, error) {
	path := filepath.Join(s.baseDir, "tools", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tool types.Tool
	if err := json.Unmarshal(data, &tool); err != nil {
		return nil, err
	}
	return &tool, nil
}

// SaveTool saves a parsed tool to disk
func (s *Storage) SaveTool(tool *types.Tool) error {
	path := filepath.Join(s.baseDir, "tools", tool.Name+".json")
	data, err := json.MarshalIndent(tool, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ToolExists checks if a tool has been parsed
func (s *Storage) ToolExists(name string) bool {
	path := filepath.Join(s.baseDir, "tools", name+".json")
	_, err := os.Stat(path)
	return err == nil
}

// SaveBashCompletion saves a bash completion script
func (s *Storage) SaveBashCompletion(name, content string) error {
	path := filepath.Join(s.baseDir, "completions", "bash", name)
	return os.WriteFile(path, []byte(content), 0644)
}

// SaveZshCompletion saves a zsh completion script
func (s *Storage) SaveZshCompletion(name, content string) error {
	path := filepath.Join(s.baseDir, "completions", "zsh", "_"+name)
	return os.WriteFile(path, []byte(content), 0644)
}

// CompletionPaths returns the paths to completion directories
func (s *Storage) CompletionPaths() (bash, zsh string) {
	return filepath.Join(s.baseDir, "completions", "bash"),
		filepath.Join(s.baseDir, "completions", "zsh")
}

// LoadConfig loads the configuration
func (s *Storage) LoadConfig() (*types.Config, error) {
	path := filepath.Join(s.baseDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := types.DefaultConfig()
			return &cfg, nil
		}
		return nil, err
	}

	var cfg types.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig saves the configuration
func (s *Storage) SaveConfig(cfg *types.Config) error {
	path := filepath.Join(s.baseDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
