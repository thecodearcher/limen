// Package oauthmicrosoft provides a Microsoft OAuth provider for the Limen OAuth plugin.
package oauthmicrosoft

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen/plugins/oauth"
)

const (
	defaultTenant    = "common"
	defaultAuthority = "https://login.microsoftonline.com"
)

func microsoftEndpoint(authority string) oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  authority + "/oauth2/v2.0/authorize",
		TokenURL: authority + "/oauth2/v2.0/token",
	}
}

// New creates a Microsoft OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		scopes: []string{"openid", "profile", "email"},
		tenant: defaultTenant,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newMicrosoftProvider(cfg)
}

type microsoftProvider struct {
	oauthConfig *oauth2.Config
	config      *config
}

func newMicrosoftProvider(cfg *config) *microsoftProvider {
	authority := strings.TrimRight(cfg.authorityURL, "/")
	if authority == "" {
		tenant := cfg.tenant
		if tenant == "" {
			tenant = defaultTenant
		}
		authority = defaultAuthority + "/" + tenant
	}
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       cfg.scopes,
		Endpoint:     microsoftEndpoint(authority),
	}
	return &microsoftProvider{oauthConfig: config, config: cfg}
}

func (m *microsoftProvider) Name() string {
	return "microsoft"
}

func (m *microsoftProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption
	for key, value := range m.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return m.oauthConfig, authOpts
}

func (m *microsoftProvider) GetUserInfo(_ context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	if token.IDToken == "" {
		return nil, errors.New("microsoft: id_token required; include openid scope")
	}
	claims, err := oauth.DecodeIDTokenClaims(token.IDToken)
	if err != nil {
		return nil, fmt.Errorf("microsoft: %w", err)
	}

	oid, _ := claims["oid"].(string)
	if oid == "" {
		return nil, errors.New("microsoft: id token missing oid claim")
	}

	email := extractEmail(claims)
	name, _ := claims["name"].(string)

	return &oauth.ProviderUserInfo{
		ID:            oid,
		Email:         email,
		EmailVerified: email != "",
		Name:          name,
		Raw:           claims,
	}, nil
}

// extractEmail returns the user's email from the ID token claims.
// "email" is preferred; falls back to "preferred_username" which Microsoft
// typically populates with the user's UPN or email address.
func extractEmail(claims map[string]any) string {
	if email, _ := claims["email"].(string); email != "" {
		return email
	}
	upn, _ := claims["preferred_username"].(string)
	return upn
}
