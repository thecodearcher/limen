package limen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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

	req := httptest.NewRequest(http.MethodPost, "/signin", http.NoBody)
	w := httptest.NewRecorder()
	auth := &AuthenticationResult{User: &User{ID: userID, Email: "create-session@test.com"}}

	result, err := l.core.CreateSession(context.Background(), req, w, auth)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Token)
	assert.NotNil(t, result.Cookie)
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
	httpCore.trustedOriginsPatterns = compileTrustedOrigins("http://localhost:8080", "https://*.example.com")

	assert.True(t, httpCore.IsTrustedOrigin("http://localhost:8080"))
	assert.True(t, httpCore.IsTrustedOrigin("https://app.example.com"))
	assert.False(t, httpCore.IsTrustedOrigin("https://evil.com"))
}
