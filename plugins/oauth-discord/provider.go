// Package oauthdiscord provides a Discord OAuth provider for the Aegis OAuth plugin.
package oauthdiscord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/aegis/plugins/oauth"
)

var discordEndpoint = oauth2.Endpoint{
	AuthURL:  "https://discord.com/oauth2/authorize",
	TokenURL: "https://discord.com/api/oauth2/token",
}

// New creates a Discord OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		scopes: []string{"identify", "email"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newDiscordProvider(cfg)
}

type discordProvider struct {
	oauthConfig *oauth2.Config
	config      *config
	httpClient  *http.Client
}

func newDiscordProvider(cfg *config) *discordProvider {
	scopes := cfg.scopes
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       scopes,
		Endpoint:     discordEndpoint,
	}
	return &discordProvider{oauthConfig: config, config: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (d *discordProvider) Name() string {
	return "discord"
}

func (d *discordProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption
	for key, value := range d.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return d.oauthConfig, authOpts
}

func (d *discordProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord: user info request failed: %s", resp.Status)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	id, _ := raw["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("discord: missing id in user info response")
	}
	username, _ := raw["username"].(string)
	email, _ := raw["email"].(string)

	avatarURL := buildAvatarURL(id, raw)

	return &oauth.ProviderUserInfo{
		ID:            id,
		Email:         email,
		EmailVerified: email != "",
		Name:          username,
		AvatarURL:     avatarURL,
		Raw:           raw,
	}, nil
}

func buildAvatarURL(userID string, raw map[string]any) string {
	avatar, _ := raw["avatar"].(string)
	if avatar == "" {
		return ""
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userID, avatar)
}
