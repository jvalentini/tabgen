package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jvalentini/tabgen/internal/config"
)

// Uninstall removes TabGen: symlinks, timers, shell hooks, and optionally data
func Uninstall(keepData bool) error {
	storage, err := config.New("")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	fmt.Println("Uninstalling TabGen...")

	// Step 1: Remove symlinks
	removeSymlinks(home)

	// Step 2: Remove timer/cron
	removeTimer(home)

	// Step 3: Remove shell hooks
	removeShellHooks(home)

	// Step 4: Remove data if requested
	if !keepData {
		baseDir := storage.BaseDir()
		if err := os.RemoveAll(baseDir); err != nil {
			fmt.Printf("Warning: failed to remove data directory: %v\n", err)
		} else {
			fmt.Printf("  ✓ Removed data directory: %s\n", baseDir)
		}
	} else {
		fmt.Printf("  ℹ Data preserved at: %s\n", storage.BaseDir())
	}

	fmt.Println("\nUninstall complete!")
	fmt.Println("Restart your shell to fully remove TabGen completions.")

	return nil
}

// removeSymlinks removes TabGen symlinks
func removeSymlinks(home string) {
	links := []string{
		filepath.Join(home, ".local", "share", "bash-completion", "completions", "tabgen-completions"),
		filepath.Join(home, ".zfunc", "tabgen-completions"),
	}

	for _, link := range links {
		if info, err := os.Lstat(link); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				os.Remove(link)
				fmt.Printf("  ✓ Removed symlink: %s\n", link)
			}
		}
	}
}

// removeTimer removes systemd timer and cron job
func removeTimer(home string) {
	// Remove systemd timer
	userDir := filepath.Join(home, ".config", "systemd", "user")
	servicePath := filepath.Join(userDir, "tabgen-scan.service")
	timerPath := filepath.Join(userDir, "tabgen-scan.timer")

	// Stop and disable timer
	exec.Command("systemctl", "--user", "stop", "tabgen-scan.timer").Run()
	exec.Command("systemctl", "--user", "disable", "tabgen-scan.timer").Run()

	if _, err := os.Stat(timerPath); err == nil {
		os.Remove(timerPath)
		fmt.Println("  ✓ Removed systemd timer")
	}
	if _, err := os.Stat(servicePath); err == nil {
		os.Remove(servicePath)
		fmt.Println("  ✓ Removed systemd service")
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()

	// Remove cron job
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	currentCron := string(output)
	if !strings.Contains(currentCron, "# tabgen daily scan") {
		return
	}

	// Filter out our cron line
	var newLines []string
	for _, line := range strings.Split(currentCron, "\n") {
		if !strings.Contains(line, "# tabgen daily scan") {
			newLines = append(newLines, line)
		}
	}

	newCron := strings.Join(newLines, "\n")
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCron)
	if err := cmd.Run(); err == nil {
		fmt.Println("  ✓ Removed cron job")
	}
}

// removeShellHooks removes TabGen hooks from shell config files
func removeShellHooks(home string) {
	removeHookFromFile(filepath.Join(home, ".bashrc"), "# TabGen completions")
	removeHookFromFile(filepath.Join(home, ".zshrc"), "# TabGen completions")
}

// removeHookFromFile removes a marked section from a file
func removeHookFromFile(path, marker string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := string(data)
	if !strings.Contains(content, marker) {
		return
	}

	// Read line by line and skip the TabGen block
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	inBlock := false
	blockLines := 0

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, marker) {
			inBlock = true
			blockLines = 0
			continue
		}

		if inBlock {
			blockLines++
			// Skip the next few lines of the block (typically 4-5 lines)
			if blockLines <= 5 && (strings.HasPrefix(strings.TrimSpace(line), "if") ||
				strings.HasPrefix(strings.TrimSpace(line), "for") ||
				strings.HasPrefix(strings.TrimSpace(line), "[") ||
				strings.HasPrefix(strings.TrimSpace(line), "fpath") ||
				strings.HasPrefix(strings.TrimSpace(line), "autoload") ||
				strings.HasPrefix(strings.TrimSpace(line), "source") ||
				strings.HasPrefix(strings.TrimSpace(line), "done") ||
				strings.HasPrefix(strings.TrimSpace(line), "fi") ||
				line == "") {
				continue
			}
			inBlock = false
		}

		result = append(result, line)
	}

	newContent := strings.Join(result, "\n")
	if newContent != content {
		os.WriteFile(path, []byte(newContent), 0644)
		fmt.Printf("  ✓ Removed hook from %s\n", filepath.Base(path))
	}
}
