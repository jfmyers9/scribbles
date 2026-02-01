# Test Strategy for pkg/lastfm

## Overview

This document outlines the testing strategy for the pkg/lastfm package.
The strategy balances comprehensive coverage with maintainability and
practicality.

## Testing Approach

### 1. Unit Tests (Primary)

**Approach**: Use `httptest.Server` to create mock Last.fm API servers.

**Rationale**:
- Fast execution (no network calls)
- Deterministic results
- Can test error conditions easily
- No API credentials required
- Can run in CI/CD without issues
- Tests actual HTTP behavior

**Coverage**:
- Client creation and configuration
- API signature calculation
- Request parameter formatting
- Response parsing (both success and error)
- Error type handling
- Retry logic
- Context cancellation
- Batch parameter formatting
- Edge cases (empty batches, oversized batches, etc.)

**Files**:
- `client_test.go` - Client creation, configuration
- `auth_test.go` - Auth service methods with mock server
- `scrobble_test.go` - Scrobble service methods with mock server
- `errors_test.go` - Error type behavior
- `signature_test.go` - API signature calculation
- `transport_test.go` - HTTP transport and retry logic

**Example Pattern**:
```go
func TestAuthService_GetToken(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        // Return mock response
    }))
    defer server.Close()

    // Create client with mock server
    client, err := lastfm.NewClient(lastfm.Config{
        APIKey:    "test_key",
        APISecret: "test_secret",
        BaseURL:   server.URL,
    })

    // Test
    token, err := client.Auth().GetToken(context.Background())
    // Assert results
}
```

### 2. Example Tests (godoc Examples)

**Approach**: Use `Example*` test functions for godoc.

**Rationale**:
- Provides executable documentation
- Appears in pkg.go.dev documentation
- Verifies API usability
- Simple to understand for users

**Coverage**:
- Basic client creation
- Authentication flow
- Single scrobble
- Batch scrobble
- Now playing update
- Error handling

**Files**:
- `examples_test.go` - All godoc examples

**Example Pattern**:
```go
func ExampleClient_Auth() {
    client, _ := lastfm.NewClient(lastfm.Config{
        APIKey:    "your-api-key",
        APISecret: "your-api-secret",
    })

    token, _ := client.Auth().GetToken(context.Background())
    fmt.Println(client.Auth().GetAuthURL(token.Token))
}
```

### 3. Integration Tests (Optional, Build Tagged)

**Approach**: Real API calls with `// +build integration` tag.

**Rationale**:
- Verifies actual API behavior
- Catches API changes
- Only runs when explicitly requested
- Requires real credentials (via environment variables)

**Coverage**:
- Complete authentication flow
- Real scrobbling
- Real batch scrobbling
- Real now playing updates
- Error responses from actual API

**Files**:
- `integration_test.go` - Real Last.fm API tests

**Build Tag**: `// +build integration`

**Run Command**: `go test -tags=integration ./pkg/lastfm/`

**Environment Variables**:
- `LASTFM_API_KEY` - API key
- `LASTFM_API_SECRET` - API secret
- `LASTFM_SESSION_KEY` - Pre-obtained session key

### 4. Benchmark Tests

**Approach**: Benchmark critical paths.

**Coverage**:
- Signature calculation
- Batch request formatting
- Response parsing

**Files**:
- `*_test.go` files with `Benchmark*` functions

## Test Organization

```
pkg/lastfm/
├── client_test.go         # Client creation, config validation
├── auth_test.go           # AuthService with mock server
├── scrobble_test.go       # ScrobbleService with mock server
├── errors_test.go         # Error types and Is/Temporary methods
├── signature_test.go      # Signature calculation tests
├── transport_test.go      # HTTP transport, retries
├── examples_test.go       # godoc examples
├── integration_test.go    # Real API tests (build tagged)
└── TEST_STRATEGY.md       # This document
```

## Mock Server Strategy

### Response Fixtures

Create helper functions to generate mock Last.fm XML responses:

```go
func mockTokenResponse() string {
    return `<?xml version="1.0" encoding="utf-8"?>
<lfm status="ok">
    <token>test_token_123</token>
</lfm>`
}

func mockErrorResponse(code int, message string) string {
    return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<lfm status="failed">
    <error code="%d">%s</error>
</lfm>`, code, message)
}
```

### Request Verification

Mock servers should verify:
- HTTP method (POST)
- Required parameters
- API signature (when authenticated)
- Content-Type header

## Coverage Goals

- **Line Coverage**: >80%
- **Critical Paths**: 100% (signature calc, auth flow, scrobbling)
- **Error Paths**: 100% (error parsing, retry logic)
- **godoc Examples**: All public APIs

## Running Tests

```bash
# Unit tests only (default)
go test ./pkg/lastfm/

# With coverage
go test -cover ./pkg/lastfm/

# With coverage report
go test -coverprofile=coverage.out ./pkg/lastfm/
go tool cover -html=coverage.out

# Integration tests (requires credentials)
LASTFM_API_KEY=xxx LASTFM_API_SECRET=yyy go test -tags=integration ./pkg/lastfm/

# Benchmarks
go test -bench=. ./pkg/lastfm/

# All tests with verbose output
go test -v ./pkg/lastfm/
```

## Test Dependencies

**Standard Library Only**:
- `net/http/httptest` - Mock HTTP servers
- `testing` - Test framework
- `context` - Context handling

**No External Test Dependencies**:
- No testify, gomock, etc. needed
- Keeps package lightweight
- Follows Go stdlib patterns

## Future Considerations

### VCR/Cassette Pattern

If we need more realistic API responses, could add:
- Record real API responses once
- Replay in tests
- Library: `gopkg.in/dnaeon/go-vcr.v3`

**Not needed initially** - httptest is sufficient.

### Contract Testing

If Last.fm provides an OpenAPI spec:
- Could validate our requests/responses
- Ensure we match the spec

**Not available** - Last.fm uses XML without OpenAPI spec.

## Summary

**Primary Strategy**: Unit tests with mock HTTP servers
**Secondary**: godoc examples for documentation
**Tertiary**: Integration tests (opt-in via build tag)

This approach provides:
- Fast, reliable tests
- Good coverage
- Executable documentation
- Optional real-world validation
- No external dependencies
