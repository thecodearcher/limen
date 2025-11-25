package aegis

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ServerSideStrategy struct {
	store  SessionStore
	config *sessionConfig
}

func NewServerSideStrategy(store SessionStore, config *sessionConfig) *ServerSideStrategy {
	return &ServerSideStrategy{
		store:  store,
		config: config,
	}
}

func (s *ServerSideStrategy) GetName() string {
	return string(SessionStrategyServerSide)
}

func (s *ServerSideStrategy) IsStateful() bool {
	return true
}

func (s *ServerSideStrategy) SupportsSlidingWindowRefresh() bool {
	return true
}

func (s *ServerSideStrategy) Create(ctx context.Context, user *User, temporaryAuth bool) (*SessionResult, error) {
	sessionToken, err := GenerateCryptoSecureRandomString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	return &SessionResult{
		Token: sessionToken,
	}, nil
}

func (s *ServerSideStrategy) Validate(ctx context.Context, request *http.Request) (*SessionValidateResult, error) {
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
		if err := s.store.Update(ctx, session.ID, session); err != nil {
			//TODO: add logger
			// This is a non-critical operation
		}
	}

	return &SessionValidateResult{
		UserID:   session.UserID,
		Metadata: session.Metadata,
		Session:  session,
	}, nil
}

func (s *ServerSideStrategy) Refresh(ctx context.Context, request *http.Request) (*SessionRefreshResult, error) {
	sessionToken, err := s.extractSessionToken(request)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	session, err := s.store.Get(ctx, sessionToken)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	if session.IsExpired(s.config.IdleTimeout) {
		s.store.Delete(ctx, sessionToken)
		return nil, ErrSessionExpired
	}

	if !session.ShouldRefresh(s.config.RefreshInterval) {
		// Session doesn't need refresh yet, return as-is
		return &SessionRefreshResult{
			Token:       session.Token,
			UserID:      session.UserID,
			ShouldStore: false,
		}, nil
	}

	newSessionID, err := GenerateCryptoSecureRandomString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new session ID: %w", err)
	}

	return &SessionRefreshResult{
		Token:          newSessionID,
		UserID:         session.UserID,
		ShouldStore:    true,
		StaleSessionID: sessionToken,
	}, nil
}

func (s *ServerSideStrategy) extractSessionToken(request *http.Request) (string, error) {
	cookie, err := request.Cookie(s.config.CookieOptions.Name)
	if err != nil {
		return "", err
	}

	sessionID := strings.TrimSpace(cookie.Value)
	if sessionID == "" {
		return "", fmt.Errorf("empty session ID")
	}

	return sessionID, nil
}
