// Package oauthgoogle provides a Google OAuth provider for the Aegis OAuth feature.
package oauthgoogle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/thecodearcher/aegis/features/oauth"
)

const providerName = "google"

// New creates a Google OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		scopes: []string{"openid", "email", "profile"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newGoogleProvider(cfg)
}

type googleProvider struct {
	oauthConfig *oauth2.Config
	config      *config
}

func newGoogleProvider(cfg *config) *googleProvider {
	scopes := cfg.scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       scopes,
		Endpoint:     google.Endpoint,
	}
	return &googleProvider{oauthConfig: config, config: cfg}
}

func (g *googleProvider) Name() string {
	return providerName
}

func (g *googleProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	authOpts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("response_type", "code"),
	}

	if g.config.prompt != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("prompt", g.config.prompt))
	}
	if g.config.accessType != "" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("access_type", g.config.accessType))
	}
	if g.config.includeGrantedScopes {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("include_granted_scopes", "true"))
	}
	return g.oauthConfig, authOpts
}

func (g *googleProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed: %s", resp.Status)
	}
	raw := map[string]any{}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	return &oauth.ProviderUserInfo{
		ID:            raw["id"].(string),
		Email:         raw["email"].(string),
		EmailVerified: raw["verified_email"].(bool),
		Name:          raw["name"].(string),
		AvatarURL:     raw["picture"].(string),
		Raw:           raw,
	}, nil
}
