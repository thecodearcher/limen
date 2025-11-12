package session

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/thecodearcher/aegis"
)

type ServerSideStrategy struct {
	store  aegis.SessionStore
	config *aegis.SessionConfig
}

func NewServerSideStrategy(store aegis.SessionStore, config *aegis.SessionConfig) *ServerSideStrategy {
	return &ServerSideStrategy{
		store:  store,
		config: config,
	}
}

func (s *ServerSideStrategy) GetName() string {
	return string(aegis.SessionStrategyServerSide)
}

func (s *ServerSideStrategy) IsStateful() bool {
	return true
}

func (s *ServerSideStrategy) Create(ctx context.Context, user *aegis.User, temporaryAuth bool) (*aegis.SessionResult, error) {
	sessionID, err := GenerateCryptoSecureRandomString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	return &aegis.SessionResult{
		Cookie: s.createCookie(sessionID, temporaryAuth),
	}, nil
}

func (s *ServerSideStrategy) Validate(ctx context.Context, request *http.Request) (*aegis.SessionValidateResult, error) {
	sessionID, err := s.extractSessionID(request)
	if err != nil {
		return nil, aegis.ErrSessionNotFound
	}

	session, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return nil, aegis.ErrSessionNotFound
	}

	if session.IsExpired(s.config.IdleTimeout) {
		s.store.Delete(ctx, sessionID)
		return nil, aegis.ErrSessionExpired
	}

	if s.config.ActivityCheckInterval > 0 && time.Now().After(session.LastAccess.Add(s.config.ActivityCheckInterval)) {
		session.Touch()
		if err := s.store.Update(ctx, session); err != nil {
			//TODO: add logger
			// This is a non-critical operation
		}
	}

	return &aegis.SessionValidateResult{
		UserID:   session.UserID,
		Metadata: session.Metadata,
	}, nil
}

func (s *ServerSideStrategy) Refresh(ctx context.Context, request *http.Request) (*aegis.SessionRefreshResult, error) {
	sessionID, err := s.extractSessionID(request)
	if err != nil {
		return nil, aegis.ErrSessionNotFound
	}

	session, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return nil, aegis.ErrSessionNotFound
	}

	if session.IsExpired(s.config.IdleTimeout) {
		s.store.Delete(ctx, sessionID)
		return nil, aegis.ErrSessionExpired
	}

	if !session.ShouldRefresh(s.config.RefreshInterval) {
		// Session doesn't need refresh yet, return as-is
		return &aegis.SessionRefreshResult{
			Token:       session.ID,
			UserID:      session.UserID,
			ShouldStore: false,
		}, nil
	}

	newSessionID, err := GenerateCryptoSecureRandomString()
	if err != nil {
		return nil, fmt.Errorf("failed to generate new session ID: %w", err)
	}

	return &aegis.SessionRefreshResult{
		Token:          newSessionID,
		UserID:         session.UserID,
		ShouldStore:    true,
		StaleSessionID: sessionID,
	}, nil
}

func (s *ServerSideStrategy) extractSessionID(request *http.Request) (string, error) {
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

func (s *ServerSideStrategy) createCookie(token string, temporaryAuth bool) *http.Cookie {
	cookieOptions := s.config.CookieOptions

	cookie := &http.Cookie{
		Name:        cookieOptions.Name,
		Value:       token,
		Path:        cookieOptions.Path,
		MaxAge:      int(s.config.Duration.Seconds()),
		HttpOnly:    cookieOptions.HTTPOnly,
		Secure:      cookieOptions.Secure,
		SameSite:    cookieOptions.SameSite,
		Partitioned: cookieOptions.Partitioned,
	}

	if temporaryAuth {
		// we override the max age for temporary auth to 5 minutes
		cookie.MaxAge = int(time.Duration(5 * time.Minute).Seconds())
	}

	return cookie
}
