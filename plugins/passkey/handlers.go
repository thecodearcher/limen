package passkey

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

type passkeyHandlers struct {
	plugin    *passkeyPlugin
	responder *limen.Responder
}

func (p *passkeyPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/passkeys",
	}
}

func (p *passkeyPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	api := newPasskeyHandlers(p, httpCore)
	routes(api, routeBuilder)
}

func routes(h *passkeyHandlers, routeBuilder *limen.RouteBuilder) {
	routeBuilder.ProtectedGET("/begin-registration", "passkey:begin-registration", h.BeginRegistration)
	routeBuilder.ProtectedPOST("/finish-registration", "passkey:finish-registration", h.FinishRegistration)

	routeBuilder.GET("/begin-authentication", "passkey:begin-authentication", h.BeginAuthentication)
	routeBuilder.POST("/finish-authentication", "passkey:finish-authentication", h.FinishAuthentication)
}

func newPasskeyHandlers(plugin *passkeyPlugin, httpCore *limen.LimenHTTPCore) *passkeyHandlers {
	return &passkeyHandlers{
		plugin:    plugin,
		responder: httpCore.Responder,
	}
}

func (h *passkeyHandlers) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[RegisterPasskeyRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("authenticator_attachment").Optional().String().In([]string{"platform", "cross-platform"})
	})

	if body == nil {
		return
	}

	user, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	credentialCreation, sessionData, err := h.plugin.BeginRegistration(r.Context(), user.User, body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	err = h.plugin.setSessionDataToCookie(w, sessionData)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, credentialCreation)
}

func (h *passkeyHandlers) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[FinishRegistrationRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("name").Optional().String().MinLength(1).MaxLength(100)
	})

	if body == nil {
		return
	}

	user, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	passkey, err := h.plugin.FinishRegistration(r, user.User, body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.plugin.deleteSessionDataFromCookie(w)
	h.responder.JSON(w, r, http.StatusOK, passkey)
}

func (h *passkeyHandlers) BeginAuthentication(w http.ResponseWriter, r *http.Request) {
	credentialAssertion, sessionData, err := h.plugin.BeginAuthentication(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	err = h.plugin.setSessionDataToCookie(w, sessionData)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, credentialAssertion)
}

func (h *passkeyHandlers) FinishAuthentication(w http.ResponseWriter, r *http.Request) {
	user, err := h.plugin.FinishAuthentication(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	authResult := &limen.AuthenticationResult{
		User: user,
	}

	sessionResult, err := h.plugin.core.CreateSession(r.Context(), r, w, authResult)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.plugin.deleteSessionDataFromCookie(w)
	h.responder.SessionResponse(w, r, h.plugin.core, authResult, sessionResult)
}
