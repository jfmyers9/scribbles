package scrobbler

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	client := New("test_key", "test_secret")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewWithSession(t *testing.T) {
	client := NewWithSession("test_key", "test_secret", "test_session")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if !client.IsAuthenticated() {
		t.Error("expected client to be authenticated")
	}
	if client.GetSessionKey() != "test_session" {
		t.Errorf("expected session key 'test_session', got '%s'", client.GetSessionKey())
	}
}

// TestAuthenticateWithToken is an integration test that requires valid API credentials
// Skip in unit tests - use for manual testing
func TestAuthenticateWithToken(t *testing.T) {
	t.Skip("Integration test - requires valid Last.fm API credentials")
}

// TestGetSession is an integration test that requires valid API credentials
// Skip in unit tests - use for manual testing
func TestGetSession(t *testing.T) {
	t.Skip("Integration test - requires valid Last.fm API credentials")
}

// TestUpdateNowPlaying is an integration test that requires valid API credentials
// Skip in unit tests - use for manual testing
func TestUpdateNowPlaying(t *testing.T) {
	t.Skip("Integration test - requires valid Last.fm API credentials and session")
}

// TestScrobbleTrack is an integration test that requires valid API credentials
// Skip in unit tests - use for manual testing
func TestScrobbleTrack(t *testing.T) {
	t.Skip("Integration test - requires valid Last.fm API credentials and session")
}

func TestScrobbleBatch(t *testing.T) {
	tests := []struct {
		name        string
		scrobbles   []Scrobble
		expectError bool
		errorMsg    string
	}{
		{
			name:      "empty batch",
			scrobbles: []Scrobble{},
		},
		{
			name:        "batch too large",
			scrobbles:   make([]Scrobble, 51),
			expectError: true,
			errorMsg:    "cannot scrobble more than 50 tracks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewWithSession("test_key", "test_secret", "test_session")
			ctx := context.Background()
			err := client.ScrobbleBatch(ctx, tt.scrobbles)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errorMsg != "" && err.Error()[:len(tt.errorMsg)] != tt.errorMsg {
					t.Errorf("expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("ScrobbleBatch failed: %v", err)
				}
			}
		})
	}
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name           string
		sessionKey     string
		authenticated  bool
	}{
		{
			name:          "with session key",
			sessionKey:    "test_session",
			authenticated: true,
		},
		{
			name:          "without session key",
			sessionKey:    "",
			authenticated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *Client
			if tt.sessionKey != "" {
				client = NewWithSession("test_key", "test_secret", tt.sessionKey)
			} else {
				client = New("test_key", "test_secret")
			}

			if client.IsAuthenticated() != tt.authenticated {
				t.Errorf("expected IsAuthenticated() = %v, got %v", tt.authenticated, client.IsAuthenticated())
			}
		})
	}
}

// TestErrorHandling is an integration test that requires network access
// Skip in unit tests - use for manual testing
func TestErrorHandling(t *testing.T) {
	t.Skip("Integration test - requires network access")
}
