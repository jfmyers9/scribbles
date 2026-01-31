package scrobbler

import (
	"context"
	"fmt"
	"time"

	"github.com/shkh/lastfm-go/lastfm"
)

// Client wraps the Last.fm API client
type Client struct {
	api *lastfm.Api
}

// New creates a new Last.fm client
func New(apiKey, apiSecret string) *Client {
	return &Client{
		api: lastfm.New(apiKey, apiSecret),
	}
}

// NewWithSession creates a new Last.fm client with an existing session key
func NewWithSession(apiKey, apiSecret, sessionKey string) *Client {
	api := lastfm.New(apiKey, apiSecret)
	api.SetSession(sessionKey)
	return &Client{
		api: api,
	}
}

// AuthenticateWithToken initiates the authentication flow
// Returns the auth URL that the user should visit
func (c *Client) AuthenticateWithToken(ctx context.Context) (token string, authURL string, err error) {
	// Get token from Last.fm
	token, err = c.api.GetToken()
	if err != nil {
		return "", "", fmt.Errorf("failed to get auth token: %w", err)
	}

	// Generate the auth URL the user should visit
	authURL = c.api.GetAuthTokenUrl(token)
	return token, authURL, nil
}

// GetSession completes the authentication flow after user authorization
// Returns the session key that should be stored for future use
func (c *Client) GetSession(ctx context.Context, token string) (sessionKey string, err error) {
	err = c.api.LoginWithToken(token)
	if err != nil {
		return "", fmt.Errorf("failed to login with token: %w", err)
	}

	sessionKey = c.api.GetSessionKey()
	if sessionKey == "" {
		return "", fmt.Errorf("received empty session key")
	}

	return sessionKey, nil
}

// UpdateNowPlaying updates the now playing status on Last.fm
func (c *Client) UpdateNowPlaying(ctx context.Context, artist, track, album string, duration time.Duration) error {
	params := lastfm.P{
		"artist": artist,
		"track":  track,
	}

	// Album is optional
	if album != "" {
		params["album"] = album
	}

	// Duration is optional but recommended (in seconds)
	if duration > 0 {
		params["duration"] = int(duration.Seconds())
	}

	_, err := c.api.Track.UpdateNowPlaying(params)
	if err != nil {
		return fmt.Errorf("failed to update now playing: %w", err)
	}

	return nil
}

// ScrobbleTrack submits a single scrobble to Last.fm
func (c *Client) ScrobbleTrack(ctx context.Context, artist, track, album string, timestamp time.Time, duration time.Duration) error {
	params := lastfm.P{
		"artist":    artist,
		"track":     track,
		"timestamp": timestamp.Unix(),
	}

	// Album is optional
	if album != "" {
		params["album"] = album
	}

	// Duration is optional but recommended (in seconds)
	if duration > 0 {
		params["duration"] = int(duration.Seconds())
	}

	result, err := c.api.Track.Scrobble(params)
	if err != nil {
		return fmt.Errorf("failed to scrobble track: %w", err)
	}

	// Check if the scrobble was accepted
	if result.Ignored != "0" {
		// Extract the ignore message if available
		if len(result.Scrobbles) > 0 && result.Scrobbles[0].IgnoredMessage.Body != "" {
			return fmt.Errorf("scrobble was ignored: %s", result.Scrobbles[0].IgnoredMessage.Body)
		}
		return fmt.Errorf("scrobble was ignored by Last.fm")
	}

	return nil
}

// ScrobbleBatch submits multiple scrobbles to Last.fm (up to 50)
func (c *Client) ScrobbleBatch(ctx context.Context, scrobbles []Scrobble) error {
	if len(scrobbles) == 0 {
		return nil
	}

	if len(scrobbles) > 50 {
		return fmt.Errorf("cannot scrobble more than 50 tracks at once (got %d)", len(scrobbles))
	}

	// Build batch parameters
	params := lastfm.P{}
	for i, s := range scrobbles {
		params[fmt.Sprintf("artist[%d]", i)] = s.Artist
		params[fmt.Sprintf("track[%d]", i)] = s.Track
		params[fmt.Sprintf("timestamp[%d]", i)] = s.Timestamp.Unix()

		if s.Album != "" {
			params[fmt.Sprintf("album[%d]", i)] = s.Album
		}

		if s.Duration > 0 {
			params[fmt.Sprintf("duration[%d]", i)] = int(s.Duration.Seconds())
		}
	}

	result, err := c.api.Track.Scrobble(params)
	if err != nil {
		return fmt.Errorf("failed to scrobble batch: %w", err)
	}

	// Check if any scrobbles were ignored
	if result.Ignored != "0" {
		return fmt.Errorf("%s scrobbles were ignored by Last.fm", result.Ignored)
	}

	return nil
}

// Scrobble represents a single scrobble to submit
type Scrobble struct {
	Artist    string
	Track     string
	Album     string
	Timestamp time.Time
	Duration  time.Duration
}

// IsAuthenticated checks if the client has a valid session
func (c *Client) IsAuthenticated() bool {
	return c.api.GetSessionKey() != ""
}

// GetSessionKey returns the current session key
func (c *Client) GetSessionKey() string {
	return c.api.GetSessionKey()
}
