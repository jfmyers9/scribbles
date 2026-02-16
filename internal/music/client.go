package music

import (
	"context"
	"time"
)

// Track represents a music track with its metadata and current state
type Track struct {
	Name     string        // Track name/title
	Artist   string        // Artist name
	Album    string        // Album name
	Duration time.Duration // Total track duration
	Position time.Duration // Current playback position
	State    PlayState     // Current playback state
}

// PlayState represents the current playback state of the music player
type PlayState int

const (
	StateStopped PlayState = iota // No track playing
	StatePlaying                  // Track is currently playing
	StatePaused                   // Track is paused
)

// String returns a human-readable representation of the PlayState
func (s PlayState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	default:
		return "unknown"
	}
}

// Client defines the interface for interacting with a music player
type Client interface {
	// GetCurrentTrack returns the currently playing/paused track, or nil if stopped
	GetCurrentTrack(ctx context.Context) (*Track, error)

	// IsRunning checks if the music player application is running
	IsRunning(ctx context.Context) (bool, error)

	// Play resumes playback
	Play(ctx context.Context) error

	// Pause pauses playback
	Pause(ctx context.Context) error

	// PlayPause toggles between play and pause
	PlayPause(ctx context.Context) error

	// NextTrack skips to the next track
	NextTrack(ctx context.Context) error

	// PreviousTrack goes to the previous track
	PreviousTrack(ctx context.Context) error
}
