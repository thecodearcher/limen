package sessionjwt

import (
	"context"
	"encoding/json"
	"fmt"
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

	signed, jti, err := p.GenerateAccessToken(auth.User, nil)
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
	rt, err := p.CreateRefreshToken(ctx, auth.User.ID, jti, family, expiresAt, nil)
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
	switch v := resolved.(type) {
	case *limen.User:
		user = v
	case map[string]any:
		user = m.plugin.core.Schema.User.FromStorage(v).(*limen.User)
	default:
		if m.plugin.config.refreshUser {
			user, err = m.plugin.core.DBAction.FindUserByID(ctx, v)
			if err != nil {
				return nil, limen.ErrSessionNotFound
			}
		} else {
			user, err = m.plugin.claimsToUser(claims, v)
			if err != nil {
				return nil, err
			}
		}
	}

	return &limen.ValidatedSession{
		User:    user,
		Session: m.plugin.claimsToSession(claims, tokenString, user.ID),
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

func (m *jwtSessionManager) ListSessions(ctx context.Context, userID any) ([]limen.Session, error) {
	tokens, err := m.plugin.findRefreshTokensByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessions := make([]limen.Session, 0, len(tokens))
	for _, rt := range tokens {
		sessions = append(sessions, limen.Session{
			ID:        rt.ID,
			Token:     rt.Token,
			UserID:    rt.UserID,
			CreatedAt: rt.CreatedAt,
			ExpiresAt: rt.ExpiresAt,
		})
	}
	return sessions, nil
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

func (m *jwtSessionManager) GetSessionData(_ context.Context, session *limen.Session, field limen.SchemaField) (any, error) {
	return m.plugin.core.Schema.Session.SessionData(session, field)
}

func (m *jwtSessionManager) UpdateSession(ctx context.Context, session *limen.Session, data map[limen.SchemaField]any) (*limen.SessionResult, error) {
	p := m.plugin

	claims := p.parseAccessTokenLenient(session.Token)
	if claims == nil {
		return nil, ErrInvalidAccessToken
	}

	sessionData, err := p.mergeSessionData(claims.Session, data)
	if err != nil {
		return nil, err
	}

	signed, newClaims, err := p.reissueAccessToken(claims, sessionData)
	if err != nil {
		return nil, err
	}

	if p.config.refreshTokenEnabled {
		if err := p.updateRefreshTokenSessionData(ctx, claims.ID, newClaims.ID, sessionData); err != nil {
			return nil, err
		}
	}

	*session = *p.claimsToSession(newClaims, signed, session.UserID)
	return p.buildSessionResult(signed, nil), nil
}

// UpdateSessions is a no-op: matching sessions would require decoding every
// refresh token's session_data, and stale values are already guarded by
// consumers re-checking the referenced resources.
func (m *jwtSessionManager) UpdateSessions(context.Context, map[limen.SchemaField]any, map[limen.SchemaField]any) error {
	return nil
}

func (p *sessionJWTPlugin) updateRefreshTokenSessionData(ctx context.Context, oldJTI, newJTI string, sessionData map[string]any) error {
	encoded, err := encodeSessionData(sessionData)
	if err != nil {
		return err
	}

	result, err := p.core.UpdateWithResult(ctx, p.refreshTokenSchema, map[limen.SchemaField]any{
		RefreshTokenSchemaJWTIDField:       newJTI,
		RefreshTokenSchemaSessionDataField: encoded,
	}, []limen.Where{
		limen.Eq(p.refreshTokenSchema.GetJWTIDField(), oldJTI),
	})
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return ErrStaleAccessToken
	}
	return nil
}

// mergeSessionData applies the update to the token's session data bag, keyed by
// session column so claimsToSession can overlay it on the stored session.
func (p *sessionJWTPlugin) mergeSessionData(current map[string]any, data map[limen.SchemaField]any) (map[string]any, error) {
	schema := p.core.Schema.Session
	merged := maps.Clone(current)
	if merged == nil {
		merged = make(map[string]any, len(data))
	}
	for field, value := range data {
		column := schema.GetField(field)
		if column == "" {
			return nil, fmt.Errorf("%w: %q", limen.ErrUnknownSessionField, field)
		}
		if sv, ok := value.(limen.SessionValue); ok {
			value = sv.Client
		}
		if value == nil {
			delete(merged, column)
			continue
		}
		merged[column] = value
	}
	if len(merged) == 0 {
		return nil, nil
	}
	return merged, nil
}

func encodeSessionData(sessionData map[string]any) (any, error) {
	if sessionData == nil {
		return nil, nil
	}
	b, err := json.Marshal(sessionData)
	if err != nil {
		return nil, fmt.Errorf("session-jwt: failed to encode session data: %w", err)
	}
	return string(b), nil
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

	schema := p.core.Schema.Session
	// raw carries the jti in place of the signed token to keep the snapshot small.
	data := map[string]any{
		schema.GetIDField():         claims.ID,
		schema.GetTokenField():      claims.ID,
		schema.GetUserIDField():     userID,
		schema.GetCreatedAtField():  issuedAt,
		schema.GetExpiresAtField():  expiresAt,
		schema.GetLastAccessField(): time.Now(),
		schema.GetMetadataField():   claimsMetadata(claims),
	}
	maps.Copy(data, claims.Session)

	session := schema.FromStorage(data).(*limen.Session)
	session.Token = rawToken
	return session
}

func (p *sessionJWTPlugin) claimsToUser(claims *LimenClaims, subject any) (*limen.User, error) {
	schema := p.core.Schema.User
	raw := map[string]any{}
	if p.config.userFromClaims != nil {
		maps.Copy(raw, p.config.userFromClaims(claims))
	}

	idField := schema.GetIDField()
	if s, ok := subject.(string); ok && p.core.IsPublicID(schema, s) {
		decoded, err := p.core.DecodePublicID(schema, s)
		if err != nil {
			return nil, ErrInvalidAccessToken
		}
		raw[p.core.PublicIDColumn(schema)] = decoded
	} else if _, ok := raw[idField]; !ok {
		raw[idField] = subject
	}

	raw[schema.GetEmailField()] = claims.Email
	raw[schema.GetEmailVerifiedAtField()] = claims.EmailVerifiedAt

	user := schema.FromStorage(raw).(*limen.User)
	if isEmptyUserID(user.ID) {
		return nil, ErrMissingUserID
	}
	return user, nil
}

func isEmptyUserID(id any) bool {
	if id == nil {
		return true
	}
	if s, ok := id.(string); ok {
		return s == ""
	}
	return false
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
