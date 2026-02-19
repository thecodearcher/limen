package aegis

import "net/http"

// CookieManager provides a unified interface for cookie operations across
// the core library and plugins. All cookies inherit security attributes
// (Secure, HttpOnly, SameSite, Path, Domain, Partitioned) from the central
// cookieConfig, so callers only specify name, value, and maxAge.
type CookieManager struct {
	base *cookieConfig
}

func newCookieManager(base *cookieConfig) *CookieManager {
	return &CookieManager{base: base}
}

// NewCookie builds an *http.Cookie that inherits security attributes from
// the central cookie configuration. The caller supplies only what varies
// per use-case: name, value, and maxAge (in seconds; use -1 to delete).
func (cm *CookieManager) NewCookie(name, value string, maxAge int) *http.Cookie {
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
func (cm *CookieManager) Set(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, cm.NewCookie(name, value, maxAge))
}

// WriteCookie writes a pre-built cookie to the response (e.g. session cookie from SessionResult).
func (cm *CookieManager) WriteCookie(w http.ResponseWriter, cookie *http.Cookie) {
	if cm == nil || cookie == nil {
		return
	}
	http.SetCookie(w, cookie)
}

// SetOnHookCtx creates a cookie and writes it via a HookContext.
func (cm *CookieManager) SetOnHookCtx(ctx *HookContext, name, value string, maxAge int) {
	ctx.SetResponseCookie(cm.NewCookie(name, value, maxAge))
}

// Delete writes a deletion cookie (MaxAge = -1, empty value).
func (cm *CookieManager) Delete(w http.ResponseWriter, name string) {
	http.SetCookie(w, cm.NewCookie(name, "", -1))
}

// ClearSessionCookie clears the session cookie from the response using the
// central cookie configuration name.
func (cm *CookieManager) ClearSessionCookie(w http.ResponseWriter) {
	if cm == nil || cm.base == nil || cm.base.sessionCookieName == "" {
		return
	}
	cm.Delete(w, cm.base.sessionCookieName)
}

// Get reads a cookie value from the request.
// Returns the value and nil on success, or ("", error) if the cookie is absent.
func (cm *CookieManager) Get(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}
