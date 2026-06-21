package limen

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Response header names used to carry session tokens in bearer/JWT transport.
const (
	HeaderSetAuthToken    = "Set-Auth-Token"    //nolint:gosec // G101 false positive: HTTP header name, not a credential
	HeaderSetRefreshToken = "Set-Refresh-Token" //nolint:gosec // G101 false positive: HTTP header name, not a credential
	HeaderExposeHeaders   = "Access-Control-Expose-Headers"
)

type Responder struct {
	cfg                *responseEnvelopeConfig
	sessionTransformer SessionTransformer
	cookies            *cookieManager
	bearerEnabled      bool
}

func newResponder(cfg *httpConfig, cookies *cookieManager, bearerEnabled bool) *Responder {
	if cfg == nil {
		cfg = &httpConfig{}
	}

	envelopeConfig := &responseEnvelopeConfig{
		mode: EnvelopeOff,
	}

	if cfg.responseEnvelope != nil {
		envelopeConfig = cfg.responseEnvelope
	}

	return &Responder{
		cfg:                envelopeConfig,
		sessionTransformer: cfg.sessionTransformer,
		cookies:            cookies,
		bearerEnabled:      bearerEnabled,
	}
}

// tryDeferResponse attempts to store response data for deferred writing.
// Returns true if the response was deferred (caller should return early).
func tryDeferResponse(w http.ResponseWriter, status int, payload any, isError bool) bool {
	rw, ok := w.(*responseWriter)
	if !ok || !rw.deferWrite {
		return false
	}
	rw.statusCode = status
	rw.payload = payload
	rw.isError = isError
	rw.written = true
	return true
}

// tryDeferRedirect stores a redirect for deferred writing when after-hooks are in use.
// Returns true if the redirect was deferred (caller should return early).
func tryDeferRedirect(w http.ResponseWriter, redirectURL string, status int) bool {
	rw, ok := w.(*responseWriter)
	if !ok || !rw.deferWrite {
		return false
	}
	rw.redirectURL = redirectURL
	rw.redirectStatus = status
	rw.written = true
	return true
}

