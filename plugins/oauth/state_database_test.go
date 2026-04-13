package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func newTestDatabaseStateStore(t *testing.T) *databaseStateStore {
	t.Helper()
	provider := &testProvider{name: "test"}
	plugin := New(WithProviders(provider), WithDatabaseState())
	limen.NewTestLimen(t, plugin)
	return newDatabaseStateStore(plugin.core, 5*time.Minute)
}

func TestDatabaseState_GenerateAndValidate(t *testing.T) {
	t.Parallel()

	store := newTestDatabaseStateStore(t)
	ctx := context.Background()

	data := map[string]any{"provider": "github", "redirect_uri": "http://localhost/cb"}

	stateToken, cookieNonce, err := store.Generate(ctx, data)
	require.NoError(t, err)
	assert.NotEmpty(t, stateToken)
	assert.NotEmpty(t, cookieNonce)

	result, err := store.Validate(ctx, stateToken, cookieNonce)
	require.NoError(t, err)
	assert.Equal(t, "github", result["provider"])
	assert.Equal(t, "http://localhost/cb", result["redirect_uri"])
}

func TestDatabaseState_SingleUse(t *testing.T) {
	t.Parallel()

	store := newTestDatabaseStateStore(t)
	ctx := context.Background()

	stateToken, cookieNonce, err := store.Generate(ctx, nil)
	require.NoError(t, err)

	_, err = store.Validate(ctx, stateToken, cookieNonce)
	require.NoError(t, err)

	_, err = store.Validate(ctx, stateToken, cookieNonce)
	assert.ErrorIs(t, err, ErrOAuthStateInvalid)
}

func TestDatabaseState_WrongNonce(t *testing.T) {
	t.Parallel()

	store := newTestDatabaseStateStore(t)
	ctx := context.Background()

	stateToken, _, err := store.Generate(ctx, nil)
	require.NoError(t, err)

	_, err = store.Validate(ctx, stateToken, "wrong-nonce")
	assert.ErrorIs(t, err, ErrOAuthStateInvalid)
}

func TestDatabaseState_EmptyInputs(t *testing.T) {
	t.Parallel()

	store := newTestDatabaseStateStore(t)
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := store.Validate(ctx, tt.stateToken, tt.cookie)
			assert.ErrorIs(t, err, ErrOAuthStateInvalid)
		})
	}
}
