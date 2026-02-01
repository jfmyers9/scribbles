package scrobbler

import (
	"context"
	"fmt"
	"time"

	"github.com/jfmyers9/scribbles/pkg/lastfm"
)

// Client wraps the Last.fm API client
type Client struct {
	client *lastfm.Client
}

// New creates a new Last.fm client
func New(apiKey, apiSecret string) *Client {
	client, err := lastfm.NewClient(lastfm.Config{
		APIKey:    apiKey,
		APISecret: apiSecret,
	})
	if err != nil {
		// This should never happen since we validate the inputs
		panic(fmt.Sprintf("failed to create lastfm client: %v", err))
	}
	return &Client{
		client: client,
	}
}

// NewWithSession creates a new Last.fm client with an existing session key
func NewWithSession(apiKey, apiSecret, sessionKey string) *Client {
	client, err := lastfm.NewClient(lastfm.Config{
		APIKey:     apiKey,
		APISecret:  apiSecret,
		SessionKey: sessionKey,
	})
	if err != nil {
		// This should never happen since we validate the inputs
		panic(fmt.Sprintf("failed to create lastfm client: %v", err))
	}
	return &Client{
		client: client,
	}
}

// AuthenticateWithToken initiates the authentication flow
// Returns the auth URL that the user should visit
func (c *Client) AuthenticateWithToken(ctx context.Context) (token string, authURL string, err error) {
	// Get token from Last.fm
	tokenResp, err := c.client.Auth().GetToken(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get auth token: %w", err)
	}

	// Generate the auth URL the user should visit
	authURL = c.client.Auth().GetAuthURL(tokenResp.Token)
	return tokenResp.Token, authURL, nil
}

// GetSession completes the authentication flow after user authorization
// Returns the session key that should be stored for future use
func (c *Client) GetSession(ctx context.Context, token string) (sessionKey string, err error) {
	session, err := c.client.Auth().GetSession(ctx, token)
	if err != nil {
		return "", fmt.Errorf("failed to login with token: %w", err)
	}

	if session.Key == "" {
		return "", fmt.Errorf("received empty session key")
	}

	// Update the client with the session key
	c.client.SetSessionKey(session.Key)

	return session.Key, nil
}

func (c *Client) UpdateNowPlaying(ctx context.Context, artist, track, album string, duration time.Duration) error {
	lfmTrack := lastfm.Track{
		Artist: artist,
		Track:  track,
		Album:  album,
	}

	if duration > 0 {
		lfmTrack.Duration = int(duration.Seconds())
	}

	_, err := c.client.Scrobble().UpdateNowPlaying(ctx, lfmTrack)
	if err != nil {
		return fmt.Errorf("failed to update now playing: %w", err)
	}

	return nil
}

func (c *Client) ScrobbleTrack(ctx context.Context, artist, track, album string, timestamp time.Time, duration time.Duration) error {
	lfmTrack := lastfm.Track{
		Artist: artist,
		Track:  track,
		Album:  album,
	}

	if duration > 0 {
		lfmTrack.Duration = int(duration.Seconds())
	}

	resp, err := c.client.Scrobble().Scrobble(ctx, lfmTrack, timestamp)
	if err != nil {
		return fmt.Errorf("failed to scrobble track: %w", err)
	}

	if resp.Ignored > 0 {
		if len(resp.Scrobbles) > 0 && resp.Scrobbles[0].IgnoredMessage.Text != "" {
			return fmt.Errorf("scrobble was ignored: %s", resp.Scrobbles[0].IgnoredMessage.Text)
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

	// Convert internal Scrobble type to pkg/lastfm Scrobble type
	lfmScrobbles := make([]lastfm.Scrobble, len(scrobbles))
	for i, s := range scrobbles {
		lfmScrobbles[i] = lastfm.Scrobble{
			Track: lastfm.Track{
				Artist:   s.Artist,
				Track:    s.Track,
				Album:    s.Album,
				Duration: int(s.Duration.Seconds()),
			},
			Timestamp: s.Timestamp,
		}
	}

	resp, err := c.client.Scrobble().ScrobbleBatch(ctx, lfmScrobbles)
	if err != nil {
		return fmt.Errorf("failed to scrobble batch: %w", err)
	}

	if resp.Ignored > 0 {
		return fmt.Errorf("%d scrobbles were ignored by Last.fm", resp.Ignored)
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
	return c.client.GetSessionKey() != ""
}

// GetSessionKey returns the current session key
func (c *Client) GetSessionKey() string {
	return c.client.GetSessionKey()
}
