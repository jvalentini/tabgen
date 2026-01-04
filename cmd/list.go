package cmd

import (
	"fmt"
	"sort"

	"github.com/justin/tabgen/internal/config"
)

// List shows discovered tools and their status
func List(showAll bool) error {
	storage, err := config.New("")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	catalog, err := storage.LoadCatalog()
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	if len(catalog.Tools) == 0 {
		fmt.Println("No tools in catalog. Run 'tabgen scan' first.")
		return nil
	}

	// Sort tool names
	names := make([]string, 0, len(catalog.Tools))
	for name := range catalog.Tools {
		names = append(names, name)
	}
	sort.Strings(names)

	// Count generated
	generated := 0
	for _, name := range names {
		entry := catalog.Tools[name]
		if entry.Generated {
			generated++
		}
	}

	fmt.Printf("Catalog: %d tools (%d with completions generated)\n\n", len(names), generated)

	if !showAll && len(names) > 50 {
		// Show just generated tools and first 20
		fmt.Println("Generated completions:")
		hasGenerated := false
		for _, name := range names {
			entry := catalog.Tools[name]
			if entry.Generated {
				hasGenerated = true
				fmt.Printf("  ✓ %s\n", name)
			}
		}
		if !hasGenerated {
			fmt.Println("  (none yet)")
		}

		fmt.Println("\nFirst 20 tools in catalog:")
		for i, name := range names {
			if i >= 20 {
				break
			}
			entry := catalog.Tools[name]
			status := " "
			if entry.Generated {
				status = "✓"
			}
			fmt.Printf("  [%s] %s\n", status, name)
		}
		fmt.Printf("\n... and %d more. Use 'tabgen list --all' to see all.\n", len(names)-20)
	} else {
		for _, name := range names {
			entry := catalog.Tools[name]
			status := " "
			if entry.Generated {
				status = "✓"
			}
			fmt.Printf("  [%s] %s\n", status, name)
		}
	}

	return nil
}
