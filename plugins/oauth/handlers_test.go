package oauth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormPostCallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		requestURL       string
		body             string
		contentType      string
		wantStatus       int
		wantPath         string
		wantQuery        map[string]string
		absentKeys       []string
		assertNoLocation bool
	}{
		{
			name:        "redirects POST form params to GET query string",
			requestURL:  "/oauth/test/callback",
			body:        url.Values{"code": {"auth-code-123"}, "state": {"state-token-456"}}.Encode(),
			contentType: "application/x-www-form-urlencoded",
			wantStatus:  http.StatusSeeOther,
			wantPath:    "/oauth/test/callback",
			wantQuery: map[string]string{
				"code":  "auth-code-123",
				"state": "state-token-456",
			},
		},
		{
			name:        "forwards error params from form body",
			requestURL:  "/oauth/test/callback",
			body:        url.Values{"state": {"state-token"}, "error": {"access_denied"}, "error_description": {"user canceled"}}.Encode(),
			contentType: "application/x-www-form-urlencoded",
			wantStatus:  http.StatusSeeOther,
			wantPath:    "/oauth/test/callback",
			wantQuery: map[string]string{
				"state":             "state-token",
				"error":             "access_denied",
				"error_description": "user canceled",
			},
			absentKeys: []string{"code"},
		},
		{
			name:        "empty form values are not included in redirect",
			requestURL:  "/oauth/test/callback",
			body:        url.Values{"state": {"state-only"}}.Encode(),
			contentType: "application/x-www-form-urlencoded",
			wantStatus:  http.StatusSeeOther,
			wantPath:    "/oauth/test/callback",
			wantQuery: map[string]string{
				"state": "state-only",
			},
			absentKeys: []string{"code", "error", "error_description"},
		},
		{
			name:        "preserves existing query params and appends all form params",
			requestURL:  "/oauth/test/callback?client_hint=abc&foo=bar",
			body:        url.Values{"code": {"auth-code-123"}, "state": {"state-token-456"}, "custom_param": {"custom-value"}}.Encode(),
			contentType: "application/x-www-form-urlencoded",
			wantStatus:  http.StatusSeeOther,
			wantPath:    "/oauth/test/callback",
			wantQuery: map[string]string{
				"client_hint":  "abc",
				"foo":          "bar",
				"code":         "auth-code-123",
				"state":        "state-token-456",
				"custom_param": "custom-value",
			},
		},
		{
			name:             "returns error when form body is malformed",
			requestURL:       "/oauth/test/callback",
			body:             "state=%zz",
			contentType:      "application/x-www-form-urlencoded",
			wantStatus:       http.StatusInternalServerError,
			assertNoLocation: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handlers := newOAuthHandlersForTest(t)
			req := newFormPostRequest(t, tt.requestURL, tt.body, tt.contentType)
			rec := httptest.NewRecorder()

			handlers.FormPostCallback(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)

			location := rec.Header().Get("Location")
			if tt.assertNoLocation {
				assert.Empty(t, location)
				return
			}

			require.NotEmpty(t, location)
			parsed, err := url.Parse(location)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, parsed.Path)

			for key, expected := range tt.wantQuery {
				assert.Equal(t, expected, parsed.Query().Get(key))
			}
			for _, key := range tt.absentKeys {
				assert.False(t, parsed.Query().Has(key))
			}
		})
	}
}

func newOAuthHandlersForTest(t *testing.T) *oauthHandlers {
	t.Helper()
	l, plugin := newTestOAuthPlugin(t)
	_ = l.Handler()
	return newOAuthHandlers(plugin, plugin.httpCore)
}

func newFormPostRequest(t *testing.T, requestURL, body, contentType string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, requestURL, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	return req
}
