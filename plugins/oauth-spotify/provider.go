// Package oauthspotify provides a Spotify OAuth provider for the Limen OAuth plugin.
package oauthspotify

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen/plugins/oauth"
)

var spotifyEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.spotify.com/authorize",
	TokenURL: "https://accounts.spotify.com/api/token",
}

// New creates a Spotify OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		clientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		clientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		scopes:       []string{"user-read-email"},
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
	raw, err := oauth.FetchUserInfoJSON(ctx, s.httpClient, "spotify", "https://api.spotify.com/v1/me", token.AccessToken, nil)
	if err != nil {
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
