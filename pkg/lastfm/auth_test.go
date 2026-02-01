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

// TestAuthService_GetToken tests the GetToken method.
func TestAuthService_GetToken(t *testing.T) {
	tests := []struct {
		name        string
		response    string
		statusCode  int
		wantToken   string
		wantErr     bool
		errContains string
	}{
		{
			name: "success",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<token>test-token-123</token>
</lfm>`,
			statusCode: http.StatusOK,
			wantToken:  "test-token-123",
			wantErr:    false,
		},
		{
			name: "api error - invalid api key",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="failed">
	<error code="10">Invalid API key</error>
</lfm>`,
			statusCode:  http.StatusOK,
			wantErr:     true,
			errContains: "error 10",
		},
		{
			name: "server error - retryable",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="failed">
	<error code="11">Service Offline</error>
</lfm>`,
			statusCode:  http.StatusOK,
			wantErr:     true,
			errContains: "error 11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Verify Content-Type
				if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
					t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", ct)
				}

				// Parse form data
				if err := r.ParseForm(); err != nil {
					t.Fatalf("failed to parse form: %v", err)
				}

				// Verify required parameters
				if method := r.FormValue("method"); method != "auth.getToken" {
					t.Errorf("expected method auth.getToken, got %s", method)
				}
				if apiKey := r.FormValue("api_key"); apiKey != "test-api-key" {
					t.Errorf("expected api_key test-api-key, got %s", apiKey)
				}
				if sig := r.FormValue("api_sig"); sig == "" {
					t.Error("expected api_sig to be present")
				}

				w.WriteHeader(tt.statusCode)
				if _, err := w.Write([]byte(tt.response)); err != nil {
					t.Fatalf("failed to write response body: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(Config{
				APIKey:    "test-api-key",
				APISecret: "test-secret",
				BaseURL:   server.URL,
			})
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			ctx := context.Background()
			token, err := client.Auth().GetToken(ctx)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if token.Token != tt.wantToken {
				t.Errorf("expected token %q, got %q", tt.wantToken, token.Token)
			}
		})
	}
}

// TestAuthService_GetAuthURL tests the GetAuthURL method.
func TestAuthService_GetAuthURL(t *testing.T) {
	client, err := NewClient(Config{
		APIKey:    "my-api-key",
		APISecret: "my-secret",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	token := "test-token-123"
	url := client.Auth().GetAuthURL(token)

	expectedURL := "https://www.last.fm/api/auth/?api_key=my-api-key&token=test-token-123"
	if url != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, url)
	}
}

// TestAuthService_GetSession tests the GetSession method.
func TestAuthService_GetSession(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		statusCode     int
		wantKey        string
		wantUsername   string
		wantSubscriber bool
		wantErr        bool
		errContains    string
	}{
		{
			name: "success - subscriber",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<session>
		<name>testuser</name>
		<key>session-key-abc123</key>
		<subscriber>1</subscriber>
	</session>
</lfm>`,
			statusCode:     http.StatusOK,
			wantKey:        "session-key-abc123",
			wantUsername:   "testuser",
			wantSubscriber: true,
			wantErr:        false,
		},
		{
			name: "success - non-subscriber",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<session>
		<name>freeuser</name>
		<key>free-session-key</key>
		<subscriber>0</subscriber>
	</session>
</lfm>`,
			statusCode:     http.StatusOK,
			wantKey:        "free-session-key",
			wantUsername:   "freeuser",
			wantSubscriber: false,
			wantErr:        false,
		},
		{
			name: "unauthorized token",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="failed">
	<error code="14">Unauthorized Token</error>
</lfm>`,
			statusCode:  http.StatusOK,
			wantErr:     true,
			errContains: "error 14",
		},
		{
			name: "expired token",
			response: `<?xml version="1.0" encoding="utf-8"?>
<lfm status="failed">
	<error code="15">Token has expired</error>
</lfm>`,
			statusCode:  http.StatusOK,
			wantErr:     true,
			errContains: "error 15",
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
				if method := r.FormValue("method"); method != "auth.getSession" {
					t.Errorf("expected method auth.getSession, got %s", method)
				}
				if token := r.FormValue("token"); token != "test-token" {
					t.Errorf("expected token test-token, got %s", token)
				}
				if apiKey := r.FormValue("api_key"); apiKey != "test-api-key" {
					t.Errorf("expected api_key test-api-key, got %s", apiKey)
				}
				if sig := r.FormValue("api_sig"); sig == "" {
					t.Error("expected api_sig to be present")
				}

				w.WriteHeader(tt.statusCode)
				if _, err := w.Write([]byte(tt.response)); err != nil {
					t.Fatalf("failed to write response body: %v", err)
				}
			}))
			defer server.Close()

			client, err := NewClient(Config{
				APIKey:    "test-api-key",
				APISecret: "test-secret",
				BaseURL:   server.URL,
			})
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			ctx := context.Background()
			session, err := client.Auth().GetSession(ctx, "test-token")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if session.Key != tt.wantKey {
				t.Errorf("expected key %q, got %q", tt.wantKey, session.Key)
			}
			if session.Username != tt.wantUsername {
				t.Errorf("expected username %q, got %q", tt.wantUsername, session.Username)
			}
			if session.Subscriber != tt.wantSubscriber {
				t.Errorf("expected subscriber %v, got %v", tt.wantSubscriber, session.Subscriber)
			}
		})
	}
}

