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
	// Add marquee flag to enable scrolling
	nowCmd.Flags().Bool("marquee", false, "Enable marquee scrolling for long text (overrides config)")
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

	// Apply width padding/marquee if requested
	width, _ := cmd.Flags().GetInt("width")
	if width == 0 {
		width = cfg.OutputWidth
	}

	marquee, _ := cmd.Flags().GetBool("marquee")
	if !marquee && !cmd.Flags().Changed("marquee") {
		// Flag not set, use config default
		marquee = cfg.MarqueeEnabled
	}

	if width > 0 {
		if marquee {
			output = marqueeText(output, width, cfg.MarqueeSpeed, cfg.MarqueeSeparator)
		} else {
			output = padToWidth(output, width)
		}
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

// extractWindow extracts a substring from text starting at startPos (in display columns)
// and returns exactly 'width' display columns. Handles Unicode characters correctly.
//
// This helper is used by marqueeText to extract a sliding window from the extended text.
// Position and width are measured in display columns (not runes), so emoji and CJK
// characters are counted by their visual width (typically 2 columns).
//
// If the extracted text is shorter than width, it's padded with spaces to ensure
// consistent output width.
func extractWindow(text string, startPos int, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(text)
	var result []rune
	currentPos := 0
	resultWidth := 0

	// Skip to start position
	for i := 0; i < len(runes) && currentPos < startPos; i++ {
		currentPos += runewidth.RuneWidth(runes[i])
	}

	// Collect runes until we reach the target width
	for i := 0; i < len(runes) && resultWidth < width; {
		// Skip past runes we've already processed to reach startPos
		runePos := 0
		for j := 0; j < i; j++ {
			runePos += runewidth.RuneWidth(runes[j])
		}

		if runePos >= startPos {
			r := runes[i]
			rw := runewidth.RuneWidth(r)

			// Don't exceed target width
			if resultWidth+rw <= width {
				result = append(result, r)
				resultWidth += rw
			} else {
				break
			}
		}
		i++
	}

	// Pad with spaces if we haven't reached target width
	if resultWidth < width {
		padding := strings.Repeat(" ", width-resultWidth)
		return string(result) + padding
	}

	return string(result)
}

// marqueeText creates a scrolling marquee effect for text that exceeds the target width.
// If text fits within width, returns static padded text.
// If text is longer, creates a scrolling window using timestamp-based positioning.
//
// Algorithm:
// 1. Create extended text: "original{separator}original" for continuous looping
// 2. Calculate scroll position: time.Now().Unix() * speed % len(extended)
//   - speed is in characters per second
//   - position wraps around to create infinite loop
//   - deterministic: same timestamp = same output (important for testing)
//
// 3. Extract a window of exactly 'width' display columns starting at position
// 4. Pad with spaces if needed to ensure exact width
//
// Interaction with tmux:
// - tmux refreshes status bar at discrete intervals (status-interval, typically 5s)
// - Each refresh calls this function with a new timestamp
// - Creates step-animation effect (not smooth scrolling)
// - Example: speed=2, interval=5s → advances 10 chars per visual update
// - Users can tune speed based on their tmux interval for optimal readability
//
// Edge cases:
// - Short text (fits in width): returns static padded text (no scrolling)
// - Very long text: will eventually cycle through entire text
// - Unicode/emoji: handled correctly using runewidth for display column calculation
func marqueeText(text string, width int, speed int, separator string) string {
	if width <= 0 {
		return text
	}

	textWidth := runewidth.StringWidth(text)

	// If text fits, just pad normally (no scrolling needed)
	if textWidth <= width {
		return padToWidth(text, width)
	}

	// Create extended text: "original + separator + original"
	// This creates a continuous loop
	extended := text + separator + text
	extendedRunes := []rune(extended)

	// Calculate scroll position based on current time
	// This creates a deterministic, timestamp-based scroll position that:
	// - Advances continuously over time (speed chars/second)
	// - Wraps around to create infinite loop (modulo totalChars)
	// - Is stateless (no need to persist position between calls)
	// - Is testable (can mock time.Now for unit tests)
	now := time.Now().Unix()
	totalChars := len(extendedRunes)
	// Position = (current_unix_time * chars_per_second) % total_chars
	// Example: speed=2, time=10s → position = 20 % totalChars
	position := int(now*int64(speed)) % totalChars

	// Build the window starting at position
	var result []rune
	resultWidth := 0

	for i := 0; i < totalChars && resultWidth < width; i++ {
		idx := (position + i) % totalChars
		r := extendedRunes[idx]
		rw := runewidth.RuneWidth(r)

		// Don't exceed target width
		if resultWidth+rw <= width {
			result = append(result, r)
			resultWidth += rw
		} else {
			break
		}
	}

	// Pad with spaces if needed to reach exact width
	if resultWidth < width {
		padding := strings.Repeat(" ", width-resultWidth)
		return string(result) + padding
	}

	return string(result)
}
