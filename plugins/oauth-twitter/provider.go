// Package oauthtwitter provides a Twitter (X) OAuth 2.0 provider for the Limen OAuth plugin.
package oauthtwitter

import (
	"context"
	"errors"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen/plugins/oauth"
)

var twitterEndpoint = oauth2.Endpoint{
	AuthURL:   "https://x.com/i/oauth2/authorize",
	TokenURL:  "https://api.x.com/2/oauth2/token",
	AuthStyle: oauth2.AuthStyleInHeader,
}

// New creates a Twitter (X) OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		scopes: []string{"users.read", "users.email", "tweet.read", "offline.access"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newTwitterProvider(cfg)
}

type twitterProvider struct {
	oauthConfig *oauth2.Config
	config      *config
	httpClient  *http.Client
}

func newTwitterProvider(cfg *config) *twitterProvider {
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       cfg.scopes,
		Endpoint:     twitterEndpoint,
	}
	return &twitterProvider{oauthConfig: config, config: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (t *twitterProvider) Name() string {
	return "twitter"
}

func (t *twitterProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption
	for key, value := range t.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return t.oauthConfig, authOpts
}

func (t *twitterProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	raw, err := oauth.FetchUserInfoJSON(ctx, t.httpClient, "twitter", "https://api.x.com/2/users/me?user.fields=profile_image_url,confirmed_email", token.AccessToken, nil)
	if err != nil {
		return nil, err
	}

	data, _ := raw["data"].(map[string]any)
	if data == nil {
		return nil, errors.New("twitter: empty data in user info response")
	}

	id, _ := data["id"].(string)
	if id == "" {
		return nil, errors.New("twitter: missing id in user info response")
	}

	name, _ := data["name"].(string)
	username, _ := data["username"].(string)
	if name == "" {
		name = username
	}
	avatarURL, _ := data["profile_image_url"].(string)
	email, _ := data["confirmed_email"].(string)

	if email == "" {
		return nil, errors.New("twitter: email is required for authentication; enable 'Request email from users' in your X app settings and ensure the user has a confirmed email")
	}

	return &oauth.ProviderUserInfo{
		ID:            id,
		Email:         email,
		EmailVerified: true,
		Name:          name,
		AvatarURL:     avatarURL,
		Raw:           data,
	}, nil
}
