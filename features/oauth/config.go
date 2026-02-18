package oauth

import (
	"context"
	"time"

	"github.com/thecodearcher/aegis"
)

type ConfigOption func(*config)

type config struct {
	secret                      []byte
	useDatabaseState            bool
	requireExplicitSignUp       bool
	providers                   map[string]Provider
	disableTokensEncryption     bool
	encryptTokens               func(secret []byte, tokens *OAuthTokens) (*OAuthTokens, error)
	decryptTokens               func(secret []byte, tokens *OAuthTokens) (*OAuthTokens, error)
	mapProfileToUser            func(info *aegis.OAuthAccountProfile) map[string]any
	getUserInfo                 func(ctx context.Context, provider string, token *TokenResponse) (*ProviderUserInfo, error)
	cookieName                  string
	cookieTTL                   time.Duration
	allowLinkingDifferentEmails bool
	disableRedirect             bool
}

// WithProvider registers an OAuth provider (e.g. Google, GitHub).
func WithProvider(p Provider) ConfigOption {
	return func(c *config) {
		if c.providers == nil {
			c.providers = make(map[string]Provider)
		}
		c.providers[p.Name()] = p
	}
}

// WithSecret sets the 32-byte secret used for encrypting OAuth tokens at rest and
// stateless state tokens. The key must be exactly 32 bytes (e.g. 32 ASCII characters);
func WithSecret(key string) ConfigOption {
	return func(c *config) {
		c.secret = []byte(key)
	}
}

// WithDatabaseState uses the database to store the OAuth state tokens.
func WithDatabaseState() ConfigOption {
	return func(c *config) {
		c.useDatabaseState = true
	}
}

// WithRequireExplicitSignUp disables new users signing in via OAuth and error if the user is not found.
func WithRequireExplicitSignUp() ConfigOption {
	return func(c *config) {
		c.requireExplicitSignUp = true
	}
}

// WithEncryptTokens sets the function to encrypt the OAuth tokens.
// The secret passed to the function is the same as the secret passed to WithSecret.
func WithEncryptTokens(encryptTokens func(secret []byte, tokens *OAuthTokens) (*OAuthTokens, error)) ConfigOption {
	return func(c *config) {
		c.encryptTokens = encryptTokens
	}
}

// WithDecryptTokens sets the function to decrypt the OAuth tokens.
// The secret passed to the function is the same as the secret passed to WithSecret.
func WithDecryptTokens(decryptTokens func(secret []byte, tokens *OAuthTokens) (*OAuthTokens, error)) ConfigOption {
	return func(c *config) {
		c.decryptTokens = decryptTokens
	}
}

// WithDisableTokensEncryption disables the encryption of OAuth tokens.
// When enabled, tokens are not encrypted and are stored in plain text.
func WithDisableTokensEncryption() ConfigOption {
	return func(c *config) {
		c.disableTokensEncryption = true
	}
}

// WithMapProfileToUser sets the function to map the OAuth profile to a user additional fields.
func WithMapProfileToUser(mapProfileToUser func(info *aegis.OAuthAccountProfile) map[string]any) ConfigOption {
	return func(c *config) {
		c.mapProfileToUser = mapProfileToUser
	}
}

// WithGetUserInfo sets the function to get the user info from the provider.
// The function is called with the provider name and token and should return the user info.
func WithGetUserInfo(getUserInfo func(ctx context.Context, provider string, token *TokenResponse) (*ProviderUserInfo, error)) ConfigOption {
	return func(c *config) {
		c.getUserInfo = getUserInfo
	}
}

// WithCookieName sets the name of the cookie used to store the OAuth state.
func WithCookieName(name string) ConfigOption {
	return func(c *config) {
		c.cookieName = name
	}
}

// WithCookieTTL sets the TTL of the cookie used to store the OAuth state.
func WithCookieTTL(ttl time.Duration) ConfigOption {
	return func(c *config) {
		c.cookieTTL = ttl
	}
}

// WithAllowLinkingDifferentEmails allows linking user with a different email to the one in the OAuth profile.
//
// Be careful with this option as it can lead to account takeover if not used correctly.
func WithAllowLinkingDifferentEmails() ConfigOption {
	return func(c *config) {
		c.allowLinkingDifferentEmails = true
	}
}

// WithDisableRedirect disables the redirect to the redirect URI after the OAuth callback.
func WithDisableRedirect() ConfigOption {
	return func(c *config) {
		c.disableRedirect = true
	}
}
