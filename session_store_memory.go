package aegis

import (
	"context"
	"fmt"
	"maps"
	"sync"
)

// MemorySessionStore implements SessionStore using in-memory storage
type MemorySessionStore struct {
	sessions map[string]*Session // map[sessionID]*Session
	users    map[string][]string // map[userID][]sessionID for efficient lookup
	mu       sync.RWMutex
}

// NewMemorySessionStore creates a new in-memory session store
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
		users:    make(map[string][]string),
	}
}

// Create creates a new session with the given ID and data
func (s *MemorySessionStore) Create(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session already exists
	if _, exists := s.sessions[session.Token]; exists {
		return nil // Session already exists, treat as success (idempotent)
	}

	// Create a copy of the session to avoid external modifications
	sessionCopy := s.copySession(session)

	// Store the session
	s.sessions[session.Token] = sessionCopy

	// Index by userID
	userIDStr := s.userIDToString(session.UserID)
	s.users[userIDStr] = append(s.users[userIDStr], session.Token)

	return nil
}

// Get retrieves a session by ID
func (s *MemorySessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	// Return a copy to prevent external modifications
	return s.copySession(session), nil
}

// Update updates an existing session
func (s *MemorySessionStore) Update(ctx context.Context, id any, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[s.userIDToString(id)]; !exists {
		return ErrSessionNotFound
	}

	// If UserID changed, update the index
	oldSession := s.sessions[s.userIDToString(id)]
	oldUserIDStr := s.userIDToString(oldSession.UserID)
	newUserIDStr := s.userIDToString(session.UserID)

	if oldUserIDStr != newUserIDStr {
		// Remove from old user index
		s.removeFromUserIndex(oldUserIDStr, session.Token)
		// Add to new user index
		s.users[newUserIDStr] = append(s.users[newUserIDStr], session.Token)
	}

	// Update the session
	s.sessions[session.Token] = s.copySession(session)

	return nil
}

// Delete removes a session by ID
func (s *MemorySessionStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	// Remove from user index
	userIDStr := s.userIDToString(session.UserID)
	s.removeFromUserIndex(userIDStr, sessionID)

	// Delete the session
	delete(s.sessions, sessionID)

	return nil
}

// DeleteByUserID removes all sessions for a specific user
func (s *MemorySessionStore) DeleteByUserID(ctx context.Context, userID any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userIDStr := s.userIDToString(userID)
	sessionIDs, exists := s.users[userIDStr]
	if !exists {
		return nil // No sessions for this user, treat as success
	}

	// Delete all sessions for this user
	for _, sessionID := range sessionIDs {
		delete(s.sessions, sessionID)
	}

	// Clear the user index
	delete(s.users, userIDStr)

	return nil
}

func (s *MemorySessionStore) List(ctx context.Context) (map[string]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.sessions, nil
}

// Helper methods

// copySession creates a deep copy of a session
func (s *MemorySessionStore) copySession(session *Session) *Session {
	copy := &Session{
		Token:      session.Token,
		UserID:     session.UserID,
		CreatedAt:  session.CreatedAt,
		ExpiresAt:  session.ExpiresAt,
		LastAccess: session.LastAccess,
		Metadata:   session.Metadata,
	}

	// Copy maps
	if session.Metadata != nil {
		copy.Metadata = make(map[string]interface{})
		maps.Copy(copy.Metadata, session.Metadata)
	} else {
		copy.Metadata = make(map[string]interface{})
	}

	return copy
}

// userIDToString converts a userID (which can be any type) to a string for indexing
func (s *MemorySessionStore) userIDToString(userID any) string {
	if userID == nil {
		return ""
	}
	if str, ok := userID.(string); ok {
		return str
	}
	// For non-string userIDs, convert to string representation
	return fmt.Sprintf("%v", userID)
}

// removeFromUserIndex removes a sessionID from a user's session list
func (s *MemorySessionStore) removeFromUserIndex(userIDStr, sessionID string) {
	sessionIDs := s.users[userIDStr]
	newSessionIDs := make([]string, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if id != sessionID {
			newSessionIDs = append(newSessionIDs, id)
		}
	}

	if len(newSessionIDs) == 0 {
		delete(s.users, userIDStr)
	} else {
		s.users[userIDStr] = newSessionIDs
	}
}
