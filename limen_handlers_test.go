package limen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestHandlersFromLimen(t *testing.T, l *Limen) *limenHandlers {
	t.Helper()
	httpCore := newTestHTTPCore(t, l)
	return newLimenHandlers(httpCore, l.core)
}

func withSessionContext(r *http.Request, session *ValidatedSession) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKeyActiveSession{}, session))
}

func TestGetSession_WithValidSession(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "a@b.com")
	handlers := newTestHandlersFromLimen(t, l)

	user, err := l.core.DBAction.FindUserByEmail(context.Background(), "a@b.com")
	assert.NoError(t, err)

	sess := seedSession(t, l, userID, "a@b.com")
	validatedSession := &ValidatedSession{
		User:    user,
		Session: &Session{Token: sess.Token, UserID: userID},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req = withSessionContext(req, validatedSession)
	w := httptest.NewRecorder()

	handlers.GetSession(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), user.Email)
}

func TestGetSession_WithoutSession(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	handlers := newTestHandlersFromLimen(t, l)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	w := httptest.NewRecorder()

	handlers.GetSession(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListSessions(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "b@c.com")
	handlers := newTestHandlersFromLimen(t, l)

	sess1 := seedSession(t, l, userID, "b@c.com")
	sess2 := seedSession(t, l, userID, "b@c.com")

	user := &User{ID: userID, Email: "b@c.com"}
	validatedSession := &ValidatedSession{
		User:    user,
		Session: &Session{Token: sess1.Token, UserID: userID},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/sessions", nil)
	req = withSessionContext(req, validatedSession)
	w := httptest.NewRecorder()

	handlers.ListSessions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), sess1.Token)
	assert.Contains(t, w.Body.String(), sess2.Token)
}

func TestSignOut(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "c@d.com")
	handlers := newTestHandlersFromLimen(t, l)

	sess := seedSession(t, l, userID, "c@d.com")

	user := &User{ID: userID, Email: "c@d.com"}
	validatedSession := &ValidatedSession{
		User:    user,
		Session: &Session{Token: sess.Token, UserID: userID},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/signout", nil)
	req = withSessionContext(req, validatedSession)
	w := httptest.NewRecorder()

	handlers.SignOut(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "limen_session" && c.MaxAge == -1 {
			found = true
		}
	}
	assert.True(t, found, "session cookie should be cleared")

	sessions, err := l.core.SessionManager.ListSessions(context.Background(), userID)
	assert.NoError(t, err)
	assert.Empty(t, sessions, "session should be deleted from DB")
}

func TestRevokeAllSessions(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	userID := seedUser(t, l, "d@e.com")
	handlers := newTestHandlersFromLimen(t, l)

	sess := seedSession(t, l, userID, "d@e.com")
	seedSession(t, l, userID, "d@e.com")
	seedSession(t, l, userID, "d@e.com")

	user := &User{ID: userID, Email: "d@e.com"}
	validatedSession := &ValidatedSession{
		User:    user,
		Session: &Session{Token: sess.Token, UserID: userID},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/revoke-sessions", nil)
	req = withSessionContext(req, validatedSession)
	w := httptest.NewRecorder()

	handlers.RevokeAllSessions(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	sessions, err := l.core.SessionManager.ListSessions(context.Background(), userID)
	assert.NoError(t, err)
	assert.Empty(t, sessions, "all sessions should be deleted from DB")
}

func TestSignOut_WithoutSession(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	handlers := newTestHandlersFromLimen(t, l)

	req := httptest.NewRequest(http.MethodPost, "/auth/signout", nil)
	w := httptest.NewRecorder()

	handlers.SignOut(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
