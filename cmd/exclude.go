package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jvalentini/tabgen/internal/config"
	"github.com/jvalentini/tabgen/internal/types"
)

// Exclude manages the exclusion list
func Exclude(action, pattern string) error {
	storage, err := config.New("")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	cfg, err := storage.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch action {
	case "list", "":
		return excludeList(cfg)
	case "add":
		return excludeAdd(storage, cfg, pattern)
	case "remove", "rm":
		return excludeRemove(storage, cfg, pattern)
	case "clear":
		return excludeClear(storage, cfg)
	default:
		return fmt.Errorf("unknown action: %s (use: list, add, remove, clear)", action)
	}
}

func excludeList(cfg *types.Config) error {
	if len(cfg.Excluded) == 0 {
		fmt.Println("No exclusions configured.")
		fmt.Println("\nUse 'tabgen exclude add <pattern>' to add patterns.")
		return nil
	}

	fmt.Printf("Excluded patterns (%d):\n", len(cfg.Excluded))
	sorted := make([]string, len(cfg.Excluded))
	copy(sorted, cfg.Excluded)
	sort.Strings(sorted)
	for _, pattern := range sorted {
		fmt.Printf("  %s\n", pattern)
	}
	return nil
}

func excludeAdd(storage *config.Storage, cfg *types.Config, pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern required: tabgen exclude add <pattern>")
	}

	// Check if already exists
	for _, p := range cfg.Excluded {
		if p == pattern {
			fmt.Printf("Pattern '%s' already excluded.\n", pattern)
			return nil
		}
	}

	cfg.Excluded = append(cfg.Excluded, pattern)
	if err := storage.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Added exclusion: %s\n", pattern)
	fmt.Println("Run 'tabgen scan' to rescan with updated exclusions.")
	return nil
}

func excludeRemove(storage *config.Storage, cfg *types.Config, pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern required: tabgen exclude remove <pattern>")
	}

	found := false
	newExcluded := make([]string, 0, len(cfg.Excluded))
	for _, p := range cfg.Excluded {
		if p == pattern {
			found = true
		} else {
			newExcluded = append(newExcluded, p)
		}
	}

	if !found {
		return fmt.Errorf("pattern '%s' not found in exclusions", pattern)
	}

	cfg.Excluded = newExcluded
	if err := storage.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed exclusion: %s\n", pattern)
	return nil
}

func excludeClear(storage *config.Storage, cfg *types.Config) error {
	count := len(cfg.Excluded)
	cfg.Excluded = []string{}
	if err := storage.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Cleared %d exclusions.\n", count)
	return nil
}

// ExcludeHelp returns usage help for the exclude command
func ExcludeHelp() string {
	return strings.TrimSpace(`
Usage: tabgen exclude <action> [pattern]

Actions:
  list           Show all excluded patterns (default)
  add <pattern>  Add a pattern to exclusions
  remove <pattern>  Remove a pattern from exclusions
  clear          Remove all exclusions

Patterns are matched against tool names. Examples:
  tabgen exclude add python2.7
  tabgen exclude add "*.dll"
`)
}
