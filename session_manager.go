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
	store        SessionStore
	config       *sessionConfig
	core         *AegisCore
	cookieConfig *cookieConfig
}

func newOpaqueSessionManager(core *AegisCore, config *sessionConfig, cookieConfig *cookieConfig) *OpaqueSessionManager {
	return &OpaqueSessionManager{
		store:        determineStore(config, core),
		config:       config,
		core:         core,
		cookieConfig: cookieConfig,
	}
}

func (m *OpaqueSessionManager) CreateSession(ctx context.Context, request *http.Request, authResult *AuthenticationResult) (*SessionResult, error) {
	token := generateCryptoSecureRandomString()

	if err := m.storeSession(ctx, request, authResult.User.ID, token); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	deliveryMethod := m.determineDeliveryMethod(request)
	result := &SessionResult{
		Token:               token,
		TokenDeliveryMethod: deliveryMethod,
	}

	if deliveryMethod == TokenDeliveryCookie {
		result.Cookie = m.createSessionCookie(token, time.Now().Add(m.config.Duration))
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

	if session.IsExpired(m.config.IdleTimeout) {
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

	if session.ShouldExtendExpiration(m.config.Duration, m.config.UpdateAge) {
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
	now := time.Now()
	newExpiresAt := now.Add(m.config.Duration)

	if err := m.store.UpdateByToken(ctx, session.Token, &SessionUpdates{
		ExpiresAt:  &newExpiresAt,
		LastAccess: &now,
	}); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	session.ExpiresAt = newExpiresAt
	session.LastAccess = now

	return &SessionResult{
		Token:               session.Token,
		Cookie:              m.createSessionCookie(session.Token, session.ExpiresAt),
		TokenDeliveryMethod: m.determineDeliveryMethod(request),
	}, nil
}

func (m *OpaqueSessionManager) createSessionCookie(token string, expiresAt time.Time) *http.Cookie {
	if m.cookieConfig == nil {
		return nil
	}
	cookieOptions := m.cookieConfig
	cookie := &http.Cookie{
		Name:        cookieOptions.name,
		Value:       token,
		Path:        cookieOptions.path,
		MaxAge:      int(time.Until(expiresAt).Seconds()),
		HttpOnly:    cookieOptions.httpOnly,
		Secure:      cookieOptions.secure,
		SameSite:    cookieOptions.sameSite,
		Partitioned: cookieOptions.partitioned,
	}

	if cookieOptions.crossSubdomain != nil && cookieOptions.crossSubdomain.enabled {
		cookie.Domain = cookieOptions.crossSubdomain.domain
	}

	return cookie
}

func (m *OpaqueSessionManager) storeSession(ctx context.Context, request *http.Request, userID any, token string) error {
	now := time.Now()
	session := &Session{
		Token:      token,
		UserID:     userID,
		CreatedAt:  now,
		ExpiresAt:  now.Add(m.config.Duration),
		LastAccess: now,
		Metadata: map[string]any{
			"ip_address": m.config.IPAddressExtractor(request),
			"user_agent": m.config.UserAgentExtractor(request),
		},
	}

	return m.store.Create(ctx, session)
}

func (m *OpaqueSessionManager) determineDeliveryMethod(request *http.Request) TokenDeliveryMethod {
	if delivery := request.Header.Get("X-Token-Transport"); delivery != "" {
		switch TokenDeliveryMethod(delivery) {
		case TokenDeliveryCookie, TokenDeliveryHeader:
			return TokenDeliveryMethod(delivery)
		}
	}

	if m.config.TokenDeliveryMethodDetector != nil {
		return m.config.TokenDeliveryMethodDetector(request)
	}

	return m.config.TokenDeliveryMethod
}

func (m *OpaqueSessionManager) extractToken(request *http.Request) (string, error) {
	if m.cookieConfig != nil {
		if cookie, err := request.Cookie(m.cookieConfig.name); err == nil {
			token := strings.TrimSpace(cookie.Value)
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
