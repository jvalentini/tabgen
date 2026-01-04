package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/justin/tabgen/internal/types"
)

// Scanner discovers executables in $PATH
type Scanner struct {
	excluded  map[string]bool
	quickMode bool // Skip --help and man checks during scan
}

// New creates a new Scanner (quick mode by default)
func New(excluded []string) *Scanner {
	ex := make(map[string]bool)
	for _, name := range excluded {
		ex[name] = true
	}
	return &Scanner{excluded: ex, quickMode: true}
}

// NewFull creates a Scanner that checks --help and man pages (slower)
func NewFull(excluded []string) *Scanner {
	s := New(excluded)
	s.quickMode = false
	return s
}

// Scan walks $PATH and returns a catalog of discovered tools
func (s *Scanner) Scan() (*types.Catalog, error) {
	catalog := &types.Catalog{
		LastScan: time.Now(),
		Tools:    make(map[string]types.CatalogEntry),
	}

	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return catalog, nil
	}

	seen := make(map[string]bool)
	dirs := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, dir := range dirs {
		if dir == "" {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // Skip unreadable directories
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()

			// Skip if already seen (earlier PATH entries take precedence)
			if seen[name] {
				continue
			}
			seen[name] = true

			// Skip excluded tools
			if s.excluded[name] {
				continue
			}

			// Skip hidden files
			if strings.HasPrefix(name, ".") {
				continue
			}

			fullPath := filepath.Join(dir, name)

			// Check if executable
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0111 == 0 {
				continue // Not executable
			}

			entry := types.CatalogEntry{
				Name:      name,
				Path:      fullPath,
				Generated: false,
				LastScan:  time.Now(),
			}

			// In quick mode, skip expensive --help and man checks
			if !s.quickMode {
				entry.HasHelp = s.checkHelp(fullPath)
				entry.HasManPage = s.checkManPage(name)
			}

			catalog.Tools[name] = entry
		}
	}

	return catalog, nil
}

// checkHelp tests if a tool responds to --help
func (s *Scanner) checkHelp(path string) bool {
	cmd := exec.Command(path, "--help")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	err := cmd.Run()
	// Many tools return non-zero for --help but still provide help
	// We just check if it doesn't completely fail
	return err == nil || cmd.ProcessState != nil
}

// checkManPage tests if a man page exists for a tool
func (s *Scanner) checkManPage(name string) bool {
	cmd := exec.Command("man", "-w", name)
	err := cmd.Run()
	return err == nil
}

// ScanSingle scans a single tool by name
func (s *Scanner) ScanSingle(name string) (*types.CatalogEntry, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, err
	}

	hasHelp := s.checkHelp(path)
	hasMan := s.checkManPage(name)

	return &types.CatalogEntry{
		Name:       name,
		Path:       path,
		Generated:  false,
		LastScan:   time.Now(),
		HasHelp:    hasHelp,
		HasManPage: hasMan,
	}, nil
}
