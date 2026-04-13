// Package oauthgoogle provides a Google OAuth provider for the Limen OAuth plugin.
package oauthgoogle

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/thecodearcher/limen/plugins/oauth"
)

// New creates a Google OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		clientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		clientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		scopes:       []string{"openid", "email", "profile"},
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
	return "google"
}

func (g *googleProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption

	for key, value := range g.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return g.oauthConfig, authOpts
}

func (g *googleProvider) GetUserInfo(_ context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	if token.IDToken == "" {
		return nil, errors.New("google: id_token required; include openid scope")
	}
	claims, err := decodeIDTokenClaims(token.IDToken)
	if err != nil {
		return nil, fmt.Errorf("google: %w", err)
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errors.New("google: id token missing sub claim")
	}
	email, _ := claims["email"].(string)
	if email == "" {
		return nil, errors.New("google: id token missing email claim")
	}
	emailVerified, _ := claims["email_verified"].(bool)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)

	return &oauth.ProviderUserInfo{
		ID:            sub,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		AvatarURL:     picture,
		Raw:           claims,
	}, nil
}

// decodeIDTokenClaims decodes the payload segment of a JWT without verification.
// Safe here because the token was obtained directly from Google's token endpoint over TLS.
func decodeIDTokenClaims(idToken string) (map[string]any, error) {
	parts := strings.SplitN(idToken, ".", 3)
	if len(parts) != 3 {
		return nil, errors.New("id token has invalid JWT format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("id token payload decode: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("id token payload unmarshal: %w", err)
	}

	return claims, nil
}
