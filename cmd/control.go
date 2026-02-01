package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/spf13/cobra"
)

// playCmd represents the play command
var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Resume playback in Apple Music",
	Long:  `Resume playback in Apple Music. If paused, starts playing the current track.`,
	RunE:  runPlay,
}

// pauseCmd represents the pause command
var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause playback in Apple Music",
	Long:  `Pause playback in Apple Music. Pauses the currently playing track.`,
	RunE:  runPause,
}

// playpauseCmd represents the playpause command
var playpauseCmd = &cobra.Command{
	Use:   "playpause",
	Short: "Toggle play/pause in Apple Music",
	Long:  `Toggle between play and pause states in Apple Music. If playing, pauses. If paused, resumes.`,
	RunE:  runPlayPause,
}

// nextCmd represents the next command
var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Skip to next track in Apple Music",
	Long:  `Skip to the next track in Apple Music. Advances to the next track in the current playlist or queue.`,
	RunE:  runNext,
}

// prevCmd represents the prev command
var prevCmd = &cobra.Command{
	Use:   "prev",
	Short: "Go to previous track in Apple Music",
	Long:  `Go to the previous track in Apple Music. Returns to the previous track in the current playlist or queue.`,
	RunE:  runPrev,
}

// shuffleCmd represents the shuffle command
var shuffleCmd = &cobra.Command{
	Use:   "shuffle [on|off]",
	Short: "Toggle or set shuffle mode in Apple Music",
	Long: `Control shuffle mode in Apple Music.

Without arguments, toggles shuffle on/off.
With 'on' or 'off' argument, explicitly sets shuffle state.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runShuffle,
}

// volumeCmd represents the volume command
var volumeCmd = &cobra.Command{
	Use:   "volume [0-100]",
	Short: "Set playback volume in Apple Music",
	Long: `Set the playback volume in Apple Music.

Volume level must be between 0 (muted) and 100 (maximum).
Without arguments, displays current volume (not yet implemented).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runVolume,
}

func init() {
	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(playpauseCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(prevCmd)
	rootCmd.AddCommand(shuffleCmd)
	rootCmd.AddCommand(volumeCmd)
}

func runPlay(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := music.NewAppleScriptClient()
	if err := client.Play(ctx); err != nil {
		return fmt.Errorf("failed to play: %w", err)
	}

	return nil
}

func runPause(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := music.NewAppleScriptClient()
	if err := client.Pause(ctx); err != nil {
		return fmt.Errorf("failed to pause: %w", err)
	}

	return nil
}

func runPlayPause(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := music.NewAppleScriptClient()
	if err := client.PlayPause(ctx); err != nil {
		return fmt.Errorf("failed to playpause: %w", err)
	}

	return nil
}

func runNext(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := music.NewAppleScriptClient()
	if err := client.NextTrack(ctx); err != nil {
		return fmt.Errorf("failed to skip to next track: %w", err)
	}

	return nil
}

func runPrev(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := music.NewAppleScriptClient()
	if err := client.PreviousTrack(ctx); err != nil {
		return fmt.Errorf("failed to go to previous track: %w", err)
	}

	return nil
}

func runShuffle(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := music.NewAppleScriptClient()

	// If no argument provided, we need to get current state and toggle
	// For now, we'll require an explicit on/off argument
	// TODO: Add GetShuffle() method to support toggle without argument
	if len(args) == 0 {
		return fmt.Errorf("shuffle requires 'on' or 'off' argument (toggle not yet supported)")
	}

	var enabled bool
	switch args[0] {
	case "on":
		enabled = true
	case "off":
		enabled = false
	default:
		return fmt.Errorf("invalid shuffle argument: %s (must be 'on' or 'off')", args[0])
	}

	if err := client.SetShuffle(ctx, enabled); err != nil {
		return fmt.Errorf("failed to set shuffle: %w", err)
	}

	return nil
}

func runVolume(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := music.NewAppleScriptClient()

	// If no argument provided, we need to get current volume
	// For now, we'll require an explicit level argument
	// TODO: Add GetVolume() method to support displaying current volume
	if len(args) == 0 {
		return fmt.Errorf("volume requires a level argument 0-100 (displaying current volume not yet supported)")
	}

	level, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid volume level: %s (must be a number 0-100)", args[0])
	}

	if err := client.SetVolume(ctx, level); err != nil {
		return fmt.Errorf("failed to set volume: %w", err)
	}

	return nil
}
