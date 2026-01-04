package scanner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jvalentini/tabgen/internal/types"
)

// Scanner discovers executables in $PATH
type Scanner struct {
	excludePatterns []string
	quickMode       bool // Skip --help and man checks during scan
}

// New creates a new Scanner (quick mode by default)
func New(excluded []string) *Scanner {
	return &Scanner{excludePatterns: excluded, quickMode: true}
}

// NewFull creates a Scanner that checks --help and man pages (slower)
func NewFull(excluded []string) *Scanner {
	s := New(excluded)
	s.quickMode = false
	return s
}

// isExcluded checks if a name matches any exclusion pattern
func (s *Scanner) isExcluded(name string) (bool, error) {
	for _, pattern := range s.excludePatterns {
		// Try glob match first
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return false, fmt.Errorf("invalid exclusion pattern %q: %w", pattern, err)
		}
		if matched {
			return true, nil
		}
		// Also try exact match
		if pattern == name {
			return true, nil
		}
	}
	return false, nil
}

// Scan walks $PATH and returns a catalog of discovered tools
// Only includes tools that appear in shell history
func (s *Scanner) Scan() (*types.Catalog, error) {
	catalog := &types.Catalog{
		LastScan: time.Now(),
		Tools:    make(map[string]types.CatalogEntry),
	}

	usedCommands, err := GetUsedCommands()
	if err != nil {
		return nil, fmt.Errorf("failed to read shell history: %w", err)
	}

	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return catalog, nil
	}

	seen := make(map[string]bool)

	for dir := range strings.SplitSeq(pathEnv, string(os.PathListSeparator)) {
		if dir == "" {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()

			if seen[name] {
				continue
			}
			seen[name] = true

			excluded, err := s.isExcluded(name)
			if err != nil {
				return nil, fmt.Errorf("checking exclusion for %s: %w", name, err)
			}
			if excluded {
				continue
			}

			if strings.HasPrefix(name, ".") {
				continue
			}

			if !usedCommands[name] {
				continue
			}

			fullPath := filepath.Join(dir, name)

			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0111 == 0 {
				continue
			}

			catalogEntry := types.CatalogEntry{
				Name:      name,
				Path:      fullPath,
				Generated: false,
				LastScan:  time.Now(),
			}

			if !s.quickMode {
				hasHelp, helpErr := s.checkHelp(fullPath)
				if helpErr != nil {
					return nil, fmt.Errorf("checking help for %s: %w", name, helpErr)
				}
				catalogEntry.HasHelp = hasHelp

				hasMan, manErr := s.checkManPage(name)
				if manErr != nil {
					return nil, fmt.Errorf("checking man page for %s: %w", name, manErr)
				}
				catalogEntry.HasManPage = hasMan
			}

			catalog.Tools[name] = catalogEntry
		}
	}

	return catalog, nil
}

// checkHelp tests if a tool responds to --help
// Returns (hasHelp, error) - error is non-nil only for permission-related failures
func (s *Scanner) checkHelp(path string) (bool, error) {
	cmd := exec.Command(path, "--help")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	err := cmd.Run()
	if err != nil {
		// Check for permission errors - these should be surfaced
		if isPermissionError(err) {
			return false, fmt.Errorf("permission denied running %s --help: %w", path, err)
		}
		// Many tools return non-zero for --help but still provide help
		// If the process ran (ProcessState exists), treat as success
		if cmd.ProcessState != nil {
			return true, nil
		}
		// Tool doesn't support --help - not an error, just no help
		return false, nil
	}
	return true, nil
}

// checkManPage tests if a man page exists for a tool
// Returns (hasManPage, error) - error is non-nil only for permission-related failures
func (s *Scanner) checkManPage(name string) (bool, error) {
	cmd := exec.Command("man", "-w", name)
	err := cmd.Run()
	if err != nil {
		// Check for permission errors
		if isPermissionError(err) {
			return false, fmt.Errorf("permission denied checking man page for %s: %w", name, err)
		}
		// No man page exists - not an error
		return false, nil
	}
	return true, nil
}

// isPermissionError checks if an error is a permission-related error
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	// Check for os.ErrPermission
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	// Check for exec errors that indicate permission issues
	if errors.Is(err, exec.ErrNotFound) {
		return false // Not found is not a permission error
	}
	// Check error message for common permission indicators
	errStr := err.Error()
	return strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "EACCES") ||
		strings.Contains(errStr, "operation not permitted")
}

// ScanSingle scans a single tool by name
func (s *Scanner) ScanSingle(name string) (*types.CatalogEntry, error) {
	// Validate input
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	path, err := exec.LookPath(name)
	if err != nil {
		return nil, fmt.Errorf("looking up %s: %w", name, err)
	}

	hasHelp, helpErr := s.checkHelp(path)
	if helpErr != nil {
		return nil, fmt.Errorf("checking help for %s: %w", name, helpErr)
	}

	hasMan, manErr := s.checkManPage(name)
	if manErr != nil {
		return nil, fmt.Errorf("checking man page for %s: %w", name, manErr)
	}

	return &types.CatalogEntry{
		Name:       name,
		Path:       path,
		Generated:  false,
		LastScan:   time.Now(),
		HasHelp:    hasHelp,
		HasManPage: hasMan,
	}, nil
}
