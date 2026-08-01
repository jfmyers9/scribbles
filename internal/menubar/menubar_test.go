package menubar

import (
	"testing"
	"time"

	"github.com/jfmyers9/scribbles/internal/music"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		snapshot Snapshot
		track    string
		scrobble string
	}{
		{
			name:     "idle",
			snapshot: Snapshot{Pending: 2},
			track:    "Now playing: Nothing",
			scrobble: "Scrobble: Waiting for a track",
		},
		{
			name: "playing",
			snapshot: Snapshot{
				Track:  &music.Track{Name: "Song", Artist: "Artist", Duration: 4 * time.Minute, State: music.StatePlaying},
				Played: 30 * time.Second,
			},
			track:    "Now playing: Artist - Song",
			scrobble: "Scrobble: 25% (target 50%)",
		},
		{
			name: "queued",
			snapshot: Snapshot{
				Track:     &music.Track{Name: "Song", Artist: "Artist", State: music.StatePaused},
				Scrobbled: true,
			},
			track:    "Paused: Artist - Song",
			scrobble: "Scrobble: Queued",
		},
		{
			name: "too short",
			snapshot: Snapshot{
				Track: &music.Track{Name: "Jingle", Artist: "Artist", Duration: 20 * time.Second, State: music.StatePlaying},
			},
			track:    "Now playing: Artist - Jingle",
			scrobble: "Scrobble: Not eligible (<30s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := render(tt.snapshot)
			if got.track != tt.track || got.scrobble != tt.scrobble {
				t.Fatalf("render() = track %q, scrobble %q; want %q, %q", got.track, got.scrobble, tt.track, tt.scrobble)
			}
		})
	}
}
