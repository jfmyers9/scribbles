package music

import "runtime"

// NewClient creates the music client appropriate for the current platform.
// Linux media players, including browser tabs, are discovered through MPRIS.
func NewClient() Client {
	if runtime.GOOS == "linux" {
		return NewMPRISClient()
	}

	return NewAppleScriptClient()
}
