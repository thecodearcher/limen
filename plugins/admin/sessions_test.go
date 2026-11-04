package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestAdminPlugin_ListUserSessions(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	first := limen.SeedTestSession(t, l, target.ID, target.Email)
	second := limen.SeedTestSession(t, l, target.ID, target.Email)

	sessions, err := plugin.ListUserSessions(t.Context(), actor, target.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	assert.ElementsMatch(t, []string{first.Token, second.Token}, []string{sessions[0].ID, sessions[1].ID})
}

func seedImpersonatedSession(t *testing.T, plugin *adminPlugin, user, impersonator *limen.User) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", http.NoBody)
	_, err := plugin.core.SessionManager.CreateSession(t.Context(), req, &limen.AuthenticationResult{User: user},
		limen.WithSessionData(map[limen.SchemaField]any{SessionSchemaImpersonatedByField: impersonator.ID}),
	)
	require.NoError(t, err)
}

func TestAdminPlugin_ListUserSessions_ImpersonatedBy(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	limen.SeedTestSession(t, l, target.ID, target.Email)
	seedImpersonatedSession(t, plugin, target, actor)

	sessions, err := plugin.ListUserSessions(t.Context(), actor, target.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	var impersonators []string
	for _, session := range sessions {
		if session.ImpersonatedBy != nil {
			impersonators = append(impersonators, session.ImpersonatedBy.Email)
		}
	}
	assert.Equal(t, []string{actor.Email}, impersonators)
}

func TestAdminPlugin_RevokeSession(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	limen.SeedTestSession(t, l, target.ID, target.Email)
	limen.SeedTestSession(t, l, target.ID, target.Email)
	sessions, err := plugin.ListUserSessions(t.Context(), actor, target.ID)
	require.NoError(t, err)

	require.NoError(t, plugin.RevokeSession(t.Context(), actor, sessions[0].ID))

	remaining, err := plugin.ListUserSessions(t.Context(), actor, target.ID)
	require.NoError(t, err)
	assert.Equal(t, []UserSession{sessions[1]}, remaining)
}

func TestAdminPlugin_RevokeUserSessions(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	limen.SeedTestSession(t, l, target.ID, target.Email)
	limen.SeedTestSession(t, l, target.ID, target.Email)

	require.NoError(t, plugin.RevokeUserSessions(t.Context(), actor, target.ID))

	remaining, err := plugin.ListUserSessions(t.Context(), actor, target.ID)
	require.NoError(t, err)
	assert.Empty(t, remaining)
}
