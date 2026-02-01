package lastfm

import (
	"context"
	"fmt"
	"time"
)

// ScrobbleService provides scrobbling operations for the Last.fm API.
type ScrobbleService struct {
	client *Client
}

const (
	// MaxBatchSize is the maximum number of scrobbles allowed in a single batch.
	MaxBatchSize = 50
)

// UpdateNowPlaying updates the "now playing" status on Last.fm.
//
// This should be called when a track starts playing. It does not count
// as a scrobble and does not affect play counts.
//
// Requires authentication (session key must be set via SetSessionKey).
//
// Example:
//
//	track := lastfm.Track{
//	    Artist: "The Beatles",
//	    Track:  "Yesterday",
//	    Album:  "Help!",
//	}
//	err := client.Scrobble().UpdateNowPlaying(ctx, track)
//	if err != nil {
//	    log.Printf("Failed to update now playing: %v", err)
//	}
func (s *ScrobbleService) UpdateNowPlaying(ctx context.Context, track Track) (*NowPlayingResponse, error) {
	if s.client.sessionKey == "" {
		return nil, fmt.Errorf("lastfm: session key required for scrobbling")
	}
	// Implementation will be added in core implementation phase
	return nil, nil
}

// Scrobble submits a single scrobble to Last.fm.
//
// A track should only be scrobbled when:
// - The track is longer than 30 seconds, AND
// - The track has been played for at least 50% of its duration OR 4 minutes
//   (whichever comes first)
//
// Requires authentication (session key must be set via SetSessionKey).
//
// Example:
//
//	track := lastfm.Track{
//	    Artist:   "The Beatles",
//	    Track:    "Yesterday",
//	    Album:    "Help!",
//	    Duration: 123,
//	}
//	timestamp := time.Now().Add(-2 * time.Minute)
//	err := client.Scrobble().Scrobble(ctx, track, timestamp)
//	if err != nil {
//	    log.Printf("Failed to scrobble: %v", err)
//	}
func (s *ScrobbleService) Scrobble(ctx context.Context, track Track, timestamp time.Time) (*ScrobbleResponse, error) {
	if s.client.sessionKey == "" {
		return nil, fmt.Errorf("lastfm: session key required for scrobbling")
	}
	scrobbles := []Scrobble{{Track: track, Timestamp: timestamp}}
	return s.ScrobbleBatch(ctx, scrobbles)
}

// ScrobbleBatch submits multiple scrobbles to Last.fm in a single request.
//
// Up to 50 scrobbles can be submitted at once. If more than 50 scrobbles
// are provided, only the first 50 will be submitted.
//
// Each scrobble should meet the same criteria as Scrobble().
//
// Requires authentication (session key must be set via SetSessionKey).
//
// Example:
//
//	scrobbles := []lastfm.Scrobble{
//	    {
//	        Track: lastfm.Track{
//	            Artist: "The Beatles",
//	            Track:  "Yesterday",
//	        },
//	        Timestamp: time.Now().Add(-10 * time.Minute),
//	    },
//	    {
//	        Track: lastfm.Track{
//	            Artist: "The Beatles",
//	            Track:  "Let It Be",
//	        },
//	        Timestamp: time.Now().Add(-5 * time.Minute),
//	    },
//	}
//	resp, err := client.Scrobble().ScrobbleBatch(ctx, scrobbles)
//	if err != nil {
//	    log.Printf("Failed to scrobble batch: %v", err)
//	}
//	fmt.Printf("Accepted: %d, Ignored: %d\n", resp.Accepted, resp.Ignored)
func (s *ScrobbleService) ScrobbleBatch(ctx context.Context, scrobbles []Scrobble) (*ScrobbleResponse, error) {
	if s.client.sessionKey == "" {
		return nil, fmt.Errorf("lastfm: session key required for scrobbling")
	}
	if len(scrobbles) == 0 {
		return &ScrobbleResponse{}, nil
	}
	if len(scrobbles) > MaxBatchSize {
		scrobbles = scrobbles[:MaxBatchSize]
	}
	// Implementation will be added in core implementation phase
	return nil, nil
}
