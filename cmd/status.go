package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jvalentini/tabgen/internal/config"
)

// Status shows the current state of TabGen installation
func Status() error {
	storage, err := config.New("")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	fmt.Println("TabGen Status")
	fmt.Println("=============")
	fmt.Println()

	// Data directory
	baseDir := storage.BaseDir()
	fmt.Printf("Data directory: %s\n", baseDir)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		fmt.Println("  Status: Not initialized (run 'tabgen scan' first)")
		return nil
	}
	fmt.Println("  Status: OK")
	fmt.Println()

	// Catalog info
	catalog, err := storage.LoadCatalog()
	if err != nil {
		fmt.Printf("Catalog: Error loading (%v)\n", err)
	} else {
		generated := 0
		for _, entry := range catalog.Tools {
			if entry.Generated {
				generated++
			}
		}
		fmt.Printf("Catalog: %d tools discovered, %d with completions\n", len(catalog.Tools), generated)
		if !catalog.LastScan.IsZero() {
			age := time.Since(catalog.LastScan)
			fmt.Printf("  Last scan: %s (%s ago)\n", catalog.LastScan.Format("2006-01-02 15:04"), formatDuration(age))
		}
	}
	fmt.Println()

	// Completion directories
	bashDir, zshDir := storage.CompletionPaths()
	bashCount := countFiles(bashDir)
	zshCount := countFiles(zshDir)
	fmt.Printf("Completions:\n")
	fmt.Printf("  Bash: %d files in %s\n", bashCount, bashDir)
	fmt.Printf("  Zsh:  %d files in %s\n", zshCount, zshDir)
	fmt.Println()

	// Symlinks
	fmt.Println("Installation:")
	checkSymlink(filepath.Join(home, ".local", "share", "bash-completion", "completions", "tabgen-completions"), "Bash symlink")
	checkSymlink(filepath.Join(home, ".zfunc", "tabgen-completions"), "Zsh symlink")

	// Timer/Cron
	checkTimer(home)

	// Shell hooks
	checkShellHook(filepath.Join(home, ".bashrc"), "Bash hook")
	checkShellHook(filepath.Join(home, ".zshrc"), "Zsh hook")

	return nil
}

// countFiles counts files in a directory
func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count
}

// checkSymlink checks if a symlink exists and is valid
func checkSymlink(path, name string) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		fmt.Printf("  [✗] %s: not installed\n", name)
		return
	}
	if err != nil {
		fmt.Printf("  [?] %s: error (%v)\n", name, err)
		return
	}
	if info.Mode()&os.ModeSymlink == 0 {
		fmt.Printf("  [!] %s: exists but not a symlink\n", name)
		return
	}

	// Check if target exists
	target, err := os.Readlink(path)
	if err != nil {
		fmt.Printf("  [!] %s: broken symlink\n", name)
		return
	}
	if _, err := os.Stat(target); os.IsNotExist(err) {
		fmt.Printf("  [!] %s: broken symlink (target missing)\n", name)
		return
	}

	fmt.Printf("  [✓] %s: %s\n", name, path)
}

// checkTimer checks for systemd timer or cron job
func checkTimer(home string) {
	// Check systemd timer
	timerPath := filepath.Join(home, ".config", "systemd", "user", "tabgen-scan.timer")
	if _, err := os.Stat(timerPath); err == nil {
		// Check if active
		cmd := exec.Command("systemctl", "--user", "is-active", "tabgen-scan.timer")
		output, _ := cmd.Output()
		status := strings.TrimSpace(string(output))
		if status == "active" {
			fmt.Printf("  [✓] Systemd timer: active\n")
		} else {
			fmt.Printf("  [!] Systemd timer: installed but %s\n", status)
		}
		return
	}

	// Check cron
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "# tabgen daily scan") {
		fmt.Printf("  [✓] Cron job: installed\n")
		return
	}

	fmt.Printf("  [✗] Timer/Cron: not installed\n")
}

// checkShellHook checks if a shell hook is installed
func checkShellHook(path, name string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("  [✗] %s: %s not found\n", name, filepath.Base(path))
		return
	}

	if strings.Contains(string(data), "# TabGen completions") {
		fmt.Printf("  [✓] %s: installed in %s\n", name, filepath.Base(path))
	} else {
		fmt.Printf("  [✗] %s: not found in %s\n", name, filepath.Base(path))
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}
