package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jfmyers9/scribbles/internal/music"
)

// TrackState represents the daemon's tracking state for the currently playing track
type TrackState struct {
	Track         *music.Track  // Currently playing track (nil if stopped)
	StartTime     time.Time     // When playback started (or resumed)
	Scrobbled     bool          // Whether this play has been scrobbled
	PausedAt      time.Time     // When track was paused (zero if not paused)
	TotalPlayTime time.Duration // Accumulated play time (excludes pauses)
}

// State manages the daemon's state with thread-safe access and persistence
type State struct {
	mu       sync.RWMutex
	current  TrackState
	filePath string // Path to state file for persistence
}

// persistedState is the JSON representation of state for disk storage
type persistedState struct {
	Track         *music.Track  `json:"track,omitempty"`
	StartTime     time.Time     `json:"start_time"`
	Scrobbled     bool          `json:"scrobbled"`
	PausedAt      time.Time     `json:"paused_at,omitempty"`
	TotalPlayTime time.Duration `json:"total_play_time"`
}

// NewState creates a new State instance
// If filePath is provided, attempts to restore state from disk
func NewState(filePath string) (*State, error) {
	s := &State{
		filePath: filePath,
	}

	// Try to restore state from disk if file exists
	if filePath != "" {
		if err := s.restore(); err != nil && !os.IsNotExist(err) {
			// Log error but continue with empty state
			// Not a fatal error - daemon can start fresh
			return s, err
		}
	}

	return s, nil
}

// SetTrack updates the current track and resets state
// This should be called when a new track starts playing
func (s *State) SetTrack(track *music.Track) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.current = TrackState{
		Track:         track,
		StartTime:     time.Now(),
		Scrobbled:     false,
		TotalPlayTime: 0,
	}

	return s.persist()
}

// UpdatePosition updates the playback position based on current track state
// Handles pause/resume by accumulating play time
func (s *State) UpdatePosition(track *music.Track) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// No current track - this is a new track
	if s.current.Track == nil {
		s.current = TrackState{
			Track:         track,
			StartTime:     time.Now(),
			Scrobbled:     false,
			TotalPlayTime: 0,
		}
		return s.persist()
	}

	// Track changed - reset state
	if !isSameTrack(s.current.Track, track) {
		s.current = TrackState{
			Track:         track,
			StartTime:     time.Now(),
			Scrobbled:     false,
			TotalPlayTime: 0,
		}
		return s.persist()
	}

	// Same track - update state based on play state
	switch track.State {
	case music.StatePlaying:
		// If we were paused, resume and accumulate play time
		if !s.current.PausedAt.IsZero() {
			// Add time played before pause to total
			pauseDuration := s.current.PausedAt.Sub(s.current.StartTime)
			s.current.TotalPlayTime += pauseDuration
			s.current.StartTime = time.Now() // Reset start time to now
			s.current.PausedAt = time.Time{} // Clear pause marker
		}
	case music.StatePaused:
		// Mark pause time if not already paused
		if s.current.PausedAt.IsZero() {
			s.current.PausedAt = time.Now()
		}
	case music.StateStopped:
		// Track stopped - reset state
		s.current = TrackState{}
	}

	return s.persist()
}

// MarkScrobbled marks the current track as scrobbled
func (s *State) MarkScrobbled() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.current.Scrobbled = true
	return s.persist()
}

// GetState returns a copy of the current state
func (s *State) GetState() TrackState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	return s.current
}

// GetPlayedDuration returns the total time the current track has been played
// This accounts for pauses and resumes
func (s *State) GetPlayedDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If currently paused, return accumulated time up to pause
	if !s.current.PausedAt.IsZero() {
		return s.current.TotalPlayTime + s.current.PausedAt.Sub(s.current.StartTime)
	}

	// If playing, return accumulated time plus current play session
	if s.current.Track != nil && s.current.Track.State == music.StatePlaying {
		return s.current.TotalPlayTime + time.Since(s.current.StartTime)
	}

	// Stopped or no track
	return s.current.TotalPlayTime
}

// Reset clears the current state
func (s *State) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.current = TrackState{}
	return s.persist()
}

// persist saves the current state to disk
// Must be called with lock held
func (s *State) persist() error {
	if s.filePath == "" {
		return nil // No persistence configured
	}

	ps := persistedState{
		Track:         s.current.Track,
		StartTime:     s.current.StartTime,
		Scrobbled:     s.current.Scrobbled,
		PausedAt:      s.current.PausedAt,
		TotalPlayTime: s.current.TotalPlayTime,
	}

	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write atomically via temp file + rename
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.filePath)
}

// restore loads state from disk
func (s *State) restore() error {
	if s.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var ps persistedState
	if err := json.Unmarshal(data, &ps); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.current = TrackState(ps)

	return nil
}

// isSameTrack compares two tracks to determine if they're the same
func isSameTrack(t1, t2 *music.Track) bool {
	if t1 == nil || t2 == nil {
		return false
	}
	return t1.Name == t2.Name &&
		t1.Artist == t2.Artist &&
		t1.Album == t2.Album
}
