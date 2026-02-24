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

// ActiveTokens holds the decrypted OAuth tokens for a user's provider account.
type ActiveTokens struct {
	AccessToken          string     `json:"access_token"`
	RefreshToken         string     `json:"refresh_token,omitempty"`
	IDToken              string     `json:"id_token,omitempty"`
	AccessTokenExpiresAt *time.Time `json:"access_token_expires_at,omitempty"`
	Scope                string     `json:"scope,omitempty"`
}