// TestAuthService_GetToken_ContextCancellation tests context cancellation.
func TestAuthService_GetToken_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<token>test-token</token>
</lfm>`)); err != nil {
			t.Fatalf("failed to write response body: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:    "test-api-key",
		APISecret: "test-secret",
		BaseURL:   server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = client.Auth().GetToken(ctx)
	if err == nil {
		t.Fatal("expected context deadline error, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected context deadline error, got %v", err)
	}
}

// TestAuthService_GetSession_ContextCancellation tests context cancellation.
func TestAuthService_GetSession_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<session>
		<name>test</name>
		<key>test-key</key>
		<subscriber>0</subscriber>
	</session>
</lfm>`)); err != nil {
			t.Fatalf("failed to write response body: %v", err)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:    "test-api-key",
		APISecret: "test-secret",
		BaseURL:   server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = client.Auth().GetSession(ctx, "test-token")
	if err == nil {
		t.Fatal("expected context deadline error, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected context deadline error, got %v", err)
	}
}

// TestAuthService_Retry tests retry logic for temporary errors.
func TestAuthService_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// First two attempts return temporary error
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<lfm status="failed">
	<error code="11">Service Offline</error>
</lfm>`)); err != nil {
				t.Fatalf("failed to write response body: %v", err)
			}
		} else {
			// Third attempt succeeds
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<token>test-token-retry</token>
</lfm>`)); err != nil {
				t.Fatalf("failed to write response body: %v", err)
			}
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:    "test-api-key",
		APISecret: "test-secret",
		BaseURL:   server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	token, err := client.Auth().GetToken(ctx)

	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}

	if token.Token != "test-token-retry" {
		t.Errorf("expected token test-token-retry, got %q", token.Token)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestAuthService_ServerError tests handling of HTTP 5xx errors.
func TestAuthService_ServerError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// First two attempts return 503
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, err := w.Write([]byte("Service Unavailable")); err != nil {
				t.Fatalf("failed to write response body: %v", err)
			}
		} else {
			// Third attempt succeeds
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
	<token>test-token-503</token>
</lfm>`)); err != nil {
				t.Fatalf("failed to write response body: %v", err)
			}
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:    "test-api-key",
		APISecret: "test-secret",
		BaseURL:   server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	token, err := client.Auth().GetToken(ctx)

	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}

	if token.Token != "test-token-503" {
		t.Errorf("expected token test-token-503, got %q", token.Token)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// Example_authFlow demonstrates the complete authentication flow.
//
// This example shows how to authenticate a user with Last.fm using the
// token-based OAuth flow.
func Example_authFlow() {
	// Create a new client with your API credentials
	client, err := NewClient(Config{
		APIKey:    "your-api-key",
		APISecret: "your-api-secret",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Step 1: Get an authentication token
	token, err := client.Auth().GetToken(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Step 2: Direct the user to authorize the token
	authURL := client.Auth().GetAuthURL(token.Token)
	fmt.Println("Please visit this URL to authorize the application:")
	fmt.Println(authURL)

	// Step 3: After the user authorizes, exchange the token for a session
	// In a real application, you would wait for user authorization here
	session, err := client.Auth().GetSession(ctx, token.Token)
	if err != nil {
		log.Fatal(err)
	}

	// Step 4: Save the session key for future use
	client.SetSessionKey(session.Key)
	fmt.Printf("Authenticated as: %s\n", session.Username)

	// The session key should be stored securely and reused for future requests
	// to avoid requiring the user to re-authenticate every time.
}

// ExampleAuthService_GetToken demonstrates how to request an authentication token.
func ExampleAuthService_GetToken() {
	client, err := NewClient(Config{
		APIKey:    "your-api-key",
		APISecret: "your-api-secret",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	token, err := client.Auth().GetToken(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Token: %s\n", token.Token)

	// Next, direct the user to the authorization URL:
	// authURL := client.Auth().GetAuthURL(token.Token)
}

// ExampleAuthService_GetAuthURL demonstrates how to generate the authorization URL.
func ExampleAuthService_GetAuthURL() {
	client, err := NewClient(Config{
		APIKey:    "your-api-key",
		APISecret: "your-api-secret",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Assume we have a token from GetToken()
	token := "example-token-123"

	authURL := client.Auth().GetAuthURL(token)
	fmt.Println("Authorization URL:", authURL)

	// Direct the user to this URL in their web browser.
	// After they authorize, you can exchange the token for a session.
}

// ExampleAuthService_GetSession demonstrates how to exchange a token for a session.
func ExampleAuthService_GetSession() {
	client, err := NewClient(Config{
		APIKey:    "your-api-key",
		APISecret: "your-api-secret",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Assume the user has authorized the token
	token := "example-authorized-token"

	session, err := client.Auth().GetSession(ctx, token)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Session Key: %s\n", session.Key)
	fmt.Printf("Username: %s\n", session.Username)
	fmt.Printf("Subscriber: %v\n", session.Subscriber)

	// Store the session key for future use
	client.SetSessionKey(session.Key)
}
