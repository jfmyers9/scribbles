//go:build !darwin

package menubar

import "context"

// Run is unavailable outside macOS. The command rejects --menu-bar before
// reaching this stub; it exists so Linux builds do not require GUI libraries.
func Run(ctx context.Context, getSnapshot func() Snapshot) {
	<-ctx.Done()
}
