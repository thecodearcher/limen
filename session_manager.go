package aegis

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OpaqueSessionManager implements SessionManager using opaque tokens.
type OpaqueSessionManager struct {
	store      SessionStore
	config     *sessionConfig
	core       *AegisCore
	cookies    *CookieManager
	cookieName string
}

func newOpaqueSessionManager(core *AegisCore, config *sessionConfig) *OpaqueSessionManager {
	var cookieName string
	if core.cookies != nil && core.config.HTTP.cookieConfig != nil {
		cookieName = core.config.HTTP.cookieConfig.sessionCookieName
	}
	return &OpaqueSessionManager{
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
func (m *OpaqueSessionManager) resolveSessionPolicy(session *Session) sessionPolicy {
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

func (m *OpaqueSessionManager) CreateSession(ctx context.Context, request *http.Request, authResult *AuthenticationResult, shortSession bool) (*SessionResult, error) {
	duration := m.config.Duration
	if m.config.ShortSessionDuration > 0 && shortSession {
		duration = m.config.ShortSessionDuration
	}

	token := generateCryptoSecureRandomString()
	expiresAt := time.Now().Add(duration)
	if err := m.storeSession(ctx, request, authResult.User.ID, token, expiresAt); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	result := &SessionResult{
		Token:        token,
		Cookie:       m.createSessionCookie(token, expiresAt),
		ShortSession: &shortSession,
	}

	return result, nil
}

func (m *OpaqueSessionManager) ValidateSession(ctx context.Context, request *http.Request) (*ValidatedSession, error) {
	token, err := m.extractToken(request)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	session, err := m.store.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	policy := m.resolveSessionPolicy(session)
	if session.IsExpired(policy.IdleTimeout) {
		m.store.DeleteByToken(ctx, token)
		return nil, ErrSessionExpired
	}

	if m.config.ActivityCheckInterval > 0 && time.Now().After(session.LastAccess.Add(m.config.ActivityCheckInterval)) {
		now := time.Now()
		if err := m.store.UpdateByToken(ctx, token, &SessionUpdates{LastAccess: &now}); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}
		session.LastAccess = now
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
		refreshed, err := m.extendSessionExpiration(ctx, request, session)
		if err != nil {
			return nil, err
		}
		result.Refreshed = refreshed
	}

	return result, nil
}

func (m *OpaqueSessionManager) RevokeSession(ctx context.Context, token string) error {
	return m.store.DeleteByToken(ctx, token)
}

func (m *OpaqueSessionManager) RevokeAllSessions(ctx context.Context, userID any) error {
	return m.store.DeleteByUserID(ctx, userID)
}

func (m *OpaqueSessionManager) extendSessionExpiration(ctx context.Context, request *http.Request, session *Session) (*SessionResult, error) {
	policy := m.resolveSessionPolicy(session)
	now := time.Now()
	newExpiresAt := now.Add(policy.Duration)

	if err := m.store.UpdateByToken(ctx, session.Token, &SessionUpdates{
		ExpiresAt:  &newExpiresAt,
		LastAccess: &now,
	}); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	session.ExpiresAt = newExpiresAt
	session.LastAccess = now

	return &SessionResult{
		Token:  session.Token,
		Cookie: m.createSessionCookie(session.Token, session.ExpiresAt),
	}, nil
}

func (m *OpaqueSessionManager) createSessionCookie(token string, expiresAt time.Time) *http.Cookie {
	if m.cookies == nil {
		return nil
	}
	return m.cookies.NewCookie(m.cookieName, token, int(time.Until(expiresAt).Seconds()))
}

func (m *OpaqueSessionManager) storeSession(ctx context.Context, request *http.Request, userID any, token string, expiresAt time.Time) error {
	now := time.Now()
	session := &Session{
		Token:      token,
		UserID:     userID,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		LastAccess: now,
		Metadata: map[string]any{
			"ip_address": m.config.IPAddressExtractor(request),
			"user_agent": m.config.UserAgentExtractor(request),
		},
	}

	return m.store.Create(ctx, session)
}

func (m *OpaqueSessionManager) extractToken(request *http.Request) (string, error) {
	if m.cookies != nil {
		if val, err := m.cookies.Get(request, m.cookieName); err == nil {
			token := strings.TrimSpace(val)
			if token != "" {
				return token, nil
			}
		}
	}

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

	return "", ErrSessionNotFound
}

func determineStore(config *sessionConfig, core *AegisCore) SessionStore {
	if config.CustomStore != nil {
		return config.CustomStore
	}

	switch config.StoreType {
	case SessionStoreTypeDatabase:
		return NewDatabaseSessionStore(core)
	case SessionStoreTypeMemory:
		fallthrough
	default:
		return NewMemorySessionStore()
	}
}
