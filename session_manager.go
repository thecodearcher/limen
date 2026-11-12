package limen

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"
)

// opaqueSessionManager implements SessionManager using opaque tokens.
type opaqueSessionManager struct {
	store      SessionStore
	config     *sessionConfig
	core       *LimenCore
	cookies    *cookieManager
	cookieName string
}

func newOpaqueSessionManager(core *LimenCore, config *sessionConfig) *opaqueSessionManager {
	var cookieName string
	if core.cookies != nil && core.config.HTTP.cookieConfig != nil {
		cookieName = core.config.HTTP.cookieConfig.sessionCookieName
	}
	return &opaqueSessionManager{
		store:      determineStore(config, core),
		config:     config,
		core:       core,
		cookies:    core.cookies,
		cookieName: cookieName,
	}
}

type sessionPolicy struct {
	Duration    time.Duration
	IdleTimeout time.Duration
	UpdateAge   time.Duration
}

// resolveSessionPolicy returns the effective policy for this session. Short sessions
// (ExpiresAt - CreatedAt < global Duration) are not extended so they stay at their fixed TTL.
func (m *opaqueSessionManager) resolveSessionPolicy(session *Session) sessionPolicy {
	p := sessionPolicy{
		Duration:    m.config.Duration,
		IdleTimeout: m.config.IdleTimeout,
		UpdateAge:   m.config.UpdateAge,
	}

	if session.ExpiresAt.Sub(session.CreatedAt) < m.config.Duration {
		p.UpdateAge = 0
	}

	return p
}

func (m *opaqueSessionManager) CreateSession(ctx context.Context, request *http.Request, authResult *AuthenticationResult, opts ...SessionCreateOption) (*SessionResult, error) {
	o := ResolveSessionCreateOptions(opts...)

	duration := m.config.Duration
	switch {
	case o.ExpiresIn > 0:
		duration = o.ExpiresIn
	case o.ShortSession && m.config.ShortSessionDuration > 0:
		duration = m.config.ShortSessionDuration
	}

	data := unwrapInternalSessionValues(o.Data)
	if _, err := m.core.Schema.Session.sessionColumns(data); err != nil {
		return nil, err
	}

	token := generateCryptoSecureRandomString()
	clock := time.Now()
	expiresAt := clock.Add(duration)
	session, err := m.storeSession(ctx, request, authResult.User.ID, token, clock, expiresAt, data)
	if err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	result := m.newSessionResult(token, expiresAt)
	result.ShortSession = &o.ShortSession
	result.Session = session
	return result, nil
}

func (m *opaqueSessionManager) ValidateSession(ctx context.Context, request *http.Request) (*ValidatedSession, error) {
	token, err := m.extractToken(request)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	session, err := m.store.Get(ctx, token)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	policy := m.resolveSessionPolicy(session)
	if session.IsExpired(policy.IdleTimeout) {
		_ = m.store.Delete(ctx, token)
		return nil, ErrSessionExpired
	}

	if m.config.ActivityCheckInterval > 0 && time.Now().After(session.LastAccess.Add(m.config.ActivityCheckInterval)) {
		session.LastAccess = time.Now()
		if err := m.store.Set(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}
	}

	user, err := m.core.DBAction.FindUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	result := &ValidatedSession{
		User:    user,
		Session: session,
	}

	if session.ShouldExtendExpiration(policy.Duration, policy.UpdateAge) {
		refreshed, err := m.extendSessionExpiration(ctx, session)
		if err != nil {
			return nil, err
		}
		result.Refreshed = refreshed
	}

	return result, nil
}

func (m *opaqueSessionManager) RevokeSession(ctx context.Context, token string) error {
	if stored, ok := m.core.VerifySignedValue(token); ok {
		token = stored
	}
	return m.store.Delete(ctx, token)
}

func (m *opaqueSessionManager) RevokeAllSessions(ctx context.Context, userID any) error {
	return m.store.DeleteByUserID(ctx, userID)
}

func (m *opaqueSessionManager) ListSessions(ctx context.Context, userID any) ([]Session, error) {
	return m.store.ListByUserID(ctx, userID)
}

