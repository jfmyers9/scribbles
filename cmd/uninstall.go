package cmd

import (
	"fmt"
	"os"

	"github.com/jfmyers9/scribbles/internal/daemon"
	"github.com/spf13/cobra"
)

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall scribbles daemon from launchd",
	Long: `Uninstall scribbles daemon from launchd and stop it from running automatically.

This command will:
  - Stop the running daemon (if any)
  - Unload the daemon from launchd
  - Remove the plist file from ~/Library/LaunchAgents/

After uninstalling, the daemon will no longer run automatically on login.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get plist path
		plistPath, err := daemon.GetPlistPath()
		if err != nil {
			return fmt.Errorf("failed to get plist path: %w", err)
		}

		// Check if plist exists
		if _, err := os.Stat(plistPath); os.IsNotExist(err) {
			fmt.Println("Daemon is not installed (plist not found)")
			return nil
		}

		// Unload the daemon
		fmt.Println("Stopping daemon...")
		if err := unloadDaemon(); err != nil {
			fmt.Printf("Warning: failed to unload daemon: %v\n", err)
			fmt.Println("Continuing with plist removal...")
		} else {
			fmt.Println("✓ Daemon stopped")
		}

		// Remove plist file
		if err := os.Remove(plistPath); err != nil {
			return fmt.Errorf("failed to remove plist file: %w", err)
		}

		fmt.Printf("✓ Removed plist from %s\n", plistPath)
		fmt.Println("\nThe scribbles daemon has been uninstalled successfully.")
		fmt.Println("It will no longer run automatically on login.")
		fmt.Println("\nTo reinstall, run:")
		fmt.Println("  scribbles install")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
