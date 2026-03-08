package sessionjwt

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/thecodearcher/aegis"
)

// AegisClaims is the JWT claims structure used by the session-jwt plugin.
// Core user fields are embedded so that ValidateSession can reconstruct a
// User without hitting the database.
type AegisClaims struct {
	jwt.RegisteredClaims
	Email         string         `json:"email,omitempty"`
	EmailVerified bool           `json:"email_verified,omitempty"`
	Custom        map[string]any `json:"custom,omitempty"`
}

func (p *sessionJWTPlugin) GenerateAccessToken(user *aegis.User) (string, string, error) {
	now := time.Now()
	expiresAt := now.Add(p.config.accessTokenDuration)
	jti := generateJTI()

	claims := AegisClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   p.config.subjectEncoder(user),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    p.config.issuer,
			Audience:  jwt.ClaimStrings(p.config.audience),
		},
	}

	if !p.config.refreshUser {
		claims.Email = user.Email
		claims.EmailVerified = user.EmailVerifiedAt != nil
	}

	if p.config.customClaims != nil {
		claims.Custom = p.config.customClaims(user)
	}

	token := jwt.NewWithClaims(p.config.signingMethod, claims)
	signed, err := token.SignedString(p.config.signingKey)
	if err != nil {
		return "", "", fmt.Errorf("session-jwt: failed to sign token: %w", err)
	}
	return signed, jti, nil
}

func (p *sessionJWTPlugin) VerifyAccessToken(tokenString string) (*AegisClaims, error) {
	keyFunc := func(token *jwt.Token) (any, error) {
		return p.config.verificationKey, nil
	}

	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{p.config.signingMethod.Alg()}),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(p.config.issuer),
		jwt.WithAudience(p.config.audience...),
	}

	token, err := jwt.ParseWithClaims(tokenString, &AegisClaims{}, keyFunc, opts...)
	if err != nil {
		return nil, ErrInvalidAccessToken
	}

	claims, ok := token.Claims.(*AegisClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidAccessToken
	}

	return claims, nil
}

func (p *sessionJWTPlugin) RefreshAccessToken(r *http.Request) (*aegis.SessionResult, *aegis.User, error) {
	if !p.config.refreshTokenEnabled {
		return nil, nil, ErrRefreshTokensDisabled
	}

	rawToken, family, err := p.extractRefreshToken(r)
	if err != nil {
		return nil, nil, err
	}
	return p.performRefresh(r.Context(), rawToken, family)
}

func (p *sessionJWTPlugin) performRefresh(ctx context.Context, rawRefreshToken string, family string) (*aegis.SessionResult, *aegis.User, error) {
	rt, err := p.FindRefreshTokenByToken(ctx, rawRefreshToken)
	if err != nil {
		if family != "" {
			if active, _ := p.FamilyHasActiveTokens(ctx, family); active {
				_ = p.DeleteRefreshTokenFamily(ctx, family)
				return nil, nil, ErrRefreshTokenReuse
			}
		}
		return nil, nil, ErrInvalidRefreshToken
	}

	if time.Now().After(rt.ExpiresAt) {
		_ = p.DeleteRefreshToken(ctx, rt.Token)
		return nil, nil, ErrInvalidRefreshToken
	}

	user, err := p.core.DBAction.FindUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("session-jwt: user not found: %w", err)
	}

	signed, jti, err := p.GenerateAccessToken(user)
	if err != nil {
		return nil, nil, err
	}

	var result *aegis.SessionResult

	if p.config.refreshTokenRotation {
		newRT, err := p.RotateRefreshToken(ctx, rt, jti)
		if err != nil {
			return nil, nil, err
		}
		result = p.buildSessionResult(signed, newRT)
	} else {
		if err := p.updateRefreshTokenJWTID(ctx, rt.Token, jti); err != nil {
			return nil, nil, err
		}
		result = p.buildSessionResult(signed, rt)
	}

	return result, user, nil
}

// parseAccessTokenLenient parses a JWT verifying the signature but tolerating
// expiry. Used by RevokeSession so that expired tokens can still have their
// JTI extracted for blacklisting / refresh token cleanup.
// Returns nil when the token is structurally invalid or the signature fails.
func (p *sessionJWTPlugin) parseAccessTokenLenient(tokenString string) *AegisClaims {
	keyFunc := func(token *jwt.Token) (any, error) {
		return p.config.verificationKey, nil
	}

	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{p.config.signingMethod.Alg()}),
	}

	token, _ := jwt.ParseWithClaims(tokenString, &AegisClaims{}, keyFunc, opts...)
	if token == nil {
		return nil
	}

	claims, ok := token.Claims.(*AegisClaims)
	if !ok {
		return nil
	}
	return claims
}
