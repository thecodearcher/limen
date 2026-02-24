package oauth

import (
	"context"

	"github.com/thecodearcher/aegis"
)

// API is the public interface for the OAuth feature.
// Use the Use function to obtain a type-safe reference from an Aegis instance.
type API interface {
	GetAuthorizationURL(ctx context.Context, providerName string, request *OAuthAuthorizeURLData) (string, string, error)

	ExchangeAuthorizationCodeForTokens(ctx context.Context, provider Provider, state, cookieNonce, code string) (*TokenResponse, map[string]any, error)

	GetUserInfoWithTokens(ctx context.Context, provider Provider, token *TokenResponse) (*aegis.OAuthAccountProfile, error)

	HandleOAuthCallback(ctx context.Context, providerName, code, state, cookieNonce string) (*aegis.OAuthAccountProfile, map[string]any, error)

	AuthenticateWithProvider(ctx context.Context, providerName, code, state, cookieNonce string) (*aegis.AuthenticationResult, map[string]any, error)

	CreateOrLinkAccount(ctx context.Context, info *aegis.OAuthAccountProfile) (*aegis.AuthenticationResult, error)

	LinkAccountToCurrentUser(ctx context.Context, user *aegis.User, info *aegis.OAuthAccountProfile) (*aegis.AuthenticationResult, error)

	ListAccountsForUser(ctx context.Context, userID any) ([]*aegis.Account, error)

	UnlinkAccount(ctx context.Context, user *aegis.User, providerName string) error

	GetAccessToken(ctx context.Context, userID any, providerName string) (*ActiveTokens, error)

	RefreshAccessToken(ctx context.Context, userID any, providerName string) (*ActiveTokens, error)
}

// Use returns a type-safe API for the OAuth feature.
// Panics if the feature was not registered in Config.Features,
// making it suitable for method chaining.
func Use(a *aegis.Aegis) API {
	return aegis.Use[API](a, aegis.FeatureOAuth)
}