func (rs Responder) JSON(w http.ResponseWriter, r *http.Request, status int, payload any) error {
	if tryDeferResponse(w, status, payload, false) {
		return nil
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	out := payload
	if rs.cfg.mode != EnvelopeOff && rs.cfg.fields.Data != "" {
		out = map[string]any{
			rs.cfg.fields.Data: payload,
		}
	}

	if message, ok := payload.(string); ok {
		out = map[string]any{
			"message": message,
		}

		if rs.cfg.mode != EnvelopeOff && rs.cfg.mode != EnvelopeWrapSuccess && rs.cfg.fields.Message != "" {
			out = map[string]any{
				rs.cfg.fields.Message: message,
			}
		}
	}

	return json.NewEncoder(w).Encode(out)
}

func (rs Responder) Error(w http.ResponseWriter, r *http.Request, err error) error {
	ae := ToLimenError(err)
	if tryDeferResponse(w, ae.Status(), ae, true) {
		return nil
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(ae.Status())

	errMsg := ae.Error()
	if errMsg == "" {
		errMsg = http.StatusText(ae.Status())
	}

	out := map[string]any{
		"message": errMsg,
	}

	if rs.cfg.mode != EnvelopeOff && rs.cfg.mode != EnvelopeWrapSuccess && rs.cfg.fields.Message != "" {
		out = map[string]any{
			rs.cfg.fields.Message: errMsg,
		}
	}

	return json.NewEncoder(w).Encode(out)
}

func (rs Responder) SessionResponse(w http.ResponseWriter, r *http.Request, core *LimenCore, result *AuthenticationResult, sessionResult *SessionResult) error {
	// Store auth result for hooks to access
	if rw, ok := w.(*responseWriter); ok {
		rw.authResult = result
	}

	if err := rs.setSessionCookies(w, sessionResult); err != nil {
		return rs.Error(w, r, err)
	}

	rs.setSessionHeaders(w, sessionResult)

	if rs.sessionTransformer != nil {
		rs.handleSessionTransformer(w, r, result, sessionResult)
		return nil
	}

	return rs.JSON(w, r, http.StatusOK, map[string]any{
		"user": SerializeModel(core.Schema.User, result.User),
	})
}

func (rs Responder) handleSessionTransformer(w http.ResponseWriter, r *http.Request, result *AuthenticationResult, sessionResult *SessionResult) {
	payload, err := rs.sessionTransformer(result.User.Raw(), sessionResult)
	if err != nil {
		rs.Error(w, r, err)
		return
	}
	rs.JSON(w, r, http.StatusOK, payload)
}

// SetHeader sets a response header
func (rs Responder) SetHeader(w http.ResponseWriter, key, value string) {
	w.Header().Set(key, value)
}

// AddHeader adds a response header (allows multiple values for same key)
func (rs Responder) AddHeader(w http.ResponseWriter, key, value string) {
	w.Header().Add(key, value)
}

// setSessionCookies sets the session cookie in the response.
func (rs Responder) setSessionCookies(w http.ResponseWriter, sessionResult *SessionResult) error {
	if sessionResult == nil {
		return nil
	}

	return rs.cookies.SetSessionCookie(w, sessionResult)
}

func (rs Responder) setSessionHeaders(w http.ResponseWriter, sessionResult *SessionResult) {
	if sessionResult == nil {
		return
	}
	if sessionResult.Cookie != nil && !rs.bearerEnabled {
		return
	}

	var exposed []string

	if sessionResult.Token != "" {
		rs.AddHeader(w, HeaderSetAuthToken, sessionResult.Token)
		exposed = append(exposed, HeaderSetAuthToken)
	}
	if sessionResult.RefreshToken != "" {
		rs.AddHeader(w, HeaderSetRefreshToken, sessionResult.RefreshToken)
		exposed = append(exposed, HeaderSetRefreshToken)
	}

	if len(exposed) > 0 {
		rs.appendExposeHeaders(w, exposed...)
	}
}

// IssuedSessionToken returns the session token a response carries.
func (rs Responder) IssuedSessionToken(h http.Header) string {
	if rs.cookies != nil && rs.cookies.base != nil {
		if name := rs.cookies.base.sessionCookieName; name != "" {
			if token := ExtractCookieValue(h, name); token != "" {
				return token
			}
		}
	}
	return h.Get(HeaderSetAuthToken)
}

// ClearSessionResponse removes any pending session from this response so no
// usable session is sent to the client.
func (rs Responder) ClearSessionResponse(w http.ResponseWriter) {
	if rs.cookies != nil {
		rs.cookies.ClearSessionResponse(w)
	}
	h := w.Header()
	h.Del(HeaderSetAuthToken)
	h.Del(HeaderSetRefreshToken)
	removeExposeHeaders(h, HeaderSetAuthToken, HeaderSetRefreshToken)
}

// appendExposeHeaders appends header names to Access-Control-Expose-Headers
// without overwriting values already set by the user's CORS middleware.
func (rs Responder) appendExposeHeaders(w http.ResponseWriter, headers ...string) {
	for _, header := range headers {
		if header == "" {
			continue
		}
		w.Header().Add(HeaderExposeHeaders, header)
	}
}

// removeExposeHeaders drops the given names from Access-Control-Expose-Headers
func removeExposeHeaders(h http.Header, names ...string) {
	values := h.Values(HeaderExposeHeaders)
	if len(values) == 0 {
		return
	}

	remove := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n != "" {
			remove[n] = struct{}{}
		}
	}

	kept := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, raw := range values {
		for part := range strings.SplitSeq(raw, ",") {
			name := strings.TrimSpace(part)
			if name == "" {
				continue
			}
			lower := strings.ToLower(name)
			if _, drop := remove[lower]; drop {
				continue
			}
			if _, exists := seen[lower]; exists {
				continue
			}
			seen[lower] = struct{}{}
			kept = append(kept, name)
		}
	}

	if len(kept) == 0 {
		h.Del(HeaderExposeHeaders)
		return
	}
	h.Set(HeaderExposeHeaders, strings.Join(kept, ", "))
}

// Redirect sends a redirect response. When the response is deferred (after-hooks in use),
// the redirect is stored and sent after hooks run so the browser receives a proper 3xx.
func (rs Responder) Redirect(w http.ResponseWriter, r *http.Request, redirectURL string, status int) {
	if tryDeferRedirect(w, redirectURL, status) {
		return
	}
	http.Redirect(w, r, redirectURL, status)
}

// RedirectWithSession sets the session cookie and redirects the client to redirectURL.
// Used by OAuth callbacks when redirect_uri is provided in the authorize request.
func (rs Responder) RedirectWithSession(w http.ResponseWriter, r *http.Request, redirectURL string, sessionResult *SessionResult) {
	if err := rs.setSessionCookies(w, sessionResult); err != nil {
		rs.Error(w, r, err)
		return
	}

	rs.setSessionHeaders(w, sessionResult)

	rs.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// SerializeModel serializes a model using its schema's Serialize method.
func SerializeModel(schema Schema, model Model) map[string]any {
	return schema.Serialize(model)
}

// SerializeAll serializes a slice of models using the given schema's Serialize method.
func SerializeAll[T Model](schema Schema, models []T) []map[string]any {
	result := make([]map[string]any, 0, len(models))
	for _, model := range models {
		result = append(result, schema.Serialize(model))
	}
	return result
}
