package aegis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type cacheSessionStore struct {
	cache  CacheAdapter
	prefix string
}

func newSecondarySessionStore(core *AegisCore) *cacheSessionStore {
	return &cacheSessionStore{
		cache:  core.CacheStore(),
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
	data, err := s.cache.Get(ctx, s.sessionKey(token))
	if err != nil {
		return nil, ErrSessionNotFound
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, ErrSessionNotFound
	}

	return &session, nil
}

func (s *cacheSessionStore) Set(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := max(time.Until(session.ExpiresAt), 0)
	if err := s.cache.Set(ctx, s.sessionKey(session.Token), data, ttl); err != nil {
		return err
	}

	return s.addToUserIndex(ctx, session)
}

func (s *cacheSessionStore) Delete(ctx context.Context, token string) error {
	sess, err := s.Get(ctx, token)
	if err != nil {
		return err
	}

	if err := s.cache.Delete(ctx, s.sessionKey(token)); err != nil {
		return err
	}

	return s.removeFromUserIndex(ctx, sess.UserID, token)
}

func (s *cacheSessionStore) ListByUserID(ctx context.Context, userID any) ([]Session, error) {
	return s.getUserSessions(ctx, userID)
}

func (s *cacheSessionStore) DeleteByUserID(ctx context.Context, userID any) error {
	sessions, err := s.getUserSessions(ctx, userID)
	if err != nil {
		return nil
	}

	for _, sess := range sessions {
		s.cache.Delete(ctx, s.sessionKey(sess.Token))
	}

	return s.cache.Delete(ctx, s.userSessionsKey(userID))
}

func (s *cacheSessionStore) getUserSessions(ctx context.Context, userID any) ([]Session, error) {
	data, err := s.cache.Get(ctx, s.userSessionsKey(userID))
	if err != nil {
		return nil, err
	}

	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *cacheSessionStore) addToUserIndex(ctx context.Context, session *Session) error {
	sessions, _ := s.getUserSessions(ctx, session.UserID)

	for i, sess := range sessions {
		if sess.Token == session.Token {
			sessions[i] = *session
			return s.saveUserIndex(ctx, session.UserID, sessions)
		}
	}

	sessions = append(sessions, *session)
	return s.saveUserIndex(ctx, session.UserID, sessions)
}

func (s *cacheSessionStore) removeFromUserIndex(ctx context.Context, userID any, token string) error {
	sessions, err := s.getUserSessions(ctx, userID)
	if err != nil {
		return nil
	}

	filtered := sessions[:0]
	for _, sess := range sessions {
		if sess.Token != token {
			filtered = append(filtered, sess)
		}
	}

	if len(filtered) == 0 {
		return s.cache.Delete(ctx, s.userSessionsKey(userID))
	}

	return s.saveUserIndex(ctx, userID, filtered)
}

func (s *cacheSessionStore) saveUserIndex(ctx context.Context, userID any, sessions []Session) error {
	data, err := json.Marshal(sessions)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, s.userSessionsKey(userID), data, 0)
}
