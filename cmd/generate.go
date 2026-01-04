package cmd

import (
	"fmt"

	"github.com/justin/tabgen/internal/config"
	"github.com/justin/tabgen/internal/generator"
	"github.com/justin/tabgen/internal/parser"
)

// GenerateOptions configures the generate command
type GenerateOptions struct {
	Tool  string // Specific tool to generate (empty = all)
	Force bool   // Force regeneration even if up-to-date
}

// Generate creates completion scripts for one or all tools
func Generate(opts GenerateOptions) error {
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

	p := parser.New()
	bashGen := generator.NewBash()
	zshGen := generator.NewZsh()

	// Determine which tools to generate
	var tools []string
	if opts.Tool != "" {
		if _, ok := catalog.Tools[opts.Tool]; !ok {
			return fmt.Errorf("tool %q not found in catalog. Run 'tabgen scan' first.", opts.Tool)
		}
		tools = []string{opts.Tool}
	} else {
		// Generate for all tools (parser will skip unparseable ones)
		for name := range catalog.Tools {
			tools = append(tools, name)
		}
	}

	if len(tools) == 0 {
		fmt.Println("No tools in catalog. Run 'tabgen scan' first.")
		return nil
	}

	fmt.Printf("Processing %d tools...\n", len(tools))

	succeeded := 0
	skipped := 0
	failed := 0

	for _, name := range tools {
		entry := catalog.Tools[name]

		// Parse the tool (also detects version)
		tool, err := p.Parse(name, entry.Path)
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", name, err)
			failed++
			continue
		}

		// Skip tools we couldn't parse
		if tool.Source == "none" {
			continue
		}

		// Check if we can skip (already generated with same version)
		if !opts.Force && entry.Generated && entry.GeneratedVersion != "" {
			if entry.GeneratedVersion == tool.Version {
				skipped++
				continue
			}
			// Version changed, will regenerate
			fmt.Printf("  ↻ %s: version changed (%s → %s)\n", name, entry.GeneratedVersion, tool.Version)
		}

		// Save parsed tool data
		if err := storage.SaveTool(tool); err != nil {
			fmt.Printf("  ✗ %s: failed to save: %v\n", name, err)
			failed++
			continue
		}

		// Generate bash completion
		bashScript := bashGen.Generate(tool)
		if err := storage.SaveBashCompletion(name, bashScript); err != nil {
			fmt.Printf("  ✗ %s: failed to save bash completion: %v\n", name, err)
			failed++
			continue
		}

		// Generate zsh completion
		zshScript := zshGen.Generate(tool)
		if err := storage.SaveZshCompletion(name, zshScript); err != nil {
			fmt.Printf("  ✗ %s: failed to save zsh completion: %v\n", name, err)
			failed++
			continue
		}

		// Update catalog with version info
		entry.Generated = true
		entry.Version = tool.Version
		entry.GeneratedVersion = tool.Version
		catalog.Tools[name] = entry

		if tool.Version != "" {
			fmt.Printf("  ✓ %s (v%s)\n", name, tool.Version)
		} else {
			fmt.Printf("  ✓ %s\n", name)
		}
		succeeded++
	}

	// Save updated catalog
	if err := storage.SaveCatalog(catalog); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Printf("\nDone: %d generated, %d skipped (up-to-date), %d failed\n", succeeded, skipped, failed)

	if succeeded > 0 {
		bashDir, zshDir := storage.CompletionPaths()
		fmt.Printf("\nCompletions saved to:\n")
		fmt.Printf("  Bash: %s\n", bashDir)
		fmt.Printf("  Zsh:  %s\n", zshDir)
	}

	return nil
}
