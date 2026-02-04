package aegis

import (
	"context"
)

// DatabaseSessionStore implements SessionStore using a database adapter
type DatabaseSessionStore struct {
	core   *AegisCore
	schema *SessionSchema
}

// NewDatabaseSessionStore creates a new database-backed session store
func NewDatabaseSessionStore(core *AegisCore) *DatabaseSessionStore {
	return &DatabaseSessionStore{
		core:   core,
		schema: core.Schema.Session,
	}
}

// Create creates a new session
func (s *DatabaseSessionStore) Create(ctx context.Context, session *Session) error {
	return s.core.DBAction.CreateSession(ctx, session, nil)
}

// GetByToken retrieves a session by token
func (s *DatabaseSessionStore) GetByToken(ctx context.Context, token string) (*Session, error) {
	return s.core.DBAction.FindSessionByToken(ctx, token)
}

// UpdateByToken updates an existing session by token
func (s *DatabaseSessionStore) UpdateByToken(ctx context.Context, token string, updates *SessionUpdates) error {
	// Convert SessionUpdates to Session for the database action
	session := &Session{}
	if updates.ExpiresAt != nil {
		session.ExpiresAt = *updates.ExpiresAt
	}
	if updates.LastAccess != nil {
		session.LastAccess = *updates.LastAccess
	}
	if updates.Metadata != nil {
		session.Metadata = updates.Metadata
	}

	return s.core.DBAction.UpdateSession(ctx, session, []Where{
		Eq(s.schema.GetTokenField(), token),
	})
}

// DeleteByToken removes a session by token
func (s *DatabaseSessionStore) DeleteByToken(ctx context.Context, token string) error {
	return s.core.DBAction.DeleteSessionByToken(ctx, token)
}

// DeleteByUserID removes all sessions for a specific user
func (s *DatabaseSessionStore) DeleteByUserID(ctx context.Context, userID any) error {
	return s.core.DBAction.DeleteSessionByUserID(ctx, userID)
}
