package aegis

import (
	"encoding/json"
	"net/http"
)

type Responder struct {
	cfg                *responseEnvelopeConfig
	sessionTransformer SessionTransformer
}

func NewResponder(cfg *HTTPConfig) *Responder {
	if cfg == nil {
		cfg = &HTTPConfig{}
	}

	envelopeConfig := &responseEnvelopeConfig{
		mode: EnvelopeOff,
	}

	if cfg.responseEnvelope != nil {
		envelopeConfig = cfg.responseEnvelope
	}

	return &Responder{cfg: envelopeConfig, sessionTransformer: cfg.sessionTransformer}
}

func (rs Responder) JSON(w http.ResponseWriter, r *http.Request, status int, payload any) error {
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

func (rs Responder) Error(w http.ResponseWriter, r *http.Request, ae *AegisError) error {
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
	if sessionResult.Cookie != nil {
		http.SetCookie(w, sessionResult.Cookie)
	}

	if rs.sessionTransformer != nil {
		payload, err := rs.sessionTransformer(result.User.Raw(), result.PendingActions, sessionResult.Token, sessionResult.RefreshToken)
		if err != nil {
			return rs.Error(w, r, err)
		}
		return rs.JSON(w, r, http.StatusOK, payload)
	}

	payload := map[string]any{
		"pending_actions": result.PendingActions,
		"user":            core.Schema.User.Serialize(result.User),
	}

	if sessionResult.Token != "" {
		payload["token"] = sessionResult.Token
	}

	if sessionResult.RefreshToken != "" {
		payload["refresh_token"] = sessionResult.RefreshToken
	}

	return rs.JSON(w, r, http.StatusOK, payload)
}
