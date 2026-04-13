// Package oauthlinkedin provides a LinkedIn OAuth 2.0 / OpenID Connect provider for the Limen OAuth plugin.
package oauthlinkedin

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen/plugins/oauth"
)

var linkedinEndpoint = oauth2.Endpoint{
	AuthURL:  "https://www.linkedin.com/oauth/v2/authorization",
	TokenURL: "https://www.linkedin.com/oauth/v2/accessToken",
}

// New creates a LinkedIn OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		clientID:     os.Getenv("LINKEDIN_CLIENT_ID"),
		clientSecret: os.Getenv("LINKEDIN_CLIENT_SECRET"),
		scopes:       []string{"openid", "profile", "email"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newLinkedInProvider(cfg)
}

type linkedInProvider struct {
	oauthConfig *oauth2.Config
	config      *config
}

func newLinkedInProvider(cfg *config) *linkedInProvider {
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       cfg.scopes,
		Endpoint:     linkedinEndpoint,
	}
	return &linkedInProvider{oauthConfig: config, config: cfg}
}

func (l *linkedInProvider) Name() string {
	return "linkedin"
}

func (l *linkedInProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption
	for key, value := range l.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return l.oauthConfig, authOpts
}

// PKCEEnabled returns false because LinkedIn does not support PKCE in the web auth-code
// flow—sending code_challenge or code_verifier causes the token exchange to fail.
func (l *linkedInProvider) PKCEEnabled() bool {
	return false
}

func (l *linkedInProvider) GetUserInfo(_ context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	if token.IDToken == "" {
		return nil, errors.New("linkedin: id_token required; include openid scope")
	}
	claims, err := oauth.DecodeIDTokenClaims(token.IDToken)
	if err != nil {
		return nil, fmt.Errorf("linkedin: %w", err)
	}

	id, _ := claims["sub"].(string)
	if id == "" {
		return nil, errors.New("linkedin: missing sub claim")
	}

	email, _ := claims["email"].(string)
	if email == "" {
		return nil, errors.New("linkedin: missing email claim")
	}
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)

	emailVerified := false
	if verified, ok := claims["email_verified"].(string); ok {
		emailVerified = verified == "true"
	}

	return &oauth.ProviderUserInfo{
		ID:            id,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		AvatarURL:     picture,
		Raw:           claims,
	}, nil
}
