/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/jfmyers9/scribbles/internal/config"
	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/spf13/cobra"
)

// nowCmd represents the now command
var nowCmd = &cobra.Command{
	Use:   "now",
	Short: "Display currently playing track from Apple Music",
	Long: `Query Apple Music and display the currently playing track.

The output format can be customized in ~/.config/scribbles/config.yaml
using a Go template. Available fields: .Name, .Artist, .Album, .Duration, .Position

Exit codes:
  0 - Track is currently playing
  1 - No track playing, paused, or Music app not running`,
	RunE: runNow,
}

func init() {
	rootCmd.AddCommand(nowCmd)

	// Add format flag to override config
	nowCmd.Flags().StringP("format", "f", "", "Output format template (overrides config)")
}

func runNow(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check for format flag override
	formatFlag, _ := cmd.Flags().GetString("format")
	if formatFlag != "" {
		cfg.OutputFormat = formatFlag
	}

	// Create music client
	client := music.NewAppleScriptClient()

	// Get current track
	track, err := client.GetCurrentTrack(ctx)
	if err != nil {
		// If Music app is not running or other error, exit with code 1
		return fmt.Errorf("failed to get current track: %w", err)
	}

	// If not playing, exit with code 1
	if track.State != music.StatePlaying {
		os.Exit(1)
		return nil
	}

	// Format and print output
	output, err := formatTrack(track, cfg.OutputFormat)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Println(output)
	return nil
}

// formatTrack applies the template to the track data
func formatTrack(track *music.Track, templateStr string) (string, error) {
	tmpl, err := template.New("output").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("invalid template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, track); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}
