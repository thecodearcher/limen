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
		schema: &core.Schema.Session,
	}
}

// Create creates a new session
func (s *DatabaseSessionStore) Create(ctx context.Context, session *Session) error {
	return s.core.DBAction.CreateSession(ctx, session, nil)
}

// Get retrieves a session by ID
func (s *DatabaseSessionStore) Get(ctx context.Context, sessionToken string) (*Session, error) {
	return s.core.DBAction.FindSessionByToken(ctx, sessionToken)
}

// Update updates an existing session
func (s *DatabaseSessionStore) Update(ctx context.Context, id any, session *Session) error {
	return s.core.DBAction.UpdateSession(ctx, session, []Where{
		Eq(s.schema.GetIDField(), id),
	})
}

// Delete removes a session by ID
func (s *DatabaseSessionStore) Delete(ctx context.Context, sessionToken string) error {
	return s.core.DBAction.DeleteSessionByToken(ctx, sessionToken)
}

// DeleteByUserID removes all sessions for a specific user
func (s *DatabaseSessionStore) DeleteByUserID(ctx context.Context, userID any) error {
	return s.core.DBAction.DeleteSessionByUserID(ctx, userID)
}
