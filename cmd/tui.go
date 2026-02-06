package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jfmyers9/scribbles/internal/config"
	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Display a terminal UI for now playing",
	Long: `Display a terminal-based user interface showing the currently playing track
from Apple Music with real-time updates.

This is a standalone TUI that polls Apple Music directly. For a TUI that
integrates with scrobbling, use 'scribbles daemon --tui' instead.

The TUI includes:
- Now playing display with track name, artist, and album
- Progress bar showing playback position
- Play state indicator (playing/paused)

Press 'q' to quit.`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	_ = cfg // Will use for configuration later

	// Create music client
	client := music.NewAppleScriptClient()

	// Create tview application
	app := tview.NewApplication()

	// Create main layout components
	nowPlaying := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	nowPlaying.SetBorder(true).
		SetTitle(" Now Playing ").
		SetTitleAlign(tview.AlignLeft)

	progress := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	progress.SetBorder(true)

	status := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[gray]Press 'q' to quit | For scrobbling: scribbles daemon --tui[-]")

	// Create layout using Flex
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nowPlaying, 0, 3, false).
		AddItem(progress, 3, 1, false).
		AddItem(status, 1, 1, false)

	// Handle keyboard input
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q', 'Q':
			app.Stop()
			return nil
		}
		return event
	})

	// Change detection caches
	var lastNowPlaying string
	var lastProgress string
	var lastBarWidth int

	// Update function to refresh display
	updateDisplay := func(track *music.Track) {
		app.QueueUpdateDraw(func() {
			var npText string
			var progText string

			if track == nil || track.State == music.StateStopped {
				npText = "\n\n[gray]No track playing[-]"
				progText = ""
			} else {
				// Build now playing text
				var sb strings.Builder
				sb.WriteString("\n")
				sb.WriteString(fmt.Sprintf("[white::b]%s[-:-:-]\n", tview.Escape(track.Name)))
				sb.WriteString(fmt.Sprintf("[yellow]%s[-]\n", tview.Escape(track.Artist)))
				sb.WriteString(fmt.Sprintf("[gray]%s[-]", tview.Escape(track.Album)))

				// Add play state indicator
				stateIcon := "[green]\u25B6[-]" // Play triangle
				if track.State == music.StatePaused {
					stateIcon = "[yellow]\u23F8[-]" // Pause icon
				}
				sb.WriteString(fmt.Sprintf("\n\n%s", stateIcon))
				npText = sb.String()

				// Build progress bar with cached width to avoid flicker
				_, _, width, _ := progress.GetInnerRect()
				barWidth := width - 14
				// Only update cached width when GetInnerRect returns a positive value
				if barWidth > 0 {
					lastBarWidth = barWidth
				}
				if lastBarWidth < 10 {
					lastBarWidth = 10
				}
				progressBar := tuiBuildProgressBar(track.Position, track.Duration, lastBarWidth)
				posStr := tuiFormatDuration(track.Position)
				durStr := tuiFormatDuration(track.Duration)
				progText = fmt.Sprintf("%s %s %s", posStr, progressBar, durStr)
			}

			if npText != lastNowPlaying {
				lastNowPlaying = npText
				nowPlaying.SetText(npText)
			}
			if progText != lastProgress {
				lastProgress = progText
				progress.SetText(progText)
			}
		})
	}

	// Start polling goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		const (
			baseInterval = 1 * time.Second
			maxInterval  = 16 * time.Second
		)
		interval := baseInterval
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Initial fetch
		track, _ := client.GetCurrentTrack(ctx)
		updateDisplay(track)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				track, err := client.GetCurrentTrack(ctx)
				if err != nil {
					updateDisplay(nil)
					// Exponential backoff on error
					if interval < maxInterval {
						interval *= 2
						if interval > maxInterval {
							interval = maxInterval
						}
						ticker.Reset(interval)
					}
					continue
				}
				// Reset to base interval on success
				if interval != baseInterval {
					interval = baseInterval
					ticker.Reset(interval)
				}
				updateDisplay(track)
			}
		}
	}()

	// Run application
	if err := app.SetRoot(flex, true).Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// tuiBuildProgressBar creates a text-based progress bar
func tuiBuildProgressBar(position, duration time.Duration, width int) string {
	if duration == 0 || width <= 0 {
		return strings.Repeat("-", width)
	}

	progress := float64(position) / float64(duration)
	if progress > 1 {
		progress = 1
	}
	if progress < 0 {
		progress = 0
	}

	filled := int(progress * float64(width))
	empty := width - filled

	bar := "[green]" + strings.Repeat("\u2588", filled) + "[-]" +
		"[gray]" + strings.Repeat("\u2591", empty) + "[-]"

	return bar
}

// tuiFormatDuration formats a duration as MM:SS
func tuiFormatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
