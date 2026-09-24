package credentialpassword

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func newJSONRequest(t *testing.T, path, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// liveCookie returns the named cookie unless the response deletes it.
func liveCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name && cookie.MaxAge >= 0 && cookie.Value != "" {
			return cookie
		}
	}
	return nil
}
