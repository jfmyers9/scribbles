package lastfm

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestScrobbleService_UpdateNowPlaying tests the UpdateNowPlaying method.
func TestScrobbleService_UpdateNowPlaying(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		statusCode  int
		track       Track
		wantErr     bool
		errContains string
	}{
		{
			name: "success",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<nowplaying>
		<artist corrected="0">The Beatles</artist>
		<track corrected="0">Yesterday</track>
		<album corrected="0">Help!</album>
		<albumArtist corrected="0">The Beatles</albumArtist>
	</nowplaying>
</lfm>`,
			statusCode: http.StatusOK,
			track: Track{
				Artist: "The Beatles",
				Track:  "Yesterday",
				Album:  "Help!",
			},
			wantErr: false,
		},
		{
			name: "with all optional fields",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<nowplaying>
		<artist corrected="0">The Beatles</artist>
		<track corrected="0">Yesterday</track>
		<album corrected="0">Help!</album>
		<albumArtist corrected="0">The Beatles</albumArtist>
	</nowplaying>
</lfm>`,
			statusCode: http.StatusOK,
			track: Track{
				Artist:      "The Beatles",
				Track:       "Yesterday",
				Album:       "Help!",
				AlbumArtist: "The Beatles",
				Duration:    125,
				TrackNumber: 1,
				MBTrackID:   "mbid-123",
			},
			wantErr: false,
		},
		{
			name: "api error - invalid session key",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="failed">
	<error code="9">Invalid session key</error>
</lfm>`,
			statusCode: http.StatusOK,
			track: Track{
				Artist: "The Beatles",
				Track:  "Yesterday",
			},
			wantErr:     true,
			errContains: "error 9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Parse form data
				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				// Verify required parameters
				if method := r.FormValue("method"); method != "track.updateNowPlaying" {
					t.Errorf("expected method track.updateNowPlaying, got %s", method)
				}
				if artist := r.FormValue("artist"); artist != tt.track.Artist {
					t.Errorf("expected artist %s, got %s", tt.track.Artist, artist)
				}
				if track := r.FormValue("track"); track != tt.track.Track {
					t.Errorf("expected track %s, got %s", tt.track.Track, track)
				}
				if sk := r.FormValue("sk"); sk != "test-session-key" {
					t.Errorf("expected sk test-session-key, got %s", sk)
				}

				// Verify optional parameters if provided
				if tt.track.Album != "" {
					if album := r.FormValue("album"); album != tt.track.Album {
						t.Errorf("expected album %s, got %s", tt.track.Album, album)
					}
				}
				if tt.track.AlbumArtist != "" {
					if albumArtist := r.FormValue("albumArtist"); albumArtist != tt.track.AlbumArtist {
						t.Errorf("expected albumArtist %s, got %s", tt.track.AlbumArtist, albumArtist)
					}
				}
				if tt.track.Duration > 0 {
					if duration := r.FormValue("duration"); duration != fmt.Sprintf("%d", tt.track.Duration) {
						t.Errorf("expected duration %d, got %s", tt.track.Duration, duration)
					}
				}

				w.WriteHeader(tt.statusCode)
				if _, err := w.Write([]byte(tt.response)); err != nil {
					t.Fatalf("failed to write response body: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(Config{
				APIKey:     "test-api-key",
				APISecret:  "test-secret",
				SessionKey: "test-session-key",
				BaseURL:    server.URL,
			})
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			ctx := context.Background()
			resp, err := client.Scrobble().UpdateNowPlaying(ctx, tt.track)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Artist != tt.track.Artist {
				t.Errorf("expected artist %s, got %s", tt.track.Artist, resp.Artist)
			}
			if resp.Track != tt.track.Track {
				t.Errorf("expected track %s, got %s", tt.track.Track, resp.Track)
			}
		})
	}
}

// TestScrobbleService_Scrobble tests the Scrobble method (single scrobble).
func TestScrobbleService_Scrobble(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		// Verify method
		if method := r.FormValue("method"); method != "track.scrobble" {
			t.Errorf("expected method track.scrobble, got %s", method)
		}

		// Verify batch parameters (single scrobble uses [0] index)
		if artist := r.FormValue("artist[0]"); artist != "The Beatles" {
			t.Errorf("expected artist[0] The Beatles, got %s", artist)
		}
		if track := r.FormValue("track[0]"); track != "Yesterday" {
			t.Errorf("expected track[0] Yesterday, got %s", track)
		}
		if timestamp := r.FormValue("timestamp[0]"); timestamp == "" {
			t.Error("expected timestamp[0] to be present")
		}

		response := `<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<scrobbles accepted="1" ignored="0">
		<scrobble>
			<artist corrected="0">The Beatles</artist>
			<track corrected="0">Yesterday</track>
			<album corrected="0">Help!</album>
			<timestamp>1234567890</timestamp>
		</scrobble>
	</scrobbles>
</lfm>`
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(response)); err != nil {
			t.Fatalf("failed to write response body: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:     "test-api-key",
		APISecret:  "test-secret",
		SessionKey: "test-session-key",
		BaseURL:    server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	track := Track{
		Artist: "The Beatles",
		Track:  "Yesterday",
		Album:  "Help!",
	}
	timestamp := time.Now()

	resp, err := client.Scrobble().Scrobble(ctx, track, timestamp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Accepted != 1 {
		t.Errorf("expected accepted 1, got %d", resp.Accepted)
	}
	if resp.Ignored != 0 {
		t.Errorf("expected ignored 0, got %d", resp.Ignored)
	}
}

// TestScrobbleService_ScrobbleBatch tests the ScrobbleBatch method.
func TestScrobbleService_ScrobbleBatch(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		statusCode   int
		scrobbles    []Scrobble
		wantAccepted int
		wantIgnored  int
		wantErr      bool
		errContains  string
	}{
		{
			name: "success - multiple scrobbles",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<scrobbles accepted="2" ignored="0">
		<scrobble>
			<artist corrected="0">The Beatles</artist>
			<track corrected="0">Yesterday</track>
			<album corrected="0">Help!</album>
			<timestamp>1234567890</timestamp>
		</scrobble>
		<scrobble>
			<artist corrected="0">The Beatles</artist>
			<track corrected="0">Let It Be</track>
			<album corrected="0">Let It Be</album>
			<timestamp>1234567950</timestamp>
		</scrobble>
	</scrobbles>
</lfm>`,
			statusCode: http.StatusOK,
			scrobbles: []Scrobble{
				{
					Track: Track{
						Artist: "The Beatles",
						Track:  "Yesterday",
						Album:  "Help!",
					},
					Timestamp: time.Unix(1234567890, 0),
				},
				{
					Track: Track{
						Artist: "The Beatles",
						Track:  "Let It Be",
						Album:  "Let It Be",
					},
					Timestamp: time.Unix(1234567950, 0),
				},
			},
			wantAccepted: 2,
			wantIgnored:  0,
			wantErr:      false,
		},
		{
			name:         "empty batch",
			scrobbles:    []Scrobble{},
			wantAccepted: 0,
			wantIgnored:  0,
			wantErr:      false,
		},
		{
			name: "api error - rate limit",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="failed">
	<error code="11">Service Offline - This service is temporarily offline. Try again later.</error>
</lfm>`,
			statusCode: http.StatusOK,
			scrobbles: []Scrobble{
				{
					Track: Track{
						Artist: "The Beatles",
						Track:  "Yesterday",
					},
					Timestamp: time.Unix(1234567890, 0),
				},
			},
			wantErr:     true,
			errContains: "error 11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle empty batch case (no server needed)
			if len(tt.scrobbles) == 0 {
				client, err := NewClient(Config{
					APIKey:     "test-api-key",
					APISecret:  "test-secret",
					SessionKey: "test-session-key",
				})
				if err != nil {
					t.Fatalf("failed to create client: %v", err)
				}

				ctx := context.Background()
				resp, err := client.Scrobble().ScrobbleBatch(ctx, tt.scrobbles)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if resp.Accepted != 0 || resp.Ignored != 0 {
					t.Errorf("expected empty response, got accepted=%d ignored=%d", resp.Accepted, resp.Ignored)
				}
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Parse form data
				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				// Verify method
				if method := r.FormValue("method"); method != "track.scrobble" {
					t.Errorf("expected method track.scrobble, got %s", method)
				}

				// Verify batch parameters
				for i, scrobble := range tt.scrobbles {
					idx := fmt.Sprintf("[%d]", i)
					if artist := r.FormValue("artist" + idx); artist != scrobble.Track.Artist {
						t.Errorf("expected artist%s %s, got %s", idx, scrobble.Track.Artist, artist)
					}
					if track := r.FormValue("track" + idx); track != scrobble.Track.Track {
						t.Errorf("expected track%s %s, got %s", idx, scrobble.Track.Track, track)
					}
					expectedTimestamp := fmt.Sprintf("%d", scrobble.Timestamp.Unix())
					if timestamp := r.FormValue("timestamp" + idx); timestamp != expectedTimestamp {
						t.Errorf("expected timestamp%s %s, got %s", idx, expectedTimestamp, timestamp)
					}
				}

				w.WriteHeader(tt.statusCode)
				if _, err := w.Write([]byte(tt.response)); err != nil {
					t.Fatalf("failed to write response body: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(Config{
				APIKey:     "test-api-key",
				APISecret:  "test-secret",
				SessionKey: "test-session-key",
				BaseURL:    server.URL,
			})
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			ctx := context.Background()
			resp, err := client.Scrobble().ScrobbleBatch(ctx, tt.scrobbles)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Accepted != tt.wantAccepted {
				t.Errorf("expected accepted %d, got %d", tt.wantAccepted, resp.Accepted)
			}
			if resp.Ignored != tt.wantIgnored {
				t.Errorf("expected ignored %d, got %d", tt.wantIgnored, resp.Ignored)
			}
		})
	}
}

// TestScrobbleService_ScrobbleBatch_MaxBatchSize tests that batch size is limited to 50.
func TestScrobbleService_ScrobbleBatch_MaxBatchSize(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		// Count how many scrobbles were sent
		count := 0
		for i := 0; i < 100; i++ {
			if r.FormValue(fmt.Sprintf("artist[%d]", i)) != "" {
				count++
			}
		}

		if count != MaxBatchSize {
			t.Errorf("expected %d scrobbles in batch, got %d", MaxBatchSize, count)
		}

		response := `<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<scrobbles accepted="50" ignored="0">
	</scrobbles>
</lfm>`
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(response)); err != nil {
			t.Fatalf("failed to write response body: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:     "test-api-key",
		APISecret:  "test-secret",
		SessionKey: "test-session-key",
		BaseURL:    server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Create 60 scrobbles (more than max)
	scrobbles := make([]Scrobble, 60)
	for i := range scrobbles {
		scrobbles[i] = Scrobble{
			Track: Track{
				Artist: fmt.Sprintf("Artist %d", i),
				Track:  fmt.Sprintf("Track %d", i),
			},
			Timestamp: time.Now(),
		}
	}

	ctx := context.Background()
	resp, err := client.Scrobble().ScrobbleBatch(ctx, scrobbles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Accepted != MaxBatchSize {
		t.Errorf("expected accepted %d, got %d", MaxBatchSize, resp.Accepted)
	}

	if requestCount != 1 {
		t.Errorf("expected 1 request, got %d", requestCount)
	}
}

// TestScrobbleService_NoSessionKey tests that methods require a session key.
func TestScrobbleService_NoSessionKey(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:    "test-api-key",
		APISecret: "test-secret",
		// No session key
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	track := Track{
		Artist: "The Beatles",
		Track:  "Yesterday",
	}

	// Test UpdateNowPlaying
	_, err = client.Scrobble().UpdateNowPlaying(ctx, track)
	if err == nil {
		t.Error("expected error for UpdateNowPlaying without session key, got nil")
	}
	if !strings.Contains(err.Error(), "session key required") {
		t.Errorf("expected error to contain 'session key required', got %v", err)
	}

	// Test Scrobble
	_, err = client.Scrobble().Scrobble(ctx, track, time.Now())
	if err == nil {
		t.Error("expected error for Scrobble without session key, got nil")
	}
	if !strings.Contains(err.Error(), "session key required") {
		t.Errorf("expected error to contain 'session key required', got %v", err)
	}

	// Test ScrobbleBatch
	_, err = client.Scrobble().ScrobbleBatch(ctx, []Scrobble{{Track: track, Timestamp: time.Now()}})
	if err == nil {
		t.Error("expected error for ScrobbleBatch without session key, got nil")
	}
	if !strings.Contains(err.Error(), "session key required") {
		t.Errorf("expected error to contain 'session key required', got %v", err)
	}
}

// TestScrobbleService_ContextCancellation tests context cancellation.
func TestScrobbleService_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`<lfm status="ok"><scrobbles accepted="1" ignored="0"></scrobbles></lfm>`)); err != nil {
			t.Fatalf("failed to write response body: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:     "test-api-key",
		APISecret:  "test-secret",
		SessionKey: "test-session-key",
		BaseURL:    server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	track := Track{
		Artist: "The Beatles",
		Track:  "Yesterday",
	}

	_, err = client.Scrobble().UpdateNowPlaying(ctx, track)
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("expected context error, got %v", err)
	}
}

// ExampleScrobbleService_UpdateNowPlaying demonstrates how to update the now playing status.
func ExampleScrobbleService_UpdateNowPlaying() {
	client, err := NewClient(Config{
		APIKey:     "your-api-key",
		APISecret:  "your-api-secret",
		SessionKey: "your-session-key",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	track := Track{
		Artist:   "The Beatles",
		Track:    "Yesterday",
		Album:    "Help!",
		Duration: 125,
	}

	resp, err := client.Scrobble().UpdateNowPlaying(ctx, track)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Now playing: %s - %s\n", resp.Artist, resp.Track)
}

// ExampleScrobbleService_Scrobble demonstrates how to scrobble a single track.
func ExampleScrobbleService_Scrobble() {
	client, err := NewClient(Config{
		APIKey:     "your-api-key",
		APISecret:  "your-api-secret",
		SessionKey: "your-session-key",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	track := Track{
		Artist:   "The Beatles",
		Track:    "Yesterday",
		Album:    "Help!",
		Duration: 125,
	}

	// Scrobble a track that was played 2 minutes ago
	timestamp := time.Now().Add(-2 * time.Minute)

	resp, err := client.Scrobble().Scrobble(ctx, track, timestamp)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Scrobbled: %d accepted, %d ignored\n", resp.Accepted, resp.Ignored)
}

// ExampleScrobbleService_ScrobbleBatch demonstrates how to scrobble multiple tracks at once.
func ExampleScrobbleService_ScrobbleBatch() {
	client, err := NewClient(Config{
		APIKey:     "your-api-key",
		APISecret:  "your-api-secret",
		SessionKey: "your-session-key",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Create a batch of scrobbles
	scrobbles := []Scrobble{
		{
			Track: Track{
				Artist: "The Beatles",
				Track:  "Yesterday",
				Album:  "Help!",
			},
			Timestamp: time.Now().Add(-10 * time.Minute),
		},
		{
			Track: Track{
				Artist: "The Beatles",
				Track:  "Let It Be",
				Album:  "Let It Be",
			},
			Timestamp: time.Now().Add(-5 * time.Minute),
		},
	}

	resp, err := client.Scrobble().ScrobbleBatch(ctx, scrobbles)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Scrobbled: %d accepted, %d ignored\n", resp.Accepted, resp.Ignored)

	// Check individual scrobble results
	for i, s := range resp.Scrobbles {
		if s.IgnoredMessage.Code != 0 {
			fmt.Printf("Scrobble %d was ignored: %s\n", i, s.IgnoredMessage.Text)
		}
	}
}
