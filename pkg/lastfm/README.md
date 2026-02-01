# Last.fm Go SDK

A modern Go client library for the Last.fm API 2.0, focusing on
authentication and scrobbling operations.

## Features

- **Clean API**: Type-safe interfaces with clear method signatures
- **Context Support**: All operations accept `context.Context` for
  cancellation and timeouts
- **Automatic Retries**: Exponential backoff for transient failures
- **Batch Operations**: Scrobble up to 50 tracks in a single request
- **Error Handling**: Structured error types with retry information
- **Zero Dependencies**: Only uses Go standard library (except tests)
- **Comprehensive Documentation**: Full godoc coverage with examples
- **Well Tested**: Extensive unit tests with mock HTTP servers

## Installation

```bash
go get github.com/jfmyers9/scribbles/pkg/lastfm
```

## Quick Start

### 1. Get API Credentials

First, get your Last.fm API credentials:

1. Visit https://www.last.fm/api/account/create
2. Create an application to get your API key and secret

### 2. Create a Client

```go
import "github.com/jfmyers9/scribbles/pkg/lastfm"

client, err := lastfm.NewClient(lastfm.Config{
    APIKey:    "your-api-key",
    APISecret: "your-api-secret",
})
if err != nil {
    log.Fatal(err)
}
```

### 3. Authenticate

Last.fm uses a token-based authentication flow:

```go
ctx := context.Background()

// Step 1: Get a token
token, err := client.Auth().GetToken(ctx)
if err != nil {
    log.Fatal(err)
}

// Step 2: Direct user to authorize
authURL := client.Auth().GetAuthURL(token.Token)
fmt.Println("Please visit:", authURL)
fmt.Print("Press enter after authorizing...")
fmt.Scanln()

// Step 3: Exchange token for session
session, err := client.Auth().GetSession(ctx, token.Token)
if err != nil {
    log.Fatal(err)
}

// Step 4: Save and use session key
client.SetSessionKey(session.Key)
// Store session.Key for future use
```

### 4. Scrobble Tracks

```go
// Update now playing
track := lastfm.Track{
    Artist: "The Beatles",
    Track:  "Yesterday",
    Album:  "Help!",
}
err := client.Scrobble().UpdateNowPlaying(ctx, track)
if err != nil {
    log.Fatal(err)
}

// Scrobble a single track
timestamp := time.Now()
err = client.Scrobble().Scrobble(ctx, track, timestamp)
if err != nil {
    log.Fatal(err)
}
```

## Usage Guide

### Authentication

The Last.fm API requires authentication for most operations. The library
provides a complete authentication flow:

```go
// Create client
client, _ := lastfm.NewClient(lastfm.Config{
    APIKey:    "your-api-key",
    APISecret: "your-api-secret",
})

// Get token
token, err := client.Auth().GetToken(ctx)
if err != nil {
    log.Fatal(err)
}

// Get authorization URL
authURL := client.Auth().GetAuthURL(token.Token)
// Direct user to authURL in their browser

// After user authorizes, exchange token for session
session, err := client.Auth().GetSession(ctx, token.Token)
if err != nil {
    log.Fatal(err)
}

// Set session key for authenticated requests
client.SetSessionKey(session.Key)

// Save session.Key to disk/database for future use
saveSessionKey(session.Key)
```

For subsequent uses, you can create the client with a saved session key:

```go
client, _ := lastfm.NewClient(lastfm.Config{
    APIKey:     "your-api-key",
    APISecret:  "your-api-secret",
    SessionKey: loadSessionKey(), // Load from storage
})
```

### Scrobbling

#### Update Now Playing

Update the current playing track:

```go
track := lastfm.Track{
    Artist:      "The Beatles",
    Track:       "Yesterday",
    Album:       "Help!",        // Optional
    AlbumArtist: "The Beatles",  // Optional
    Duration:    125,             // Optional, seconds
    TrackNumber: 1,               // Optional
    MusicBrainzID: "...",         // Optional
}

err := client.Scrobble().UpdateNowPlaying(ctx, track)
if err != nil {
    log.Fatal(err)
}
```

#### Scrobble a Single Track

Submit a completed scrobble:

```go
track := lastfm.Track{
    Artist: "The Beatles",
    Track:  "Yesterday",
}

// Timestamp when track started playing
timestamp := time.Now().Add(-2 * time.Minute)

err := client.Scrobble().Scrobble(ctx, track, timestamp)
if err != nil {
    log.Fatal(err)
}
```

