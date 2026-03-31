package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

type tokenRefresherProvider struct {
	testProvider
	lastRefreshToken string
	response         *TokenResponse
}

func (p *tokenRefresherProvider) RefreshToken(_ context.Context, refreshToken string) (*TokenResponse, error) {
	p.lastRefreshToken = refreshToken
	return p.response, nil
}

func TestGetAccessToken(t *testing.T) {
	t.Parallel()

	l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
	ctx := context.Background()
	user := seedOAuthUser(t, l, "tokens@example.com")

	t.Run("account not found", func(t *testing.T) {
		tokens, err := plugin.GetAccessToken(ctx, user.ID, "missing")
		assert.Nil(t, tokens)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("returns active tokens", func(t *testing.T) {
		expiresAt := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
		seedOAuthAccount(t, plugin, user.ID, "test", "provider-token-1")
		account, err := plugin.findAccountByUserIDAndProvider(ctx, user.ID, "test")
		require.NoError(t, err)
		require.NoError(t, plugin.core.Update(ctx, plugin.accountSchema, &limen.Account{
			AccessToken:          "stored-at",
			RefreshToken:         "stored-rt",
			IDToken:              "stored-idt",
			Scope:                "openid profile",
			AccessTokenExpiresAt: &expiresAt,
		}, []limen.Where{
			limen.Eq(plugin.accountSchema.GetIDField(), account.ID),
		}))

		tokens, err := plugin.GetAccessToken(ctx, user.ID, "test")
		require.NoError(t, err)
		require.NotNil(t, tokens)
		assert.Equal(t, "stored-at", tokens.AccessToken)
		assert.Equal(t, "stored-rt", tokens.RefreshToken)
		assert.Equal(t, "stored-idt", tokens.IDToken)
		assert.Equal(t, "openid profile", tokens.Scope)
		require.NotNil(t, tokens.AccessTokenExpiresAt)
		assert.True(t, tokens.AccessTokenExpiresAt.Equal(expiresAt))
	})
}

func TestRefreshAccessToken(t *testing.T) {
	t.Parallel()

	refresher := &tokenRefresherProvider{
		testProvider: testProvider{name: "refresher"},
		response: &TokenResponse{
			AccessToken: "new-at",
		},
	}

	l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption(), WithProvider(refresher))
	ctx := context.Background()
	user := seedOAuthUser(t, l, "refresh@example.com")

	t.Run("provider not found", func(t *testing.T) {
		tokens, err := plugin.RefreshAccessToken(ctx, user.ID, "missing")
		assert.Nil(t, tokens)
		assert.ErrorIs(t, err, ErrProviderNotFound)
	})

	t.Run("account not found", func(t *testing.T) {
		tokens, err := plugin.RefreshAccessToken(ctx, user.ID, "refresher")
		assert.Nil(t, tokens)
		assert.ErrorIs(t, err, ErrAccountNotFound)
	})

	t.Run("no refresh token", func(t *testing.T) {
		userNoRefresh := seedOAuthUser(t, l, "refresh-empty@example.com")
		now := time.Now()
		require.NoError(t, plugin.core.Create(ctx, plugin.accountSchema, &limen.Account{
			UserID:            userNoRefresh.ID,
			Provider:          "refresher",
			ProviderAccountID: "provider-refresh-empty",
			AccessToken:       "existing-at",
			RefreshToken:      "",
			Scope:             "openid email",
			CreatedAt:         now,
			UpdatedAt:         now,
		}, nil))

		tokens, err := plugin.RefreshAccessToken(ctx, userNoRefresh.ID, "refresher")
		assert.Nil(t, tokens)
		assert.ErrorIs(t, err, ErrNoRefreshToken)
	})

	t.Run("preserves refresh token and scope fallback", func(t *testing.T) {
		user2 := seedOAuthUser(t, l, "refresh-fallback@example.com")
		seedOAuthAccount(t, plugin, user2.ID, "refresher", "provider-refresh-fallback")
		account, err := plugin.findAccountByUserIDAndProvider(ctx, user2.ID, "refresher")
		require.NoError(t, err)
		require.NoError(t, plugin.core.Update(ctx, plugin.accountSchema, &limen.Account{
			RefreshToken: "old-refresh-token",
			Scope:        "old-scope",
		}, []limen.Where{
			limen.Eq(plugin.accountSchema.GetIDField(), account.ID),
		}))

		refresher.response = &TokenResponse{
			AccessToken: "fresh-access-token",
			Scope:       "",
		}

		tokens, err := plugin.RefreshAccessToken(ctx, user2.ID, "refresher")
		require.NoError(t, err)
		require.NotNil(t, tokens)
		assert.Equal(t, "fresh-access-token", tokens.AccessToken)
		assert.Equal(t, "old-refresh-token", tokens.RefreshToken)
		assert.Equal(t, "old-scope", tokens.Scope)
		assert.Equal(t, "old-refresh-token", refresher.lastRefreshToken)

		stored, err := plugin.GetAccessToken(ctx, user2.ID, "refresher")
		require.NoError(t, err)
		assert.Equal(t, "fresh-access-token", stored.AccessToken)
		assert.Equal(t, "old-refresh-token", stored.RefreshToken)
	})
}
