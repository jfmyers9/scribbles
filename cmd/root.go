/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information (set via ldflags during build)
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "scribbles",
	Short: "Apple Music scrobbler for Last.fm",
	Long: `scribbles is an Apple Music scrobbler for Last.fm.

It runs as a background daemon that monitors Apple Music playback
and scrobbles tracks to Last.fm according to Last.fm's scrobbling rules.

It also provides a CLI command to query the currently playing track,
useful for displaying in tmux status lines or other status bars.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildDate),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here if needed
}


