package aegis

import (
	"context"
	"fmt"
	"maps"
	"sync"
)

// MemorySessionStore implements SessionStore using in-memory storage.
type MemorySessionStore struct {
	sessions     map[string]*Session
	userSessions map[string]map[string]bool
	mu           sync.RWMutex
}

// NewMemorySessionStore creates a new in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions:     make(map[string]*Session),
		userSessions: make(map[string]map[string]bool),
	}
}

// Create creates a new session with the given data.
// If a session with the same token already exists, it returns nil (idempotent).
func (s *MemorySessionStore) Create(ctx context.Context, session *Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}
	if session.Token == "" {
		return fmt.Errorf("session token cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.Token]; exists {
		return nil
	}

	s.sessions[session.Token] = s.copySession(session)

	userIDStr := s.userIDToString(session.UserID)
	if s.userSessions[userIDStr] == nil {
		s.userSessions[userIDStr] = make(map[string]bool)
	}
	s.userSessions[userIDStr][session.Token] = true

	return nil
}

// Get retrieves a session by token.
// Returns ErrSessionNotFound if the session does not exist.
func (s *MemorySessionStore) Get(ctx context.Context, sessionToken string) (*Session, error) {
	if sessionToken == "" {
		return nil, ErrSessionNotFound
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionToken]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return s.copySession(session), nil
}

// Update updates an existing session.
func (s *MemorySessionStore) Update(ctx context.Context, id any, session *Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}
	if session.Token == "" {
		return fmt.Errorf("session token cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.Token]; !exists {
		return ErrSessionNotFound
	}

	s.sessions[session.Token] = s.copySession(session)

	return nil
}

// Delete removes a session by token.
// Returns ErrSessionNotFound if the session does not exist.
func (s *MemorySessionStore) Delete(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return ErrSessionNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionToken]
	if !exists {
		return ErrSessionNotFound
	}

	userIDStr := s.userIDToString(session.UserID)
	if sessions, ok := s.userSessions[userIDStr]; ok {
		delete(sessions, sessionToken)
		if len(sessions) == 0 {
			delete(s.userSessions, userIDStr)
		}
	}

	delete(s.sessions, sessionToken)

	return nil
}

// DeleteByUserID removes all sessions for a specific user.
// Returns nil if the user has no sessions (idempotent operation).
func (s *MemorySessionStore) DeleteByUserID(ctx context.Context, userID any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userIDStr := s.userIDToString(userID)
	sessionTokens, exists := s.userSessions[userIDStr]
	if !exists {
		return nil
	}

	for token := range sessionTokens {
		delete(s.sessions, token)
	}

	delete(s.userSessions, userIDStr)

	return nil
}

func (s *MemorySessionStore) copySession(session *Session) *Session {
	if session == nil {
		return nil
	}

	copy := &Session{
		ID:         session.ID,
		Token:      session.Token,
		UserID:     session.UserID,
		CreatedAt:  session.CreatedAt,
		ExpiresAt:  session.ExpiresAt,
		LastAccess: session.LastAccess,
		Metadata:   make(map[string]any),
	}

	if session.Metadata != nil {
		maps.Copy(copy.Metadata, session.Metadata)
	}

	return copy
}

func (s *MemorySessionStore) userIDToString(userID any) string {
	if userID == nil {
		return ""
	}
	if str, ok := userID.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", userID)
}
