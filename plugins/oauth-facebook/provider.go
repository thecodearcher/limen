// Package oauthfacebook provides a Facebook OAuth provider for the Aegis OAuth plugin.
package oauthfacebook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/aegis/plugins/oauth"
)

var facebookEndpoint = oauth2.Endpoint{
	AuthURL:  "https://www.facebook.com/v25.0/dialog/oauth",
	TokenURL: "https://graph.facebook.com/v25.0/oauth/access_token",
}

// New creates a Facebook OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		scopes: []string{"email", "public_profile"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newFacebookProvider(cfg)
}

type facebookProvider struct {
	oauthConfig *oauth2.Config
	config      *config
	httpClient  *http.Client
}

func newFacebookProvider(cfg *config) *facebookProvider {
	scopes := cfg.scopes
	if len(scopes) == 0 {
		scopes = []string{"email", "public_profile"}
	}
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       scopes,
		Endpoint:     facebookEndpoint,
	}
	return &facebookProvider{oauthConfig: config, config: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (f *facebookProvider) Name() string {
	return "facebook"
}

func (f *facebookProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption
	for key, value := range f.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return f.oauthConfig, authOpts
}

func (f *facebookProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://graph.facebook.com/me?fields=id,name,email,picture.type(large)", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook: user info request failed: %s", resp.Status)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	id, _ := raw["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("facebook: missing id in user info response")
	}
	name, _ := raw["name"].(string)
	email, _ := raw["email"].(string)

	avatarURL := extractPictureURL(raw)

	return &oauth.ProviderUserInfo{
		ID:            id,
		Email:         email,
		EmailVerified: email != "",
		Name:          name,
		AvatarURL:     avatarURL,
		Raw:           raw,
	}, nil
}

// Facebook returns picture as {"picture":{"data":{"url":"..."}}}
func extractPictureURL(raw map[string]any) string {
	pic, ok := raw["picture"].(map[string]any)
	if !ok {
		return ""
	}
	data, ok := pic["data"].(map[string]any)
	if !ok {
		return ""
	}
	url, _ := data["url"].(string)
	return url
}
