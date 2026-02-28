package oauth

import (
	"context"
	"fmt"
	"time"

	"github.com/thecodearcher/aegis"
)

// GetAccessToken retrieves and decrypts the stored OAuth tokens for a user's
// provider account. Returns the decrypted access token, refresh token, ID token,
// expiry, and scope.
func (o *oauthPlugin) GetAccessToken(ctx context.Context, userID any, providerName string) (*ActiveTokens, error) {
	account, err := o.findAccountByUserIDAndProvider(ctx, userID, providerName)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	tokens, err := o.decryptTokens(account)
	if err != nil {
		return nil, err
	}

	return &ActiveTokens{
		AccessToken:          tokens.AccessToken,
		RefreshToken:         tokens.RefreshToken,
		IDToken:              tokens.IDToken,
		AccessTokenExpiresAt: account.AccessTokenExpiresAt,
		Scope:                account.Scope,
	}, nil
}

// RefreshAccessToken uses the stored refresh token to obtain a new access token
// from the provider. The new tokens are encrypted and persisted back to the database.
// If the provider implements TokenRefresher, its custom logic is used; otherwise
// the standard oauth2 refresh flow is used.
func (o *oauthPlugin) RefreshAccessToken(ctx context.Context, userID any, providerName string) (*ActiveTokens, error) {
	provider, ok := o.providers[providerName]
	if !ok {
		return nil, ErrProviderNotFound
	}

	account, err := o.findAccountByUserIDAndProvider(ctx, userID, providerName)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	tokens, err := o.decryptTokens(account)
	if err != nil {
		return nil, err
	}

	fmt.Printf("tokens: %+v\n", account.RefreshToken)

	if tokens.RefreshToken == "" {
		return nil, ErrNoRefreshToken
	}

	tokenResp, err := o.refreshProviderToken(ctx, provider, tokens.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Preserve the existing refresh token if the provider didn't issue a new one.
	if tokenResp.RefreshToken == "" {
		tokenResp.RefreshToken = tokens.RefreshToken
	}

	profile := &aegis.OAuthAccountProfile{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
	}

	encrypted, err := o.encryptTokens(profile)
	if err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if !tokenResp.ExpiresAt.IsZero() {
		expiresAt = &tokenResp.ExpiresAt
	}

	updated := &aegis.Account{
		AccessToken:          encrypted.AccessToken,
		RefreshToken:         encrypted.RefreshToken,
		IDToken:              encrypted.IDToken,
		AccessTokenExpiresAt: expiresAt,
		UpdatedAt:            time.Now(),
	}
	if tokenResp.Scope != "" {
		updated.Scope = tokenResp.Scope
	}

	if err := o.core.Update(ctx, o.accountSchema, updated, []aegis.Where{
		aegis.Eq(o.accountSchema.GetIDField(), account.ID),
	}); err != nil {
		return nil, err
	}

	scope := tokenResp.Scope
	if scope == "" {
		scope = account.Scope
	}

	return &ActiveTokens{
		AccessToken:          tokenResp.AccessToken,
		RefreshToken:         tokenResp.RefreshToken,
		IDToken:              tokenResp.IDToken,
		AccessTokenExpiresAt: expiresAt,
		Scope:                scope,
	}, nil
}

func (o *oauthPlugin) refreshProviderToken(ctx context.Context, provider Provider, refreshToken string) (*TokenResponse, error) {
	if refresher, ok := provider.(TokenRefresher); ok {
		return refresher.RefreshToken(ctx, refreshToken)
	}
	config, _ := o.getProviderConfig(provider)
	return RefreshToken(ctx, config, refreshToken)
}
