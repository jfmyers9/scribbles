package scrobbler

import (
	"time"
)

// Last.fm scrobbling rules constants
const (
	// MinimumTrackDuration is the minimum track length required for scrobbling (30 seconds)
	MinimumTrackDuration = 30 * time.Second

	// ScrobblePercentage is the percentage of track that must be played (50%)
	ScrobblePercentage = 0.5

	// MaxScrobbleThreshold is the maximum time that needs to be played (4 minutes)
	MaxScrobbleThreshold = 4 * time.Minute
)

// ShouldScrobble determines if a track should be scrobbled based on Last.fm rules:
// 1. Track must be longer than 30 seconds
// 2. Track must have been played for at least 50% of its duration OR 4 minutes, whichever comes first
//
// Parameters:
//   - trackDuration: Total duration of the track
//   - playedDuration: How long the track has been played
//
// Returns:
//   - true if the track should be scrobbled
//   - false if the track should not be scrobbled
func ShouldScrobble(trackDuration, playedDuration time.Duration) bool {
	// Rule 1: Track must be longer than 30 seconds
	if trackDuration < MinimumTrackDuration {
		return false
	}

	// Rule 2: Calculate the scrobble threshold
	// The threshold is the minimum of:
	//   - 50% of track duration
	//   - 4 minutes
	threshold := time.Duration(float64(trackDuration) * ScrobblePercentage)
	if threshold > MaxScrobbleThreshold {
		threshold = MaxScrobbleThreshold
	}

	// Check if we've played enough to scrobble
	return playedDuration >= threshold
}

// ScrobbleThreshold calculates the exact time threshold at which a track should be scrobbled
// This is useful for daemon logic to know when to trigger a scrobble
func ScrobbleThreshold(trackDuration time.Duration) time.Duration {
	// Track must be at least 30 seconds
	if trackDuration < MinimumTrackDuration {
		// Return a value that can never be met
		return time.Duration(-1)
	}

	// Calculate 50% of duration
	threshold := time.Duration(float64(trackDuration) * ScrobblePercentage)

	// Cap at 4 minutes
	if threshold > MaxScrobbleThreshold {
		threshold = MaxScrobbleThreshold
	}

	return threshold
}

// IsEligible checks if a track is eligible for scrobbling based on its duration alone
// This can be used to quickly filter out tracks that are too short before tracking them
func IsEligible(trackDuration time.Duration) bool {
	return trackDuration >= MinimumTrackDuration
}
