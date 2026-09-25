package credentialpassword

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireEmailVerification_IssuesChallengeWithoutSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		seedUser   bool
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "sign-up",
			path:       "/auth/signup/credential",
			body:       `{"email":"user@test.com","password":"Password1"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unverified sign-in",
			seedUser:   true,
			path:       "/auth/signin/credential",
			body:       `{"credential":"user@test.com","password":"Password1"}`,
			wantStatus: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, plugin := newTestLimenAndPlugin(t, WithRequireEmailVerification(true))
			if tt.seedUser {
				seedTestUser(t, plugin, "user@test.com", "Password1")
			}

			w := httptest.NewRecorder()
			l.Handler().ServeHTTP(w, newJSONRequest(t, tt.path, tt.body))

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.NotNil(t, liveCookie(w, "limen_email_verify"))
			assert.Nil(t, liveCookie(w, "limen_session"))
			if tt.wantStatus == http.StatusForbidden {
				assert.Contains(t, w.Body.String(), `"code":"email_not_verified"`)
			}
		})
	}
}
