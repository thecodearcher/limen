package aegis

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OpaqueTokenStrategy struct {
	store        SessionStore
	config       *sessionConfig
	cookieConfig *cookieConfig
}

func NewOpaqueTokenStrategy(store SessionStore, config *sessionConfig, cookieConfig *cookieConfig) *OpaqueTokenStrategy {
	return &OpaqueTokenStrategy{
		store:        store,
		config:       config,
		cookieConfig: cookieConfig,
	}
}

func (s *OpaqueTokenStrategy) GetName() string {
	return string(SessionStrategyOpaqueToken)
}

func (s *OpaqueTokenStrategy) IsStateful() bool {
	return true
}

func (s *OpaqueTokenStrategy) SupportsExpirationExtension() bool {
	return true
}

func (s *OpaqueTokenStrategy) Create(ctx context.Context, user *User, options *SessionCreateOptions) (*SessionCreateResult, error) {
	sessionToken, err := generateCryptoSecureRandomString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	return &SessionCreateResult{
		SessionValue: sessionToken,
		Token:        sessionToken,
	}, nil
}

func (s *OpaqueTokenStrategy) Validate(ctx context.Context, request *http.Request) (*SessionValidateResult, error) {
	sessionID, err := s.extractSessionToken(request)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	session, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	if session.IsExpired(s.config.IdleTimeout) {
		s.store.Delete(ctx, sessionID)
		return nil, ErrSessionExpired
	}

	if s.config.ActivityCheckInterval > 0 && time.Now().After(session.LastAccess.Add(s.config.ActivityCheckInterval)) {
		session.Touch()
		if err := s.store.Update(ctx, session.ID, &Session{LastAccess: session.LastAccess}); err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}
	}

	return &SessionValidateResult{
		UserID: session.UserID,
		AegisSession: AegisSession{
			Session: session,
		},
	}, nil
}

func (s *OpaqueTokenStrategy) extractSessionToken(request *http.Request) (string, error) {
	if s.cookieConfig != nil {
		if cookie, err := request.Cookie(s.cookieConfig.name); err == nil {
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
