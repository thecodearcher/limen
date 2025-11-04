package session

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/thecodearcher/aegis"
)

// MemorySessionStore implements SessionStore using in-memory storage
type MemorySessionStore struct {
	sessions map[string]*aegis.Session // map[sessionID]*Session
	users    map[string][]string       // map[userID][]sessionID for efficient lookup
	mu       sync.RWMutex
}

// NewMemorySessionStore creates a new in-memory session store
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*aegis.Session),
		users:    make(map[string][]string),
	}
}

// Create creates a new session with the given ID and data
func (s *MemorySessionStore) Create(ctx context.Context, session *aegis.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session already exists
	if _, exists := s.sessions[session.ID]; exists {
		return nil // Session already exists, treat as success (idempotent)
	}

	// Create a copy of the session to avoid external modifications
	sessionCopy := s.copySession(session)

	// Store the session
	s.sessions[session.ID] = sessionCopy

	// Index by userID
	userIDStr := s.userIDToString(session.UserID)
	s.users[userIDStr] = append(s.users[userIDStr], session.ID)

	return nil
}

// Get retrieves a session by ID
func (s *MemorySessionStore) Get(ctx context.Context, sessionID string) (*aegis.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, aegis.ErrSessionNotFound
	}

	// Return a copy to prevent external modifications
	return s.copySession(session), nil
}

// Update updates an existing session
func (s *MemorySessionStore) Update(ctx context.Context, session *aegis.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[session.ID]; !exists {
		return aegis.ErrSessionNotFound
	}

	// If UserID changed, update the index
	oldSession := s.sessions[session.ID]
	oldUserIDStr := s.userIDToString(oldSession.UserID)
	newUserIDStr := s.userIDToString(session.UserID)

	if oldUserIDStr != newUserIDStr {
		// Remove from old user index
		s.removeFromUserIndex(oldUserIDStr, session.ID)
		// Add to new user index
		s.users[newUserIDStr] = append(s.users[newUserIDStr], session.ID)
	}

	// Update the session
	s.sessions[session.ID] = s.copySession(session)

	return nil
}

// Delete removes a session by ID
func (s *MemorySessionStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return aegis.ErrSessionNotFound
	}

	// Remove from user index
	userIDStr := s.userIDToString(session.UserID)
	s.removeFromUserIndex(userIDStr, sessionID)

	// Delete the session
	delete(s.sessions, sessionID)

	return nil
}

// DeleteByUserID removes all sessions for a specific user
func (s *MemorySessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userIDStr := userID
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

// Cleanup removes expired sessions
func (s *MemorySessionStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var expiredIDs []string

	// Find all expired sessions
	for sessionID, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			expiredIDs = append(expiredIDs, sessionID)
		}
	}

	// Delete expired sessions
	for _, sessionID := range expiredIDs {
		session := s.sessions[sessionID]
		userIDStr := s.userIDToString(session.UserID)
		s.removeFromUserIndex(userIDStr, sessionID)
		delete(s.sessions, sessionID)
	}

	return nil
}

// Exists checks if a session exists
func (s *MemorySessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.sessions[sessionID]
	return exists, nil
}

// Count returns the number of active sessions for a user
func (s *MemorySessionStore) Count(ctx context.Context, userID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionIDs, exists := s.users[userID]
	if !exists {
		return 0, nil
	}

	// Count only non-expired sessions
	now := time.Now()
	count := 0
	for _, sessionID := range sessionIDs {
		if session, ok := s.sessions[sessionID]; ok && !now.After(session.ExpiresAt) {
			count++
		}
	}

	return count, nil
}

// Helper methods

// copySession creates a deep copy of a session
func (s *MemorySessionStore) copySession(session *aegis.Session) *aegis.Session {
	copy := &aegis.Session{
		ID:         session.ID,
		UserID:     session.UserID,
		CreatedAt:  session.CreatedAt,
		ExpiresAt:  session.ExpiresAt,
		LastAccess: session.LastAccess,
		IPAddress:  session.IPAddress,
		UserAgent:  session.UserAgent,
	}

	// Copy maps
	if session.Data != nil {
		copy.Data = make(map[string]interface{})
		maps.Copy(copy.Data, session.Data)
	} else {
		copy.Data = make(map[string]interface{})
	}

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