#### Batch Scrobbling

Submit up to 50 scrobbles at once:

```go
scrobbles := []lastfm.Scrobble{
    {
        Track: lastfm.Track{
            Artist: "The Beatles",
            Track:  "Yesterday",
            Album:  "Help!",
        },
        Timestamp: time.Now().Add(-10 * time.Minute),
    },
    {
        Track: lastfm.Track{
            Artist: "The Beatles",
            Track:  "Let It Be",
            Album:  "Let It Be",
        },
        Timestamp: time.Now().Add(-5 * time.Minute),
    },
}

response, err := client.Scrobble().ScrobbleBatch(ctx, scrobbles)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Scrobbled %d tracks\n", response.Accepted)
fmt.Printf("Ignored %d tracks\n", response.Ignored)
```

### Error Handling

The library provides structured errors with retry information:

```go
err := client.Scrobble().Scrobble(ctx, track, timestamp)
if err != nil {
    var lastfmErr *lastfm.Error
    if errors.As(err, &lastfmErr) {
        fmt.Printf("Last.fm error %d: %s\n", lastfmErr.Code, lastfmErr.Message)

        if lastfmErr.Temporary() {
            // This error is temporary, retry the request
            // (Note: retries are automatic, this is just for logging)
        }
    } else {
        // Network error or other non-Last.fm error
        fmt.Printf("Error: %v\n", err)
    }
}
```

The client automatically retries transient errors with exponential backoff:

- Last.fm error codes 11 (service offline) and 16 (service temporarily
  unavailable)
- HTTP 5xx server errors
- Network errors (connection failures, timeouts)

### Context Support

All API methods accept a `context.Context` for cancellation and timeouts:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

token, err := client.Auth().GetToken(ctx)

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    // Cancel after user hits Ctrl+C
    <-sigChan
    cancel()
}()

err := client.Scrobble().Scrobble(ctx, track, timestamp)
```

### Configuration

The client can be configured with custom settings:

```go
client, err := lastfm.NewClient(lastfm.Config{
    // Required
    APIKey:    "your-api-key",
    APISecret: "your-api-secret",

    // Optional
    SessionKey: "saved-session-key",  // For authenticated requests

    // Custom HTTP client (for timeouts, proxies, etc.)
    HTTPClient: &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            Proxy: http.ProxyFromEnvironment,
        },
    },

    // Custom base URL (for testing)
    BaseURL: "https://ws.audioscrobbler.com/2.0/",

    // Optional logger (must implement lastfm.Logger interface)
    Logger: myLogger,
})
```

### Logging

You can provide a custom logger that implements the `Logger` interface:

```go
type Logger interface {
    Printf(format string, v ...interface{})
}

// Example: use standard log package
client, _ := lastfm.NewClient(lastfm.Config{
    APIKey:    "your-api-key",
    APISecret: "your-api-secret",
    Logger:    log.Default(),
})
```

## API Coverage

Currently implemented:

- **Authentication**
  - `auth.getToken` - Get authentication token
  - `auth.getSession` - Exchange token for session key

- **Scrobbling**
  - `track.updateNowPlaying` - Update now playing status
  - `track.scrobble` - Submit scrobbles (batch up to 50)

## Examples

See the [godoc examples](https://pkg.go.dev/github.com/jfmyers9/scribbles/pkg/lastfm#pkg-examples)
for complete runnable examples:

- Authentication flow
- Getting a token
- Getting an auth URL
- Getting a session
- Updating now playing
- Scrobbling a single track
- Batch scrobbling

## Testing

The library includes comprehensive tests using mock HTTP servers:

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# View coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Last.fm API Documentation

For more information about the Last.fm API:

- [API Docs](https://www.last.fm/api/intro)
- [Scrobbling Guide](https://www.last.fm/api/scrobbling)
- [Authentication](https://www.last.fm/api/authentication)

## License

MIT License - see [LICENSE](../../LICENSE) for details.

## Contributing

Contributions are welcome! Please ensure:

- All tests pass: `go test ./...`
- Code is formatted: `go fmt ./...`
- Linting passes: `golangci-lint run`
- New features include tests and godoc examples
- Public APIs have godoc comments

## Related Projects

- [scribbles](https://github.com/jfmyers9/scribbles) - Apple Music
  scrobbler using this library
- [Last.fm API](https://www.last.fm/api) - Official API documentation
