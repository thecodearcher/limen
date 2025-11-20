package aegis

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type SessionManager struct {
	store      SessionStore
	config     *SessionConfig
	core       *AegisCore
	strategies map[SessionStrategyType]SessionStrategy
}

const temporaryAuthKey = "temp_auth"

func newSessionManager(core *AegisCore) *SessionManager {
	return &SessionManager{
		store:  determineStore(core.Session, core),
		config: core.Session,
		core:   core,
	}
}

func (m *SessionManager) determineStrategy(config SessionStrategyType) SessionStrategy {
	switch config {
	case SessionStrategyJWT:
		return NewJWTStrategy(m.core, m.config)
	case SessionStrategyServerSide:
		fallthrough
	default:
		return NewServerSideStrategy(m.store, m.config)
	}
}

func (m *SessionManager) CreateSession(ctx context.Context, request *http.Request, authResult *AuthenticationResult) (*SessionResult, error) {
	strategy := m.determineStrategyForRequest(request)
	temporaryAuth := len(authResult.PendingActions) > 0
	result, err := strategy.Create(ctx, authResult.User, temporaryAuth)
	if err != nil {
		return nil, err
	}

	if !strategy.IsStateful() {
		return result, nil
	}

	session := &Session{
		Token:      result.Token,
		UserID:     authResult.User.ID,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(m.config.Duration),
		LastAccess: time.Now(),
		Metadata: map[string]any{
			"ip_address": m.config.IPAddressExtractor(request),
			"user_agent": m.config.UserAgentExtractor(request),
		},
	}

	if temporaryAuth {
		session.Metadata[temporaryAuthKey] = true
		session.ExpiresAt = time.Now().Add(m.config.TemporaryAuthDuration)
	}

	if err := m.store.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return result, nil
}

func (m *SessionManager) ValidateSession(ctx context.Context, request *http.Request) (*SessionValidateResult, error) {
	strategy := m.determineStrategyForRequest(request)

	validateResult, err := strategy.Validate(ctx, request)
	if err != nil {
		return nil, err
	}

	user, err := m.core.DBAction.FindUserByID(ctx, validateResult.UserID)
	if err != nil {
		return nil, err
	}

	return &SessionValidateResult{
		UserID:   validateResult.UserID,
		User:     user,
		Session:  validateResult.Session,
		Metadata: validateResult.Metadata,
	}, nil
}

func (m *SessionManager) RefreshSession(ctx context.Context, request *http.Request) (*SessionRefreshResult, error) {
	strategy := m.determineStrategyForRequest(request)
	result, err := strategy.Refresh(ctx, request)
	if err != nil {
		return nil, err
	}

	if result.ShouldStore {
		session := &Session{
			Token:      result.Token,
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
func (m *SessionManager) Revoke(ctx context.Context, request *http.Request, sessionID string) error {
	strategy := m.determineStrategyForRequest(request)
	if !strategy.IsStateful() {
		return fmt.Errorf("cannot revoke stateless session")
	}
	return m.store.Delete(ctx, sessionID)
}

func (m *SessionManager) RevokeAll(ctx context.Context, request *http.Request, userID string) error {
	strategy := m.determineStrategyForRequest(request)
	if !strategy.IsStateful() {
		return fmt.Errorf("cannot revoke stateless session")
	}
	return m.store.DeleteByUserID(ctx, userID)
}

func (m *SessionManager) determineTokenModeFromRequest(request *http.Request) SessionStrategyType {
	transport := request.Header.Get("X-Session-Transport")

	switch transport {
	case "jwt":
		return SessionStrategyJWT
	case "cookie":
		return SessionStrategyServerSide
	case "hybrid":
		return SessionStrategyHybrid
	default:
		return ""
	}
}

func (m *SessionManager) determineStrategyForRequest(request *http.Request) SessionStrategy {
	strategy := m.determineTokenModeFromRequest(request)
	if strategy == "" {
		strategy = m.config.Strategy
	}

	return m.determineStrategy(strategy)
}

func determineStore(config *SessionConfig, core *AegisCore) SessionStore {
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
