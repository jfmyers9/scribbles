package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jfmyers9/scribbles/internal/daemon"
	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/rivo/tview"
)

const maxRecentTracks = 5

// Config holds TUI configuration options
type Config struct {
	RefreshRate time.Duration // How often to refresh the display
	Theme       string        // Color theme
}

// DefaultConfig returns the default TUI configuration
func DefaultConfig() Config {
	return Config{
		RefreshRate: 500 * time.Millisecond,
		Theme:       "default",
	}
}

// RecentTrack stores info about a recently played track
type RecentTrack struct {
	Name      string
	Artist    string
	Scrobbled bool
	PlayedAt  time.Time
}

// App is the TUI application for displaying music playback
type App struct {
	app        *tview.Application
	nowPlaying *tview.TextView
	progress   *tview.TextView
	status     *tview.TextView
	scrobble   *tview.TextView
	recent     *tview.TextView

	// Configuration
	config Config

	// Music client for controls
	musicClient music.Client

	// Mutex protects shared state accessed by both the channel consumer
	// goroutine and the ticker goroutine in handleUpdates.
	mu sync.Mutex

	// Current state (guarded by mu)
	currentTrack *music.Track
	trackState   *daemon.TrackState
	pendingCount int

	// Session stats (guarded by mu)
	sessionStart    time.Time
	tracksPlayed    int
	scrobblesSubmit int
	lastScrobbled   bool // tracks scrobble transition for accurate counting

	// Ring buffer for recent tracks (avoids allocation on every track change)
	recentBuf   [maxRecentTracks]RecentTrack
	recentCount int // total tracks added (recentCount % maxRecentTracks = next write index)

	// Last-rendered content for change detection
	lastNowPlaying string
	lastProgress   string
	lastScrobble   string
	lastRecent     string

	// Cached progress bar width to stabilize change detection.
	// Updated only when GetInnerRect returns a positive value.
	lastBarWidth int

	// Context cancel function
	cancelFunc context.CancelFunc
}

// New creates a new TUI application with default config
func New() *App {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new TUI application with the given config
func NewWithConfig(cfg Config) *App {
	a := &App{
		app:          tview.NewApplication(),
		config:       cfg,
		sessionStart: time.Now(),
	}
	a.setupUI()
	return a
}

// SetMusicClient sets the music client for playback controls
func (a *App) SetMusicClient(client music.Client) {
	a.musicClient = client
}

// setupUI creates the UI layout
func (a *App) setupUI() {
	// Now playing panel
	a.nowPlaying = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.nowPlaying.SetBorder(true).
		SetTitle(" Now Playing ").
		SetTitleAlign(tview.AlignLeft)

	// Progress bar
	a.progress = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.progress.SetBorder(true)

	// Scrobble status
	a.scrobble = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.scrobble.SetBorder(true).
		SetTitle(" Scrobble ").
		SetTitleAlign(tview.AlignLeft)

	// Recent tracks
	a.recent = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.recent.SetBorder(true).
		SetTitle(" Recent ").
		SetTitleAlign(tview.AlignLeft)

	// Status bar
	a.status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[gray]q:quit  space:play/pause  n:next  p:prev[-]")

	// Create layout
	// Top row: now playing (takes most space)
	// Middle row: progress bar
	// Bottom row: scrobble status | recent tracks
	// Footer: status bar

	bottomRow := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(a.scrobble, 0, 1, false).
		AddItem(a.recent, 0, 1, false)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.nowPlaying, 0, 3, false).
		AddItem(a.progress, 3, 1, false).
		AddItem(bottomRow, 7, 1, false).
		AddItem(a.status, 1, 1, false)

	// Handle keyboard input
	a.app.SetInputCapture(a.handleKeyEvent)

	a.app.SetRoot(flex, true)
}

// handleKeyEvent processes keyboard input
func (a *App) handleKeyEvent(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'q', 'Q':
		a.app.Stop()
		return nil
	case ' ':
		// Play/pause toggle
		if a.musicClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = a.musicClient.PlayPause(ctx)
		}
		return nil
	case 'n', 'N':
		// Next track
		if a.musicClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = a.musicClient.NextTrack(ctx)
		}
		return nil
	case 'p', 'P':
		// Previous track
		if a.musicClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = a.musicClient.PreviousTrack(ctx)
		}
		return nil
	}
	return event
}

