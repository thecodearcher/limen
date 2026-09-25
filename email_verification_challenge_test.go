package limen

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailVerificationChallenge_Expired(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		l := newTestLimenWithEmailVerification(t,
			WithEmailVerification(WithVerificationChallengeTTL(time.Minute)),
		)
		user := SeedTestUser(t, l, "expired@test.com")

		w := httptest.NewRecorder()
		require.NoError(t, l.IssueEmailVerificationChallenge(w, user.Email))
		time.Sleep(time.Minute)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/email-verifications", http.NoBody)
		copyResponseCookies(t, w, req)

		_, err := l.ResolveEmailVerificationChallenge(req)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})
}
