package limen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", http.NoBody)
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

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", http.NoBody)
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

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/sessions", http.NoBody)
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

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/signout", http.NoBody)
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

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/revoke-sessions", http.NoBody)
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

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/signout", http.NoBody)
	w := httptest.NewRecorder()

	handlers.SignOut(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequestEmailVerificationHandler_RejectsBareEmailByDefault(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithEmailVerification(t)
	req := jsonRequest(t, http.MethodPost, "/auth/email-verifications", `{"email":"ghost@test.com"}`)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequestEmailVerificationHandler_RequestByEmailAlwaysOK(t *testing.T) {
	t.Parallel()

	var sentTo string
	l := newTestLimenWithEmailVerification(t,
		WithEmailVerification(
			WithEmailVerificationRequestByEmail(),
			WithSendEmailVerificationMail(func(email, _ string) {
				sentTo = email
			}),
		),
	)
	SeedTestUser(t, l, "unverified@test.com")
	verified := SeedTestUser(t, l, "verified@test.com")
	verification, err := l.RequestEmailVerification(t.Context(), verified, false)
	require.NoError(t, err)
	email, err := l.VerifyEmail(t.Context(), verification.Value)
	require.NoError(t, err)
	require.Equal(t, verified.Email, email)

	tests := []struct {
		name       string
		email      string
		wantSentTo string
	}{
		{name: "unverified account", email: "unverified@test.com", wantSentTo: "unverified@test.com"},
		{name: "unknown email", email: "ghost@test.com"},
		{name: "verified account", email: "verified@test.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentTo = ""
			req := jsonRequest(t, http.MethodPost, "/auth/email-verifications", `{"email":"`+tt.email+`"}`)
			w := httptest.NewRecorder()
			l.Handler().ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.wantSentTo, sentTo)
		})
	}
}

func TestRequestEmailVerificationHandler_ProofOverridesBodyEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		attachProof func(t *testing.T, l *Limen, owner *User, req *http.Request)
	}{
		{
			name: "challenge cookie",
			attachProof: func(t *testing.T, l *Limen, owner *User, req *http.Request) {
				issued := httptest.NewRecorder()
				require.NoError(t, l.IssueEmailVerificationChallenge(issued, owner.Email))
				copyResponseCookies(t, issued, req)
			},
		},
		{
			name: "session",
			attachProof: func(t *testing.T, l *Limen, owner *User, req *http.Request) {
				req.AddCookie(SeedTestSession(t, l, owner.ID, owner.Email).Cookie)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var sentTo string
			l := newTestLimenWithEmailVerification(t,
				WithEmailVerification(
					WithEmailVerificationRequestByEmail(),
					WithSendEmailVerificationMail(func(email, _ string) {
						sentTo = email
					}),
				),
			)
			owner := SeedTestUser(t, l, "owner@test.com")
			SeedTestUser(t, l, "other@test.com")

			req := jsonRequest(t, http.MethodPost, "/auth/email-verifications", `{"email":"other@test.com"}`)
			tt.attachProof(t, l, owner, req)
			w := httptest.NewRecorder()
			l.Handler().ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "owner@test.com", sentTo)
		})
	}
}

func TestVerifyEmailHandler_ClearsChallengeAndAutoSignsIn(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithEmailVerification(t,
		WithEmailVerification(WithAutoSignInAfterVerification()),
	)
	user := SeedTestUser(t, l, "autosign@test.com")
	verification, err := l.RequestEmailVerification(t.Context(), user, false)
	require.NoError(t, err)

	issued := httptest.NewRecorder()
	require.NoError(t, l.IssueEmailVerificationChallenge(issued, user.Email))

	req := jsonRequest(t, http.MethodPost, "/auth/verify-email", `{"token":"`+verification.Value+`"}`)
	copyResponseCookies(t, issued, req)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, findCookie(w, "limen_session"))
	cleared := findCookie(w, defaultEmailVerificationChallengeCookieName)
	require.NotNil(t, cleared)
	assert.Negative(t, cleared.MaxAge)
}
