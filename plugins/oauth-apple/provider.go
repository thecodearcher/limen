// Package oauthapple provides an Apple Sign In OAuth provider for the Limen OAuth plugin.
package oauthapple

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen/plugins/oauth"
)

var appleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://appleid.apple.com/auth/authorize",
	TokenURL: "https://appleid.apple.com/auth/token",
}

// New creates an Apple OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		clientID:     os.Getenv("APPLE_CLIENT_ID"),
		clientSecret: os.Getenv("APPLE_CLIENT_SECRET"),
		scopes:       []string{"name", "email"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newAppleProvider(cfg)
}

type appleProvider struct {
	oauthConfig *oauth2.Config
	config      *config
}

func newAppleProvider(cfg *config) *appleProvider {
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       cfg.scopes,
		Endpoint:     appleEndpoint,
	}
	return &appleProvider{oauthConfig: oauthCfg, config: cfg}
}

func (a *appleProvider) Name() string {
	return "apple"
}

func (a *appleProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption
	for key, value := range a.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return a.oauthConfig, authOpts
}

// ResponseMode returns form_post because Apple delivers the authorization
// response (including the first-login user payload) as a POST body.
func (a *appleProvider) ResponseMode() oauth.ResponseMode {
	return oauth.ResponseModeFormPost
}

func (a *appleProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	if token.IDToken == "" {
		return nil, errors.New("apple: id_token required; include email scope")
	}
	claims, err := oauth.DecodeIDTokenClaims(token.IDToken)
	if err != nil {
		return nil, fmt.Errorf("apple: %w", err)
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errors.New("apple: missing sub claim")
	}

	email, _ := claims["email"].(string)
	if email == "" {
		return nil, errors.New("apple: missing email claim")
	}

	emailVerified := false
	if v, ok := claims["email_verified"].(string); ok {
		emailVerified = v == "true"
	} else if v, ok := claims["email_verified"].(bool); ok {
		emailVerified = v
	}

	name := extractNameFromParams(oauth.CallbackParams(ctx))

	return &oauth.ProviderUserInfo{
		ID:            sub,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		Raw:           claims,
	}, nil
}

// extractNameFromParams reads the user's name from the Apple first-login
// "user" callback parameter.
// Apple sends this JSON only on the very first authorization for a user.
func extractNameFromParams(params url.Values) string {
	if params == nil {
		return ""
	}
	raw := params.Get("user")
	if raw == "" {
		return ""
	}

	var payload struct {
		Name struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"name"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}

	first := payload.Name.FirstName
	last := payload.Name.LastName
	switch {
	case first != "" && last != "":
		return first + " " + last
	case first != "":
		return first
	default:
		return last
	}
}
