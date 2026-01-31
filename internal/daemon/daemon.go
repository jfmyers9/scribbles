package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/jfmyers9/scribbles/internal/scrobbler"
	"github.com/rs/zerolog"
)

// Config holds daemon configuration
type Config struct {
	PollInterval      time.Duration // How often to poll Music app
	StateFile         string        // Path to state persistence file
	QueueDB           string        // Path to scrobble queue database
	ProcessInterval   time.Duration // How often to process scrobble queue
	ScrobbleThreshold float64       // Percentage threshold (0.0-1.0) for scrobbling
}

// Daemon coordinates the music poller, state tracking, and scrobbling
type Daemon struct {
	config   Config
	client   music.Client
	scrobble *scrobbler.Client
	queue    *scrobbler.Queue
	state    *State
	poller   *Poller
	logger   zerolog.Logger
}

// New creates a new Daemon instance
func New(cfg Config, musicClient music.Client, scrobbleClient *scrobbler.Client, logger zerolog.Logger) (*Daemon, error) {
	// Create state
	state, err := NewState(cfg.StateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}

	// Create queue
	queue, err := scrobbler.NewQueue(cfg.QueueDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue: %w", err)
	}

	// Create poller
	poller := NewPoller(musicClient, cfg.PollInterval, logger)

	return &Daemon{
		config:   cfg,
		client:   musicClient,
		scrobble: scrobbleClient,
		queue:    queue,
		state:    state,
		poller:   poller,
		logger:   logger.With().Str("component", "daemon").Logger(),
	}, nil
}

// Run starts the daemon and blocks until shutdown signal received
func (d *Daemon) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Handle first signal gracefully, second signal forces exit
	go func() {
		<-sigChan
		d.logger.Info().Msg("Shutdown signal received, initiating graceful shutdown")
		cancel()

		// Second signal forces exit
		<-sigChan
		d.logger.Warn().Msg("Second shutdown signal received, forcing exit")
		os.Exit(1)
	}()

	// Run the daemon
	if err := d.run(ctx); err != nil && err != context.Canceled {
		return err
	}

	return nil
}

// run is the main daemon loop
func (d *Daemon) run(ctx context.Context) error {
	d.logger.Info().Msg("Starting daemon")

	var wg sync.WaitGroup
	updates := make(chan TrackUpdate, 10)

	// Start poller
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.poller.Run(ctx, updates); err != nil && err != context.Canceled {
			d.logger.Error().Err(err).Msg("Poller error")
		}
	}()

	// Start queue processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.processQueue(ctx); err != nil && err != context.Canceled {
			d.logger.Error().Err(err).Msg("Queue processor error")
		}
	}()

	// Start scrobble checker
	wg.Add(1)
	go func() {
		defer wg.Done()
		d.checkScrobbleEligibility(ctx)
	}()

	// Main loop: handle track updates
	wg.Add(1)
	go func() {
		defer wg.Done()
		d.handleUpdates(ctx, updates)
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	d.logger.Info().Msg("Daemon stopped")
	return nil
}

// handleUpdates processes track updates from the poller
func (d *Daemon) handleUpdates(ctx context.Context, updates <-chan TrackUpdate) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			if update.Err != nil {
				// Log error but continue
				d.logger.Debug().Err(update.Err).Msg("Track update error")
				continue
			}

			if err := d.handleTrackUpdate(update.Track); err != nil {
				d.logger.Error().Err(err).Msg("Failed to handle track update")
			}
		}
	}
}

// handleTrackUpdate processes a single track update
func (d *Daemon) handleTrackUpdate(track *music.Track) error {
	currentState := d.state.GetState()

	// No track playing - reset state if needed
	if track == nil || track.State == music.StateStopped {
		if currentState.Track != nil {
			d.logger.Info().Msg("Music stopped")
			return d.state.Reset()
		}
		return nil
	}

	// Check if track changed
	trackChanged := currentState.Track == nil ||
		!isSameTrack(currentState.Track, track)

	if trackChanged {
		d.logger.Info().
			Str("track", track.Name).
			Str("artist", track.Artist).
			Msg("Track changed")

		// Update state with new track
		if err := d.state.SetTrack(track); err != nil {
			return fmt.Errorf("failed to set track: %w", err)
		}

		// Update Now Playing on Last.fm
		ctx := context.Background()
		if err := d.scrobble.UpdateNowPlaying(ctx, track.Artist, track.Name, track.Album, track.Duration); err != nil {
			d.logger.Warn().Err(err).Msg("Failed to update Now Playing")
			// Not a fatal error, continue
		}

		return nil
	}

	// Same track - update position
	return d.state.UpdatePosition(track)
}

// checkScrobbleEligibility periodically checks if current track is ready to scrobble
func (d *Daemon) checkScrobbleEligibility(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.checkAndScrobble(); err != nil {
				d.logger.Error().Err(err).Msg("Failed to check scrobble eligibility")
			}
		}
	}
}

