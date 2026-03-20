package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testOAuthSecret = []byte("01234567890123456789012345678901")

func TestStatelessState_GenerateAndValidate(t *testing.T) {
	t.Parallel()

	store := newStatelessStateStore(testOAuthSecret, 5*time.Minute)
	ctx := context.Background()

	data := map[string]any{"provider": "google", "redirect_uri": "http://localhost:3000/callback"}

	stateToken, cookieValue, err := store.Generate(ctx, data)
	assert.NoError(t, err)
	assert.NotEmpty(t, stateToken)
	assert.NotEmpty(t, cookieValue)

	result, err := store.Validate(ctx, stateToken, cookieValue)
	assert.NoError(t, err)
	assert.Equal(t, "google", result["provider"])
	assert.Equal(t, "http://localhost:3000/callback", result["redirect_uri"])
}

func TestStatelessState_Expired(t *testing.T) {
	t.Parallel()

	store := newStatelessStateStore(testOAuthSecret, -1*time.Second) // already expired
	ctx := context.Background()

	stateToken, cookieValue, err := store.Generate(ctx, nil)
	assert.NoError(t, err)

	_, err = store.Validate(ctx, stateToken, cookieValue)
	assert.ErrorIs(t, err, ErrOAuthStateInvalid)
}

func TestStatelessState_TamperedState(t *testing.T) {
	t.Parallel()

	store := newStatelessStateStore(testOAuthSecret, 5*time.Minute)
	ctx := context.Background()

	_, cookieValue, err := store.Generate(ctx, nil)
	assert.NoError(t, err)

	_, err = store.Validate(ctx, "tampered-token", cookieValue)
	assert.ErrorIs(t, err, ErrOAuthStateInvalid)
}

func TestStatelessState_EmptyInputs(t *testing.T) {
	t.Parallel()

	store := newStatelessStateStore(testOAuthSecret, 5*time.Minute)
	ctx := context.Background()

	tests := []struct {
		name       string
		stateToken string
		cookie     string
	}{
		{name: "empty state", stateToken: "", cookie: "something"},
		{name: "empty cookie", stateToken: "something", cookie: ""},
		{name: "both empty", stateToken: "", cookie: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := store.Validate(ctx, tt.stateToken, tt.cookie)
			assert.ErrorIs(t, err, ErrOAuthStateInvalid)
		})
	}
}

func TestStatelessState_WrongSecret(t *testing.T) {
	t.Parallel()

	store1 := newStatelessStateStore(testOAuthSecret, 5*time.Minute)
	store2 := newStatelessStateStore([]byte("99999999999999999999999999999999"), 5*time.Minute)
	ctx := context.Background()

	stateToken, cookieValue, err := store1.Generate(ctx, nil)
	assert.NoError(t, err)

	_, err = store2.Validate(ctx, stateToken, cookieValue)
	assert.ErrorIs(t, err, ErrOAuthStateInvalid)
}
