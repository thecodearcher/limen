package oauthgithub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen/plugins/oauth"
)

func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		clientID:     os.Getenv("GITHUB_CLIENT_ID"),
		clientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		scopes:       []string{"read:user", "user:email"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newGitHubProvider(cfg)
}

var githubEndpoint = oauth2.Endpoint{
	AuthURL:  "https://github.com/login/oauth/authorize",
	TokenURL: "https://github.com/login/oauth/access_token",
}

type githubProvider struct {
	oauthConfig *oauth2.Config
	config      *config
	httpClient  *http.Client
}

func newGitHubProvider(cfg *config) *githubProvider {
	scopes := cfg.scopes
	if len(scopes) == 0 {
		scopes = []string{"read:user", "user:email"}
	}
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       scopes,
		Endpoint:     githubEndpoint,
	}
	return &githubProvider{oauthConfig: config, config: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (g *githubProvider) Name() string {
	return "github"
}

func (g *githubProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	authOpts := []oauth2.AuthCodeOption{}
	for key, value := range g.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return g.oauthConfig, authOpts
}

func (g *githubProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	raw, err := oauth.FetchUserInfoJSON(ctx, g.httpClient, "github", "https://api.github.com/user", token.AccessToken, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, err
	}

	email := raw["email"]
	if email == nil || email == "" {
		email, _ = g.fetchPrimaryEmail(ctx, token.AccessToken)
	}

	id, _ := raw["id"].(float64)
	name, _ := raw["name"].(string)
	avatarURL, _ := raw["avatar_url"].(string)
	return &oauth.ProviderUserInfo{
		ID:            fmt.Sprintf("%d", int64(id)),
		Email:         email.(string),
		EmailVerified: email != "",
		Name:          name,
		AvatarURL:     avatarURL,
		Raw:           raw,
	}, nil
}

func (g *githubProvider) fetchPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", nil
}
