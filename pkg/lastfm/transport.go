package lastfm

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Base represents the root XML response from Last.fm API.
type Base struct {
	XMLName xml.Name `xml:"lfm"`
	Status  string   `xml:"status,attr"`
	Inner   []byte   `xml:",innerxml"`
}

// APIError represents an error response from the Last.fm API.
type APIError struct {
	Code    int    `xml:"code,attr"`
	Message string `xml:",chardata"`
}

const (
	apiStatusOK     = "ok"
	apiStatusFailed = "failed"
)

// call makes an HTTP request to the Last.fm API with retry logic.
//
// It handles:
// - Request construction with proper headers
// - Signature calculation for authenticated requests
// - Response parsing (XML)
// - Error handling and retry logic
// - Context cancellation
func (c *Client) call(ctx context.Context, method string, params map[string]string, requiresAuth bool) ([]byte, error) {
	// Build request parameters
	reqParams := make(map[string]string)
	for k, v := range params {
		reqParams[k] = v
	}
	reqParams["method"] = method
	reqParams["api_key"] = c.apiKey

	// Add session key for authenticated requests
	if requiresAuth {
		if c.sessionKey == "" {
			return nil, ErrNoSessionKey
		}
		reqParams["sk"] = c.sessionKey
	}

	// Calculate signature
	signature := calculateSignature(reqParams, c.apiSecret)

	// Build form data
	formData := url.Values{}
	for k, v := range reqParams {
		formData.Add(k, v)
	}
	formData.Add("api_sig", signature)

	// Retry with exponential backoff
	var lastErr error
	backoff := 1 * time.Second
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		c.logDebugf("lastfm: calling %s (attempt %d/%d)", method, i+1, maxRetries)

		// Make the HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, strings.NewReader(formData.Encode()))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "scribbles/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if shouldRetryNetworkError(err) && i < maxRetries-1 {
				c.logDebugf("lastfm: network error, retrying: %v", err)
				if !sleep(ctx, backoff) {
					return nil, ctx.Err()
				}
				backoff = nextBackoff(backoff)
				continue
			}
			return nil, fmt.Errorf("http request failed: %w", err)
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		// Handle HTTP status codes
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d %s", resp.StatusCode, resp.Status)
			if i < maxRetries-1 {
				c.logDebugf("lastfm: server error, retrying: %v", lastErr)
				if !sleep(ctx, backoff) {
					return nil, ctx.Err()
				}
				backoff = nextBackoff(backoff)
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Parse XML response
		var base Base
		if err := xml.Unmarshal(body, &base); err != nil {
			return nil, fmt.Errorf("failed to parse XML response: %w", err)
		}

		// Check for API errors
		if base.Status == apiStatusFailed {
			var apiErr APIError
			if err := xml.Unmarshal(base.Inner, &apiErr); err != nil {
				return nil, fmt.Errorf("failed to parse error response: %w", err)
			}

			lastfmErr := &Error{
				Code:    apiErr.Code,
				Message: apiErr.Message,
			}

			// Retry temporary errors
			if lastfmErr.Temporary() && i < maxRetries-1 {
				c.logDebugf("lastfm: temporary error, retrying: %v", lastfmErr)
				lastErr = lastfmErr
				if !sleep(ctx, backoff) {
					return nil, ctx.Err()
				}
				backoff = nextBackoff(backoff)
				continue
			}

			return nil, lastfmErr
		}

		// Success
		c.logDebugf("lastfm: %s succeeded", method)
		return base.Inner, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// shouldRetryNetworkError checks if a network error is retryable.
func shouldRetryNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network errors
	if _, ok := err.(net.Error); ok {
		return true
	}

	// Check for URL errors (which may contain network errors)
	if urlErr, ok := err.(*url.Error); ok {
		if _, ok := urlErr.Err.(net.Error); ok {
			return true
		}
		if netErr, ok := urlErr.Err.(net.Error); ok && netErr.Timeout() {
			return true
		}
	}

	return false
}

// sleep waits for the specified duration or until context is cancelled.
// Returns true if sleep completed, false if context was cancelled.
func sleep(ctx context.Context, duration time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(duration):
		return true
	}
}

// nextBackoff calculates the next backoff duration with exponential increase.
// Maximum backoff is capped at 30 seconds.
func nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > 30*time.Second {
		return 30 * time.Second
	}
	return next
}
