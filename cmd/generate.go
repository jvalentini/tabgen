package cmd

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/justin/tabgen/internal/config"
	"github.com/justin/tabgen/internal/generator"
	"github.com/justin/tabgen/internal/parser"
	"github.com/justin/tabgen/internal/types"
)

// GenerateOptions configures the generate command
type GenerateOptions struct {
	Tool    string // Specific tool to generate (empty = all)
	Force   bool   // Force regeneration even if up-to-date
	Workers int    // Number of concurrent workers (default: NumCPU)
}

// toolResult holds the outcome of processing a single tool
type toolResult struct {
	Name             string
	Status           string // "success", "skipped", "failed"
	Version          string
	GeneratedVersion string
	ContentHash      string // Hash of parsed tool content
	Error            error
	Message          string
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

	// Set default workers
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	// Don't use more workers than tools
	if workers > len(tools) {
		workers = len(tools)
	}

	// Create channels
	toolChan := make(chan string, len(tools))
	resultChan := make(chan toolResult, len(tools))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processTools(toolChan, resultChan, catalog, storage, opts.Force)
		}()
	}

	// Send tools to workers
	for _, name := range tools {
		toolChan <- name
	}
	close(toolChan)

	// Wait for workers to finish, then close results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	succeeded := 0
	skipped := 0
	failed := 0

	catalogUpdates := make(map[string]types.CatalogEntry)

	for result := range resultChan {
		switch result.Status {
		case "success":
			if result.Version != "" {
				fmt.Printf("  ✓ %s (v%s)\n", result.Name, result.Version)
			} else {
				fmt.Printf("  ✓ %s\n", result.Name)
			}
			succeeded++
			// Queue catalog update
			entry := catalog.Tools[result.Name]
			entry.Generated = true
			entry.Version = result.Version
			entry.GeneratedVersion = result.GeneratedVersion
			entry.ContentHash = result.ContentHash
			catalogUpdates[result.Name] = entry
		case "skipped":
			skipped++
		case "failed":
			fmt.Printf("  ✗ %s: %v\n", result.Name, result.Error)
			failed++
		case "version_changed", "hash_changed":
			fmt.Printf("  ↻ %s: %s\n", result.Name, result.Message)
			if result.Version != "" {
				fmt.Printf("  ✓ %s (v%s)\n", result.Name, result.Version)
			} else {
				fmt.Printf("  ✓ %s\n", result.Name)
			}
			succeeded++
			// Queue catalog update
			entry := catalog.Tools[result.Name]
			entry.Generated = true
			entry.Version = result.Version
			entry.GeneratedVersion = result.GeneratedVersion
			entry.ContentHash = result.ContentHash
			catalogUpdates[result.Name] = entry
		}
	}

	// Apply catalog updates
	for name, entry := range catalogUpdates {
		catalog.Tools[name] = entry
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

// processTools is the worker function that processes tools from the input channel
func processTools(toolChan <-chan string, resultChan chan<- toolResult, catalog *types.Catalog, storage *config.Storage, force bool) {
	p := parser.New()
	bashGen := generator.NewBash()
	zshGen := generator.NewZsh()

	for name := range toolChan {
		entry := catalog.Tools[name]
		result := toolResult{Name: name}

		// Parse the tool (also detects version)
		tool, err := p.Parse(name, entry.Path)
		if err != nil {
			result.Status = "failed"
			result.Error = err
			resultChan <- result
			continue
		}

		// Skip tools we couldn't parse
		if tool.Source == "none" {
			continue
		}

		// Compute content hash for cache invalidation
		contentHash := tool.ContentHash()

		// Check if we can skip (already generated with same version AND content hash)
		if !force && entry.Generated && entry.GeneratedVersion != "" {
			versionMatch := entry.GeneratedVersion == tool.Version
			hashMatch := entry.ContentHash != "" && entry.ContentHash == contentHash

			if versionMatch && hashMatch {
				result.Status = "skipped"
				resultChan <- result
				continue
			}

			// Explain why we're regenerating
			if !versionMatch {
				result.Status = "version_changed"
				result.Message = fmt.Sprintf("version changed (%s → %s)", entry.GeneratedVersion, tool.Version)
			} else if !hashMatch {
				result.Status = "hash_changed"
				result.Message = "help output changed"
			}
		} else {
			result.Status = "success"
		}

		// Save parsed tool data
		if err := storage.SaveTool(tool); err != nil {
			result.Status = "failed"
			result.Error = fmt.Errorf("failed to save: %w", err)
			resultChan <- result
			continue
		}

		// Generate bash completion
		bashScript := bashGen.Generate(tool)
		if err := storage.SaveBashCompletion(name, bashScript); err != nil {
			result.Status = "failed"
			result.Error = fmt.Errorf("failed to save bash completion: %w", err)
			resultChan <- result
			continue
		}

		// Generate zsh completion
		zshScript := zshGen.Generate(tool)
		if err := storage.SaveZshCompletion(name, zshScript); err != nil {
			result.Status = "failed"
			result.Error = fmt.Errorf("failed to save zsh completion: %w", err)
			resultChan <- result
			continue
		}

		result.Version = tool.Version
		result.GeneratedVersion = tool.Version
		result.ContentHash = contentHash
		resultChan <- result
	}
}
