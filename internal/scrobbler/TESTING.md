# Last.fm Integration Testing

This document describes how to test the Last.fm integration in scribbles.

## Unit Tests

Unit tests can be run without any credentials:

```bash
go test ./internal/scrobbler/
```

These tests verify the basic functionality and parameter handling without making real API calls.

## Integration Tests

Integration tests require valid Last.fm API credentials and make real API calls.

### Prerequisites

1. Create a Last.fm API account at: https://www.last.fm/api/account/create
2. Note your API Key and API Secret

### Running Integration Tests

#### Step 1: Test Authentication

```bash
export LASTFM_API_KEY="your_api_key"
export LASTFM_API_SECRET="your_api_secret"
go test -tags=integration -v -run TestIntegration_LastFmAuth ./internal/scrobbler/
```

This will output an auth URL. Visit the URL in your browser to authorize the application.

#### Step 2: Get Session Key

After authorizing, copy the token from the test output and run:

```bash
export LASTFM_TOKEN="token_from_step_1"
go test -tags=integration -v -run TestIntegration_GetSession ./internal/scrobbler/
```

This will output a session key. Save this for future tests.

#### Step 3: Test Now Playing

```bash
export LASTFM_SESSION_KEY="session_key_from_step_2"
go test -tags=integration -v -run TestIntegration_UpdateNowPlaying ./internal/scrobbler/
```

Check your Last.fm profile - it should show "Test Artist - Test Track" as now playing.

#### Step 4: Test Scrobbling

```bash
go test -tags=integration -v -run TestIntegration_Scrobble ./internal/scrobbler/
```

Check your Last.fm profile - you should see a scrobble for "Test Artist - Test Track".

#### Step 5: Test Batch Scrobbling

```bash
go test -tags=integration -v -run TestIntegration_ScrobbleBatch ./internal/scrobbler/
```

Check your Last.fm profile - you should see multiple test scrobbles.

### Run All Integration Tests

To run all integration tests at once (after obtaining credentials):

```bash
export LASTFM_API_KEY="your_api_key"
export LASTFM_API_SECRET="your_api_secret"
export LASTFM_SESSION_KEY="your_session_key"
go test -tags=integration -v ./internal/scrobbler/
```

Note: The auth and token tests will be skipped when LASTFM_SESSION_KEY is set.

## Manual Testing with CLI

You can also test the integration using the CLI:

### Test Authentication Flow

```bash
./scribbles auth
```

Follow the prompts to authenticate with Last.fm. Your session key will be saved to `~/.config/scribbles/config.yaml`.

### Verify Configuration

```bash
cat ~/.config/scribbles/config.yaml
```

You should see your API key, secret, and session key in the config file.

## Error Cases to Test

When testing integration, verify these error cases are handled correctly:

1. **Invalid API credentials**: Use incorrect API key/secret
2. **Unauthorized token**: Try to get session with an unauthorized token
3. **Invalid session key**: Use an invalid session key for scrobbling
4. **Network errors**: Disconnect network and verify error handling
5. **Track too short**: Try to scrobble a track < 30 seconds (should be rejected by Last.fm)
6. **Timestamp too old**: Try to scrobble a track > 2 weeks old (should be rejected by Last.fm)

## Cleanup

After testing, you may want to remove test scrobbles from your Last.fm profile. You can do this manually from the Last.fm website.
