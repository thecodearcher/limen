package session

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/thecodearcher/aegis"
)

type SessionManager struct {
	store      aegis.SessionStore
	config     *aegis.SessionConfig
	core       *aegis.AegisCore
	strategies map[aegis.SessionStrategyType]aegis.SessionStrategy
}

func NewSessionManager(store aegis.SessionStore, config *aegis.SessionConfig, core *aegis.AegisCore) *SessionManager {
	return &SessionManager{
		store:  store,
		config: config,
		core:   core,
	}
}

func (m *SessionManager) determineStrategy(config aegis.SessionStrategyType) aegis.SessionStrategy {
	switch config {
	case aegis.SessionStrategyServerSide:
		return NewServerSideStrategy(m.store, m.config)
	case aegis.SessionStrategyJWT:
		return NewJWTStrategy(m.core, m.config)
	default:
		return nil
	}
}

func (m *SessionManager) CreateSession(ctx context.Context, user *aegis.User, request *http.Request) (*aegis.SessionResult, error) {
	strategy := m.determineStrategyFromRequest(request)
	result, err := strategy.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	if !strategy.IsStateful() {
		return result, nil
	}

	session := &aegis.Session{
		ID:         result.Token,
		UserID:     user.ID,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(m.config.Duration),
		LastAccess: time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	if err := m.store.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return result, nil
}

func (m *SessionManager) ValidateSession(ctx context.Context, request *http.Request) (*aegis.SessionValidateResult, error) {
	strategy := m.determineStrategyFromRequest(request)

	return strategy.Validate(ctx, request)
}

func (m *SessionManager) RefreshSession(ctx context.Context, request *http.Request) (*aegis.SessionRefreshResult, error) {
	strategy := m.determineStrategyFromRequest(request)
	result, err := strategy.Refresh(ctx, request)
	if err != nil {
		return nil, err
	}

	if result.ShouldStore {
		session := &aegis.Session{
			ID:         result.Token,
			UserID:     result.UserID,
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().Add(m.config.Duration),
			LastAccess: time.Now(),
			Metadata:   make(map[string]interface{}),
		}

		if err := m.store.Create(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to store session: %w", err)
		}

		if err := m.store.Delete(ctx, result.StaleSessionID); err != nil {
			return nil, fmt.Errorf("failed to delete stale session: %w", err)
		}
	}

	return result, nil
}

// Revoke revokes a session
func (s *ServerSideStrategy) Revoke(ctx context.Context, sessionID string) error {
	if !s.IsStateful() {
		return fmt.Errorf("cannot revoke stateless session")
	}
	return s.store.Delete(ctx, sessionID)
}

func (s *ServerSideStrategy) RevokeAll(ctx context.Context, userID string) error {
	if !s.IsStateful() {
		return fmt.Errorf("cannot revoke stateless session")
	}
	return s.store.DeleteByUserID(ctx, userID)
}

func (m *SessionManager) determineTokenModeFromRequest(request *http.Request) aegis.SessionStrategyType {
	transport := request.Header.Get("X-Session-Transport")

	switch transport {
	case "jwt":
		return aegis.SessionStrategyJWT
	case "cookie":
		return aegis.SessionStrategyServerSide
	default:
		return aegis.SessionStrategyHybrid
	}
}

func (m *SessionManager) determineStrategyFromRequest(request *http.Request) aegis.SessionStrategy {
	strategy := m.determineTokenModeFromRequest(request)
	return m.determineStrategy(strategy)
}

func (m *SessionManager) createCookie(token string) *http.Cookie {
	cookieOptions := m.config.CookieOptions

	return &http.Cookie{
		Name:        cookieOptions.Name,
		Value:       token,
		Path:        cookieOptions.Path,
		MaxAge:      int(m.config.Duration.Seconds()),
		HttpOnly:    cookieOptions.HTTPOnly,
		Secure:      cookieOptions.Secure,
		SameSite:    cookieOptions.SameSite,
		Partitioned: cookieOptions.Partitioned,
	}
}
