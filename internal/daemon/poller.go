package daemon

import (
	"context"
	"time"

	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/rs/zerolog"
)

// TrackUpdate represents an update from the music client
type TrackUpdate struct {
	Track *music.Track // Current track (nil if stopped/no track)
	Err   error        // Error from music client
}

// Poller polls the music client at regular intervals
type Poller struct {
	client   music.Client
	interval time.Duration
	logger   zerolog.Logger
}

// NewPoller creates a new Poller instance
func NewPoller(client music.Client, interval time.Duration, logger zerolog.Logger) *Poller {
	return &Poller{
		client:   client,
		interval: interval,
		logger:   logger.With().Str("component", "poller").Logger(),
	}
}

// Run starts the polling loop and sends updates to the provided channel
// Blocks until context is cancelled
func (p *Poller) Run(ctx context.Context, updates chan<- TrackUpdate) error {
	p.logger.Info().
		Dur("interval", p.interval).
		Msg("Starting poller")

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Poll immediately on start
	p.poll(ctx, updates)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info().Msg("Poller stopped")
			return ctx.Err()
		case <-ticker.C:
			p.poll(ctx, updates)
		}
	}
}

// poll queries the music client and sends an update
func (p *Poller) poll(ctx context.Context, updates chan<- TrackUpdate) {
	track, err := p.client.GetCurrentTrack(ctx)
	if err != nil {
		p.logger.Debug().Err(err).Msg("Error getting current track")
		// Send error update (non-blocking)
		select {
		case updates <- TrackUpdate{Err: err}:
		case <-ctx.Done():
		}
		return
	}

	// Send update (non-blocking)
	select {
	case updates <- TrackUpdate{Track: track}:
		if track != nil {
			p.logger.Debug().
				Str("track", track.Name).
				Str("artist", track.Artist).
				Str("state", track.State.String()).
				Msg("Poll update")
		}
	case <-ctx.Done():
	}
}
