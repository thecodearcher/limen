package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	if resp.Scope == "" {
		resp.Scope = strings.Join(config.Scopes, ",")
	}
	return resp, nil
}

// RefreshToken uses the standard oauth2.TokenSource to exchange a refresh token
// for a new access token via the provider's token endpoint.
func RefreshToken(ctx context.Context, config *oauth2.Config, refreshToken string) (*TokenResponse, error) {
	src := config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	tok, err := src.Token()
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

// FetchUserInfoJSON performs a GET request to the given URL with a Bearer token
// and decodes the JSON response into a map. Shared by REST-based OAuth providers.
func FetchUserInfoJSON(ctx context.Context, client *http.Client, providerName, url, accessToken string, extraHeaders map[string]string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: user info request failed: %s", providerName, resp.Status)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// DecodeIDTokenClaims decodes the payload segment of a JWT without verification.
//
// NOTE: This does not verify the token, so it is not safe to use for any purpose other than to get the claims
// from the id_token returned by the provider.
func DecodeIDTokenClaims(idToken string) (map[string]any, error) {
	parts := strings.SplitN(idToken, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("id_token has invalid JWT format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("id_token payload decode: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("id_token payload unmarshal: %w", err)
	}
	return claims, nil
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
