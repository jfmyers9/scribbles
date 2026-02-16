package scrobbler

import (
	"testing"
	"time"
)

func TestShouldScrobble(t *testing.T) {
	tests := []struct {
		name           string
		trackDuration  time.Duration
		playedDuration time.Duration
		shouldScrobble bool
		description    string
	}{
		{
			name:           "track too short (29 seconds)",
			trackDuration:  29 * time.Second,
			playedDuration: 29 * time.Second,
			shouldScrobble: false,
			description:    "tracks under 30 seconds should never scrobble",
		},
		{
			name:           "track exactly 30 seconds, fully played",
			trackDuration:  30 * time.Second,
			playedDuration: 30 * time.Second,
			shouldScrobble: true,
			description:    "30 second track played for 30 seconds (100%) should scrobble",
		},
		{
			name:           "track exactly 30 seconds, played 15 seconds (50%)",
			trackDuration:  30 * time.Second,
			playedDuration: 15 * time.Second,
			shouldScrobble: true,
			description:    "30 second track played for 15 seconds (50%) should scrobble",
		},
		{
			name:           "track exactly 30 seconds, played 14 seconds (under 50%)",
			trackDuration:  30 * time.Second,
			playedDuration: 14 * time.Second,
			shouldScrobble: false,
			description:    "30 second track played for 14 seconds (46%) should not scrobble",
		},
		{
			name:           "3 minute track, played 90 seconds (50%)",
			trackDuration:  3 * time.Minute,
			playedDuration: 90 * time.Second,
			shouldScrobble: true,
			description:    "3 minute track played for 90 seconds (50%) should scrobble",
		},
		{
			name:           "3 minute track, played 89 seconds (just under 50%)",
			trackDuration:  3 * time.Minute,
			playedDuration: 89 * time.Second,
			shouldScrobble: false,
			description:    "3 minute track played for 89 seconds (49.4%) should not scrobble",
		},
		{
			name:           "8 minute track, played 4 minutes (50%)",
			trackDuration:  8 * time.Minute,
			playedDuration: 4 * time.Minute,
			shouldScrobble: true,
			description:    "8 minute track played for 4 minutes (50%) should scrobble (hits max threshold)",
		},
		{
			name:           "8 minute track, played 3 minutes 59 seconds",
			trackDuration:  8 * time.Minute,
			playedDuration: 3*time.Minute + 59*time.Second,
			shouldScrobble: false,
			description:    "8 minute track just under 4 minutes should not scrobble",
		},
		{
			name:           "10 minute track, played 4 minutes (40%)",
			trackDuration:  10 * time.Minute,
			playedDuration: 4 * time.Minute,
			shouldScrobble: true,
			description:    "10 minute track at 4 minutes (40%) should scrobble (max threshold)",
		},
		{
			name:           "10 minute track, played 5 minutes (50%)",
			trackDuration:  10 * time.Minute,
			playedDuration: 5 * time.Minute,
			shouldScrobble: true,
			description:    "10 minute track at 5 minutes should scrobble (above threshold)",
		},
		{
			name:           "1 hour track, played 4 minutes",
			trackDuration:  60 * time.Minute,
			playedDuration: 4 * time.Minute,
			shouldScrobble: true,
			description:    "very long track should scrobble at 4 minutes regardless of percentage",
		},
		{
			name:           "1 hour track, played 3 minutes",
			trackDuration:  60 * time.Minute,
			playedDuration: 3 * time.Minute,
			shouldScrobble: false,
			description:    "very long track under 4 minutes should not scrobble",
		},
		{
			name:           "short track, not played at all",
			trackDuration:  3 * time.Minute,
			playedDuration: 0,
			shouldScrobble: false,
			description:    "track not played at all should not scrobble",
		},
		{
			name:           "exactly 4 minute track, played 2 minutes",
			trackDuration:  4 * time.Minute,
			playedDuration: 2 * time.Minute,
			shouldScrobble: true,
			description:    "4 minute track at 50% (2 minutes) should scrobble",
		},
		{
			name:           "exactly 8 minute track, played exactly 4 minutes",
			trackDuration:  8 * time.Minute,
			playedDuration: 4 * time.Minute,
			shouldScrobble: true,
			description:    "8 minute track at exactly 4 minutes should scrobble (boundary case)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldScrobble(tt.trackDuration, tt.playedDuration)
			if result != tt.shouldScrobble {
				t.Errorf("%s: ShouldScrobble(%v, %v) = %v, want %v",
					tt.description,
					tt.trackDuration,
					tt.playedDuration,
					result,
					tt.shouldScrobble,
				)
			}
		})
	}
}

