package limen

import (
	"fmt"
	"net/http"
	"strings"
)

// cookieManager provides a unified interface for cookie operations across
// the core library and plugins. All cookies inherit security attributes
// (Secure, HttpOnly, SameSite, Path, Domain, Partitioned) from the central
// cookieConfig, so callers only specify name, value, and maxAge.
type cookieManager struct {
	base   *cookieConfig
	secret []byte
}

func newCookieManager(base *cookieConfig, secret []byte) *cookieManager {
	return &cookieManager{base: base, secret: secret}
}

// NewCookie builds an *http.Cookie that inherits security attributes from
// the central cookie configuration. The caller supplies only what varies
// per use-case: name, value, and maxAge (in seconds; use -1 to delete).
func (cm *cookieManager) NewCookie(name, value string, maxAge int) *http.Cookie {
	cookie := &http.Cookie{
		Name:        name,
		Value:       value,
		MaxAge:      maxAge,
		Path:        cm.base.path,
		HttpOnly:    cm.base.httpOnly,
		Secure:      cm.base.secure,
		SameSite:    cm.base.sameSite,
		Partitioned: cm.base.partitioned,
	}

	if cm.base.crossSubdomain != nil && cm.base.crossSubdomain.enabled {
		cookie.Domain = cm.base.crossSubdomain.domain
	}

	return cookie
}

// Set creates a cookie and writes it to the response.
func (cm *cookieManager) Set(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, cm.NewCookie(name, value, maxAge))
}

// WriteCookie writes a pre-built cookie to the response (e.g. session cookie from SessionResult).
func (cm *cookieManager) writeCookie(w http.ResponseWriter, cookie *http.Cookie) {
	if cm == nil || cookie == nil {
		return
	}
	http.SetCookie(w, cookie)
}

// SetOnHookCtx creates a cookie and writes it via a HookContext.
func (cm *cookieManager) SetOnHookCtx(ctx *HookContext, name, value string, maxAge int) {
	ctx.SetResponseCookie(cm.NewCookie(name, value, maxAge))
}

// Delete writes a deletion cookie (MaxAge = -1, empty value).
func (cm *cookieManager) Delete(w http.ResponseWriter, name string) {
	http.SetCookie(w, cm.NewCookie(name, "", -1))
}

// DeleteSessionCookie writes a session deletion cookie (Max-Age=-1, empty value).
// It tells the browser to remove an existing session cookie.
func (cm *cookieManager) DeleteSessionCookie(w http.ResponseWriter) {
	if cm == nil || cm.base == nil || cm.base.sessionCookieName == "" {
		return
	}
	cm.Delete(w, cm.base.sessionCookieName)
}

// ClearSessionResponse removes any pending session Set-Cookie header already queued
// on this response, then writes a session deletion cookie.
// Use this when intercepting a response that may have just issued a session cookie.
func (cm *cookieManager) ClearSessionResponse(w http.ResponseWriter) {
	if cm == nil || cm.base == nil {
		return
	}
	if cm.base.sessionCookieName != "" {
		removeResponseCookie(w.Header(), cm.base.sessionCookieName)
	}
	cm.DeleteSessionCookie(w)
}

// removeResponseCookie removes matching Set-Cookie entries from response headers only.
// It prevents the cookie from being sent in this response; it does not delete browser state.
func removeResponseCookie(h http.Header, name string) {
	cookies := h.Values("Set-Cookie")
	if len(cookies) == 0 {
		return
	}
	h.Del("Set-Cookie")
	for _, c := range cookies {
		// Cookie format is "name=value; ..." - keep everything but the target name.
		if !strings.HasPrefix(c, name+"=") {
			h.Add("Set-Cookie", c)
		}
	}
}

// Get reads a cookie value from the request.
// Returns the value and nil on success, or ("", error) if the cookie is absent.
func (cm *cookieManager) Get(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// SetSignedCookie sets a signed cookie
func (cm *cookieManager) SetSignedCookie(w http.ResponseWriter, name, value string, maxAge int) error {
	cookie := cm.NewCookie(name, value, maxAge)
	if cm.secret == nil {
		return fmt.Errorf("secret is nil")
	}
	encoded, err := EncryptXChaCha(cookie.Value, cm.secret, nil)
	if err != nil {
		return fmt.Errorf("failed to encrypt cookie: %w", err)
	}
	cookie.Value = encoded
	cm.Set(w, name, encoded, maxAge)
	return nil
}

// GetSignedCookie gets a signed cookie
func (cm *cookieManager) GetSignedCookie(r *http.Request, name string) (string, error) {
	value, err := cm.Get(r, name)
	if err != nil {
		return "", err
	}

	decoded, err := DecryptXChaCha(value, cm.secret, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt cookie: %w", err)
	}

	return decoded, nil
}

func (cm *cookieManager) SetSessionCookie(w http.ResponseWriter, sessionResult *SessionResult) error {
	if sessionResult == nil {
		return nil
	}

	if sessionResult.Cookie != nil {
		cm.writeCookie(w, sessionResult.Cookie)
	}

	for _, extra := range sessionResult.ExtraCookies {
		cm.writeCookie(w, extra)
	}

	if sessionResult.ShortSession != nil && *sessionResult.ShortSession {
		if err := cm.SetSignedCookie(w, shortSessionCookieName, "true", int(shortSessionMaxAge.Seconds())); err != nil {
			return fmt.Errorf("failed to set short session cookie: %w", err)
		}
	}
	return nil
}

// checkIsShortSession reads and decodes the short session cookie from the request.
// Returns false if cookie is absent, expired, or invalid (no error; treat as no short session).
func (cm *cookieManager) checkIsShortSession(r *http.Request) bool {
	value, err := cm.GetSignedCookie(r, shortSessionCookieName)
	if err != nil {
		return false
	}
	return value == "true"
}
