// Package lastfm provides a client library for the Last.fm API 2.0.
//
// # Overview
//
// This package implements a modern Go client for the Last.fm API, focusing
// on authentication and scrobbling operations. It provides a clean, type-safe
// API with context support, proper error handling, and retry logic.
//
// # Installation
//
//	go get github.com/jfmyers9/scribbles/pkg/lastfm
//
// # Quick Start
//
// First, create a client with your API credentials:
//
//	import "github.com/jfmyers9/scribbles/pkg/lastfm"
//
//	client, err := lastfm.NewClient(lastfm.Config{
//	    APIKey:    "your-api-key",
//	    APISecret: "your-api-secret",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Authentication
//
// Last.fm uses a token-based authentication flow:
//
//  1. Get a token from Last.fm
//  2. Direct the user to authorize the token
//  3. Exchange the token for a session key
//  4. Store and reuse the session key
//
// Example:
//
//	// Step 1: Get token
//	token, err := client.Auth().GetToken(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Step 2: User authorizes
//	fmt.Println("Please visit:", client.Auth().GetAuthURL(token.Token))
//	fmt.Print("Press enter after authorizing...")
//	fmt.Scanln()
//
//	// Step 3: Get session
//	session, err := client.Auth().GetSession(ctx, token.Token)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Step 4: Save and use session key
//	client.SetSessionKey(session.Key)
//	// Store session.Key for future use
//
// # Scrobbling
//
// Once authenticated, you can scrobble tracks and update now playing status:
//
//	// Update now playing
//	track := lastfm.Track{
//	    Artist: "The Beatles",
//	    Track:  "Yesterday",
//	    Album:  "Help!",
//	}
//	err := client.Scrobble().UpdateNowPlaying(ctx, track)
//
//	// Scrobble a single track
//	err = client.Scrobble().Scrobble(ctx, track, time.Now())
//
//	// Batch scrobble (up to 50 tracks)
//	scrobbles := []lastfm.Scrobble{
//	    {Track: track1, Timestamp: time1},
//	    {Track: track2, Timestamp: time2},
//	}
//	resp, err := client.Scrobble().ScrobbleBatch(ctx, scrobbles)
//
// # Error Handling
//
// The package provides structured errors with retry information:
//
//	resp, err := client.Scrobble().Scrobble(ctx, track, timestamp)
//	if err != nil {
//	    var lastfmErr *lastfm.Error
//	    if errors.As(err, &lastfmErr) {
//	        if lastfmErr.Temporary() {
//	            // Retry the request
//	        }
//	    }
//	}
//
// # Context Support
//
// All API methods accept a context.Context for cancellation and timeouts:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	token, err := client.Auth().GetToken(ctx)
//
// # Configuration
//
// The client can be configured with custom HTTP clients, base URLs (for testing),
// and optional loggers:
//
//	client, err := lastfm.NewClient(lastfm.Config{
//	    APIKey:     "your-api-key",
//	    APISecret:  "your-api-secret",
//	    SessionKey: "saved-session-key",
//	    HTTPClient: &http.Client{Timeout: 30 * time.Second},
//	    Logger:     myLogger, // Implements lastfm.Logger interface
//	})
//
// # API Coverage
//
// Currently implemented:
//   - Authentication (auth.getToken, auth.getSession)
//   - Scrobbling (track.scrobble, track.updateNowPlaying)
//
// # Last.fm API Documentation
//
// For more information about the Last.fm API:
// https://www.last.fm/api/scrobbling
package lastfm
