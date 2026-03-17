package oauth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
)

type databaseStateStore struct {
	core *limen.LimenCore
	ttl  time.Duration
}

func newDatabaseStateStore(core *limen.LimenCore, ttl time.Duration) *databaseStateStore {
	return &databaseStateStore{core: core, ttl: ttl}
}

func (s *databaseStateStore) Generate(ctx context.Context, data map[string]any) (string, string, error) {
	stateToken := generateRandomString()
	cookieNonce := generateRandomString()

	value := statePayload{
		CookieNonce: cookieNonce,
		ExpiresAt:   time.Now().Add(s.ttl).Unix(),
		Data:        data,
	}
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return "", "", fmt.Errorf("oauth: failed to marshal state value: %w", err)
	}

	_, err = s.core.DBAction.CreateVerification(ctx, oauthStateAction, stateToken, string(valueJSON), s.ttl)
	if err != nil {
		return "", "", fmt.Errorf("oauth: failed to store state: %w", err)
	}

	return stateToken, cookieNonce, nil
}

func (s *databaseStateStore) Validate(ctx context.Context, stateToken string, cookieNonce string) (map[string]any, error) {
	if stateToken == "" || cookieNonce == "" {
		return nil, ErrOAuthStateInvalid
	}

	verification, err := s.core.DBAction.FindVerificationByAction(ctx, oauthStateAction, stateToken)
	if err != nil {
		return nil, ErrOAuthStateInvalid
	}

	var value statePayload
	if err := json.Unmarshal([]byte(verification.Value), &value); err != nil {
		_ = s.core.DBAction.DeleteVerification(ctx, verification.ID)
		return nil, ErrOAuthStateInvalid
	}

	isExpired := time.Now().After(verification.ExpiresAt)
	if subtle.ConstantTimeCompare([]byte(value.CookieNonce), []byte(cookieNonce)) != 1 || isExpired {
		_ = s.core.DBAction.DeleteVerification(ctx, verification.ID)
		return nil, ErrOAuthStateInvalid
	}

	if err := s.core.DBAction.DeleteVerification(ctx, verification.ID); err != nil {
		return nil, fmt.Errorf("oauth: failed to consume state token: %w", err)
	}

	return value.Data, nil
}
