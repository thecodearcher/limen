package passkey

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

type passkeyHandlers struct {
	plugin    *passkeyPlugin
	httpCore  *limen.LimenHTTPCore
	responder *limen.Responder
}

func newPasskeyHandlers(plugin *passkeyPlugin, httpCore *limen.LimenHTTPCore) *passkeyHandlers {
	return &passkeyHandlers{
		plugin:    plugin,
		httpCore:  httpCore,
		responder: httpCore.Responder,
	}
}

func (h *passkeyHandlers) GenerateRegistrationOptions(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	options, err := h.plugin.BeginRegistration(r.Context(), r, w, name)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, options)
}

func (h *passkeyHandlers) VerifyRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		h.responder.Error(w, r, ErrInvalidResponse)
		return
	}
	defer r.Body.Close()
	passkey, err := h.plugin.FinishRegistration(r.Context(), r, r.Body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, passkey)
}

func (h *passkeyHandlers) GenerateAuthenticationOptions(w http.ResponseWriter, r *http.Request) {
	options, err := h.plugin.BeginAuthentication(r.Context(), w)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, options)
}

func (h *passkeyHandlers) VerifyAuthentication(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		h.responder.Error(w, r, ErrInvalidResponse)
		return
	}
	defer r.Body.Close()
	result, err := h.plugin.FinishAuthentication(r.Context(), r, r.Body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	sessionResult, err := h.plugin.core.CreateSession(r.Context(), r, w, result)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.SessionResponse(w, r, h.plugin.core, result, sessionResult)
}

func (h *passkeyHandlers) ListPasskeys(w http.ResponseWriter, r *http.Request) {
	session, err := requireValidatedSession(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	passkeys, err := h.plugin.ListPasskeys(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, passkeys)
}

func (h *passkeyHandlers) DeletePasskey(w http.ResponseWriter, r *http.Request) {
	session, err := requireValidatedSession(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("id", data["id"])
	})
	if body == nil {
		return
	}
	if err := h.plugin.DeletePasskey(r.Context(), session.User.ID, body["id"]); err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, map[string]any{"success": true})
}

func (h *passkeyHandlers) UpdatePasskey(w http.ResponseWriter, r *http.Request) {
	session, err := requireValidatedSession(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.
			RequiredString("id", data["id"]).
			RequiredString("name", data["name"])
	})
	if body == nil {
		return
	}
	updated, err := h.plugin.UpdatePasskey(r.Context(), session.User.ID, body["id"], body["name"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, updated)
}
