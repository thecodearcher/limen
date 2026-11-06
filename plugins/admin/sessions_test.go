package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
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

func validatedSession(t *testing.T, plugin *adminPlugin, user *limen.User) *limen.ValidatedSession {
	t.Helper()
	sessions, err := plugin.core.SessionManager.ListSessions(t.Context(), user.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	return &limen.ValidatedSession{User: user, Session: &sessions[0]}
}

func TestAdminPlugin_ImpersonateUser(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t, WithImpersonationDuration(10*time.Minute))
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	limen.SeedTestSession(t, l, actor.ID, actor.Email)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/impersonate", http.NoBody)

	user, result, err := plugin.ImpersonateUser(t.Context(), req, validatedSession(t, plugin, actor), target.ID)
	require.NoError(t, err)
	assert.Equal(t, target.ID, user.ID)

	sessions, err := plugin.ListUserSessions(t.Context(), actor, target.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, result.Token, sessions[0].ID)
	assert.Equal(t, actor.Email, sessions[0].ImpersonatedBy.Email)
	assert.WithinDuration(t, time.Now().Add(10*time.Minute), sessions[0].ExpiresAt, time.Minute)
}

func TestAdminPlugin_ImpersonateUser_Access(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name          string
		opts          []ConfigOption
		actorRole     string
		targetRole    string
		self          bool
		impersonating bool
		banned        bool
		wantErr       error
	}{
		{name: "actor without permission", actorRole: roleNameUser, wantErr: ErrInsufficientPermission},
		{name: "self", actorRole: roleNameAdmin, self: true, wantErr: ErrCannotImpersonateSelf},
		{name: "admin target", actorRole: roleNameAdmin, targetRole: roleNameAdmin, wantErr: ErrInsufficientPermission},
		{name: "admin target with impersonate-admins", actorRole: "owner", targetRole: roleNameAdmin},
		{name: "already impersonating", actorRole: roleNameAdmin, impersonating: true, wantErr: ErrAlreadyImpersonating},
		{name: "banned target", actorRole: roleNameAdmin, banned: true, wantErr: ErrUserBanned},
		{name: "secondary admin role target", actorRole: roleNameAdmin, targetRole: "support", opts: []ConfigOption{WithAdminRoles(roleNameAdmin, "support")}, wantErr: ErrInsufficientPermission},
		{name: "impersonation disabled", actorRole: roleNameAdmin, opts: []ConfigOption{WithImpersonation(false)}, wantErr: ErrImpersonationNotEnabled},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestAdminPluginWithRoles(t, map[string]access.Permissions{
				"owner":   access.P("user:impersonate,impersonate-admins"),
				"support": access.P("user:read"),
			}, tt.opts...)

			actor := limen.SeedTestUser(t, l, "actor@test.com")
			assignRole(t, plugin, actor.ID, tt.actorRole)
			target := limen.SeedTestUser(t, l, "target@test.com")
			if tt.targetRole != "" {
				assignRole(t, plugin, target.ID, tt.targetRole)
			}
			if tt.self {
				target = actor
			}
			if tt.banned {
				_, err := plugin.BanUser(t.Context(), actor, target.ID, &BanUserRequest{})
				require.NoError(t, err)
			}
			if tt.impersonating {
				seedImpersonatedSession(t, plugin, actor, limen.SeedTestUser(t, l, "other@test.com"))
			} else {
				limen.SeedTestSession(t, l, actor.ID, actor.Email)
			}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/impersonate", http.NoBody)

			_, _, err := plugin.ImpersonateUser(t.Context(), req, validatedSession(t, plugin, actor), target.ID)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAdminPlugin_StopImpersonating(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	actor := seedAdmin(t, l, plugin, "actor@test.com")
	target := limen.SeedTestUser(t, l, "target@test.com")
	seedImpersonatedSession(t, plugin, target, actor)

	impersonator, err := plugin.StopImpersonating(t.Context(), validatedSession(t, plugin, target))
	require.NoError(t, err)
	assert.Equal(t, actor.ID, impersonator.ID)

	sessions, err := plugin.ListUserSessions(t.Context(), actor, target.ID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestAdminPlugin_StopImpersonating_NotImpersonating(t *testing.T) {
	t.Parallel()

	l, plugin := newTestAdminPlugin(t)
	user := limen.SeedTestUser(t, l, "user@test.com")
	limen.SeedTestSession(t, l, user.ID, user.Email)

	_, err := plugin.StopImpersonating(t.Context(), validatedSession(t, plugin, user))
	assert.ErrorIs(t, err, ErrNotImpersonating)
}
