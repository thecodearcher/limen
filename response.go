package aegis

import (
	"encoding/json"
	"net/http"
)

type Responder struct {
	cfg                *responseEnvelopeConfig
	sessionTransformer SessionTransformer
	cookieConfig       *cookieConfig
}

func newResponder(cfg *httpConfig) *Responder {
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
		cookieConfig:       cfg.cookieConfig,
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
	rs.setSessionCookies(w, sessionResult)
	rs.setSessionHeaders(w, sessionResult)

	if rs.sessionTransformer != nil {
		return rs.handleSessionTransformer(w, r, result, sessionResult)
	}

	return rs.JSON(w, r, http.StatusOK, map[string]any{
		"user": core.Schema.User.Serialize(result.User),
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

// SetCookie sets a cookie on the response
func (rs Responder) SetCookie(w http.ResponseWriter, cookie *http.Cookie) {
	if cookie == nil {
		return
	}
	http.SetCookie(w, cookie)
}

// DeleteCookie removes a cookie by setting MaxAge to -1
func (rs Responder) DeleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:   name,
		Value:  "",
		MaxAge: -1,
	})
}

// ClearSessionCookies clears the session cookie from the response.
func (rs Responder) ClearSessionCookies(w http.ResponseWriter) {
	if rs.cookieConfig == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     rs.cookieConfig.name,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: rs.cookieConfig.httpOnly,
		Secure:   rs.cookieConfig.secure,
		SameSite: rs.cookieConfig.sameSite,
		Path:     rs.cookieConfig.path,
	})
}

// setSessionCookies sets the session cookie in the response if the delivery method is TokenDeliveryCookie.
func (rs Responder) setSessionCookies(w http.ResponseWriter, sessionResult *SessionResult) {
	if sessionResult == nil || sessionResult.Cookie == nil || sessionResult.TokenDeliveryMethod != TokenDeliveryCookie {
		return
	}
	http.SetCookie(w, sessionResult.Cookie)
}

// setSessionHeaders sets the session header in the response if the delivery method is TokenDeliveryHeader.
func (rs Responder) setSessionHeaders(w http.ResponseWriter, sessionResult *SessionResult) {
	if sessionResult == nil || sessionResult.TokenDeliveryMethod != TokenDeliveryHeader {
		return
	}
	w.Header().Set("Set-Aegis-Token", sessionResult.Token)
}
