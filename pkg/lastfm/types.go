package lastfm

import (
	"time"
)

// Track represents a music track for scrobbling or now playing updates.
type Track struct {
	Artist      string    // Required: Artist name
	Track       string    // Required: Track name
	Album       string    // Optional: Album name
	AlbumArtist string    // Optional: Album artist (if different from track artist)
	Duration    int       // Optional: Track duration in seconds
	TrackNumber int       // Optional: Track number on album
	MBTrackID   string    // Optional: MusicBrainz track ID
}

// Scrobble represents a single scrobble with timestamp.
type Scrobble struct {
	Track     Track     // The track being scrobbled
	Timestamp time.Time // When the track was played
}

// Token represents an authentication token from auth.getToken.
type Token struct {
	Token string // The authentication token
}

// Session represents an authenticated session from auth.getSession.
type Session struct {
	Key        string // Session key for authenticated requests
	Username   string // Last.fm username
	Subscriber bool   // Whether user is a subscriber
}

// NowPlayingResponse represents the response from track.updateNowPlaying.
type NowPlayingResponse struct {
	Artist      string
	Track       string
	Album       string
	AlbumArtist string
	IgnoredMessage struct {
		Code int
		Text string
	}
}

// ScrobbleResponse represents the response from track.scrobble.
type ScrobbleResponse struct {
	Accepted int // Number of scrobbles accepted
	Ignored  int // Number of scrobbles ignored
	Scrobbles []struct {
		Artist    string
		Track     string
		Album     string
		Timestamp int64
		IgnoredMessage struct {
			Code int
			Text string
		}
	}
}
