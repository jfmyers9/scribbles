package discord

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/jfmyers9/scribbles/internal/music"
)

type fakeRPC struct {
	activities []Activity
	closed     bool
	failNext   error
}

func (f *fakeRPC) SetActivity(a Activity) error {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}
	f.activities = append(f.activities, a)
	return nil
}

func (f *fakeRPC) Close() { f.closed = true }

func newTestPresence() (*Presence, *fakeRPC) {
	fake := &fakeRPC{}
	p := &Presence{
		appID:  "test",
		logger: zerolog.Nop(),
		connect: func(string) (rpcClient, error) {
			return fake, nil
		},
	}
	return p, fake
}

func playingTrack(name, artist, album string) *music.Track {
	return &music.Track{
		Name: name, Artist: artist, Album: album,
		Duration: 3 * time.Minute, Position: 30 * time.Second,
		State: music.StatePlaying,
	}
}

func TestDedup_SkipsDuplicateUpdates(t *testing.T) {
	p, fake := newTestPresence()
	track := playingTrack("Song", "Artist", "Album")

	p.handleTrack(track)
	p.handleTrack(track)
	p.handleTrack(track)

	if len(fake.activities) != 1 {
		t.Fatalf("expected 1 SetActivity call, got %d", len(fake.activities))
	}
}

func TestDedup_SendsOnTrackChange(t *testing.T) {
	p, fake := newTestPresence()

	p.handleTrack(playingTrack("Song A", "Artist", "Album"))
	p.handleTrack(playingTrack("Song B", "Artist", "Album"))

	if len(fake.activities) != 2 {
		t.Fatalf("expected 2 SetActivity calls, got %d", len(fake.activities))
	}
	if fake.activities[0].Details != "Song A" {
		t.Errorf("first activity details = %q, want %q", fake.activities[0].Details, "Song A")
	}
	if fake.activities[1].Details != "Song B" {
		t.Errorf("second activity details = %q, want %q", fake.activities[1].Details, "Song B")
	}
}

func TestClearsOnPause(t *testing.T) {
	p, fake := newTestPresence()

	p.handleTrack(playingTrack("Song", "Artist", "Album"))
	p.handleTrack(&music.Track{
		Name: "Song", Artist: "Artist", Album: "Album",
		State: music.StatePaused,
	})

	// First call sets activity, second clears it (empty Activity)
	if len(fake.activities) != 2 {
		t.Fatalf("expected 2 SetActivity calls, got %d", len(fake.activities))
	}
	if fake.activities[1].Details != "" {
		t.Errorf("clear activity should have empty details, got %q", fake.activities[1].Details)
	}
}

func TestClearsOnNilTrack(t *testing.T) {
	p, fake := newTestPresence()

	p.handleTrack(playingTrack("Song", "Artist", "Album"))
	p.handleTrack(nil)

	if len(fake.activities) != 2 {
		t.Fatalf("expected 2 SetActivity calls, got %d", len(fake.activities))
	}
}

func TestNoClearWhenAlreadyStopped(t *testing.T) {
	p, fake := newTestPresence()

	// Never played — pause/nil should not trigger a clear
	p.handleTrack(nil)
	p.handleTrack(&music.Track{State: music.StatePaused})

	if len(fake.activities) != 0 {
		t.Fatalf("expected 0 SetActivity calls, got %d", len(fake.activities))
	}
}

func TestReconnectsAfterError(t *testing.T) {
	connectCount := 0
	fake := &fakeRPC{}
	p := &Presence{
		appID:  "test",
		logger: zerolog.Nop(),
		connect: func(string) (rpcClient, error) {
			connectCount++
			fake = &fakeRPC{}
			return fake, nil
		},
	}

	track := playingTrack("Song", "Artist", "Album")
	p.handleTrack(track)
	if connectCount != 1 {
		t.Fatalf("expected 1 connect, got %d", connectCount)
	}

	// Simulate connection failure on next SetActivity
	fake.failNext = errors.New("broken pipe")
	p.last = lastActivity{} // reset dedup so we actually try
	p.handleTrack(track)

	// Should have disconnected (close called on old client)
	// Next call should reconnect
	p.handleTrack(track)
	if connectCount != 2 {
		t.Fatalf("expected 2 connects after error, got %d", connectCount)
	}
}

func TestRunStopsOnContextCancel(t *testing.T) {
	p, fake := newTestPresence()
	// Pre-connect so close is observable
	p.client = fake

	ctx, cancel := context.WithCancel(context.Background())
	updates := make(chan TrackUpdate, 1)
	done := make(chan struct{})

	go func() {
		p.Run(ctx, updates)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after context cancel")
	}

	if !fake.closed {
		t.Error("expected client to be closed on context cancel")
	}
}

func TestActivityFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(itunesResponse{
			Results: []itunesResult{
				{ArtworkURL100: "https://example.com/art/100x100bb.jpg"},
			},
		})
	}))
	defer srv.Close()

	p, fake := newTestPresence()
	p.artwork = newArtworkLookup()
	p.artwork.endpoint = srv.URL

	p.handleTrack(playingTrack("Bohemian Rhapsody", "Queen", "A Night at the Opera"))

	if len(fake.activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(fake.activities))
	}
	a := fake.activities[0]
	if a.Type != 2 {
		t.Errorf("type = %d, want 2 (Listening)", a.Type)
	}
	if a.Name != "Apple Music" {
		t.Errorf("name = %q, want %q", a.Name, "Apple Music")
	}
	if a.Details != "Bohemian Rhapsody" {
		t.Errorf("details = %q, want %q", a.Details, "Bohemian Rhapsody")
	}
	if a.State != "by Queen" {
		t.Errorf("state = %q, want %q", a.State, "by Queen")
	}
	if a.Assets == nil || a.Assets.LargeText != "A Night at the Opera" {
		t.Errorf("large_text = %q, want %q", a.Assets.LargeText, "A Night at the Opera")
	}
	if a.Assets == nil || a.Assets.LargeImage != "https://example.com/art/600x600bb.jpg" {
		t.Errorf("large_image = %q, want artwork URL", a.Assets.LargeImage)
	}
	if a.Timestamps == nil || a.Timestamps.Start == nil || a.Timestamps.End == nil {
		t.Fatal("expected timestamps with start and end")
	}
}
