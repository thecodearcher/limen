package sessionjwt

import (
	"context"
	"maps"
	"net/http"
	"time"

	"github.com/thecodearcher/limen"
)

// jwtSessionManager implements limen.SessionManager using JWTs as access
// tokens and opaque refresh tokens stored in the database.
type jwtSessionManager struct {
	plugin *sessionJWTPlugin
}

func (m *jwtSessionManager) CreateSession(ctx context.Context, r *http.Request, auth *limen.AuthenticationResult, shortSession bool) (*limen.SessionResult, error) {
	p := m.plugin

	signed, jti, err := p.GenerateAccessToken(auth.User)
	if err != nil {
		return nil, err
	}

	if !p.config.refreshTokenEnabled {
		return p.buildSessionResult(signed, nil), nil
	}

	var expiresAt *time.Time
	if shortSession {
		shortDuration := min(p.config.refreshTokenDuration, 24*time.Hour)
		exp := time.Now().Add(shortDuration)
		expiresAt = &exp
	}

	family := generateJTI()
	rt, err := p.CreateRefreshToken(ctx, auth.User.ID, jti, family, expiresAt)
	if err != nil {
		return nil, err
	}

	return p.buildSessionResult(signed, rt), nil
}

func (m *jwtSessionManager) ValidateSession(ctx context.Context, r *http.Request) (*limen.ValidatedSession, error) {
	tokenString, err := m.plugin.extractToken(r)
	if err != nil {
		return nil, limen.ErrSessionNotFound
	}

	claims, err := m.plugin.VerifyAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	if blocked, _ := m.plugin.IsBlacklisted(ctx, claims.ID); blocked {
		return nil, ErrTokenRevoked
	}

	resolved, err := m.plugin.config.subjectResolver(claims.Subject)
	if err != nil {
		return nil, err
	}

	var user *limen.User
	if m.plugin.config.refreshUser {
		user, err = m.plugin.fetchUser(ctx, resolved)
		if err != nil {
			return nil, limen.ErrSessionNotFound
		}
	} else {
		user = m.plugin.claimsToUser(claims, resolved)
	}

	return &limen.ValidatedSession{
		User:    user,
		Session: m.plugin.claimsToSession(claims, tokenString, resolved),
	}, nil
}

func (m *jwtSessionManager) RevokeSession(ctx context.Context, token string) error {
	p := m.plugin

	claims := p.parseAccessTokenLenient(token)
	if claims == nil || claims.ID == "" {
		return nil
	}

	_ = p.deleteRefreshTokensByJTI(ctx, claims.ID)
	if p.config.blacklistEnabled && claims.ExpiresAt != nil {
		_ = p.AddToBlacklist(ctx, claims.ID, claims.ExpiresAt.Time)
	}
	return nil
}

func (m *jwtSessionManager) RevokeAllSessions(ctx context.Context, userID any) error {
	p := m.plugin

	if p.config.blacklistEnabled {
		tokens, err := p.findRefreshTokensByUserID(ctx, userID)
		if err == nil {
			for _, rt := range tokens {
				_ = p.AddToBlacklist(ctx, rt.JWTID, rt.ExpiresAt)
			}
		}
	}

	return p.DeleteRefreshTokensByUserID(ctx, userID)
}

func (p *sessionJWTPlugin) fetchUser(ctx context.Context, resolved any) (*limen.User, error) {
	if user, ok := resolved.(*limen.User); ok {
		return user, nil
	}
	return p.core.DBAction.FindUserByID(ctx, resolved)
}

func (p *sessionJWTPlugin) buildSessionResult(jwtString string, rt *RefreshToken) *limen.SessionResult {
	result := &limen.SessionResult{Token: jwtString}
	if rt != nil {
		result.RefreshToken = encodeRefreshTokenValue(rt.Token, rt.Family)
	}
	return result
}

func (p *sessionJWTPlugin) deleteRefreshTokensByJTI(ctx context.Context, jwtID string) error {
	return p.core.Delete(ctx, p.refreshTokenSchema, []limen.Where{
		limen.Eq(p.refreshTokenSchema.GetJWTIDField(), jwtID),
	})
}

func (p *sessionJWTPlugin) findRefreshTokensByUserID(ctx context.Context, userID any) ([]*RefreshToken, error) {
	models, err := p.core.FindMany(ctx, p.refreshTokenSchema, []limen.Where{
		limen.Eq(p.refreshTokenSchema.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}
	tokens := make([]*RefreshToken, 0, len(models))
	for _, m := range models {
		tokens = append(tokens, m.(*RefreshToken))
	}
	return tokens, nil
}

func (p *sessionJWTPlugin) updateRefreshTokenJWTID(ctx context.Context, token string, jwtID string) error {
	rt := &RefreshToken{Token: token, JWTID: jwtID}
	return p.core.Update(ctx, p.refreshTokenSchema, rt, []limen.Where{
		limen.Eq(p.refreshTokenSchema.GetTokenField(), token),
	})
}

func (p *sessionJWTPlugin) claimsToSession(claims *LimenClaims, rawToken string, userID any) *limen.Session {
	var issuedAt, expiresAt time.Time
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time
	}
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return &limen.Session{
		ID:         claims.ID,
		Token:      rawToken,
		UserID:     userID,
		CreatedAt:  issuedAt,
		ExpiresAt:  expiresAt,
		LastAccess: time.Now(),
		Metadata:   claimsMetadata(claims),
	}
}

func (p *sessionJWTPlugin) claimsToUser(claims *LimenClaims, userID any) *limen.User {
	var emailVerifiedAt *time.Time
	if claims.EmailVerified {
		now := time.Now()
		emailVerifiedAt = &now
	}

	return &limen.User{
		ID:              userID,
		Email:           claims.Email,
		EmailVerifiedAt: emailVerifiedAt,
	}
}

func claimsMetadata(claims *LimenClaims) map[string]any {
	m := make(map[string]any)
	if claims.Issuer != "" {
		m["iss"] = claims.Issuer
	}
	if len(claims.Audience) > 0 {
		m["aud"] = []string(claims.Audience)
	}
	maps.Copy(m, claims.Custom)
	return m
}
