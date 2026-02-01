package lastfm

import (
	"context"
	"encoding/xml"
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

	params := map[string]string{
		"artist": track.Artist,
		"track":  track.Track,
		"sk":     s.client.sessionKey,
	}

	// Add optional parameters
	if track.Album != "" {
		params["album"] = track.Album
	}
	if track.AlbumArtist != "" {
		params["albumArtist"] = track.AlbumArtist
	}
	if track.Duration > 0 {
		params["duration"] = fmt.Sprintf("%d", track.Duration)
	}
	if track.TrackNumber > 0 {
		params["trackNumber"] = fmt.Sprintf("%d", track.TrackNumber)
	}
	if track.MBTrackID != "" {
		params["mbid"] = track.MBTrackID
	}

	resp, err := s.client.call(ctx, "track.updateNowPlaying", params, true)
	if err != nil {
		return nil, err
	}

	nowPlaying, err := unmarshalNowPlaying(resp)
	if err != nil {
		return nil, fmt.Errorf("lastfm: failed to parse now playing response: %w", err)
	}

	return nowPlaying, nil
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

	params := map[string]string{
		"sk": s.client.sessionKey,
	}

	// Add batch parameters with indexed keys
	for i, scrobble := range scrobbles {
		idx := fmt.Sprintf("[%d]", i)
		params["artist"+idx] = scrobble.Track.Artist
		params["track"+idx] = scrobble.Track.Track
		params["timestamp"+idx] = fmt.Sprintf("%d", scrobble.Timestamp.Unix())

		// Add optional parameters
		if scrobble.Track.Album != "" {
			params["album"+idx] = scrobble.Track.Album
		}
		if scrobble.Track.AlbumArtist != "" {
			params["albumArtist"+idx] = scrobble.Track.AlbumArtist
		}
		if scrobble.Track.Duration > 0 {
			params["duration"+idx] = fmt.Sprintf("%d", scrobble.Track.Duration)
		}
		if scrobble.Track.TrackNumber > 0 {
			params["trackNumber"+idx] = fmt.Sprintf("%d", scrobble.Track.TrackNumber)
		}
		if scrobble.Track.MBTrackID != "" {
			params["mbid"+idx] = scrobble.Track.MBTrackID
		}
	}

	resp, err := s.client.call(ctx, "track.scrobble", params, true)
	if err != nil {
		return nil, err
	}

	scrobbleResp, err := unmarshalScrobbles(resp)
	if err != nil {
		return nil, fmt.Errorf("lastfm: failed to parse scrobble response: %w", err)
	}

	return scrobbleResp, nil
}

// nowPlayingResponse represents the XML response from track.updateNowPlaying.
type nowPlayingResponse struct {
	Artist      string `xml:"nowplaying>artist"`
	Track       string `xml:"nowplaying>track"`
	Album       string `xml:"nowplaying>album"`
	AlbumArtist string `xml:"nowplaying>albumArtist"`
	IgnoredMessage struct {
		Code int    `xml:"code,attr"`
		Text string `xml:",chardata"`
	} `xml:"nowplaying>ignoredMessage"`
}

// unmarshalNowPlaying parses the XML response from track.updateNowPlaying.
func unmarshalNowPlaying(data []byte) (*NowPlayingResponse, error) {
	// Wrap inner XML in root element for proper unmarshaling
	wrapped := []byte("<root>" + string(data) + "</root>")

	var resp nowPlayingResponse
	if err := xml.Unmarshal(wrapped, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal now playing response: %w", err)
	}

	return &NowPlayingResponse{
		Artist:      resp.Artist,
		Track:       resp.Track,
		Album:       resp.Album,
		AlbumArtist: resp.AlbumArtist,
		IgnoredMessage: struct {
			Code int
			Text string
		}{
			Code: resp.IgnoredMessage.Code,
			Text: resp.IgnoredMessage.Text,
		},
	}, nil
}

// scrobbleResponse represents the XML response from track.scrobble.
type scrobbleResponse struct {
	Scrobbles struct {
		Accepted  string `xml:"accepted,attr"`
		Ignored   string `xml:"ignored,attr"`
		Scrobbles []struct {
			Artist    string `xml:"artist"`
			Track     string `xml:"track"`
			Album     string `xml:"album"`
			Timestamp string `xml:"timestamp"`
			IgnoredMessage struct {
				Code int    `xml:"code,attr"`
				Text string `xml:",chardata"`
			} `xml:"ignoredMessage"`
		} `xml:"scrobble"`
	} `xml:"scrobbles"`
}

// unmarshalScrobbles parses the XML response from track.scrobble.
func unmarshalScrobbles(data []byte) (*ScrobbleResponse, error) {
	// Wrap inner XML in root element for proper unmarshaling
	wrapped := []byte("<root>" + string(data) + "</root>")

	var resp scrobbleResponse
	if err := xml.Unmarshal(wrapped, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scrobble response: %w", err)
	}

	// Parse accepted and ignored counts
	accepted := 0
	ignored := 0
	if resp.Scrobbles.Accepted != "" {
		fmt.Sscanf(resp.Scrobbles.Accepted, "%d", &accepted)
	}
	if resp.Scrobbles.Ignored != "" {
		fmt.Sscanf(resp.Scrobbles.Ignored, "%d", &ignored)
	}

	result := &ScrobbleResponse{
		Accepted:  accepted,
		Ignored:   ignored,
		Scrobbles: make([]struct {
			Artist    string
			Track     string
			Album     string
			Timestamp int64
			IgnoredMessage struct {
				Code int
				Text string
			}
		}, len(resp.Scrobbles.Scrobbles)),
	}

	for i, s := range resp.Scrobbles.Scrobbles {
		var timestamp int64
		if s.Timestamp != "" {
			fmt.Sscanf(s.Timestamp, "%d", &timestamp)
		}

		result.Scrobbles[i].Artist = s.Artist
		result.Scrobbles[i].Track = s.Track
		result.Scrobbles[i].Album = s.Album
		result.Scrobbles[i].Timestamp = timestamp
		result.Scrobbles[i].IgnoredMessage.Code = s.IgnoredMessage.Code
		result.Scrobbles[i].IgnoredMessage.Text = s.IgnoredMessage.Text
	}

	return result, nil
}
