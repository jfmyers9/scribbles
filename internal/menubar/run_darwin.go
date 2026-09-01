//go:build darwin

package menubar

import (
	"context"
	"time"

	"github.com/getlantern/systray"
)

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
