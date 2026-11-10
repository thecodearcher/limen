package limen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"
)

type cacheSessionStore struct {
	cache  CacheAdapter
	schema *SessionSchema
	config *SchemaConfig
	prefix string
}

func newSecondarySessionStore(core *LimenCore) *cacheSessionStore {
	return &cacheSessionStore{
		cache:  core.CacheStore(),
		schema: core.Schema.Session,
		config: core.Schema,
		prefix: core.CacheKeyPrefix(),
	}
}

func (s *cacheSessionStore) sessionKey(token string) string {
	return fmt.Sprintf("%s:session:t:%s", s.prefix, token)
}

func (s *cacheSessionStore) userSessionsKey(userID any) string {
	return fmt.Sprintf("%s:session:u:%v", s.prefix, userID)
}

func (s *cacheSessionStore) Get(ctx context.Context, token string) (*Session, error) {
	stored, err := s.loadSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return s.schema.FromStorage(stored).(*Session), nil
}

func (s *cacheSessionStore) Set(ctx context.Context, data any) error {
	switch v := data.(type) {
	case *Session:
		stored, err := s.loadSessionOrEmpty(ctx, v.Token)
		if err != nil {
			return err
		}

		maps.Copy(stored, s.schema.ToStorage(v))
		stored[s.schema.GetIDField()] = v.ID
		return s.saveSession(ctx, v.Token, stored)
	case map[SchemaField]any:
		token, err := sessionPayloadToken(v)
		if err != nil {
			return err
		}

		stored, err := s.loadSessionOrEmpty(ctx, token)
		if err != nil {
			return err
		}

		columns, err := s.schema.sessionColumns(v)
		if err != nil {
			return err
		}

		maps.Copy(stored, columns)
		return s.saveSession(ctx, token, stored)
	default:
		return errUnsupportedSessionData(data)
	}
}

func (s *cacheSessionStore) loadSessionOrEmpty(ctx context.Context, token string) (map[string]any, error) {
	stored, err := s.loadSession(ctx, token)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return make(map[string]any), nil
		}
		return nil, err
	}
	return stored, nil
}

func (s *cacheSessionStore) loadSession(ctx context.Context, token string) (map[string]any, error) {
	data, err := s.cache.Get(ctx, s.sessionKey(token))
	if err != nil {
		return nil, ErrSessionNotFound
	}

	var stored map[string]any
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	return s.rehydrateSession(stored)
}

func (s *cacheSessionStore) saveSession(ctx context.Context, token string, stored map[string]any) error {
	session := s.schema.FromStorage(stored).(*Session)

	data, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := max(time.Until(session.ExpiresAt), 0)
	if err := s.cache.Set(ctx, s.sessionKey(token), data, ttl); err != nil {
		return err
	}

	return s.addToUserIndex(ctx, session)
}

// rehydrateSession restores the Go types the JSON round trip degraded on the
// columns SessionSchema.FromStorage asserts.
func (s *cacheSessionStore) rehydrateSession(stored map[string]any) (map[string]any, error) {
	for _, column := range []string{s.schema.GetCreatedAtField(), s.schema.GetExpiresAtField(), s.schema.GetLastAccessField()} {
		switch v := stored[column].(type) {
		case time.Time:
		case string:
			parsed, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				return nil, err
			}
			stored[column] = parsed
		default:
			return nil, fmt.Errorf("session store: column %q holds %T, want time", column, v)
		}
	}

	if _, ok := stored[s.schema.GetTokenField()].(string); !ok {
		return nil, fmt.Errorf("session store: column %q is not a string", s.schema.GetTokenField())
	}

	for _, column := range []string{s.schema.GetIDField(), s.schema.GetUserIDField()} {
		stored[column] = s.config.NormalizeIDValue(stored[column])
	}
	return stored, nil
}

func (s *cacheSessionStore) Delete(ctx context.Context, token string) error {
	sess, err := s.Get(ctx, token)
	if err != nil && !errors.Is(err, ErrSessionNotFound) {
		return err
	}

	if sess == nil {
		return nil
	}

	if err := s.cache.Delete(ctx, s.sessionKey(token)); err != nil {
		return err
	}

	return s.removeFromUserIndex(ctx, sess.UserID, token)
}

func (s *cacheSessionStore) ListByUserID(ctx context.Context, userID any) ([]Session, error) {
	entries, err := s.getUserIndex(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	sessions := make([]Session, 0, len(entries))
	for _, entry := range entries {
		if !entry.ExpiresAt.After(now) {
			continue
		}
		session, err := s.Get(ctx, entry.Token)
		if errors.Is(err, ErrSessionNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}
	return sessions, nil
}

func (s *cacheSessionStore) DeleteByUserID(ctx context.Context, userID any) error {
	entries, err := s.getUserIndex(ctx, userID)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		_ = s.cache.Delete(ctx, s.sessionKey(entry.Token))
	}

	return s.cache.Delete(ctx, s.userSessionsKey(userID))
}

type sessionIndexEntry struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *cacheSessionStore) getUserIndex(ctx context.Context, userID any) ([]sessionIndexEntry, error) {
	data, err := s.cache.Get(ctx, s.userSessionsKey(userID))
	if err != nil {
		return nil, err
	}

	var entries []sessionIndexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *cacheSessionStore) addToUserIndex(ctx context.Context, session *Session) error {
	entries, _ := s.getUserIndex(ctx, session.UserID)

	i := slices.IndexFunc(entries, func(entry sessionIndexEntry) bool { return entry.Token == session.Token })
	if i != -1 && entries[i].ExpiresAt.Equal(session.ExpiresAt) {
		return nil
	}

	entries = slices.DeleteFunc(entries, func(entry sessionIndexEntry) bool { return entry.Token == session.Token })
	entries = append(entries, sessionIndexEntry{Token: session.Token, ExpiresAt: session.ExpiresAt})
	return s.saveUserIndex(ctx, session.UserID, entries)
}

func (s *cacheSessionStore) removeFromUserIndex(ctx context.Context, userID any, token string) error {
	entries, err := s.getUserIndex(ctx, userID)
	if err != nil {
		return nil
	}

	entries = slices.DeleteFunc(entries, func(entry sessionIndexEntry) bool { return entry.Token == token })
	return s.saveUserIndex(ctx, userID, entries)
}

func (s *cacheSessionStore) saveUserIndex(ctx context.Context, userID any, entries []sessionIndexEntry) error {
	now := time.Now()
	entries = slices.DeleteFunc(entries, func(entry sessionIndexEntry) bool { return !entry.ExpiresAt.After(now) })
	if len(entries) == 0 {
		return s.cache.Delete(ctx, s.userSessionsKey(userID))
	}

	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}

	furthest := slices.MaxFunc(entries, func(a, b sessionIndexEntry) int { return a.ExpiresAt.Compare(b.ExpiresAt) })
	return s.cache.Set(ctx, s.userSessionsKey(userID), data, time.Until(furthest.ExpiresAt))
}
