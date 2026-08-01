package menubar

import (
	"context"
	"fmt"
	"time"

	"github.com/getlantern/systray"
	"github.com/jfmyers9/scribbles/internal/music"
	"github.com/jfmyers9/scribbles/internal/scrobbler"
)

// Snapshot is the daemon state shown in the menu bar menu.
type Snapshot struct {
	Track     *music.Track
	Scrobbled bool
	Played    time.Duration
	Pending   int
}

// Run displays the menu bar item until ctx is canceled or Quit is selected.
func Run(ctx context.Context, getSnapshot func() Snapshot) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	exit := make(chan struct{})
	go func() {
		select {
		case <-runCtx.Done():
			systray.Quit()
		case <-exit:
		}
	}()

	systray.Run(func() {
		systray.SetTitle("♫ Active")
		systray.SetTooltip("Scribbles is active")

		status := systray.AddMenuItem("Scribbles is active", "")
		track := systray.AddMenuItem("Now playing: Nothing", "")
		scrobble := systray.AddMenuItem("Scrobble: Waiting for a track", "")
		queue := systray.AddMenuItem("Queue: 0 pending", "")
		status.Disable()
		track.Disable()
		scrobble.Disable()
		queue.Disable()
		systray.AddSeparator()
		quit := systray.AddMenuItem("Quit Scribbles", "Stop the daemon")

		update := func() {
			view := render(getSnapshot())
			systray.SetTitle(view.title)
			systray.SetTooltip(view.tooltip)
			status.SetTitle(view.status)
			track.SetTitle(view.track)
			scrobble.SetTitle(view.scrobble)
			queue.SetTitle(view.queue)
		}
		update()

		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-runCtx.Done():
					return
				case <-exit:
					return
				case <-ticker.C:
					update()
				}
			}
		}()

		go func() {
			<-quit.ClickedCh
			systray.Quit()
		}()
	}, func() {
		close(exit)
	})
}

type view struct {
	title    string
	tooltip  string
	status   string
	track    string
	scrobble string
	queue    string
}

func render(snapshot Snapshot) view {
	if snapshot.Track == nil || snapshot.Track.State == music.StateStopped {
		return view{
			title:    "♫ Active",
			tooltip:  "Scribbles is active - nothing playing",
			status:   "Scribbles is active",
			track:    "Now playing: Nothing",
			scrobble: "Scrobble: Waiting for a track",
			queue:    fmt.Sprintf("Queue: %d pending", snapshot.Pending),
		}
	}

	track := snapshot.Track.Artist + " - " + snapshot.Track.Name
	state := "Now playing: " + track
	if snapshot.Track.State == music.StatePaused {
		state = "Paused: " + track
	}

	scrobble := "Scrobble: Waiting"
	if snapshot.Scrobbled {
		scrobble = "Scrobble: Queued"
	} else {
		threshold := scrobbler.ScrobbleThreshold(snapshot.Track.Duration)
		if threshold < 0 {
			scrobble = "Scrobble: Not eligible (<30s)"
		} else if threshold > 0 {
			progress := int(snapshot.Played * 100 / threshold)
			if progress > 100 {
				progress = 100
			}
			scrobble = fmt.Sprintf("Scrobble: %d%% (target 50%%)", progress)
		}
	}

	return view{
		title:    "♫ Active",
		tooltip:  "Scribbles is active - " + track,
		status:   "Scribbles is active",
		track:    state,
		scrobble: scrobble,
		queue:    fmt.Sprintf("Queue: %d pending", snapshot.Pending),
	}
}
