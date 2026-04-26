// Package oauthconsentkeys provides a ConsentKeys OAuth/OIDC provider for the Limen OAuth plugin.
package oauthconsentkeys

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen/plugins/oauth"
)

const defaultDiscoveryURL = "https://api.consentkeys.com/.well-known/openid-configuration"

// New creates a ConsentKeys OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		clientID:     os.Getenv("CONSENTKEYS_CLIENT_ID"),
		clientSecret: os.Getenv("CONSENTKEYS_CLIENT_SECRET"),
		scopes:       []string{"openid", "profile", "email"},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	cfg.resolveDiscovery(defaultDiscoveryURL)
	cfg.validateEndpoints()

	return newConsentKeysProvider(cfg)
}

type consentKeysProvider struct {
	oauthConfig *oauth2.Config
	config      *config
	httpClient  *http.Client
}

func newConsentKeysProvider(cfg *config) *consentKeysProvider {
	scopes := cfg.scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.authorizationURL,
			TokenURL: cfg.tokenURL,
		},
	}
	return &consentKeysProvider{oauthConfig: config, config: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *consentKeysProvider) Name() string {
	return "consentkeys"
}

func (c *consentKeysProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption
	for key, value := range c.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return c.oauthConfig, authOpts
}

func (c *consentKeysProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	raw, err := oauth.FetchUserInfoJSON(ctx, c.httpClient, "consentkeys", c.config.userInfoURL, token.AccessToken, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, err
	}

	return mapUserInfo(raw)
}

func mapUserInfo(raw map[string]any) (*oauth.ProviderUserInfo, error) {
	id := stringClaim(raw, "sub")
	if id == "" {
		return nil, fmt.Errorf("consentkeys: missing sub in user info response")
	}

	name := stringClaim(raw, "name")
	if name == "" {
		name = stringClaim(raw, "preferred_username")
	}

	return &oauth.ProviderUserInfo{
		ID:            id,
		Email:         stringClaim(raw, "email"),
		EmailVerified: boolClaim(raw, "email_verified"),
		Name:          name,
		AvatarURL:     stringClaim(raw, "picture"),
		Raw:           raw,
	}, nil
}

func stringClaim(raw map[string]any, key string) string {
	value, _ := raw[key].(string)
	return value
}

func boolClaim(raw map[string]any, key string) bool {
	switch value := raw[key].(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(value)
		return err == nil && parsed
	default:
		return false
	}
}
