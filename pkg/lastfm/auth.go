package lastfm

import (
	"context"
)

// AuthService provides authentication operations for the Last.fm API.
type AuthService struct {
	client *Client
}

// GetToken requests an authentication token from Last.fm.
//
// This is the first step in the authentication flow. After obtaining a token,
// the user must authorize it by visiting the URL returned by GetAuthURL.
//
// Example:
//
//	token, err := client.Auth().GetToken(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Visit:", client.Auth().GetAuthURL(token.Token))
func (a *AuthService) GetToken(ctx context.Context) (*Token, error) {
	// Implementation will be added in core implementation phase
	return nil, nil
}

// GetAuthURL returns the URL where users authorize the token.
//
// After calling GetToken, direct the user to this URL to authorize
// the application. Once authorized, call GetSession to exchange the
// token for a session key.
//
// Example:
//
//	authURL := client.Auth().GetAuthURL(token.Token)
//	fmt.Println("Please visit:", authURL)
func (a *AuthService) GetAuthURL(token string) string {
	return "https://www.last.fm/api/auth/?api_key=" + a.client.apiKey + "&token=" + token
}

// GetSession exchanges an authorized token for a session key.
//
// After the user has authorized the token at the URL from GetAuthURL,
// call this method to exchange the token for a permanent session key.
// The session key should be stored and used for all future authenticated
// requests.
//
// Example:
//
//	session, err := client.Auth().GetSession(ctx, token.Token)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	client.SetSessionKey(session.Key)
//	// Store session.Key for future use
func (a *AuthService) GetSession(ctx context.Context, token string) (*Session, error) {
	// Implementation will be added in core implementation phase
	return nil, nil
}
