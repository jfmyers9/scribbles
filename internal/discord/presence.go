package discord

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/jfmyers9/scribbles/internal/music"
)

// TrackUpdate represents a music player state change.
// Mirrors daemon.TrackUpdate to avoid import cycles.
type TrackUpdate struct {
	Track *music.Track
	Err   error
}

type rpcClient interface {
	SetActivity(Activity) error
	Close() error
}

// Presence manages Discord Rich Presence updates.
type Presence struct {
	appID   string
	logger  zerolog.Logger
	client  rpcClient
	connect func(string) (rpcClient, error)
	last    lastActivity
	artwork *artworkLookup
}

type lastActivity struct {
	name, artist, album string
	playing             bool
}

func New(appID string, logger zerolog.Logger) *Presence {
	return &Presence{
		appID:  appID,
		logger: logger.With().Str("component", "discord").Logger(),
		connect: func(appID string) (rpcClient, error) {
			return ipcConnect(appID)
		},
		artwork: newArtworkLookup(),
	}
}

// Run consumes TrackUpdates and sets Discord Rich Presence.
// Connects lazily on first playing track. If Discord isn't
// running, logs the error and retries on the next update.
func (p *Presence) Run(ctx context.Context, updates <-chan TrackUpdate) {
	for {
		select {
		case <-ctx.Done():
			p.close()
			return
		case u, ok := <-updates:
			if !ok {
				p.close()
				return
			}
			if u.Err != nil {
				continue
			}
			p.handleTrack(u.Track)
		}
	}
}

func (p *Presence) handleTrack(track *music.Track) {
	if track == nil || track.State != music.StatePlaying {
		if p.last.playing {
			p.clearActivity()
			p.last = lastActivity{}
		}
		return
	}

	cur := lastActivity{
		name: track.Name, artist: track.Artist,
		album: track.Album, playing: true,
	}
	if cur == p.last {
		return
	}

	if err := p.ensureConnected(); err != nil {
		p.logger.Warn().Err(err).Msg("Discord not available")
		return
	}

	start := time.Now().Add(-track.Position)
	end := start.Add(track.Duration)
	startUnix := start.Unix()
	endUnix := end.Unix()

	var largeImage string
	if p.artwork != nil {
		largeImage = p.artwork.Lookup(track.Artist, track.Album)
	}

	err := p.client.SetActivity(Activity{
		Type:    2, // Listening
		Name:    "Apple Music",
		Details: track.Name,
		State:   "by " + track.Artist,
		Timestamps: &Timestamps{
			Start: &startUnix,
			End:   &endUnix,
		},
		Assets: &Assets{
			LargeImage: largeImage,
			LargeText:  track.Album,
			SmallImage: "scribbles",
			SmallText:  "scribbles",
		},
	})
	if err != nil {
		p.logger.Warn().Err(err).Msg("Failed to set activity")
		p.close()
		return
	}
	p.last = cur
}

func (p *Presence) ensureConnected() error {
	if p.client != nil {
		return nil
	}
	client, err := p.connect(p.appID)
	if err != nil {
		return err
	}
	p.logger.Info().Msg("Connected to Discord")
	p.client = client
	return nil
}

func (p *Presence) clearActivity() {
	if p.client == nil {
		return
	}
	if err := p.client.SetActivity(Activity{}); err != nil {
		p.logger.Debug().Err(err).Msg("Failed to clear activity")
		p.close()
	}
}

func (p *Presence) close() {
	if p.client == nil {
		return
	}
	_ = p.client.Close()
	p.client = nil
}
