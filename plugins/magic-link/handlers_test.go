package magiclink

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestMagicLinkHandler_RejectsUntrustedRedirectURIs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		fieldName string
	}{
		{
			name:      "redirect_uri",
			body:      `{"email":"u@test.com","redirect_uri":"https://evil.example/welcome"}`,
			fieldName: "redirect_uri",
		},
		{
			name:      "new_user_redirect_uri",
			body:      `{"email":"u@test.com","new_user_redirect_uri":"https://evil.example/welcome"}`,
			fieldName: "new_user_redirect_uri",
		},
		{
			name:      "error_redirect_uri",
			body:      `{"email":"u@test.com","error_redirect_uri":"https://evil.example/error"}`,
			fieldName: "error_redirect_uri",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l, _ := newTestLimenAndPlugin(t)

			req := newJSONRequest(t, http.MethodPost, "/auth/magic-link/signin", tt.body)
			w := httptest.NewRecorder()
			l.Handler().ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Contains(t, w.Body.String(), tt.fieldName+" is not trusted")
		})
	}
}

func TestVerifyMagicLinkHandler_RedirectSelection(t *testing.T) {
	t.Parallel()

	const baseURL = "http://localhost:8080"

	tests := []struct {
		name             string
		email            string
		opts             *RequestMagicLinkOptions
		expectRedirect   bool
		expectedLocation string
	}{
		{
			name:           "returns session JSON when no redirect provided",
			email:          "fallback@test.com",
			opts:           nil,
			expectRedirect: false,
		},
		{
			name:             "uses trusted redirect URI",
			email:            "callback@test.com",
			opts:             &RequestMagicLinkOptions{RedirectURI: baseURL + "/welcome"},
			expectRedirect:   true,
			expectedLocation: baseURL + "/welcome",
		},
		{
			name:  "prefers new-user redirect URI for new users",
			email: "brand-new@test.com",
			opts: &RequestMagicLinkOptions{
				RedirectURI:        baseURL + "/default",
				NewUserRedirectURI: baseURL + "/welcome-new",
			},
			expectRedirect:   true,
			expectedLocation: baseURL + "/welcome-new",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var token string
			l, plugin := newTestLimenAndPlugin(t, WithSendMagicLink(func(msg MagicLinkMessage) {
				token = msg.Token
			}))
			_, err := plugin.RequestMagicLink(context.Background(), tt.email, tt.opts)
			require.NoError(t, err)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/magic-link/verify?token="+token, http.NoBody)
			w := httptest.NewRecorder()
			l.Handler().ServeHTTP(w, req)

			if tt.expectRedirect {
				assert.Equal(t, http.StatusSeeOther, w.Code)
				assert.Equal(t, tt.expectedLocation, w.Header().Get("Location"))
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Empty(t, w.Header().Get("Location"))
				assert.Contains(t, w.Body.String(), tt.email)
			}
			assert.NotEmpty(t, w.Result().Cookies(), "successful verify must set a session cookie")
		})
	}
}

func TestVerifyMagicLinkHandler_MissingToken(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimenAndPlugin(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/magic-link/verify", http.NoBody)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "token is required")
}
