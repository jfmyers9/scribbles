package music

import (
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
