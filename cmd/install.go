package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jfmyers9/scribbles/internal/daemon"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install scribbles daemon as a launchd agent",
	Long: `Install scribbles daemon as a launchd agent that runs automatically on login.

This command will:
  - Generate a launchd plist file for the scribbles daemon
  - Install it to ~/Library/LaunchAgents/
  - Load the agent with launchctl
  - Start the daemon automatically

The daemon will run in the background and automatically scrobble tracks
from Apple Music to Last.fm.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the path to the current executable
		binaryPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		// Resolve symlinks to get the actual binary path
		binaryPath, err = filepath.EvalSymlinks(binaryPath)
		if err != nil {
			return fmt.Errorf("failed to resolve executable path: %w", err)
		}

		// Get the log path
		logPath, err := daemon.GetDefaultLogPath()
		if err != nil {
			return fmt.Errorf("failed to get log path: %w", err)
		}

		// Create log directory if it doesn't exist
		if err := os.MkdirAll(logPath, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Get home directory for working directory
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		// Generate plist
		config := daemon.PlistConfig{
			BinaryPath:       binaryPath,
			LogPath:          logPath,
			WorkingDirectory: home,
		}

		plistContent, err := daemon.GeneratePlist(config)
		if err != nil {
			return fmt.Errorf("failed to generate plist: %w", err)
		}

		// Get plist path
		plistPath, err := daemon.GetPlistPath()
		if err != nil {
			return fmt.Errorf("failed to get plist path: %w", err)
		}

		// Create LaunchAgents directory if it doesn't exist
		launchAgentsDir := filepath.Dir(plistPath)
		if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
			return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
		}

		// Check if plist already exists
		if _, err := os.Stat(plistPath); err == nil {
			fmt.Println("Daemon is already installed. Uninstalling first...")
			// Try to unload the existing daemon
			if err := unloadDaemon(); err != nil {
				fmt.Printf("Warning: failed to unload existing daemon: %v\n", err)
			}
		}

		// Write plist file
		if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
			return fmt.Errorf("failed to write plist file: %w", err)
		}

		fmt.Printf("✓ Installed plist to %s\n", plistPath)

		// Load the daemon with launchctl
		if err := loadDaemon(plistPath); err != nil {
			return fmt.Errorf("failed to load daemon: %w", err)
		}

		fmt.Println("✓ Daemon loaded and started successfully")
		fmt.Printf("✓ Logs will be written to %s\n", logPath)
		fmt.Println("\nThe scribbles daemon is now running and will start automatically on login.")
		fmt.Println("\nYou can check the daemon status with:")
		fmt.Println("  launchctl list | grep scribbles")
		fmt.Println("\nTo uninstall, run:")
		fmt.Println("  scribbles uninstall")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

// loadDaemon loads the daemon using launchctl
func loadDaemon(plistPath string) error {
	// Get current user ID for launchctl bootstrap
	uidCmd := exec.Command("id", "-u")
	uidOutput, err := uidCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}
	uid := string(uidOutput)
	uid = uid[:len(uid)-1] // Remove trailing newline

	// Use launchctl bootstrap to load the agent
	domain := fmt.Sprintf("gui/%s", uid)
	cmd := exec.Command("launchctl", "bootstrap", domain, plistPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if already loaded
		if len(output) > 0 {
			outputStr := string(output)
			// Bootstrap returns error if already loaded, which is OK
			if len(outputStr) > 0 {
				return fmt.Errorf("launchctl bootstrap failed: %s", outputStr)
			}
		}
		return fmt.Errorf("failed to run launchctl bootstrap: %w", err)
	}

	return nil
}

// unloadDaemon unloads the daemon using launchctl
func unloadDaemon() error {
	// Get current user ID for launchctl bootout
	uidCmd := exec.Command("id", "-u")
	uidOutput, err := uidCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}
	uid := string(uidOutput)
	uid = uid[:len(uid)-1] // Remove trailing newline

	// Use launchctl bootout to unload the agent
	domain := fmt.Sprintf("gui/%s", uid)
	serviceName := fmt.Sprintf("%s/com.scribbles.daemon", domain)
	cmd := exec.Command("launchctl", "bootout", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Bootout may fail if not loaded, which is OK
		if len(output) > 0 {
			outputStr := string(output)
			// Ignore "Could not find service" errors
			if len(outputStr) > 0 {
				fmt.Printf("Warning: %s\n", outputStr)
			}
		}
	}

	return nil
}