func TestScrobbleThreshold(t *testing.T) {
	tests := []struct {
		name          string
		trackDuration time.Duration
		expected      time.Duration
		description   string
	}{
		{
			name:          "track too short",
			trackDuration: 29 * time.Second,
			expected:      time.Duration(-1),
			description:   "tracks under 30 seconds should return -1",
		},
		{
			name:          "exactly 30 seconds",
			trackDuration: 30 * time.Second,
			expected:      15 * time.Second,
			description:   "30 second track should have 15 second threshold (50%)",
		},
		{
			name:          "3 minute track",
			trackDuration: 3 * time.Minute,
			expected:      90 * time.Second,
			description:   "3 minute track should have 90 second threshold (50%)",
		},
		{
			name:          "8 minute track (at boundary)",
			trackDuration: 8 * time.Minute,
			expected:      4 * time.Minute,
			description:   "8 minute track should have 4 minute threshold (50%)",
		},
		{
			name:          "9 minute track (above boundary)",
			trackDuration: 9 * time.Minute,
			expected:      4 * time.Minute,
			description:   "9 minute track should cap at 4 minute threshold (not 4.5 minutes)",
		},
		{
			name:          "10 minute track",
			trackDuration: 10 * time.Minute,
			expected:      4 * time.Minute,
			description:   "10 minute track should cap at 4 minute threshold (not 5 minutes)",
		},
		{
			name:          "1 hour track",
			trackDuration: 60 * time.Minute,
			expected:      4 * time.Minute,
			description:   "very long track should cap at 4 minute threshold",
		},
		{
			name:          "exactly 4 minute track",
			trackDuration: 4 * time.Minute,
			expected:      2 * time.Minute,
			description:   "4 minute track should have 2 minute threshold (50%)",
		},
		{
			name:          "7 minute 59 second track",
			trackDuration: 7*time.Minute + 59*time.Second,
			expected:      time.Duration(float64(7*time.Minute+59*time.Second) * 0.5),
			description:   "just under 8 minute track should use 50% rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScrobbleThreshold(tt.trackDuration)
			if result != tt.expected {
				t.Errorf("%s: ScrobbleThreshold(%v) = %v, want %v",
					tt.description,
					tt.trackDuration,
					result,
					tt.expected,
				)
			}
		})
	}
}

func TestIsEligible(t *testing.T) {
	tests := []struct {
		name          string
		trackDuration time.Duration
		eligible      bool
	}{
		{
			name:          "track too short",
			trackDuration: 29 * time.Second,
			eligible:      false,
		},
		{
			name:          "exactly 30 seconds",
			trackDuration: 30 * time.Second,
			eligible:      true,
		},
		{
			name:          "31 seconds",
			trackDuration: 31 * time.Second,
			eligible:      true,
		},
		{
			name:          "3 minute track",
			trackDuration: 3 * time.Minute,
			eligible:      true,
		},
		{
			name:          "very long track",
			trackDuration: 60 * time.Minute,
			eligible:      true,
		},
		{
			name:          "zero duration",
			trackDuration: 0,
			eligible:      false,
		},
		{
			name:          "1 second",
			trackDuration: 1 * time.Second,
			eligible:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEligible(tt.trackDuration)
			if result != tt.eligible {
				t.Errorf("IsEligible(%v) = %v, want %v",
					tt.trackDuration,
					result,
					tt.eligible,
				)
			}
		})
	}
}

// Benchmark tests to ensure rules calculations are fast
func BenchmarkShouldScrobble(b *testing.B) {
	trackDuration := 3 * time.Minute
	playedDuration := 90 * time.Second

	for i := 0; i < b.N; i++ {
		ShouldScrobble(trackDuration, playedDuration)
	}
}

func BenchmarkScrobbleThreshold(b *testing.B) {
	trackDuration := 8 * time.Minute

	for i := 0; i < b.N; i++ {
		ScrobbleThreshold(trackDuration)
	}
}

func BenchmarkIsEligible(b *testing.B) {
	trackDuration := 3 * time.Minute

	for i := 0; i < b.N; i++ {
		IsEligible(trackDuration)
	}
}