func (m *opaqueSessionManager) GetSessionData(_ context.Context, session *Session, field SchemaField) (any, error) {
	return m.core.Schema.Session.SessionData(session, field)
}

func (m *opaqueSessionManager) UpdateSession(ctx context.Context, session *Session, data map[SchemaField]any) (*SessionResult, error) {
	payload := unwrapInternalSessionValues(data)
	if _, err := m.core.Schema.Session.sessionColumns(payload); err != nil {
		return nil, err
	}
	payload[SessionSchemaTokenField] = session.Token
	payload[SchemaIDField] = session.ID

	if err := m.store.Set(ctx, payload); err != nil {
		return nil, err
	}

	fresh, err := m.store.Get(ctx, session.Token)
	if err != nil {
		return nil, err
	}
	*session = *fresh
	return nil, nil
}

func (m *opaqueSessionManager) UpdateSessions(ctx context.Context, data map[SchemaField]any, match map[SchemaField]any) error {
	bulk, ok := m.store.(SessionBulkUpdater)
	if !ok {
		return nil
	}
	return bulk.UpdateSessions(ctx, unwrapInternalSessionValues(data), unwrapInternalSessionValues(match))
}

func unwrapInternalSessionValues(data map[SchemaField]any) map[SchemaField]any {
	unwrapped := make(map[SchemaField]any, len(data))
	for field, value := range data {
		if sv, ok := value.(SessionValue); ok {
			value = sv.Internal
		}
		unwrapped[field] = value
	}
	return unwrapped
}

func (m *opaqueSessionManager) extendSessionExpiration(ctx context.Context, session *Session) (*SessionResult, error) {
	policy := m.resolveSessionPolicy(session)
	now := time.Now()

	session.ExpiresAt = now.Add(policy.Duration)
	session.LastAccess = now

	if err := m.store.Set(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return m.newSessionResult(session.Token, session.ExpiresAt), nil
}

func (m *opaqueSessionManager) newSessionResult(token string, expiresAt time.Time) *SessionResult {
	signed := m.core.SignValue(token)
	result := &SessionResult{Token: signed}
	if m.cookies != nil {
		result.Cookie = m.cookies.NewCookie(m.cookieName, signed, int(time.Until(expiresAt).Seconds()))
	}
	return result
}

func (m *opaqueSessionManager) storeSession(ctx context.Context, request *http.Request, userID any, token string, clock, expiresAt time.Time, data map[SchemaField]any) (*Session, error) {
	session := &Session{
		Token:      token,
		UserID:     userID,
		CreatedAt:  clock,
		ExpiresAt:  expiresAt,
		LastAccess: clock,
		Metadata: map[string]any{
			"ip_address": m.config.IPAddressExtractor(request),
			"user_agent": m.config.UserAgentExtractor(request),
		},
	}

	maps.Copy(data, m.core.Schema.Session.storageFields(session))
	if err := m.store.Set(ctx, data); err != nil {
		return nil, err
	}

	session.raw, _ = m.core.Schema.Session.sessionColumns(data)
	return session, nil
}

func (m *opaqueSessionManager) extractToken(request *http.Request) (string, error) {
	token, ok := m.core.VerifySignedValue(m.requestToken(request))
	if !ok {
		return "", ErrSessionNotFound
	}
	return token, nil
}

func (m *opaqueSessionManager) requestToken(request *http.Request) string {
	if m.cookies != nil {
		if val, err := m.cookies.Get(request, m.cookieName); err == nil {
			if token := strings.TrimSpace(val); token != "" {
				return token
			}
		}
	}

	if m.config.BearerEnabled {
		parts := strings.SplitN(request.Header.Get("Authorization"), " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	return ""
}

func determineStore(config *sessionConfig, core *LimenCore) SessionStore {
	if config.CustomStore != nil {
		return config.CustomStore
	}

	if config.StoreType == StoreTypeDatabase {
		return newDatabaseSessionStore(core)
	}
	return newSecondarySessionStore(core)
}
