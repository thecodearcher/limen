package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"time"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/aegis"
)

// BuildAuthCodeURL builds the OAuth2 authorization URL using the provider's config.
// state and verifier are required for CSRF and PKCE; authOpts add provider-specific params (e.g. AccessTypeOffline).
func BuildAuthCodeURL(config *oauth2.Config, state, verifier string, authOpts ...oauth2.AuthCodeOption) string {
	opts := make([]oauth2.AuthCodeOption, 0, len(authOpts)+2)
	opts = append(opts, authOpts...)
	if verifier != "" {
		opts = append(opts,
			oauth2.S256ChallengeOption(verifier),
		)
	}
	return config.AuthCodeURL(state, opts...)
}

// ExchangeCode exchanges an authorization code for tokens using the provider's config.
// codeVerifier is required when PKCE was used on the authorization URL.
func ExchangeCode(ctx context.Context, config *oauth2.Config, code, codeVerifier string) (*TokenResponse, error) {
	var opts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		opts = append(opts, oauth2.VerifierOption(codeVerifier))
	}
	tok, err := config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, err
	}

	resp := &TokenResponse{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		ExpiresAt:    tok.Expiry,
	}

	if extra, ok := tok.Extra("id_token").(string); ok {
		resp.IDToken = extra
	}
	if scope, ok := tok.Extra("scope").(string); ok && scope != "" {
		resp.Scope = scope
	}
	return resp, nil
}

// generateCodeVerifier creates a cryptographically random PKCE code_verifier
func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// generateRandomString generates a cryptographically secure random string
func generateRandomString() string {
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	return hex.EncodeToString(randomBytes)
}

func newAccountFromOAuthProfile(userID any, profile *aegis.OAuthAccountProfile, tokens *OAuthTokens) *aegis.Account {
	now := time.Now()
	return &aegis.Account{
		UserID:               userID,
		Provider:             profile.Provider,
		ProviderAccountID:    profile.ProviderAccountID,
		AccessToken:          tokens.AccessToken,
		RefreshToken:         tokens.RefreshToken,
		AccessTokenExpiresAt: profile.AccessTokenExpiresAt,
		Scope:                profile.Scope,
		IDToken:              tokens.IDToken,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}
