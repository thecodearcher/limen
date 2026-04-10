package limen

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestCookieManager(t *testing.T) *CookieManager {
	t.Helper()
	cfg := &cookieConfig{
		sessionCookieName: "limen_session",
		path:              "/",
		secure:            true,
		httpOnly:          true,
		sameSite:          http.SameSiteLaxMode,
		partitioned:       false,
		crossSubdomain:    &crossDomainConfig{enabled: false},
	}
	return newCookieManager(cfg, TestSecret)
}

func TestCookieManager_NewCookie_Attributes(t *testing.T) {
	t.Parallel()

	cm := newTestCookieManager(t)
	cookie := cm.NewCookie("test", "value", 3600)

	assert.Equal(t, "test", cookie.Name)
	assert.Equal(t, "value", cookie.Value)
	assert.Equal(t, 3600, cookie.MaxAge)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.Secure)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}

func TestCookieManager_NewCookie_CrossSubdomain(t *testing.T) {
	t.Parallel()

	cfg := &cookieConfig{
		sessionCookieName: "limen_session",
		path:              "/",
		secure:            true,
		httpOnly:          true,
		sameSite:          http.SameSiteLaxMode,
		crossSubdomain:    &crossDomainConfig{enabled: true, domain: ".example.com"},
	}
	cm := newCookieManager(cfg, TestSecret)
	cookie := cm.NewCookie("test", "val", 100)

	assert.Equal(t, ".example.com", cookie.Domain)
}

func TestCookieManager_SetAndGet(t *testing.T) {
	t.Parallel()

	cm := newTestCookieManager(t)
	w := httptest.NewRecorder()

	cm.Set(w, "mykey", "myval", 600)

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "mykey", cookies[0].Name)
	assert.Equal(t, "myval", cookies[0].Value)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.AddCookie(cookies[0])
	val, err := cm.Get(req, "mykey")
	assert.NoError(t, err)
	assert.Equal(t, "myval", val)
}

func TestCookieManager_Get_Missing(t *testing.T) {
	t.Parallel()

	cm := newTestCookieManager(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)

	_, err := cm.Get(req, "nonexistent")
	assert.Error(t, err)
}

func TestCookieManager_Delete(t *testing.T) {
	t.Parallel()

	cm := newTestCookieManager(t)
	w := httptest.NewRecorder()

	cm.Delete(w, "to-delete")

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "to-delete", cookies[0].Name)
	assert.Equal(t, -1, cookies[0].MaxAge)
	assert.Equal(t, "", cookies[0].Value)
}

func TestCookieManager_ClearSessionCookie(t *testing.T) {
	t.Parallel()

	cm := newTestCookieManager(t)
	w := httptest.NewRecorder()

	cm.ClearSessionCookie(w)

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "limen_session", cookies[0].Name)
	assert.Equal(t, -1, cookies[0].MaxAge)
}

func TestCookieManager_SignedCookie_RoundTrip(t *testing.T) {
	t.Parallel()

	cm := newTestCookieManager(t)
	w := httptest.NewRecorder()

	err := cm.SetSignedCookie(w, "signed", "secret-data", 600)
	assert.NoError(t, err)

	cookies := w.Result().Cookies()
	assert.NotEmpty(t, cookies)

	var signedCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "signed" {
			signedCookie = c
			break
		}
	}
	assert.NotNil(t, signedCookie)
	assert.NotEqual(t, "secret-data", signedCookie.Value, "value should be encrypted")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.AddCookie(signedCookie)
	val, err := cm.GetSignedCookie(req, "signed")
	assert.NoError(t, err)
	assert.Equal(t, "secret-data", val)
}
