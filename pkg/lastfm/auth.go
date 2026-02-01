package lastfm

import (
	"context"
	"encoding/xml"
	"fmt"
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
	resp, err := a.client.call(ctx, "auth.getToken", nil, false)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := unmarshalToken(resp, &token); err != nil {
		return nil, err
	}

	return &token, nil
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
	params := map[string]string{
		"token": token,
	}

	resp, err := a.client.call(ctx, "auth.getSession", params, false)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := unmarshalSession(resp, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// tokenResponse represents the XML response from auth.getToken.
type tokenResponse struct {
	Token string `xml:"token"`
}

// unmarshalToken parses the auth.getToken XML response.
func unmarshalToken(data []byte, token *Token) error {
	// Wrap inner XML in root element for proper unmarshaling
	wrapped := []byte("<root>" + string(data) + "</root>")

	var resp tokenResponse
	if err := xml.Unmarshal(wrapped, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	token.Token = resp.Token
	return nil
}

// sessionResponse represents the XML response from auth.getSession.
type sessionResponse struct {
	Name       string `xml:"session>name"`
	Key        string `xml:"session>key"`
	Subscriber int    `xml:"session>subscriber"`
}

// unmarshalSession parses the auth.getSession XML response.
func unmarshalSession(data []byte, session *Session) error {
	// Wrap inner XML in root element for proper unmarshaling
	wrapped := []byte("<root>" + string(data) + "</root>")

	var resp sessionResponse
	if err := xml.Unmarshal(wrapped, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal session response: %w", err)
	}

	session.Key = resp.Key
	session.Username = resp.Name
	session.Subscriber = resp.Subscriber == 1
	return nil
}
