package scrobbler

import (
	"context"
	"fmt"
	"net"
	"net/url"
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

func (c *Client) UpdateNowPlaying(ctx context.Context, artist, track, album string, duration time.Duration) error {
	params := lastfm.P{
		"artist": artist,
		"track":  track,
	}

	if album != "" {
		params["album"] = album
	}

	if duration > 0 {
		params["duration"] = int(duration.Seconds())
	}

	err := retryWithBackoff(ctx, 3, func() error {
		_, err := c.api.Track.UpdateNowPlaying(params)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to update now playing: %w", err)
	}

	return nil
}

func (c *Client) ScrobbleTrack(ctx context.Context, artist, track, album string, timestamp time.Time, duration time.Duration) error {
	params := lastfm.P{
		"artist":    artist,
		"track":     track,
		"timestamp": timestamp.Unix(),
	}

	if album != "" {
		params["album"] = album
	}

	if duration > 0 {
		params["duration"] = int(duration.Seconds())
	}

	var result lastfm.TrackScrobble

	err := retryWithBackoff(ctx, 3, func() error {
		var err error
		result, err = c.api.Track.Scrobble(params)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to scrobble track: %w", err)
	}

	if result.Ignored != "0" {
		if len(result.Scrobbles) > 0 && result.Scrobbles[0].IgnoredMessage.Body != "" {
			return fmt.Errorf("scrobble was ignored: %s", result.Scrobbles[0].IgnoredMessage.Body)
		}
		return fmt.Errorf("scrobble was ignored by Last.fm")
	}

	return nil
}

func (c *Client) ScrobbleBatch(ctx context.Context, scrobbles []Scrobble) error {
	if len(scrobbles) == 0 {
		return nil
	}

	if len(scrobbles) > 50 {
		return fmt.Errorf("cannot scrobble more than 50 tracks at once (got %d)", len(scrobbles))
	}

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

	var result lastfm.TrackScrobble

	err := retryWithBackoff(ctx, 3, func() error {
		var err error
		result, err = c.api.Track.Scrobble(params)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to scrobble batch: %w", err)
	}

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

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if _, ok := err.(net.Error); ok {
		return true
	}

	if _, ok := err.(*url.Error); ok {
		return true
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	return false
}

func retryWithBackoff(ctx context.Context, maxRetries int, fn func() error) error {
	var lastErr error
	backoff := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetryableError(err) {
			return err
		}

		if i == maxRetries-1 {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
