package passkey

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestBeginAuthenticationAllowCredentials(t *testing.T) {
	t.Parallel()

	const credentialID = "dGVzdC1jcmVkZW50aWFsLWlk"

	tests := []struct {
		name               string
		withSession        bool
		withPasskey        bool
		wantAllowCredCount int
	}{
		{
			name:               "no session stays discoverable",
			withSession:        false,
			withPasskey:        false,
			wantAllowCredCount: 0,
		},
		{
			name:               "session without passkeys stays discoverable",
			withSession:        true,
			withPasskey:        false,
			wantAllowCredCount: 0,
		},
		{
			name:               "session with passkeys fills allowCredentials",
			withSession:        true,
			withPasskey:        true,
			wantAllowCredCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin, l := newTestPasskeyPlugin(t)
			user := limen.SeedTestUser(t, l, tt.name+"@example.com")

			if tt.withPasskey {
				seedTestPasskey(t, plugin, user.ID, credentialID)
			}

			req := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/passkeys/begin-authentication",
				http.NoBody,
			)
			if tt.withSession {
				session := limen.SeedTestSession(t, l, user.ID, user.Email)
				req.AddCookie(session.Cookie)
			}

			assertion, challenge, err := plugin.BeginAuthentication(req)
			require.NoError(t, err)
			require.NotNil(t, assertion)
			require.NotNil(t, challenge)

			assert.Len(t, assertion.Response.AllowedCredentials, tt.wantAllowCredCount)
			if tt.wantAllowCredCount > 0 {
				wantID, err := base64.RawURLEncoding.DecodeString(credentialID)
				require.NoError(t, err)
				assert.Equal(t, wantID, []byte(assertion.Response.AllowedCredentials[0].CredentialID))
				assert.NotEmpty(t, challenge.Session.UserID)
			} else {
				assert.Empty(t, challenge.Session.UserID)
			}
		})
	}
}

func TestDiscoverableUserHandlerUsesCredentialID(t *testing.T) {
	t.Parallel()

	plugin, l := newTestPasskeyPlugin(t)
	user := limen.SeedTestUser(t, l, "cred-id@example.com")
	const credentialID = "Y3JlZC1pZC1maXJzdA"
	seedTestPasskey(t, plugin, user.ID, credentialID)

	rawID, err := base64.RawURLEncoding.DecodeString(credentialID)
	require.NoError(t, err)

	got, err := plugin.discoverableUserHandler(t.Context())(rawID, []byte("wrong-handle"))
	require.NoError(t, err)
	require.Equal(t, user.ID, got.(*webAuthnUser).User().ID)

	_, err = plugin.discoverableUserHandler(t.Context())(nil, []byte("wrong-handle"))
	require.ErrorIs(t, err, ErrUnknownPasskey)
}

func TestFinishAuthenticationReturnsUser(t *testing.T) {
	t.Parallel()

	plugin, _ := newTestPasskeyPlugin(t)
	cred := loadFixtureCredential(t)
	column := plugin.core.PublicIDColumn(plugin.core.Schema.User)
	require.NoError(t, plugin.core.DBAction.CreateUser(t.Context(), &limen.User{Email: "assert@example.com"}, map[string]any{
		column: cred.UserHandle,
	}))
	user, err := plugin.core.DBAction.FindUserByEmail(t.Context(), "assert@example.com")
	require.NoError(t, err)
	seedFixturePasskey(t, plugin, user.ID)

	req := requestWithChallenge(
		t,
		plugin,
		http.MethodPost,
		"/passkeys/finish-authentication",
		loadAssertionResponse(t),
		loadAssertionChallenge(t),
	)

	gotUser, err := plugin.FinishAuthentication(req)
	require.NoError(t, err)
	require.Equal(t, user.ID, gotUser.ID)
}
