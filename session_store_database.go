package limen

import (
	"context"
)

type databaseSessionStore struct {
	core   *LimenCore
	schema *SessionSchema
}

func newDatabaseSessionStore(core *LimenCore) *databaseSessionStore {
	return &databaseSessionStore{
		core:   core,
		schema: core.Schema.Session,
	}
}

func (s *databaseSessionStore) Get(ctx context.Context, token string) (*Session, error) {
	return s.core.DBAction.FindSessionByToken(ctx, token)
}

func (s *databaseSessionStore) Set(ctx context.Context, session *Session) error {
	if session.ID == nil {
		return s.core.DBAction.CreateSession(ctx, session, nil)
	}
	return s.core.DBAction.UpdateSession(ctx, session, []Where{
		Eq(s.schema.GetTokenField(), session.Token),
	})
}

func (s *databaseSessionStore) Delete(ctx context.Context, token string) error {
	return s.core.DBAction.DeleteSessionByToken(ctx, token)
}

func (s *databaseSessionStore) DeleteByUserID(ctx context.Context, userID any) error {
	return s.core.DBAction.DeleteSessionByUserID(ctx, userID)
}

func (s *databaseSessionStore) ListByUserID(ctx context.Context, userID any) ([]Session, error) {
	return s.core.DBAction.ListSessionsByUserID(ctx, userID)
}
