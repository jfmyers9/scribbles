package music

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func TestTrackFromMPRISProperties(t *testing.T) {
	props := map[string]dbus.Variant{
		"PlaybackStatus": dbus.MakeVariant("Playing"),
		"Position":       dbus.MakeVariant(int64(42_500_000)),
		"Metadata": dbus.MakeVariant(map[string]dbus.Variant{
			"xesam:title":  dbus.MakeVariant("Example Song"),
			"xesam:artist": dbus.MakeVariant([]string{"First Artist", "Second Artist"}),
			"xesam:album":  dbus.MakeVariant("Example Album"),
			"mpris:length": dbus.MakeVariant(int64(180_000_000)),
		}),
	}

	track, err := trackFromMPRISProperties(props)
	if err != nil {
		t.Fatalf("trackFromMPRISProperties() error = %v", err)
	}
	if track == nil {
		t.Fatal("trackFromMPRISProperties() returned nil")
	}
	if track.Name != "Example Song" || track.Artist != "First Artist, Second Artist" || track.Album != "Example Album" {
		t.Fatalf("unexpected metadata: %+v", track)
	}
	if track.Duration != 3*time.Minute || track.Position != 42*time.Second+500*time.Millisecond {
		t.Fatalf("unexpected timing: duration=%v position=%v", track.Duration, track.Position)
	}
	if track.State != StatePlaying {
		t.Fatalf("state = %v, want playing", track.State)
	}
}

func TestTrackFromMPRISPropertiesStopped(t *testing.T) {
	track, err := trackFromMPRISProperties(map[string]dbus.Variant{
		"PlaybackStatus": dbus.MakeVariant("Stopped"),
	})
	if err != nil {
		t.Fatalf("trackFromMPRISProperties() error = %v", err)
	}
	if track != nil {
		t.Fatalf("trackFromMPRISProperties() = %+v, want nil", track)
	}
}

func TestTrackFromMPRISPropertiesRequiresScrobbleMetadata(t *testing.T) {
	props := map[string]dbus.Variant{
		"PlaybackStatus": dbus.MakeVariant("Paused"),
		"Metadata": dbus.MakeVariant(map[string]dbus.Variant{
			"xesam:title": dbus.MakeVariant("Title Without Artist"),
		}),
	}

	track, err := trackFromMPRISProperties(props)
	if err != nil {
		t.Fatalf("trackFromMPRISProperties() error = %v", err)
	}
	if track != nil {
		t.Fatalf("trackFromMPRISProperties() = %+v, want nil", track)
	}
}

func TestAppleMusicDurationLookupAndCache(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.URL.Query().Get("term"); got != "Sun June Leave The City" {
			t.Errorf("term = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[
			{"trackName":"Leave The City","artistName":"Sun June","collectionName":"Other Album","trackTimeMillis":123000},
			{"trackName":"Leave The City","artistName":"Sun June","collectionName":"Just Be Simple / Leave The City - Single","trackTimeMillis":268749}
		]}`))
	}))
	defer server.Close()

	client := NewMPRISClient()
	client.searchURL = server.URL
	track := &Track{
		Name:   "Leave The City",
		Artist: "Sun June",
		Album:  "Just Be Simple / Leave The City - Single",
	}

	for range 2 {
		if got := client.appleMusicDuration(context.Background(), track); got != 268749*time.Millisecond {
			t.Fatalf("appleMusicDuration() = %v", got)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("search requests = %d, want 1", got)
	}
}

func TestIsAppleMusicMetadata(t *testing.T) {
	props := map[string]dbus.Variant{
		"Metadata": dbus.MakeVariant(map[string]dbus.Variant{
			"xesam:url": dbus.MakeVariant("https://music.apple.com/us/home"),
		}),
	}
	if !isAppleMusicMetadata(props) {
		t.Fatal("music.apple.com metadata was not recognized")
	}
}
