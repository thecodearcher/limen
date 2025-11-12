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

func NewSessionManager(core *aegis.AegisCore) *SessionManager {
	return &SessionManager{
		store:  determineStore(core.Session, core),
		config: core.Session,
		core:   core,
	}
}

func (m *SessionManager) determineStrategy(config aegis.SessionStrategyType) aegis.SessionStrategy {
	switch config {
	case aegis.SessionStrategyJWT:
		return NewJWTStrategy(m.core, m.config)
	case aegis.SessionStrategyServerSide:
		fallthrough
	default:
		return NewServerSideStrategy(m.store, m.config)
	}
}

func (m *SessionManager) CreateSession(ctx context.Context, request *http.Request, authResult *aegis.AuthenticationResult) (*aegis.SessionResult, error) {
	strategy := m.determineStrategyForRequest(request)
	temporaryAuth := len(authResult.PendingActions) > 0
	result, err := strategy.Create(ctx, authResult.User, temporaryAuth)
	if err != nil {
		return nil, err
	}

	if !strategy.IsStateful() {
		return result, nil
	}

	session := &aegis.Session{
		ID:         result.Token,
		UserID:     authResult.User.ID,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(m.config.Duration),
		LastAccess: time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	if temporaryAuth {
		session.Metadata["temp_auth"] = true
		session.ExpiresAt = time.Now().Add(time.Duration(5 * time.Minute))
	}

	if err := m.store.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return result, nil
}

func (m *SessionManager) ValidateSession(ctx context.Context, request *http.Request) (*aegis.SessionValidateResult, error) {
	strategy := m.determineStrategyForRequest(request)

	return strategy.Validate(ctx, request)
}

func (m *SessionManager) RefreshSession(ctx context.Context, request *http.Request) (*aegis.SessionRefreshResult, error) {
	strategy := m.determineStrategyForRequest(request)
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
	case "hybrid":
		return aegis.SessionStrategyHybrid
	default:
		return ""
	}
}

func (m *SessionManager) determineStrategyForRequest(request *http.Request) aegis.SessionStrategy {
	strategy := m.determineTokenModeFromRequest(request)
	if strategy == "" {
		strategy = m.config.Strategy
	}

	return m.determineStrategy(strategy)
}

func determineStore(config *aegis.SessionConfig, core *aegis.AegisCore) aegis.SessionStore {
	if config.CustomStore != nil {
		return config.CustomStore
	}

	switch config.StoreType {
	case aegis.SessionStoreTypeDatabase:
		return NewDatabaseSessionStore(core)
	case aegis.SessionStoreTypeMemory:
		fallthrough
	default:
		return NewMemorySessionStore()
	}
}
