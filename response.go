package aegis

import (
	"encoding/json"
	"net/http"
)

type Responder struct {
	cfg                *responseEnvelopeConfig
	sessionTransformer SessionTransformer
	cookies            *CookieManager
}

func newResponder(cfg *httpConfig, cookies *CookieManager) *Responder {
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

	if rs.cfg.serializer != nil {
		body, _ := json.Marshal(payload)
		return rs.cfg.serializer(w, r, status, body, nil)
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
	ae := ToAegisError(err)
	if tryDeferResponse(w, ae.Status(), ae, true) {
		return nil
	}

	if rs.cfg.serializer != nil {
		return rs.cfg.serializer(w, r, ae.Status(), nil, ae)
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

func (rs Responder) SessionResponse(w http.ResponseWriter, r *http.Request, core *AegisCore, result *AuthenticationResult, sessionResult *SessionResult) error {
	// Store auth result for hooks to access
	if rw, ok := w.(*responseWriter); ok {
		rw.authResult = result
	}

	if err := rs.setSessionCookies(w, sessionResult); err != nil {
		return rs.Error(w, r, err)
	}

	if rs.sessionTransformer != nil {
		if err := rs.handleSessionTransformer(w, r, result, sessionResult); err != nil {
			return rs.Error(w, r, err)
		}
	}

	return rs.JSON(w, r, http.StatusOK, map[string]any{
		"user": SerializeModel(core.Schema.User, result.User),
	})
}

func (rs Responder) handleSessionTransformer(w http.ResponseWriter, r *http.Request, result *AuthenticationResult, sessionResult *SessionResult) error {
	payload, err := rs.sessionTransformer(result.User.Raw(), sessionResult)
	if err != nil {
		return rs.Error(w, r, err)
	}
	return rs.JSON(w, r, http.StatusOK, payload)
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
	if sessionResult == nil || sessionResult.Cookie == nil {
		return nil
	}

	return rs.cookies.SetSessionCookie(w, sessionResult)
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
	rs.Redirect(w, r, redirectURL, http.StatusFound)
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
