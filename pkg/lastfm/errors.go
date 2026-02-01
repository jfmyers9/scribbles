package lastfm

import (
	"fmt"
)

// Error represents a Last.fm API error.
//
// The Error type provides structured error information including
// the Last.fm error code and message. It implements error, and
// provides additional methods for retry logic.
type Error struct {
	Code    int    // Last.fm error code
	Message string // Error message from Last.fm
}

// Error returns the error message.
func (e *Error) Error() string {
	return fmt.Sprintf("lastfm: error %d: %s", e.Code, e.Message)
}

// Is checks if the target error is a Last.fm error.
//
// This allows errors.Is() to work with *Error types.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Temporary returns true if the error is temporary and the request
// should be retried.
//
// The following Last.fm error codes are considered temporary:
//   - 11: Service Offline - temporarily unavailable
//   - 16: Service Temporarily Unavailable
//
// Network errors and timeouts should also be considered temporary
// but are not represented by this type.
func (e *Error) Temporary() bool {
	switch e.Code {
	case 11: // Service Offline
		return true
	case 16: // Service Temporarily Unavailable
		return true
	default:
		return false
	}
}

// Common Last.fm error codes.
const (
	ErrCodeInvalidService       = 2
	ErrCodeInvalidMethod        = 3
	ErrCodeAuthenticationFailed = 4
	ErrCodeInvalidFormat        = 5
	ErrCodeInvalidParameters    = 6
	ErrCodeInvalidResourceSpec  = 7
	ErrCodeOperationFailed      = 8
	ErrCodeInvalidSessionKey    = 9
	ErrCodeInvalidAPIKey        = 10
	ErrCodeServiceOffline       = 11
	ErrCodeSubscribersOnly      = 12
	ErrCodeInvalidSignature     = 13
	ErrCodeUnauthorizedToken    = 14
	ErrCodeExpiredToken         = 15
	ErrCodeTempUnavailable      = 16
	ErrCodeRateLimitExceeded    = 29
)

// Predefined errors for common cases.
var (
	// ErrNoSessionKey is returned when an operation requires authentication
	// but no session key has been set.
	ErrNoSessionKey = fmt.Errorf("lastfm: session key required")

	// ErrInvalidConfig is returned when client configuration is invalid.
	ErrInvalidConfig = fmt.Errorf("lastfm: invalid configuration")
)

// isRetryableError determines if an error should trigger a retry.
//
// This is used internally by the transport layer to implement
// retry logic with exponential backoff.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a Last.fm API error
	var lastfmErr *Error
	if e, ok := err.(*Error); ok {
		lastfmErr = e
	}

	if lastfmErr != nil {
		return lastfmErr.Temporary()
	}

	// Network errors are generally retryable but are not
	// represented by *Error. The transport layer handles
	// network error retry separately.
	return false
}
