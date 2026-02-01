// Package lastfm provides a client for the Last.fm API 2.0.
//
// This package implements the Last.fm API for authentication,
// scrobbling, and other operations. It is designed to be used
// as a standalone SDK.
//
// Example usage:
//
//	import "github.com/jfmyers9/scribbles/pkg/lastfm"
//
//	client := lastfm.NewClient(lastfm.Config{
//	    APIKey:    "your-api-key",
//	    APISecret: "your-api-secret",
//	})
//
//	token, err := client.Auth().GetToken(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println("Authorize at:", client.Auth().GetAuthURL(token.Token))
package lastfm

import (
	"fmt"
	"net/http"
)

// Config holds client configuration.
type Config struct {
	APIKey     string       // Required: Last.fm API key
	APISecret  string       // Required: Last.fm API secret
	SessionKey string       // Optional: Session key for authenticated requests
	HTTPClient *http.Client // Optional: HTTP client (defaults to http.DefaultClient)
	BaseURL    string       // Optional: Base URL for API (defaults to Last.fm API, used for testing)
	Logger     Logger       // Optional: Logger interface for debug logging
}

// Logger is an optional interface for logging.
type Logger interface {
	// Debugf logs a debug message with format and arguments.
	Debugf(format string, args ...interface{})
}

// Client is the main entry point for Last.fm API operations.
type Client struct {
	apiKey     string
	apiSecret  string
	sessionKey string
	httpClient *http.Client
	baseURL    string
	logger     Logger

	auth     *AuthService
	scrobble *ScrobbleService
}

const (
	// DefaultBaseURL is the default Last.fm API endpoint.
	DefaultBaseURL = "https://ws.audioscrobbler.com/2.0/"
)

// NewClient creates a new Last.fm API client.
//
// Returns an error if required configuration (APIKey, APISecret) is missing.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("lastfm: APIKey is required")
	}
	if cfg.APISecret == "" {
		return nil, fmt.Errorf("lastfm: APISecret is required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	c := &Client{
		apiKey:     cfg.APIKey,
		apiSecret:  cfg.APISecret,
		sessionKey: cfg.SessionKey,
		httpClient: httpClient,
		baseURL:    baseURL,
		logger:     cfg.Logger,
	}

	c.auth = &AuthService{client: c}
	c.scrobble = &ScrobbleService{client: c}

	return c, nil
}

// Auth returns the authentication service.
func (c *Client) Auth() *AuthService {
	return c.auth
}

// Scrobble returns the scrobbling service.
func (c *Client) Scrobble() *ScrobbleService {
	return c.scrobble
}

// SetSessionKey sets the session key for authenticated requests.
func (c *Client) SetSessionKey(key string) {
	c.sessionKey = key
}

// GetSessionKey returns the current session key.
func (c *Client) GetSessionKey() string {
	return c.sessionKey
}

// logDebugf logs a debug message if a logger is configured.
func (c *Client) logDebugf(format string, args ...interface{}) {
	if c.logger != nil {
		c.logger.Debugf(format, args...)
	}
}
