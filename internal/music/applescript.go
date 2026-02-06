package music

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// AppleScriptClient implements the Client interface using AppleScript to query Apple Music
type AppleScriptClient struct{}

// NewAppleScriptClient creates a new AppleScript-based music client
func NewAppleScriptClient() *AppleScriptClient {
	return &AppleScriptClient{}
}

// IsRunning checks if the Music app is currently running
func (c *AppleScriptClient) IsRunning(ctx context.Context) (bool, error) {
	script := `tell application "System Events" to (name of processes) contains "Music"`

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check if Music is running: %w", err)
	}

	result := strings.TrimSpace(string(output))
	return result == "true", nil
}

// GetCurrentTrack returns the currently playing or paused track from Apple Music
// This uses a single osascript call that checks if Music is running and queries
// track data atomically, avoiding the overhead of two separate subprocess spawns.
func (c *AppleScriptClient) GetCurrentTrack(ctx context.Context) (*Track, error) {
	script := `
tell application "System Events"
	if not ((name of processes) contains "Music") then
		return "not_running"
	end if
end tell
tell application "Music"
	if player state is stopped then
		return "stopped"
	else
		set trackName to name of current track
		set trackArtist to artist of current track
		set trackAlbum to album of current track
		set trackDuration to duration of current track
		set playerPos to player position
		set playerState to player state as string

		return trackName & "|||" & trackArtist & "|||" & trackAlbum & "|||" & trackDuration & "|||" & playerPos & "|||" & playerState
	end if
end tell`

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		// If there's an error, try to extract the error message
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("osascript error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute osascript: %w", err)
	}

	result := strings.TrimSpace(string(output))

	// Handle not running or stopped states
	if result == "not_running" || result == "stopped" {
		return nil, nil
	}

	// Parse the result
	track, err := parseTrackOutput(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse track output: %w", err)
	}

	return track, nil
}

// parseTrackOutput parses the delimited output from the AppleScript
func parseTrackOutput(output string) (*Track, error) {
	// Split by our custom delimiter
	parts := strings.Split(output, "|||")
	if len(parts) != 6 {
		return nil, fmt.Errorf("expected 6 parts, got %d: %q", len(parts), output)
	}

	name := strings.TrimSpace(parts[0])
	artist := strings.TrimSpace(parts[1])
	album := strings.TrimSpace(parts[2])
	durationStr := strings.TrimSpace(parts[3])
	positionStr := strings.TrimSpace(parts[4])
	stateStr := strings.TrimSpace(parts[5])

	// Parse duration (in seconds as float)
	durationSec, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration %q: %w", durationStr, err)
	}

	// Parse position (in seconds as float)
	positionSec, err := strconv.ParseFloat(positionStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse position %q: %w", positionStr, err)
	}

	// Parse state
	var state PlayState
	switch stateStr {
	case "playing":
		state = StatePlaying
	case "paused":
		state = StatePaused
	case "stopped":
		state = StateStopped
	default:
		return nil, fmt.Errorf("unknown player state: %q", stateStr)
	}

	return &Track{
		Name:     name,
		Artist:   artist,
		Album:    album,
		Duration: secondsToDuration(durationSec),
		Position: secondsToDuration(positionSec),
		State:    state,
	}, nil
}

// secondsToDuration converts seconds (as float) to time.Duration
func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

// Play resumes playback in Apple Music
func (c *AppleScriptClient) Play(ctx context.Context) error {
	script := `tell application "Music" to play`
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to play: %w", err)
	}
	return nil
}

// Pause pauses playback in Apple Music
func (c *AppleScriptClient) Pause(ctx context.Context) error {
	script := `tell application "Music" to pause`
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pause: %w", err)
	}
	return nil
}

// PlayPause toggles between play and pause in Apple Music
func (c *AppleScriptClient) PlayPause(ctx context.Context) error {
	script := `tell application "Music" to playpause`
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to playpause: %w", err)
	}
	return nil
}

// NextTrack skips to the next track in Apple Music
func (c *AppleScriptClient) NextTrack(ctx context.Context) error {
	script := `tell application "Music" to next track`
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to skip to next track: %w", err)
	}
	return nil
}

// PreviousTrack goes back to the previous track in Apple Music
func (c *AppleScriptClient) PreviousTrack(ctx context.Context) error {
	script := `tell application "Music" to back track`
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to go to previous track: %w", err)
	}
	return nil
}

// SetShuffle enables or disables shuffle mode in Apple Music
func (c *AppleScriptClient) SetShuffle(ctx context.Context, enabled bool) error {
	script := fmt.Sprintf(`tell application "Music" to set shuffle enabled to %t`, enabled)
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set shuffle: %w", err)
	}
	return nil
}

// SetVolume sets the playback volume in Apple Music (0-100)
func (c *AppleScriptClient) SetVolume(ctx context.Context, level int) error {
	if level < 0 || level > 100 {
		return fmt.Errorf("volume level must be between 0 and 100, got %d", level)
	}
	script := fmt.Sprintf(`tell application "Music" to set sound volume to %d`, level)
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set volume: %w", err)
	}
	return nil
}
