package aegis

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type SessionManager struct {
	store        SessionStore
	config       *sessionConfig
	core         *AegisCore
	strategies   map[SessionStrategyType]SessionStrategy
	cookieConfig *cookieConfig
}

const temporaryAuthKey = "temp_auth"

func newSessionManager(core *AegisCore, config *sessionConfig) *SessionManager {
	return &SessionManager{
		store:      determineStore(config, core),
		config:     config,
		core:       core,
		strategies: make(map[SessionStrategyType]SessionStrategy),
	}
}

func (m *SessionManager) determineStrategy(config SessionStrategyType) SessionStrategy {
	if strategy, ok := m.strategies[config]; ok {
		return strategy
	}

	return NewOpaqueTokenStrategy(m.store, m.config, m.cookieConfig)
}

func (m *SessionManager) RegisterStrategy(strategyType SessionStrategyType, strategy SessionStrategy) {
	m.strategies[strategyType] = strategy
}

func (m *SessionManager) CreateSession(ctx context.Context, request *http.Request, authResult *AuthenticationResult) (*SessionResult, error) {
	strategy := m.determineStrategyForRequest(request)
	temporaryAuth := len(authResult.PendingActions) > 0
	options := &SessionCreateOptions{
		Duration:      m.config.Duration,
		TemporaryAuth: temporaryAuth,
	}

	if temporaryAuth {
		options.Duration = m.config.TemporaryAuthDuration
	}

	createResult, err := strategy.Create(ctx, authResult.User, options)
	if err != nil {
		return nil, err
	}
	sessionOptions := createResult.SessionOptions

	if sessionOptions == nil {
		sessionOptions = &SessionOptions{
			Duration: options.Duration,
		}
	}

	if createResult.SessionValue != "" {
		duration := sessionOptions.Duration
		if duration == 0 {
			duration = options.Duration
		}
		if err := m.storeSession(ctx, request, authResult.User.ID, createResult.SessionValue, duration, temporaryAuth); err != nil {
			return nil, fmt.Errorf("failed to store session: %w", err)
		}
	}

	deliveryMethod := m.determineDeliveryMethod(request)
	result := &SessionResult{
		Token:               createResult.Token,
		TokenDeliveryMethod: deliveryMethod,
	}

	if deliveryMethod == TokenDeliveryCookie {
		result.Cookie = m.createSessionCookie(result.Token, time.Now().Add(m.config.Duration))
	}

	return result, nil
}

func (m *SessionManager) ValidateSession(ctx context.Context, request *http.Request) (*AegisSession, error) {
	strategy := m.determineStrategyForRequest(request)

	validateResult, err := strategy.Validate(ctx, request)
	if err != nil {
		return nil, err
	}

	user, err := m.findUserFromValidSession(ctx, validateResult)
	if err != nil {
		return nil, err
	}

	result := &AegisSession{
		User:    user,
		Session: validateResult.Session,
	}

	if strategy.SupportsExpirationExtension() && validateResult.Session.ShouldExtendExpiration(m.config.Duration, m.config.UpdateAge) {
		extensionResult, err := m.extendSessionExpiration(ctx, strategy, validateResult.Session)
		if err != nil {
			return nil, err
		}
		result.SessionExtensionResult = extensionResult
	}

	return result, nil
}

// Revoke revokes a session
func (m *SessionManager) Revoke(ctx context.Context, request *http.Request, sessionID string) error {
	strategy := m.determineStrategyForRequest(request)
	if !strategy.IsStateful() {
		return nil
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

func (m *SessionManager) RevokeAllCookies(responseWriter http.ResponseWriter) {
	if m.cookieConfig == nil {
		return
	}
	sessionCookie := &http.Cookie{
		Name:     m.cookieConfig.name,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: m.cookieConfig.httpOnly,
		Secure:   m.cookieConfig.secure,
		SameSite: m.cookieConfig.sameSite,
		Path:     m.cookieConfig.path,
	}

	http.SetCookie(responseWriter, sessionCookie)
}

func (m *SessionManager) determineStrategyForRequest(_ *http.Request) SessionStrategy {
	return m.determineStrategy(m.config.Strategy)
}

// extendSessionExpiration extends the session expiration time if the strategy
// supports sliding window extension and the session is due for extension i.e UpdateAge is reached.
// Returns a SessionExtensionResult if the session was extended, nil otherwise.
func (m *SessionManager) extendSessionExpiration(ctx context.Context, strategy SessionStrategy, session *Session) (*SessionExtensionResult, error) {
	if !strategy.SupportsExpirationExtension() {
		return nil, nil
	}

	if !session.ShouldExtendExpiration(m.config.Duration, m.config.UpdateAge) {
		return nil, nil
	}

	session.ExpiresAt = time.Now().Add(m.config.Duration)
	session.LastAccess = time.Now()

	if err := m.store.Update(ctx, session.ID, &Session{
		ExpiresAt:  session.ExpiresAt,
		LastAccess: session.LastAccess,
	}); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &SessionExtensionResult{
		Token:  session.Token,
		Cookie: m.createSessionCookie(session.Token, session.ExpiresAt),
	}, nil
}

func (m *SessionManager) createSessionCookie(token string, expiresAt time.Time) *http.Cookie {
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

func (m *SessionManager) findUserFromValidSession(ctx context.Context, result *SessionValidateResult) (*User, error) {
	if result.User != nil {
		return result.User, nil
	}

	return m.core.DBAction.FindUserByID(ctx, result.UserID)
}

func (m *SessionManager) storeSession(ctx context.Context, request *http.Request, userID any, value string, duration time.Duration, temporaryAuth bool) error {
	session := &Session{
		Token:      value,
		UserID:     userID,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(duration),
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

	return m.store.Create(ctx, session)
}

func (m *SessionManager) determineDeliveryMethod(request *http.Request) TokenDeliveryMethod {
	// Check for explicit override in request header
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
