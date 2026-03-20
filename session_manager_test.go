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

func typedSessionManager(t *testing.T, l *Limen) *opaqueSessionManager {
	t.Helper()
	sm, ok := l.core.SessionManager.(*opaqueSessionManager)
	require.True(t, ok, "expected opaqueSessionManager")
	return sm
}

func TestOpaqueSessionManager_CreateSession(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "a@b.com")

	req := httptest.NewRequest(http.MethodPost, "/signin", nil)
	auth := &AuthenticationResult{User: &User{ID: userID, Email: "a@b.com"}}

	result, err := l.core.SessionManager.CreateSession(context.Background(), req, auth, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Token)
	assert.NotNil(t, result.Cookie)

	validateReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	validateReq.AddCookie(result.Cookie)
	validated, err := l.core.SessionManager.ValidateSession(context.Background(), validateReq)
	assert.NoError(t, err)
	assert.Equal(t, userID, validated.User.ID)
}

func TestOpaqueSessionManager_CreateShortSession(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithSessionConfig(t, WithSessionShortDuration(1*time.Hour))
	userID := seedUser(t, l, "b@c.com")

	req := httptest.NewRequest(http.MethodPost, "/signin", nil)
	auth := &AuthenticationResult{User: &User{ID: userID, Email: "b@c.com"}}

	result, err := l.core.SessionManager.CreateSession(context.Background(), req, auth, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Token)

	validateReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	validateReq.AddCookie(result.Cookie)
	validated, err := l.core.SessionManager.ValidateSession(context.Background(), validateReq)
	assert.NoError(t, err)
	ttl := validated.Session.ExpiresAt.Sub(validated.Session.CreatedAt)
	assert.InDelta(t, time.Hour.Seconds(), ttl.Seconds(), 1)
}

func TestOpaqueSessionManager_ValidateSession(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "c@d.com")
	sess := seedSession(t, l, userID, "c@d.com")

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(sess.Cookie)

	validated, err := l.core.SessionManager.ValidateSession(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, validated)
	assert.Equal(t, userID, validated.User.ID)
	assert.Equal(t, sess.Token, validated.Session.Token)
}

func TestOpaqueSessionManager_ValidateSession_Expired(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "exp@test.com")

	ctx := context.Background()
	past := time.Now().Add(-48 * time.Hour)
	err := l.core.DBAction.CreateSession(ctx, &Session{
		Token:      "expired-token",
		UserID:     userID,
		CreatedAt:  past.Add(-7 * 24 * time.Hour),
		ExpiresAt:  past,
		LastAccess: past,
	}, nil)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "limen_session", Value: "expired-token"})

	_, err = l.core.SessionManager.ValidateSession(context.Background(), req)
	assert.ErrorIs(t, err, ErrSessionExpired)
}

func TestOpaqueSessionManager_ValidateSession_NotFound(t *testing.T) {
	t.Parallel()

	testData := []struct {
		name   string
		cookie *http.Cookie
	}{
		{name: "nonexistent", cookie: &http.Cookie{Name: "limen_session", Value: "nonexistent"}},
		{name: "empty", cookie: nil},
	}
	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			l := newTestLimen(t)
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			_, err := l.core.SessionManager.ValidateSession(context.Background(), req)
			assert.ErrorIs(t, err, ErrSessionNotFound)
		})
	}
}

func TestOpaqueSessionManager_ExtractToken_Bearer(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithSessionConfig(t, WithBearerEnabled())
	sm := typedSessionManager(t, l)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer my-token-123")

	token, err := sm.extractToken(req)
	assert.NoError(t, err)
	assert.Equal(t, "my-token-123", token)
}

func TestOpaqueSessionManager_ExtractToken_CookiePriority(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithSessionConfig(t, WithBearerEnabled())
	sm := typedSessionManager(t, l)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "limen_session", Value: "cookie-token"})
	req.Header.Set("Authorization", "Bearer bearer-token")

	token, err := sm.extractToken(req)
	assert.NoError(t, err)
	assert.Equal(t, "cookie-token", token, "cookie should take priority over bearer")
}

func TestOpaqueSessionManager_RevokeSession(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "rev@test.com")
	sess := seedSession(t, l, userID, "rev@test.com")
	sm := typedSessionManager(t, l)
	err := sm.RevokeSession(context.Background(), sess.Token)
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(sess.Cookie)
	_, err = sm.ValidateSession(context.Background(), req)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestOpaqueSessionManager_RevokeAllSessions(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "revall@test.com")

	for range 3 {
		seedSession(t, l, userID, "revall@test.com")
	}

	sm := typedSessionManager(t, l)
	err := sm.RevokeAllSessions(context.Background(), userID)
	assert.NoError(t, err)

	sessions, err := sm.ListSessions(context.Background(), userID)
	assert.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestOpaqueSessionManager_ListSessions(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "list@test.com")

	seedSession(t, l, userID, "list@test.com")
	seedSession(t, l, userID, "list@test.com")

	sessions, err := l.core.SessionManager.ListSessions(context.Background(), userID)
	assert.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestResolveSessionPolicy_ShortSession(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithSessionConfig(t, WithSessionShortDuration(1*time.Hour))
	sm := typedSessionManager(t, l)

	now := time.Now()
	shortSession := &Session{
		CreatedAt: now,
		ExpiresAt: now.Add(1 * time.Hour),
	}

	policy := sm.resolveSessionPolicy(shortSession)
	assert.Equal(t, time.Duration(0), policy.UpdateAge, "short sessions should not be extended")
}

func TestResolveSessionPolicy_NormalSession(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	sm := typedSessionManager(t, l)

	now := time.Now()
	normalSession := &Session{
		CreatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}

	policy := sm.resolveSessionPolicy(normalSession)
	assert.Equal(t, 24*time.Hour, policy.UpdateAge)
}
