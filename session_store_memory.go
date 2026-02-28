package aegis

import (
	"context"
	"fmt"
	"maps"
	"sync"
)

// MemorySessionStore implements SessionStore using in-memory storage.
type MemorySessionStore struct {
	sessions     map[string]*Session        // token -> session
	sessionsByID map[string]*Session        // id -> session (for O(1) ID lookups)
	userSessions map[string]map[string]bool // userID -> set of tokens
	mu           sync.RWMutex
}

// NewMemorySessionStore creates a new in-memory session store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions:     make(map[string]*Session),
		sessionsByID: make(map[string]*Session),
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

	copied := s.copySession(session)
	s.sessions[session.Token] = copied

	// Maintain ID index if ID is set
	if session.ID != nil {
		idStr := s.idToString(session.ID)
		s.sessionsByID[idStr] = copied
	}

	userIDStr := s.userIDToString(session.UserID)
	if s.userSessions[userIDStr] == nil {
		s.userSessions[userIDStr] = make(map[string]bool)
	}
	s.userSessions[userIDStr][session.Token] = true

	return nil
}

// GetByToken retrieves a session by token.
// Returns ErrSessionNotFound if the session does not exist.
func (s *MemorySessionStore) GetByToken(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, ErrSessionNotFound
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[token]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return s.copySession(session), nil
}

// UpdateByToken updates an existing session by token with partial data.
// Only non-nil fields in the updates parameter are applied.
func (s *MemorySessionStore) UpdateByToken(ctx context.Context, token string, updates *SessionUpdates) error {
	if updates == nil {
		return fmt.Errorf("updates cannot be nil")
	}
	if token == "" {
		return ErrSessionNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.sessions[token]
	if !exists {
		return ErrSessionNotFound
	}

	// Apply non-nil fields from updates
	if updates.ExpiresAt != nil {
		existing.ExpiresAt = *updates.ExpiresAt
	}
	if updates.LastAccess != nil {
		existing.LastAccess = *updates.LastAccess
	}
	if updates.Metadata != nil {
		if existing.Metadata == nil {
			existing.Metadata = make(map[string]any)
		}
		maps.Copy(existing.Metadata, updates.Metadata)
	}

	return nil
}

// DeleteByToken removes a session by token.
// Returns ErrSessionNotFound if the session does not exist.
func (s *MemorySessionStore) DeleteByToken(ctx context.Context, token string) error {
	if token == "" {
		return ErrSessionNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[token]
	if !exists {
		return ErrSessionNotFound
	}

	// Remove from user sessions index
	userIDStr := s.userIDToString(session.UserID)
	if sessions, ok := s.userSessions[userIDStr]; ok {
		delete(sessions, token)
		if len(sessions) == 0 {
			delete(s.userSessions, userIDStr)
		}
	}

	// Remove from ID index
	if session.ID != nil {
		idStr := s.idToString(session.ID)
		delete(s.sessionsByID, idStr)
	}

	delete(s.sessions, token)

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
		// Remove from ID index before deleting session
		if session, ok := s.sessions[token]; ok && session.ID != nil {
			idStr := s.idToString(session.ID)
			delete(s.sessionsByID, idStr)
		}
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

func (s *MemorySessionStore) idToString(id any) string {
	if id == nil {
		return ""
	}
	if str, ok := id.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", id)
}
