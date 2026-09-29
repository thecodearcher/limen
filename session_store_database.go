package limen

import (
	"context"
	"fmt"
	"maps"
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
	session, err := s.core.FindOne(ctx, s.schema, []Where{
		Eq(s.schema.GetTokenField(), token),
	}, nil)
	if err != nil {
		return nil, err
	}
	return session.(*Session), nil
}

func (s *databaseSessionStore) Set(ctx context.Context, data any) error {
	switch v := data.(type) {
	case *Session:
		if v.ID != nil {
			res, err := s.core.UpdateWithResult(ctx, s.schema, v, []Where{Eq(s.schema.GetIDField(), v.ID)})
			if err != nil || res.RowsAffected > 0 {
				return err
			}
		}
		return s.core.Create(ctx, s.schema, v, nil)
	case map[SchemaField]any:
		if v[SchemaIDField] == nil {
			return s.core.Create(ctx, s.schema, v, nil)
		}
		return s.write(ctx, maps.Clone(v))
	default:
		return errUnsupportedSessionData(data)
	}
}

func (s *databaseSessionStore) write(ctx context.Context, payload map[SchemaField]any) error {
	token, err := sessionPayloadToken(payload)
	if err != nil {
		return err
	}

	update := maps.Clone(payload)
	delete(update, SessionSchemaTokenField)
	delete(update, SchemaIDField)
	if len(update) == 0 {
		return nil
	}

	result, err := s.core.UpdateWithResult(ctx, s.schema, update, []Where{
		Eq(s.schema.GetTokenField(), token),
	})
	if err != nil {
		return err
	}

	if result.RowsAffected > 0 {
		return nil
	}
	return s.core.Create(ctx, s.schema, payload, nil)
}

func (s *databaseSessionStore) UpdateSessions(ctx context.Context, data map[SchemaField]any, match map[SchemaField]any) error {
	columns, err := s.schema.sessionColumns(match)
	if err != nil {
		return err
	}

	conditions := make([]Where, 0, len(columns))
	for column, value := range columns {
		conditions = append(conditions, Eq(column, value))
	}
	return s.core.Update(ctx, s.schema, data, conditions)
}

func sessionPayloadToken(payload map[SchemaField]any) (string, error) {
	token, ok := payload[SessionSchemaTokenField].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("session store: session data requires %q", SessionSchemaTokenField)
	}
	return token, nil
}

func errUnsupportedSessionData(data any) error {
	return fmt.Errorf("session store: unsupported data type %T, want *limen.Session or map[limen.SchemaField]any", data)
}

func (s *databaseSessionStore) Delete(ctx context.Context, token string) error {
	return s.core.Delete(ctx, s.schema, []Where{
		Eq(s.schema.GetTokenField(), token),
	})
}

func (s *databaseSessionStore) DeleteByUserID(ctx context.Context, userID any) error {
	return s.core.Delete(ctx, s.schema, []Where{
		Eq(s.schema.GetUserIDField(), userID),
	})
}

func (s *databaseSessionStore) ListByUserID(ctx context.Context, userID any) ([]Session, error) {
	models, err := s.core.FindMany(ctx, s.schema, []Where{
		Eq(s.schema.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}
	sessions := make([]Session, 0, len(models))
	for _, m := range models {
		sessions = append(sessions, *m.(*Session))
	}
	return sessions, nil
}
