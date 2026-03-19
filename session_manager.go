package limen

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// opaqueSessionManager implements SessionManager using opaque tokens.
type opaqueSessionManager struct {
	store      SessionStore
	config     *sessionConfig
	core       *LimenCore
	cookies    *CookieManager
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

func (m *opaqueSessionManager) CreateSession(ctx context.Context, request *http.Request, authResult *AuthenticationResult, shortSession bool) (*SessionResult, error) {
	duration := m.config.Duration
	if m.config.ShortSessionDuration > 0 && shortSession {
		duration = m.config.ShortSessionDuration
	}

	token := generateCryptoSecureRandomString()
	clock := time.Now()
	expiresAt := clock.Add(duration)
	if err := m.storeSession(ctx, request, authResult.User.ID, token, clock, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	result := &SessionResult{
		Token:        token,
		Cookie:       m.createSessionCookie(token, expiresAt),
		ShortSession: &shortSession,
	}

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
		m.store.Delete(ctx, token)
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
	return m.store.Delete(ctx, token)
}

func (m *opaqueSessionManager) RevokeAllSessions(ctx context.Context, userID any) error {
	return m.store.DeleteByUserID(ctx, userID)
}

func (m *opaqueSessionManager) ListSessions(ctx context.Context, userID any) ([]Session, error) {
	return m.store.ListByUserID(ctx, userID)
}

func (m *opaqueSessionManager) extendSessionExpiration(ctx context.Context, session *Session) (*SessionResult, error) {
	policy := m.resolveSessionPolicy(session)
	now := time.Now()

	session.ExpiresAt = now.Add(policy.Duration)
	session.LastAccess = now

	if err := m.store.Set(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &SessionResult{
		Token:  session.Token,
		Cookie: m.createSessionCookie(session.Token, session.ExpiresAt),
	}, nil
}

func (m *opaqueSessionManager) createSessionCookie(token string, expiresAt time.Time) *http.Cookie {
	if m.cookies == nil {
		return nil
	}
	return m.cookies.NewCookie(m.cookieName, token, int(time.Until(expiresAt).Seconds()))
}

func (m *opaqueSessionManager) storeSession(ctx context.Context, request *http.Request, userID any, token string, clock, expiresAt time.Time) error {
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

	return m.store.Set(ctx, session)
}

func (m *opaqueSessionManager) extractToken(request *http.Request) (string, error) {
	if m.cookies != nil {
		if val, err := m.cookies.Get(request, m.cookieName); err == nil {
			token := strings.TrimSpace(val)
			if token != "" {
				return token, nil
			}
		}
	}

	if m.config.BearerEnabled {
		authHeader := request.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token := strings.TrimSpace(parts[1])
				if token != "" {
					return token, nil
				}
			}
		}
	}

	return "", ErrSessionNotFound
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
