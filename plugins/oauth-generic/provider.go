package oauthgeneric

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen/plugins/oauth"
)

// New creates a generic OAuth provider that implements oauth.Provider.
// Panics if required options are missing: name, clientID, clientSecret, authorizationURL, tokenURL.
// For user info, provide one of: WithGetUserInfo (full custom), WithUserInfoURL + WithMapUserInfo,
// or WithMapUserInfo alone (will use the id_token claims when the provider returns one).
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	cfg.resolveDiscovery()
	cfg.validate()
	cfg.resolveDefaults()

	return &genericProvider{
		config: cfg,
		scopes: cfg.scopes,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type genericProvider struct {
	config *config
	scopes []string
	client *http.Client
}

func (g *genericProvider) Name() string {
	return g.config.name
}

func (g *genericProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	endpoint := oauth2.Endpoint{
		AuthURL:  g.config.authorizationURL,
		TokenURL: g.config.tokenURL,
	}
	cfg := &oauth2.Config{
		ClientID:     g.config.clientID,
		ClientSecret: g.config.clientSecret,
		RedirectURL:  g.config.redirectURL,
		Scopes:       g.scopes,
		Endpoint:     endpoint,
	}

	var authOpts []oauth2.AuthCodeOption
	for key, value := range g.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return cfg, authOpts
}

func (g *genericProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	if g.config.getUserInfo != nil {
		return g.config.getUserInfo(ctx, token)
	}
	if token.IDToken != "" {
		return g.userInfoFromIDToken(token.IDToken)
	}
	return g.fetchUserInfoFromURL(ctx, token)
}

// userInfoFromIDToken decodes the id_token JWT payload and passes the claims to mapUserInfo.
func (g *genericProvider) userInfoFromIDToken(idToken string) (*oauth.ProviderUserInfo, error) {
	claims, err := oauth.DecodeIDTokenClaims(idToken)
	if err != nil {
		return nil, err
	}
	info, err := g.config.mapUserInfo(claims)
	if err != nil {
		return nil, err
	}
	info.Raw = claims
	return info, nil
}

func (g *genericProvider) fetchUserInfoFromURL(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.config.userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed: %s", resp.Status)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	info, err := g.config.mapUserInfo(raw)
	if err != nil {
		return nil, err
	}
	info.Raw = raw
	return info, nil
}

func (g *genericProvider) BuildAuthorizationURL(ctx context.Context, state, codeVerifier, callbackRedirectURI string) (string, error) {
	if g.config.buildAuthorizationURL != nil {
		return g.config.buildAuthorizationURL(ctx, state, codeVerifier, callbackRedirectURI)
	}
	cfg, authOpts := g.OAuth2Config()
	cfg.RedirectURL = callbackRedirectURI
	return oauth.BuildAuthCodeURL(cfg, state, codeVerifier, authOpts...), nil
}

func (g *genericProvider) ExchangeAuthorizationCode(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth.TokenResponse, error) {
	if g.config.exchangeTokens != nil {
		return g.config.exchangeTokens(ctx, code, codeVerifier, redirectURI)
	}
	cfg, _ := g.OAuth2Config()
	cfg.RedirectURL = redirectURI
	return oauth.ExchangeCode(ctx, cfg, code, codeVerifier)
}

// RefreshToken implements oauth.TokenRefresher. When WithRefreshTokens is set, the custom
// function is used; otherwise the standard oauth2 token refresh flow is used.
func (g *genericProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth.TokenResponse, error) {
	if g.config.refreshTokens != nil {
		return g.config.refreshTokens(ctx, refreshToken)
	}
	cfg, _ := g.OAuth2Config()
	return oauth.RefreshToken(ctx, cfg, refreshToken)
}
