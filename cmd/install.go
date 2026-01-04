package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/justin/tabgen/internal/config"
)

// Install sets up TabGen: symlinks, timers, and shell hooks
func Install(skipTimer bool) error {
	storage, err := config.New("")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	fmt.Println("Installing TabGen...")

	// Step 1: Create symlinks for completions
	if err := installSymlinks(storage, home); err != nil {
		return err
	}

	// Step 2: Set up timer/cron for daily scans
	if !skipTimer {
		if err := installTimer(storage, home); err != nil {
			fmt.Printf("Warning: failed to set up timer: %v\n", err)
			fmt.Println("You can run 'tabgen scan' manually instead.")
		}
	}

	// Step 3: Create shell hooks
	if err := installShellHooks(storage, home); err != nil {
		return err
	}

	fmt.Println("\nInstallation complete!")
	fmt.Println("\nTo activate completions, restart your shell or run:")
	fmt.Println("  source ~/.bashrc  # for bash")
	fmt.Println("  source ~/.zshrc   # for zsh")

	return nil
}

// installSymlinks creates symlinks from standard completion dirs to TabGen's
func installSymlinks(storage *config.Storage, home string) error {
	bashSrc, zshSrc := storage.CompletionPaths()

	// Bash completion directory
	bashDest := filepath.Join(home, ".local", "share", "bash-completion", "completions")
	if err := os.MkdirAll(bashDest, 0755); err != nil {
		return fmt.Errorf("failed to create bash completion dir: %w", err)
	}

	// Create a symlink for each completion file (or a source file)
	bashLink := filepath.Join(bashDest, "tabgen-completions")
	if err := createSymlink(bashSrc, bashLink); err != nil {
		fmt.Printf("Warning: could not create bash symlink: %v\n", err)
	} else {
		fmt.Printf("  ✓ Bash completions linked: %s\n", bashLink)
	}

	// Zsh completion directory
	zshDest := filepath.Join(home, ".zfunc")
	if err := os.MkdirAll(zshDest, 0755); err != nil {
		return fmt.Errorf("failed to create zsh completion dir: %w", err)
	}

	zshLink := filepath.Join(zshDest, "tabgen-completions")
	if err := createSymlink(zshSrc, zshLink); err != nil {
		fmt.Printf("Warning: could not create zsh symlink: %v\n", err)
	} else {
		fmt.Printf("  ✓ Zsh completions linked: %s\n", zshLink)
	}

	return nil
}

// createSymlink creates or updates a symlink
func createSymlink(src, dest string) error {
	// Remove existing symlink or file
	if _, err := os.Lstat(dest); err == nil {
		os.Remove(dest)
	}
	return os.Symlink(src, dest)
}

// installTimer sets up systemd user timer or cron
func installTimer(storage *config.Storage, home string) error {
	// Check if systemd user instance is available
	if hasSystemdUser() {
		return installSystemdTimer(storage, home)
	}

	// Fall back to cron
	return installCron(storage)
}

// hasSystemdUser checks if systemd user instance is available
func hasSystemdUser() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	cmd := exec.Command("systemctl", "--user", "status")
	err := cmd.Run()
	return err == nil
}

// installSystemdTimer installs a systemd user timer
func installSystemdTimer(storage *config.Storage, home string) error {
	userDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return err
	}

	// Get the tabgen binary path
	tabgenPath, err := os.Executable()
	if err != nil {
		tabgenPath = "tabgen" // Fall back to PATH lookup
	}

	// Write service file
	serviceContent := fmt.Sprintf(`[Unit]
Description=TabGen completion scanner

[Service]
Type=oneshot
ExecStart=%s scan
`, tabgenPath)

	servicePath := filepath.Join(userDir, "tabgen-scan.service")
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return err
	}

	// Write timer file
	timerContent := `[Unit]
Description=Daily TabGen scan

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
`
	timerPath := filepath.Join(userDir, "tabgen-scan.timer")
	if err := os.WriteFile(timerPath, []byte(timerContent), 0644); err != nil {
		return err
	}

	// Enable and start the timer
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd user daemon: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "enable", "tabgen-scan.timer").Run(); err != nil {
		return fmt.Errorf("failed to enable tabgen-scan.timer: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "start", "tabgen-scan.timer").Run(); err != nil {
		return fmt.Errorf("failed to start tabgen-scan.timer: %w", err)
	}

	fmt.Println("  ✓ Systemd timer installed (daily scan)")
	return nil
}

// installCron adds a cron job for daily scanning
func installCron(storage *config.Storage) error {
	tabgenPath, err := os.Executable()
	if err != nil {
		tabgenPath = "tabgen"
	}

	cronLine := fmt.Sprintf("0 4 * * * %s scan >/dev/null 2>&1 # tabgen daily scan\n", tabgenPath)

	// Get current crontab
	cmd := exec.Command("crontab", "-l")
	output, _ := cmd.Output()
	currentCron := string(output)

	// Check if already installed
	if strings.Contains(currentCron, "# tabgen daily scan") {
		fmt.Println("  ✓ Cron job already installed")
		return nil
	}

	// Add our line
	newCron := currentCron + cronLine

	// Install new crontab
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCron)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install cron job: %w", err)
	}

	fmt.Println("  ✓ Cron job installed (daily scan at 4am)")
	return nil
}

// installShellHooks adds shell startup hooks
func installShellHooks(storage *config.Storage, home string) error {
	bashSrc, zshSrc := storage.CompletionPaths()

	// Bash hook
	bashrcPath := filepath.Join(home, ".bashrc")
	bashHook := fmt.Sprintf(`
# TabGen completions
if [ -d "%s" ]; then
    for f in "%s"/*; do
        [ -f "$f" ] && source "$f"
    done
fi
`, bashSrc, bashSrc)

	if err := appendIfNotPresent(bashrcPath, bashHook, "# TabGen completions"); err != nil {
		fmt.Printf("Warning: could not update .bashrc: %v\n", err)
	} else {
		fmt.Println("  ✓ Bash hook added to ~/.bashrc")
	}

	// Zsh hook
	zshrcPath := filepath.Join(home, ".zshrc")
	zshHook := fmt.Sprintf(`
# TabGen completions
if [ -d "%s" ]; then
    fpath=("%s" $fpath)
    autoload -Uz compinit && compinit -C
fi
`, zshSrc, zshSrc)

	if err := appendIfNotPresent(zshrcPath, zshHook, "# TabGen completions"); err != nil {
		fmt.Printf("Warning: could not update .zshrc: %v\n", err)
	} else {
		fmt.Println("  ✓ Zsh hook added to ~/.zshrc")
	}

	return nil
}

// appendIfNotPresent appends content to a file if marker is not present
func appendIfNotPresent(path, content, marker string) error {
	// Read existing content
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if already present
	if strings.Contains(string(existing), marker) {
		return nil // Already installed
	}

	// Append content
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}