// checkAndScrobble checks if current track should be scrobbled and adds to queue
func (d *Daemon) checkAndScrobble() error {
	state := d.state.GetState()

	// No track or already scrobbled
	if state.Track == nil || state.Scrobbled {
		return nil
	}

	// Check if track is eligible for scrobbling
	playedDuration := d.state.GetPlayedDuration()
	if !scrobbler.ShouldScrobble(state.Track.Duration, playedDuration) {
		return nil
	}

	// Track is ready to scrobble
	d.logger.Info().
		Str("track", state.Track.Name).
		Str("artist", state.Track.Artist).
		Dur("played", playedDuration).
		Msg("Scrobbling track")

	// Add to queue
	timestamp := time.Now()
	ctx := context.Background()
	scrobble := scrobbler.Scrobble{
		Track:     state.Track.Name,
		Artist:    state.Track.Artist,
		Album:     state.Track.Album,
		Duration:  state.Track.Duration,
		Timestamp: timestamp,
	}
	if _, err := d.queue.Add(ctx, scrobble); err != nil {
		return fmt.Errorf("failed to add to queue: %w", err)
	}

	// Mark as scrobbled in state
	if err := d.state.MarkScrobbled(); err != nil {
		return fmt.Errorf("failed to mark scrobbled: %w", err)
	}

	return nil
}

// processQueue periodically processes pending scrobbles in the queue
func (d *Daemon) processQueue(ctx context.Context) error {
	ticker := time.NewTicker(d.config.ProcessInterval)
	defer ticker.Stop()

	// Process immediately on start
	d.processPendingScrobbles()

	for {
		select {
		case <-ctx.Done():
			// Final processing before shutdown
			d.logger.Info().Msg("Processing final scrobbles before shutdown")
			d.processPendingScrobbles()
			return ctx.Err()
		case <-ticker.C:
			d.processPendingScrobbles()
		}
	}
}

// processPendingScrobbles submits pending scrobbles to Last.fm
func (d *Daemon) processPendingScrobbles() {
	ctx := context.Background()
	pending, err := d.queue.GetPending(ctx, 50) // Last.fm allows batch of 50
	if err != nil {
		d.logger.Error().Err(err).Msg("Failed to get pending scrobbles")
		return
	}

	if len(pending) == 0 {
		return
	}

	d.logger.Info().Int("count", len(pending)).Msg("Processing pending scrobbles")

	// Submit in batch if more than one
	if len(pending) == 1 {
		queuedScrobble := pending[0]
		err := d.scrobble.ScrobbleTrack(
			ctx,
			queuedScrobble.Artist,
			queuedScrobble.TrackName,
			queuedScrobble.Album,
			queuedScrobble.Timestamp,
			queuedScrobble.Duration,
		)

		if err != nil {
			d.logger.Warn().
				Err(err).
				Int64("id", queuedScrobble.ID).
				Str("track", queuedScrobble.TrackName).
				Msg("Failed to scrobble")
			if markErr := d.queue.MarkError(ctx, queuedScrobble.ID, err.Error()); markErr != nil {
				d.logger.Error().Err(markErr).Msg("Failed to mark scrobble error")
			}
		} else {
			d.logger.Info().
				Str("track", queuedScrobble.TrackName).
				Str("artist", queuedScrobble.Artist).
				Msg("Scrobbled successfully")
			if markErr := d.queue.MarkScrobbled(ctx, queuedScrobble.ID); markErr != nil {
				d.logger.Error().Err(markErr).Msg("Failed to mark scrobble as completed")
			}
		}
	} else {
		// Batch scrobble - convert QueuedScrobble to Scrobble
		scrobbles := make([]scrobbler.Scrobble, len(pending))
		for i, qs := range pending {
			scrobbles[i] = scrobbler.Scrobble{
				Artist:    qs.Artist,
				Track:     qs.TrackName,
				Album:     qs.Album,
				Timestamp: qs.Timestamp,
				Duration:  qs.Duration,
			}
		}

		if err := d.scrobble.ScrobbleBatch(ctx, scrobbles); err != nil {
			d.logger.Warn().
				Err(err).
				Int("count", len(pending)).
				Msg("Batch scrobble failed")

			// Mark all as error
			for _, s := range pending {
				if markErr := d.queue.MarkError(ctx, s.ID, err.Error()); markErr != nil {
					d.logger.Error().Err(markErr).Int64("id", s.ID).Msg("Failed to mark scrobble error")
				}
			}
		} else {
			d.logger.Info().
				Int("count", len(pending)).
				Msg("Batch scrobbled successfully")

			// Mark all as scrobbled
			ids := make([]int64, len(pending))
			for i, s := range pending {
				ids[i] = s.ID
			}
			if markErr := d.queue.MarkScrobbledBatch(ctx, ids); markErr != nil {
				d.logger.Error().Err(markErr).Msg("Failed to mark batch as scrobbled")
			}
		}
	}
}

// Shutdown gracefully shuts down the daemon
func (d *Daemon) Shutdown() error {
	d.logger.Info().Msg("Shutting down daemon")

	ctx := context.Background()

	// Cleanup old records
	if _, err := d.queue.Cleanup(ctx, 7*24*time.Hour); err != nil {
		d.logger.Warn().Err(err).Msg("Failed to cleanup queue")
	}

	// Close queue
	if err := d.queue.Close(); err != nil {
		return fmt.Errorf("failed to close queue: %w", err)
	}

	return nil
}
