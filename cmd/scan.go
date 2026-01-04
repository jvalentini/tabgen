package cmd

import (
	"fmt"
	"time"

	"github.com/justin/tabgen/internal/config"
	"github.com/justin/tabgen/internal/scanner"
)

// Scan walks $PATH and discovers executable tools
func Scan() error {
	storage, err := config.New("")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Load config for exclusions
	cfg, _ := storage.LoadConfig()

	// Load existing catalog to preserve generated status
	existingCatalog, _ := storage.LoadCatalog()

	fmt.Println("Scanning $PATH for executables...")
	if len(cfg.Excluded) > 0 {
		fmt.Printf("  (excluding %d patterns)\n", len(cfg.Excluded))
	}
	start := time.Now()

	s := scanner.New(cfg.Excluded)
	catalog, err := s.Scan()
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Preserve generated status from existing catalog
	for name, entry := range catalog.Tools {
		if existing, ok := existingCatalog.Tools[name]; ok {
			entry.Generated = existing.Generated
			catalog.Tools[name] = entry
		}
	}

	if err := storage.SaveCatalog(catalog); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	elapsed := time.Since(start)

	fmt.Printf("Found %d executables in %v\n", len(catalog.Tools), elapsed.Round(time.Millisecond))
	fmt.Printf("Catalog saved to %s/catalog.json\n", storage.BaseDir())
	fmt.Printf("\nRun 'tabgen generate <tool>' to create completions for a specific tool.")
	fmt.Printf("\nRun 'tabgen generate' to process all tools (may take a while).\n")

	return nil
}
