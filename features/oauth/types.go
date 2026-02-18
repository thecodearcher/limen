package oauth

import (
	"time"
)

// TokenResponse holds the OAuth2 token exchange response.
type TokenResponse struct {
	AccessToken    string
	RefreshToken   string
	ExpiresAt      time.Time
	TokenType      string
	Scope          string
	IDToken        string
	AdditionalData map[string]any
}

// ProviderUserInfo holds the user profile returned by an OAuth provider.
type ProviderUserInfo struct {
	ID            string
	Email         string
	EmailVerified bool
	Name          string
	AvatarURL     string
	Raw           map[string]any
}

type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
}

type OAuthAuthorizeURLData struct {
	AdditionalData   map[string]any
	RedirectURI      string
	ErrorRedirectURI string
}
