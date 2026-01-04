package cmd

import (
	"fmt"

	"github.com/justin/tabgen/internal/config"
	"github.com/justin/tabgen/internal/generator"
	"github.com/justin/tabgen/internal/parser"
)

// Generate creates completion scripts for one or all tools
func Generate(toolName string) error {
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
	if toolName != "" {
		if _, ok := catalog.Tools[toolName]; !ok {
			return fmt.Errorf("tool %q not found in catalog. Run 'tabgen scan' first.", toolName)
		}
		tools = []string{toolName}
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

	fmt.Printf("Generating completions for %d tools...\n", len(tools))

	succeeded := 0
	failed := 0

	for _, name := range tools {
		entry := catalog.Tools[name]

		// Parse the tool
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

		// Update catalog
		entry.Generated = true
		catalog.Tools[name] = entry

		fmt.Printf("  ✓ %s\n", name)
		succeeded++
	}

	// Save updated catalog
	if err := storage.SaveCatalog(catalog); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Printf("\nDone: %d succeeded, %d failed\n", succeeded, failed)

	if succeeded > 0 {
		bashDir, zshDir := storage.CompletionPaths()
		fmt.Printf("\nCompletions saved to:\n")
		fmt.Printf("  Bash: %s\n", bashDir)
		fmt.Printf("  Zsh:  %s\n", zshDir)
		fmt.Printf("\nTo use, add to your shell config:\n")
		fmt.Printf("  Bash: source %s/<tool>\n", bashDir)
		fmt.Printf("  Zsh:  fpath=(%s $fpath); autoload -Uz compinit && compinit\n", zshDir)
	}

	return nil
}
