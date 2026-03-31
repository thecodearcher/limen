package oauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestListAccountsForUser(t *testing.T) {
	t.Parallel()

	l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
	ctx := context.Background()

	user := seedOAuthUser(t, l, "list@example.com")
	seedOAuthAccount(t, plugin, user.ID, "google", "g-123")
	seedOAuthAccount(t, plugin, user.ID, "github", "gh-456")

	accounts, err := plugin.ListAccountsForUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)

	providers := map[string]bool{}
	for _, acc := range accounts {
		providers[acc.Provider] = true
	}
	assert.True(t, providers["google"])
	assert.True(t, providers["github"])
}

func TestUnlinkAccount(t *testing.T) {
	t.Parallel()

	l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
	ctx := context.Background()

	user := seedOAuthUser(t, l, "unlink@example.com")
	seedOAuthAccount(t, plugin, user.ID, "google", "g-unlink")
	seedOAuthAccount(t, plugin, user.ID, "github", "gh-unlink")

	err := plugin.UnlinkAccount(ctx, user, "google")
	require.NoError(t, err)

	accounts, err := plugin.ListAccountsForUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, accounts, 1)
	assert.Equal(t, "github", accounts[0].Provider)
}

func TestUnlinkAccount_CannotUnlinkOnly(t *testing.T) {
	t.Parallel()

	l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
	ctx := context.Background()

	user := seedOAuthUser(t, l, "only@example.com")
	seedOAuthAccount(t, plugin, user.ID, "google", "g-only")

	err := plugin.UnlinkAccount(ctx, user, "google")
	assert.ErrorIs(t, err, ErrCannotUnlinkOnlyAccount)
}

func TestUnlinkAccount_NoAccounts(t *testing.T) {
	t.Parallel()

	l, plugin := newTestOAuthPlugin(t, WithDisableTokensEncryption())
	ctx := context.Background()

	user := seedOAuthUser(t, l, "noacc@example.com")

	err := plugin.UnlinkAccount(ctx, &limen.User{ID: user.ID}, "google")
	assert.ErrorIs(t, err, ErrAccountNotFound)
}
