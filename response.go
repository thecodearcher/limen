package aegis

import (
	"encoding/json"
	"net/http"
)

type Responder struct {
	cfg *responseEnvelopeConfig
}

func NewResponder(cfg *responseEnvelopeConfig) Responder {
	if cfg == nil {
		cfg = &responseEnvelopeConfig{
			mode: EnvelopeOff,
		}
	}
	return Responder{cfg: cfg}
}

func (rs Responder) JSON(w http.ResponseWriter, r *http.Request, status int, payload any) error {
	if rs.cfg.formatter != nil {
		body, _ := json.Marshal(payload)
		return rs.cfg.formatter(w, r, status, body, nil)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	out := payload
	if rs.cfg.mode != EnvelopeOff && rs.cfg.fields.Data != "" {
		out = map[string]any{
			rs.cfg.fields.Data: payload,
		}
	}

	return json.NewEncoder(w).Encode(out)
}

func (rs Responder) Error(w http.ResponseWriter, r *http.Request, ae AegisError) error {
	if rs.cfg.formatter != nil {
		return rs.cfg.formatter(w, r, ae.Status(), nil, ae)
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
