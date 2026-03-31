package credentialpassword

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func newTestLimenWithPlugin(t *testing.T, opts ...ConfigOption) *credentialPasswordPlugin {
	t.Helper()

	_, plugin := newTestLimenAndPlugin(t, opts...)
	return plugin
}

func newTestLimenAndPlugin(t *testing.T, opts ...ConfigOption) (*limen.Limen, *credentialPasswordPlugin) {
	t.Helper()

	plugin := New(opts...)
	l, _ := limen.NewTestLimen(t, plugin)
	return l, plugin
}

func seedTestUser(t *testing.T, api API, email, password string) *limen.User {
	t.Helper()
	pw := password
	result, err := api.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    email,
		Password: &pw,
	}, nil)
	require.NoError(t, err)
	return result.User
}

func seedOAuthTestUser(t *testing.T, plugin *credentialPasswordPlugin, email string) *limen.User {
	t.Helper()

	err := plugin.dbAction.CreateUser(context.Background(), &limen.User{Email: email}, nil)
	require.NoError(t, err)

	user, err := plugin.dbAction.FindUserByEmail(context.Background(), email)
	require.NoError(t, err)
	return user
}
