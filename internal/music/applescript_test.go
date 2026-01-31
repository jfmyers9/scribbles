package music

import (
	"context"
	"testing"
	"time"
)

// TestAppleScriptClient_Integration tests the AppleScript client against the real Music app
// This is an integration test and requires Apple Music to be installed
func TestAppleScriptClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewAppleScriptClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test IsRunning
	t.Run("IsRunning", func(t *testing.T) {
		running, err := client.IsRunning(ctx)
		if err != nil {
			t.Fatalf("IsRunning() failed: %v", err)
		}
		t.Logf("Music app running: %v", running)
	})

	// Test GetCurrentTrack
	t.Run("GetCurrentTrack", func(t *testing.T) {
		track, err := client.GetCurrentTrack(ctx)
		if err != nil {
			t.Fatalf("GetCurrentTrack() failed: %v", err)
		}

		if track == nil {
			t.Log("No track currently playing (Music not running or stopped)")
			return
		}

		// Validate track data
		if track.Name == "" {
			t.Error("Track name is empty")
		}
		if track.Artist == "" {
			t.Error("Track artist is empty")
		}
		if track.Duration <= 0 {
			t.Errorf("Invalid track duration: %v", track.Duration)
		}
		if track.Position < 0 {
			t.Errorf("Invalid track position: %v", track.Position)
		}
		if track.Position > track.Duration {
			t.Errorf("Position (%v) exceeds duration (%v)", track.Position, track.Duration)
		}
		if track.State != StatePlaying && track.State != StatePaused {
			t.Errorf("Unexpected state: %v", track.State)
		}

		t.Logf("Current track:")
		t.Logf("  Name: %s", track.Name)
		t.Logf("  Artist: %s", track.Artist)
		t.Logf("  Album: %s", track.Album)
		t.Logf("  Duration: %v", track.Duration)
		t.Logf("  Position: %v", track.Position)
		t.Logf("  State: %v", track.State)
	})
}

// TestParseTrackOutput tests the parsing logic with various inputs
func TestParseTrackOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Track
		wantErr bool
	}{
		{
			name:  "valid playing track",
			input: "Bohemian Rhapsody|||Queen|||A Night at the Opera|||354.0|||120.5|||playing",
			want: &Track{
				Name:     "Bohemian Rhapsody",
				Artist:   "Queen",
				Album:    "A Night at the Opera",
				Duration: 354 * time.Second,
				Position: 120*time.Second + 500*time.Millisecond,
				State:    StatePlaying,
			},
		},
		{
			name:  "valid paused track",
			input: "Stairway to Heaven|||Led Zeppelin|||Led Zeppelin IV|||482.0|||45.0|||paused",
			want: &Track{
				Name:     "Stairway to Heaven",
				Artist:   "Led Zeppelin",
				Album:    "Led Zeppelin IV",
				Duration: 482 * time.Second,
				Position: 45 * time.Second,
				State:    StatePaused,
			},
		},
		{
			name:  "track with special characters",
			input: "Don't Stop Believin'|||Journey|||Escape|||251.0|||30.0|||playing",
			want: &Track{
				Name:     "Don't Stop Believin'",
				Artist:   "Journey",
				Album:    "Escape",
				Duration: 251 * time.Second,
				Position: 30 * time.Second,
				State:    StatePlaying,
			},
		},
		{
			name:  "track with empty album",
			input: "Test Track|||Test Artist||||||180.0|||60.0|||playing",
			want: &Track{
				Name:     "Test Track",
				Artist:   "Test Artist",
				Album:    "",
				Duration: 180 * time.Second,
				Position: 60 * time.Second,
				State:    StatePlaying,
			},
		},
		{
			name:    "invalid - wrong number of parts",
			input:   "Track|||Artist|||Album",
			wantErr: true,
		},
		{
			name:    "invalid - bad duration",
			input:   "Track|||Artist|||Album|||bad|||60.0|||playing",
			wantErr: true,
		},
		{
			name:    "invalid - bad position",
			input:   "Track|||Artist|||Album|||180.0|||bad|||playing",
			wantErr: true,
		},
		{
			name:    "invalid - unknown state",
			input:   "Track|||Artist|||Album|||180.0|||60.0|||unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTrackOutput(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("parseTrackOutput() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("parseTrackOutput() unexpected error: %v", err)
				return
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Artist != tt.want.Artist {
				t.Errorf("Artist = %q, want %q", got.Artist, tt.want.Artist)
			}
			if got.Album != tt.want.Album {
				t.Errorf("Album = %q, want %q", got.Album, tt.want.Album)
			}
			if got.Duration != tt.want.Duration {
				t.Errorf("Duration = %v, want %v", got.Duration, tt.want.Duration)
			}
			if got.Position != tt.want.Position {
				t.Errorf("Position = %v, want %v", got.Position, tt.want.Position)
			}
			if got.State != tt.want.State {
				t.Errorf("State = %v, want %v", got.State, tt.want.State)
			}
		})
	}
}

// TestPlayState_String tests the String method on PlayState
func TestPlayState_String(t *testing.T) {
	tests := []struct {
		state PlayState
		want  string
	}{
		{StateStopped, "stopped"},
		{StatePlaying, "playing"},
		{StatePaused, "paused"},
		{PlayState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("PlayState.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