// Run starts the TUI with a track update channel from the daemon
func (a *App) Run(ctx context.Context, updates <-chan daemon.TrackUpdate, stateGetter func() daemon.TrackState, playedGetter func() time.Duration) error {
	// Create cancellable context
	ctx, a.cancelFunc = context.WithCancel(ctx)

	// Start update goroutine
	go a.handleUpdates(ctx, updates, stateGetter, playedGetter)

	// Run application
	if err := a.app.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// handleUpdates processes track updates and refreshes the display.
// It splits work into two goroutines: one consumes channel updates (state only),
// and a single ticker drives all redraws to prevent queued redraw buildup.
// All shared App fields are protected by a.mu.
func (a *App) handleUpdates(ctx context.Context, updates <-chan daemon.TrackUpdate, stateGetter func() daemon.TrackState, playedGetter func() time.Duration) {
	var lastTrackName string

	// Channel consumer goroutine: updates track info but does NOT trigger redraws.
	// The ticker goroutine is the sole caller of stateGetter() and refresh().
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-updates:
				if update.Err != nil {
					continue
				}

				a.mu.Lock()
				// Check for track change
				if update.Track != nil && update.Track.Name != lastTrackName {
					// Add previous track to recent list
					if a.currentTrack != nil && lastTrackName != "" {
						a.addToRecentTracks(a.currentTrack, a.trackState)
						a.tracksPlayed++
					}
					lastTrackName = update.Track.Name
				}

				a.currentTrack = update.Track
				a.mu.Unlock()
			}
		}
	}()

	// Single refresh ticker: the only source of redraws
	refreshRate := a.config.RefreshRate
	if refreshRate <= 0 {
		refreshRate = 500 * time.Millisecond
	}
	ticker := time.NewTicker(refreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.app.Stop()
			return
		case <-ticker.C:
			a.mu.Lock()
			if stateGetter != nil {
				state := stateGetter()
				a.trackState = &state

				// Increment scrobble count on transition from not-scrobbled to scrobbled
				if a.trackState.Scrobbled && !a.lastScrobbled {
					a.scrobblesSubmit++
				}
				a.lastScrobbled = a.trackState.Scrobbled
			}
			a.mu.Unlock()
			a.refresh(playedGetter)
		}
	}
}

// addToRecentTracks adds a track to the ring buffer of recent tracks.
// Must be called with a.mu held.
func (a *App) addToRecentTracks(track *music.Track, state *daemon.TrackState) {
	if track == nil {
		return
	}

	scrobbled := false
	if state != nil {
		scrobbled = state.Scrobbled
	}

	// Write into ring buffer at the current position
	idx := a.recentCount % maxRecentTracks
	a.recentBuf[idx] = RecentTrack{
		Name:      track.Name,
		Artist:    track.Artist,
		Scrobbled: scrobbled,
		PlayedAt:  time.Now(),
	}
	a.recentCount++
}

// getRecentTracks returns recent tracks in most-recent-first order.
// Must be called with a.mu held.
func (a *App) getRecentTracks() []RecentTrack {
	n := a.recentCount
	if n > maxRecentTracks {
		n = maxRecentTracks
	}
	result := make([]RecentTrack, n)
	for i := 0; i < n; i++ {
		// Walk backwards from the most recently written slot
		idx := (a.recentCount - 1 - i) % maxRecentTracks
		result[i] = a.recentBuf[idx]
	}
	return result
}

// refresh updates all UI components
func (a *App) refresh(playedGetter func() time.Duration) {
	a.app.QueueUpdateDraw(func() {
		a.mu.Lock()
		defer a.mu.Unlock()

		a.updateNowPlaying()
		a.updateProgress(playedGetter)
		a.updateScrobbleStatus(playedGetter)
		a.updateRecentTracks()
	})
}

// updateNowPlaying updates the now playing panel
func (a *App) updateNowPlaying() {
	var text string

	if a.currentTrack == nil || a.currentTrack.State == music.StateStopped {
		text = "\n\n[gray]No track playing[-]"
	} else {
		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("[white::b]%s[-:-:-]\n", tview.Escape(a.currentTrack.Name)))
		sb.WriteString(fmt.Sprintf("[yellow]%s[-]\n", tview.Escape(a.currentTrack.Artist)))
		sb.WriteString(fmt.Sprintf("[gray]%s[-]", tview.Escape(a.currentTrack.Album)))

		// Play state indicator
		stateIcon := "[green]\u25B6[-]" // Play triangle
		if a.currentTrack.State == music.StatePaused {
			stateIcon = "[yellow]\u23F8[-]" // Pause icon
		}
		sb.WriteString(fmt.Sprintf("\n\n%s", stateIcon))
		text = sb.String()
	}

	if text != a.lastNowPlaying {
		a.lastNowPlaying = text
		a.nowPlaying.SetText(text)
	}
}

