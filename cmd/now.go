/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/jfmyers9/scribbles/internal/config"
	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/mattn/go-runewidth"
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
	// Add width flag to set fixed output width
	nowCmd.Flags().IntP("width", "w", 0, "Fixed output width (0=disabled, overrides config)")
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

	// Apply width padding if requested
	width, _ := cmd.Flags().GetInt("width")
	if width == 0 {
		width = cfg.OutputWidth
	}
	if width > 0 {
		output = padToWidth(output, width)
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

// padToWidth pads or truncates text to a fixed display width.
// Width is measured in display columns, accounting for Unicode characters.
// If width <= 0, returns text unchanged.
// If text is longer than width, truncates with "..." suffix.
// If text is shorter than width, pads with spaces.
func padToWidth(text string, width int) string {
	if width <= 0 {
		return text // no padding requested
	}

	currentWidth := runewidth.StringWidth(text)

	if currentWidth > width {
		// Truncate with "..." suffix
		// We need to manually truncate and add "..." then pad if needed
		ellipsis := "..."
		ellipsisWidth := runewidth.StringWidth(ellipsis)

		if width <= ellipsisWidth {
			// If width is too small, just return ellipsis truncated to width
			return runewidth.Truncate(ellipsis, width, "")
		}

		// Truncate to (width - ellipsisWidth) and add ellipsis
		truncated := runewidth.Truncate(text, width-ellipsisWidth, "")
		result := truncated + ellipsis

		// Ensure we're exactly at the target width (in case truncate was imprecise)
		resultWidth := runewidth.StringWidth(result)
		if resultWidth < width {
			padding := strings.Repeat(" ", width-resultWidth)
			return result + padding
		} else if resultWidth > width {
			// Shouldn't happen, but handle it just in case
			return runewidth.Truncate(result, width, "")
		}
		return result
	} else if currentWidth < width {
		// Pad with spaces
		padding := strings.Repeat(" ", width-currentWidth)
		return text + padding
	}

	return text // exactly the right width
}
