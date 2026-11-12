package limen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimenCore_GetPlugin_NotFound(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	_, ok := l.core.GetPlugin("nonexistent")
	assert.False(t, ok)
}

func TestLimenCore_GetBaseURLWithPluginPath_NoPlugin(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	result := l.core.GetBaseURLWithPluginPath("nonexistent", "/foo")
	assert.Empty(t, result, "should return empty string when plugin not found")
}

func TestLimenCore_CreateSession(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "create-session@test.com")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", http.NoBody)
	w := httptest.NewRecorder()
	auth := &AuthenticationResult{User: &User{ID: userID, Email: "create-session@test.com"}}

	result, err := l.core.CreateSession(context.Background(), req, w, auth)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Token)
	assert.NotNil(t, result.Cookie)
}

type guardPlugin struct {
	testPlugin
	blockedUserID any
}

func (p *guardPlugin) Name() PluginName { return "guard-test" }

func (p *guardPlugin) BeforeSessionCreate(_ context.Context, auth *AuthenticationResult) error {
	if auth.User.ID == p.blockedUserID {
		return ErrForbidden
	}
	return nil
}

func TestLimenCore_CreateSession_Guard(t *testing.T) {
	t.Parallel()

	guard := &guardPlugin{}
	l := newTestLimen(t, guard)
	blocked := seedUser(t, l, "blocked@test.com")
	allowed := seedUser(t, l, "allowed@test.com")
	guard.blockedUserID = blocked

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", http.NoBody)

	_, err := l.core.CreateSession(t.Context(), req, httptest.NewRecorder(), &AuthenticationResult{User: &User{ID: blocked}})
	assert.ErrorIs(t, err, ErrForbidden)
	sessions, err := l.core.SessionManager.ListSessions(t.Context(), blocked)
	assert.NoError(t, err)
	assert.Empty(t, sessions)

	result, err := l.core.CreateSession(t.Context(), req, httptest.NewRecorder(), &AuthenticationResult{User: &User{ID: allowed}})
	require.NoError(t, err)
	assert.NotEmpty(t, result.Token)
}

func TestLimen_Use_Panics_NotRegistered(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	assert.Panics(t, func() {
		Use[Plugin](l, "nonexistent")
	})
}

func TestLimen_TryUse_NotRegistered(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	_, ok := TryUse[Plugin](l, "nonexistent")
	assert.False(t, ok)
}

func TestLimenCore_Use_ReturnsPlugin(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t, newTestPlugin(t))
	plugin := Use[*testPlugin](l, "test")
	assert.Equal(t, "test-method-on-plugin", plugin.TestMethodOnPlugin())
}

func TestLimenHTTPCore_IsTrustedOrigin(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	httpCore := newTestHTTPCore(t, l)
	httpCore.trustedOriginsPatterns = compileTrustedOrigins("http://localhost:8080", "https://*.example.com", "test.com")

	assert.True(t, httpCore.IsTrustedOrigin("http://localhost:8080"))
	assert.True(t, httpCore.IsTrustedOrigin("https://app.example.com"))
	assert.True(t, httpCore.IsTrustedOrigin("https://test.com"))
	assert.True(t, httpCore.IsTrustedOrigin("http://localhost:8080/foo/bar"))
	assert.True(t, httpCore.IsTrustedOrigin("https://app.example.com/foo/bar"))
	assert.False(t, httpCore.IsTrustedOrigin("http://app.example.com"))
	assert.False(t, httpCore.IsTrustedOrigin("https://example.com"))
	assert.False(t, httpCore.IsTrustedOrigin("http://localhost:2080"))
	assert.False(t, httpCore.IsTrustedOrigin("https://evil.com"))
}

func rotateValidatedSession(user *User, sess *SessionResult) *ValidatedSession {
	return &ValidatedSession{
		User:    user,
		Session: &Session{Token: sess.Token, UserID: user.ID},
	}
}

func rotateRequest(t *testing.T, sess *SessionResult) *http.Request {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/session", nil)
	if sess.Cookie != nil {
		r.AddCookie(sess.Cookie)
	}
	return r
}

func TestRotateSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		revokeAll        bool
		wantOtherRevoked bool
	}{
		{name: "revoke current only", revokeAll: false, wantOtherRevoked: false},
		{name: "revoke all sessions", revokeAll: true, wantOtherRevoked: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, core := NewTestLimen(t)
			email := tt.name + "@example.com"
			user := SeedTestUser(t, l, email)
			currentSess := SeedTestSession(t, l, user.ID, email)
			otherSess := SeedTestSession(t, l, user.ID, email)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/rotate", nil)
			req.AddCookie(currentSess.Cookie)
			w := httptest.NewRecorder()

			_, newSess, err := core.RotateSession(req, w, rotateValidatedSession(user, currentSess), tt.revokeAll)
			require.NoError(t, err)
			assert.NotEqual(t, currentSess.Token, newSess.Token, "new session token must differ from old")

			_, err = l.GetSession(rotateRequest(t, currentSess))
			assert.Error(t, err, "current session must be revoked after rotation")

			_, err = l.GetSession(rotateRequest(t, otherSess))
			if tt.wantOtherRevoked {
				assert.Error(t, err, "other sessions must be revoked when revokeAll is true")
			} else {
				assert.NoError(t, err, "other sessions must be preserved when revokeAll is false")
			}

			_, err = l.GetSession(rotateRequest(t, newSess))
			assert.NoError(t, err, "newly issued session must be valid")
		})
	}
}

func TestLimenCore_DeleteUser(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "delete@test.com")
	seedSession(t, l, userID, "delete@test.com")
	err := l.core.Create(t.Context(), l.core.Schema.Account, &Account{
		UserID:            userID,
		Provider:          "github",
		ProviderAccountID: "1",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}, nil)
	require.NoError(t, err)

	require.NoError(t, l.core.DeleteUser(t.Context(), userID))

	_, err = l.core.DBAction.FindUserByID(t.Context(), userID)
	assert.ErrorIs(t, err, ErrRecordNotFound)
	sessions, err := l.core.SessionManager.ListSessions(t.Context(), userID)
	assert.NoError(t, err)
	assert.Empty(t, sessions)
	_, err = l.core.FindOne(t.Context(), l.core.Schema.Account, []Where{
		Eq(l.core.Schema.Account.GetUserIDField(), userID),
	}, nil)
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

func TestLimenCore_DeleteUser_NotFound(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)

	err := l.core.DeleteUser(t.Context(), "missing")
	assert.ErrorIs(t, err, ErrRecordNotFound)
}