// updateProgress updates the progress bar
func (a *App) updateProgress(playedGetter func() time.Duration) {
	var text string

	if a.currentTrack == nil || a.currentTrack.State == music.StateStopped {
		text = ""
	} else {
		_, _, width, _ := a.progress.GetInnerRect()
		barWidth := width - 14 // Account for time display
		// Only update cached width when GetInnerRect returns a positive value,
		// avoiding flicker from transient zero-width during layout.
		if barWidth > 0 {
			a.lastBarWidth = barWidth
		}
		if a.lastBarWidth < 10 {
			a.lastBarWidth = 10
		}

		progressBar := buildProgressBar(a.currentTrack.Position, a.currentTrack.Duration, a.lastBarWidth)
		posStr := formatDuration(a.currentTrack.Position)
		durStr := formatDuration(a.currentTrack.Duration)
		text = fmt.Sprintf("%s %s %s", posStr, progressBar, durStr)
	}

	if text != a.lastProgress {
		a.lastProgress = text
		a.progress.SetText(text)
	}
}

// updateScrobbleStatus updates the scrobble status panel
func (a *App) updateScrobbleStatus(playedGetter func() time.Duration) {
	var sb strings.Builder

	if a.trackState == nil || a.currentTrack == nil || a.currentTrack.State == music.StateStopped {
		sb.WriteString("[gray]No track[-]\n\n")
		sb.WriteString(fmt.Sprintf("Pending: %d\n", a.pendingCount))
		sb.WriteString(fmt.Sprintf("Session: %s", formatDuration(time.Since(a.sessionStart))))
	} else {
		// Scrobble progress
		if a.trackState.Scrobbled {
			sb.WriteString("[green]\u2713 Scrobbled[-]\n")
		} else if a.currentTrack.Duration > 0 && playedGetter != nil {
			played := playedGetter()
			threshold := a.currentTrack.Duration / 2
			if threshold > 4*time.Minute {
				threshold = 4 * time.Minute
			}
			progress := float64(played) / float64(threshold) * 100
			if progress > 100 {
				progress = 100
			}

			// Visual progress indicator
			barWidth := 10
			filled := int(progress / 100 * float64(barWidth))
			bar := strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", barWidth-filled)
			sb.WriteString(fmt.Sprintf("[yellow]%s %.0f%%[-]\n", bar, progress))
		} else {
			sb.WriteString("[gray]Waiting...[-]\n")
		}

		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("Pending: %d\n", a.pendingCount))
		sb.WriteString(fmt.Sprintf("Session: %s", formatDuration(time.Since(a.sessionStart))))
	}

	text := sb.String()
	if text != a.lastScrobble {
		a.lastScrobble = text
		a.scrobble.SetText(text)
	}
}

// updateRecentTracks updates the recent tracks panel
func (a *App) updateRecentTracks() {
	var sb strings.Builder

	tracks := a.getRecentTracks()
	if len(tracks) == 0 {
		sb.WriteString("[gray]No recent tracks[-]")
	} else {
		for i, track := range tracks {
			if i > 0 {
				sb.WriteString("\n")
			}

			// Scrobble indicator
			if track.Scrobbled {
				sb.WriteString("[green]\u2713[-] ")
			} else {
				sb.WriteString("[red]\u2717[-] ")
			}

			// Truncate name if too long
			name := track.Name
			if len(name) > 20 {
				name = name[:17] + "..."
			}
			sb.WriteString(fmt.Sprintf("[white]%s[-]", tview.Escape(name)))
		}
	}

	text := sb.String()
	if text != a.lastRecent {
		a.lastRecent = text
		a.recent.SetText(text)
	}
}

// SetPendingCount updates the pending scrobble count
func (a *App) SetPendingCount(count int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pendingCount = count
}

// Stop stops the TUI application
func (a *App) Stop() {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	a.app.Stop()
}

// buildProgressBar creates a text-based progress bar
func buildProgressBar(position, duration time.Duration, width int) string {
	if duration == 0 || width <= 0 {
		return strings.Repeat("-", width)
	}

	progress := float64(position) / float64(duration)
	if progress > 1 {
		progress = 1
	}
	if progress < 0 {
		progress = 0
	}

	filled := int(progress * float64(width))
	empty := width - filled

	bar := "[green]" + strings.Repeat("\u2588", filled) + "[-]" +
		"[gray]" + strings.Repeat("\u2591", empty) + "[-]"

	return bar
}

// formatDuration formats a duration as MM:SS or HH:MM:SS for longer durations
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
