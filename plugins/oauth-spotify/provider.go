// Package oauthspotify provides a Spotify OAuth provider for the Aegis OAuth plugin.
package oauthspotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/aegis/plugins/oauth"
)

var spotifyEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.spotify.com/authorize",
	TokenURL: "https://accounts.spotify.com/api/token",
}

// New creates a Spotify OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		scopes: []string{"user-read-email"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newSpotifyProvider(cfg)
}

type spotifyProvider struct {
	oauthConfig *oauth2.Config
	config      *config
	httpClient  *http.Client
}

func newSpotifyProvider(cfg *config) *spotifyProvider {
	scopes := cfg.scopes
	if len(scopes) == 0 {
		scopes = []string{"user-read-email"}
	}
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       scopes,
		Endpoint:     spotifyEndpoint,
	}
	return &spotifyProvider{oauthConfig: config, config: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (s *spotifyProvider) Name() string {
	return "spotify"
}

func (s *spotifyProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption
	for key, value := range s.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return s.oauthConfig, authOpts
}

func (s *spotifyProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.spotify.com/v1/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify: user info request failed: %s", resp.Status)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	id, _ := raw["id"].(string)
	if id == "" {
		return nil, errors.New("spotify: missing id in user info response")
	}
	email, _ := raw["email"].(string)
	if email == "" {
		return nil, errors.New("spotify: missing email in user info response; include user-read-email scope")
	}

	name, _ := raw["display_name"].(string)
	if name == "" {
		name = id
	}

	return &oauth.ProviderUserInfo{
		ID:            id,
		Email:         email,
		EmailVerified: email != "",
		Name:          name,
		AvatarURL:     extractAvatarURL(raw),
		Raw:           raw,
	}, nil
}

// Spotify returns images as an array like [{"url":"...","height":300,"width":300}].
func extractAvatarURL(raw map[string]any) string {
	images, ok := raw["images"].([]any)
	if !ok || len(images) == 0 {
		return ""
	}
	first, ok := images[0].(map[string]any)
	if !ok {
		return ""
	}
	url, _ := first["url"].(string)
	return url
}
