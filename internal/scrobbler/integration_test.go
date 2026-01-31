// +build integration

package scrobbler

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestIntegration_LastFmAuth tests the authentication flow
// Run with: go test -tags=integration -v ./internal/scrobbler/
// Requires: LASTFM_API_KEY and LASTFM_API_SECRET environment variables
func TestIntegration_LastFmAuth(t *testing.T) {
	apiKey := os.Getenv("LASTFM_API_KEY")
	apiSecret := os.Getenv("LASTFM_API_SECRET")

	if apiKey == "" || apiSecret == "" {
		t.Skip("Skipping integration test: LASTFM_API_KEY and LASTFM_API_SECRET must be set")
	}

	client := New(apiKey, apiSecret)
	ctx := context.Background()

	// Test getting auth token
	token, authURL, err := client.AuthenticateWithToken(ctx)
	if err != nil {
		t.Fatalf("Failed to get auth token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	if authURL == "" {
		t.Error("Expected non-empty auth URL")
	}

	t.Logf("Auth URL: %s", authURL)
	t.Log("Please visit the URL above to authorize, then set LASTFM_TOKEN env var and run the session test")
}

// TestIntegration_GetSession tests getting a session key
// Run with: LASTFM_TOKEN=<token> go test -tags=integration -v -run TestIntegration_GetSession ./internal/scrobbler/
// Requires: LASTFM_API_KEY, LASTFM_API_SECRET, and LASTFM_TOKEN environment variables
func TestIntegration_GetSession(t *testing.T) {
	apiKey := os.Getenv("LASTFM_API_KEY")
	apiSecret := os.Getenv("LASTFM_API_SECRET")
	token := os.Getenv("LASTFM_TOKEN")

	if apiKey == "" || apiSecret == "" || token == "" {
		t.Skip("Skipping integration test: LASTFM_API_KEY, LASTFM_API_SECRET, and LASTFM_TOKEN must be set")
	}

	client := New(apiKey, apiSecret)
	ctx := context.Background()

	sessionKey, err := client.GetSession(ctx, token)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if sessionKey == "" {
		t.Error("Expected non-empty session key")
	}

	t.Logf("Session key: %s", sessionKey)
	t.Log("Save this session key for future tests")
}

// TestIntegration_UpdateNowPlaying tests updating now playing status
// Run with: LASTFM_SESSION_KEY=<key> go test -tags=integration -v -run TestIntegration_UpdateNowPlaying ./internal/scrobbler/
// Requires: LASTFM_API_KEY, LASTFM_API_SECRET, and LASTFM_SESSION_KEY environment variables
func TestIntegration_UpdateNowPlaying(t *testing.T) {
	apiKey := os.Getenv("LASTFM_API_KEY")
	apiSecret := os.Getenv("LASTFM_API_SECRET")
	sessionKey := os.Getenv("LASTFM_SESSION_KEY")

	if apiKey == "" || apiSecret == "" || sessionKey == "" {
		t.Skip("Skipping integration test: LASTFM_API_KEY, LASTFM_API_SECRET, and LASTFM_SESSION_KEY must be set")
	}

	client := NewWithSession(apiKey, apiSecret, sessionKey)
	ctx := context.Background()

	err := client.UpdateNowPlaying(ctx, "Test Artist", "Test Track", "Test Album", 3*time.Minute)
	if err != nil {
		t.Fatalf("Failed to update now playing: %v", err)
	}

	t.Log("Successfully updated now playing")
}

// TestIntegration_Scrobble tests scrobbling a track
// Run with: LASTFM_SESSION_KEY=<key> go test -tags=integration -v -run TestIntegration_Scrobble ./internal/scrobbler/
// Requires: LASTFM_API_KEY, LASTFM_API_SECRET, and LASTFM_SESSION_KEY environment variables
func TestIntegration_Scrobble(t *testing.T) {
	apiKey := os.Getenv("LASTFM_API_KEY")
	apiSecret := os.Getenv("LASTFM_API_SECRET")
	sessionKey := os.Getenv("LASTFM_SESSION_KEY")

	if apiKey == "" || apiSecret == "" || sessionKey == "" {
		t.Skip("Skipping integration test: LASTFM_API_KEY, LASTFM_API_SECRET, and LASTFM_SESSION_KEY must be set")
	}

	client := NewWithSession(apiKey, apiSecret, sessionKey)
	ctx := context.Background()

	// Scrobble a track from 5 minutes ago
	timestamp := time.Now().Add(-5 * time.Minute)
	err := client.ScrobbleTrack(ctx, "Test Artist", "Test Track", "Test Album", timestamp, 3*time.Minute)
	if err != nil {
		t.Fatalf("Failed to scrobble track: %v", err)
	}

	t.Log("Successfully scrobbled track")
}

// TestIntegration_ScrobbleBatch tests batch scrobbling
// Run with: LASTFM_SESSION_KEY=<key> go test -tags=integration -v -run TestIntegration_ScrobbleBatch ./internal/scrobbler/
// Requires: LASTFM_API_KEY, LASTFM_API_SECRET, and LASTFM_SESSION_KEY environment variables
func TestIntegration_ScrobbleBatch(t *testing.T) {
	apiKey := os.Getenv("LASTFM_API_KEY")
	apiSecret := os.Getenv("LASTFM_API_SECRET")
	sessionKey := os.Getenv("LASTFM_SESSION_KEY")

	if apiKey == "" || apiSecret == "" || sessionKey == "" {
		t.Skip("Skipping integration test: LASTFM_API_KEY, LASTFM_API_SECRET, and LASTFM_SESSION_KEY must be set")
	}

	client := NewWithSession(apiKey, apiSecret, sessionKey)
	ctx := context.Background()

	// Create batch of scrobbles
	now := time.Now()
	scrobbles := []Scrobble{
		{
			Artist:    "Test Artist 1",
			Track:     "Test Track 1",
			Album:     "Test Album 1",
			Timestamp: now.Add(-10 * time.Minute),
			Duration:  3 * time.Minute,
		},
		{
			Artist:    "Test Artist 2",
			Track:     "Test Track 2",
			Album:     "Test Album 2",
			Timestamp: now.Add(-7 * time.Minute),
			Duration:  4 * time.Minute,
		},
	}

	err := client.ScrobbleBatch(ctx, scrobbles)
	if err != nil {
		t.Fatalf("Failed to scrobble batch: %v", err)
	}

	t.Logf("Successfully scrobbled %d tracks", len(scrobbles))
}
