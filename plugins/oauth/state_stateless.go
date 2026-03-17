package oauth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
)

type statelessStateStore struct {
	secret []byte
	ttl    time.Duration
}

func newStatelessStateStore(secret []byte, ttl time.Duration) *statelessStateStore {
	return &statelessStateStore{secret: secret, ttl: ttl}
}

func (s *statelessStateStore) Generate(_ context.Context, data map[string]any) (string, string, error) {
	stateToken := generateRandomString()

	payload := statePayload{
		CookieNonce: stateToken,
		ExpiresAt:   time.Now().Add(s.ttl).Unix(),
		Data:        data,
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("oauth: failed to marshal state payload: %w", err)
	}

	cookieValue, err := limen.EncryptXChaCha(string(plaintext), s.secret, nil)
	if err != nil {
		return "", "", fmt.Errorf("oauth: failed to encrypt state: %w", err)
	}

	return stateToken, cookieValue, nil
}

func (s *statelessStateStore) Validate(_ context.Context, stateToken string, cookieValue string) (map[string]any, error) {
	if stateToken == "" || cookieValue == "" {
		return nil, ErrOAuthStateInvalid
	}

	plaintext, err := limen.DecryptXChaCha(cookieValue, s.secret, nil)
	if err != nil {
		return nil, ErrOAuthStateInvalid
	}

	var payload statePayload
	if err := json.Unmarshal([]byte(plaintext), &payload); err != nil {
		return nil, ErrOAuthStateInvalid
	}

	if time.Now().After(time.Unix(payload.ExpiresAt, 0)) {
		return nil, ErrOAuthStateInvalid
	}

	if subtle.ConstantTimeCompare([]byte(payload.CookieNonce), []byte(stateToken)) != 1 {
		return nil, ErrOAuthStateInvalid
	}

	return payload.Data, nil
}
