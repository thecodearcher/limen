package passkey

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginInit_DerivesRPIDFromBaseURL(t *testing.T) {
	t.Parallel()

	_, plugin := newTestLimenAndPlugin(t)
	require.NotNil(t, plugin.webauthn)
	// The default test Limen base URL is http://localhost:8080.
	assert.Equal(t, "localhost", plugin.config.rpID)
	assert.NotEmpty(t, plugin.config.origins, "origins should default to the base URL")
}

func TestPluginInit_RespectsWithRPID(t *testing.T) {
	t.Parallel()

	_, plugin := newTestLimenAndPlugin(t,
		WithRPID("example.com"),
		WithRPName("Example"),
		WithOrigins("https://example.com"),
	)
	assert.Equal(t, "example.com", plugin.config.rpID)
	assert.Equal(t, "Example", plugin.config.rpName)
	assert.Equal(t, []string{"https://example.com"}, plugin.config.origins)
}

func TestGenerateRegistrationOptions_RequiresSession(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimenAndPlugin(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/passkey/generate-register-options", http.NoBody)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}

func TestGenerateAuthenticationOptions_PublicEndpoint(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimenAndPlugin(t,
		WithRPID("localhost"),
		WithOrigins("http://localhost:8080"),
	)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/passkey/generate-authenticate-options", http.NoBody)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// The response must include a challenge field.
	assert.Contains(t, w.Body.String(), "challenge")
	// A signed challenge cookie must be set so the verify endpoint
	// can correlate the assertion to its stored state.
	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == defaultChallengeCookieName {
			found = true
			break
		}
	}
	assert.True(t, found, "challenge cookie should be set")
}

func TestVerifyAuthentication_MissingCookieReturnsChallengeNotFound(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimenAndPlugin(t)

	req := newJSONRequest(t, http.MethodPost, "/auth/passkey/verify-authentication", `{"response":{}}`)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "challenge")
}

func TestVerifyRegistration_RequiresSession(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimenAndPlugin(t)

	req := newJSONRequest(t, http.MethodPost, "/auth/passkey/verify-registration", `{"response":{}}`)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListPasskeys_RequiresSession(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimenAndPlugin(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/passkey/list", http.NoBody)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeletePasskey_RequiresSession(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimenAndPlugin(t)

	req := newJSONRequest(t, http.MethodPost, "/auth/passkey/delete", `{"id":"1"}`)
	w := httptest.NewRecorder()
	l.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
