package twofactor

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func newTestPlugin(t *testing.T, revokeOthers bool) (*twoFactorPlugin, *limen.Limen) {
	t.Helper()
	l, core := limen.NewTestLimen(t)
	p := &twoFactorPlugin{
		core:   core,
		config: &config{revokeOtherSessionsOnStateChange: revokeOthers},
	}
	return p, l
}

func seedUserAndSession(t *testing.T, l *limen.Limen, email string) (*limen.User, *limen.SessionResult) {
	t.Helper()
	user := limen.SeedTestUser(t, l, email)
	sess := limen.SeedTestSession(t, l, user.ID, email)
	return user, sess
}

func TestRotateSession_RevokesCurrentAndIssuesNew(t *testing.T) {
	t.Parallel()
	p, l := newTestPlugin(t, false)
	user, oldSess := seedUserAndSession(t, l, "rotate@example.com")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/two-factor/finalize-setup", nil)
	req.AddCookie(oldSess.Cookie)
	w := httptest.NewRecorder()

	validated := &limen.ValidatedSession{
		User:    user,
		Session: &limen.Session{Token: oldSess.Token, UserID: user.ID},
	}

	_, newSess, err := p.rotateSession(req, w, validated)
	require.NoError(t, err)
	assert.NotEqual(t, oldSess.Token, newSess.Token, "new session token must differ from old")

	_, err = l.GetSession(requestWithToken(t, oldSess))
	assert.Error(t, err, "old session token must be invalid after rotation")

	_, err = l.GetSession(requestWithToken(t, newSess))
	assert.NoError(t, err, "new session token must be valid")
}

func TestRotateSession_RevokeOthersEnabled(t *testing.T) {
	t.Parallel()
	p, l := newTestPlugin(t, true)
	user, currentSess := seedUserAndSession(t, l, "others@example.com")
	otherSess := limen.SeedTestSession(t, l, user.ID, "others@example.com")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/two-factor/finalize-setup", nil)
	req.AddCookie(currentSess.Cookie)
	w := httptest.NewRecorder()

	validated := &limen.ValidatedSession{
		User:    user,
		Session: &limen.Session{Token: currentSess.Token, UserID: user.ID},
	}

	_, newSess, err := p.rotateSession(req, w, validated)
	require.NoError(t, err)
	assert.NotEqual(t, currentSess.Token, newSess.Token)

	_, err = l.GetSession(requestWithToken(t, currentSess))
	assert.Error(t, err, "old current session must be revoked")

	_, err = l.GetSession(requestWithToken(t, otherSess))
	assert.Error(t, err, "other sessions must be revoked when revokeOtherSessions is true")

	_, err = l.GetSession(requestWithToken(t, newSess))
	assert.NoError(t, err, "newly issued session must be valid")
}

func TestRotateSession_RevokeOthersDisabled_KeepsOtherSessions(t *testing.T) {
	t.Parallel()
	p, l := newTestPlugin(t, false)
	user, currentSess := seedUserAndSession(t, l, "keep@example.com")
	otherSess := limen.SeedTestSession(t, l, user.ID, "keep@example.com")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/two-factor/disable", nil)
	req.AddCookie(currentSess.Cookie)
	w := httptest.NewRecorder()

	validated := &limen.ValidatedSession{
		User:    user,
		Session: &limen.Session{Token: currentSess.Token, UserID: user.ID},
	}

	_, newSess, err := p.rotateSession(req, w, validated)
	require.NoError(t, err)

	_, err = l.GetSession(requestWithToken(t, currentSess))
	assert.Error(t, err, "old current session must be revoked")

	_, err = l.GetSession(requestWithToken(t, otherSess))
	assert.NoError(t, err, "other sessions must be preserved when revokeOtherSessions is false")

	_, err = l.GetSession(requestWithToken(t, newSess))
	assert.NoError(t, err, "newly issued session must be valid")
}

func requestWithToken(t *testing.T, sess *limen.SessionResult) *http.Request {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/session", nil)
	if sess.Cookie != nil {
		r.AddCookie(sess.Cookie)
	}
	return r
}
