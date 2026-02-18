package oauth

import "context"

type statePayload struct {
	CookieNonce string         `json:"cn"`
	ExpiresAt   int64          `json:"exp"`
	Data        map[string]any `json:"d"`
}

type StateStore interface {
	// Generate creates a new state token and a cookie value (nonce for database store,
	// encrypted payload for stateless store).
	Generate(ctx context.Context, data map[string]any) (stateToken string, cookieValue string, err error)

	// Validate verifies the state token and cookie value, ensures the token has not
	// expired, and returns the data map that was passed to Generate.
	Validate(ctx context.Context, stateToken string, cookieValue string) (map[string]any, error)
}
