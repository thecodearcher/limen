package passkey

import (
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"

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
	if h.plugin.config.requireSession {
		routeBuilder.ProtectedGET("/begin-registration", "passkey:begin-registration", h.BeginRegistration)
		routeBuilder.ProtectedPOST("/finish-registration", "passkey:finish-registration", h.FinishRegistration)
	} else {
		routeBuilder.GET("/begin-registration", "passkey:begin-registration", h.BeginRegistration)
		routeBuilder.POST("/finish-registration", "passkey:finish-registration", h.FinishRegistration)
	}

	routeBuilder.GET("/begin-authentication", "passkey:begin-authentication", h.BeginAuthentication)
	routeBuilder.POST("/finish-authentication", "passkey:finish-authentication", h.FinishAuthentication)

	routeBuilder.ProtectedGET("/", "passkey:list", h.ListPasskeys)
	routeBuilder.ProtectedPATCH("/:id", "passkey:update", h.UpdatePasskey)
	routeBuilder.ProtectedDELETE("/:id", "passkey:delete", h.DeletePasskey)
}

func newPasskeyHandlers(plugin *passkeyPlugin, httpCore *limen.LimenHTTPCore) *passkeyHandlers {
	return &passkeyHandlers{
		plugin:    plugin,
		responder: httpCore.Responder,
	}
}

func (h *passkeyHandlers) validatePasskeyIDParam(v *limen.Validator) {
	v.Param("id").Required().Custom(func(value any, _ map[string]any) error {
		return limen.ValidateClientIDValue(h.plugin.core, h.plugin.passkeySchema, value)
	})
}

func (h *passkeyHandlers) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[RegisterPasskeyRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("authenticator_attachment").Optional().String().In([]string{"platform", "cross-platform"})
		if !h.plugin.config.requireSession {
			v.Field("context").Optional().String()
		}
	})

	if body == nil {
		return
	}

	user, err := h.optionalSessionUser(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	var credentialCreation *protocol.CredentialCreation
	var cookie *PasskeyChallenge

	if user != nil {
		credentialCreation, cookie, err = h.plugin.BeginRegistration(r, user, body)
	} else {
		credentialCreation, cookie, err = h.plugin.BeginPublicRegistration(r, body)
	}
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	if err := h.plugin.setChallengeCookie(w, cookie); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, credentialCreation)
}

func (h *passkeyHandlers) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[FinishRegistrationRequest](w, r, h.responder, func(v *limen.Validator) {
		v.Field("name").Optional().String().MinLength(1).MaxLength(100)
		v.Field("create_session").Optional().Boolean()
	})

	if body == nil {
		return
	}

	user, err := h.optionalSessionUser(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	var passkey *Passkey
	var createdUser *limen.User
	if user != nil {
		passkey, err = h.plugin.FinishRegistration(r, user, body)
	} else {
		createdUser, passkey, err = h.plugin.FinishPublicRegistration(r, body)
	}
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.plugin.deleteChallengeCookie(w)

	if createdUser != nil && body.CreateSession {
		authResult := &limen.AuthenticationResult{User: createdUser}
		sessionResult, err := h.plugin.core.CreateSession(r.Context(), r, w, authResult)
		if err != nil {
			h.responder.Error(w, r, err)
			return
		}
		h.responder.SessionResponse(w, r, h.plugin.core, authResult, sessionResult)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.passkeySchema, passkey))
}

func (h *passkeyHandlers) optionalSessionUser(r *http.Request) (*limen.User, error) {
	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err == nil {
		return session.User, nil
	}
	if h.plugin.config.requireSession {
		return nil, ErrSessionRequired
	}

	validated, validateErr := h.plugin.core.SessionManager.ValidateSession(r.Context(), r)
	if validateErr != nil {
		return nil, nil
	}
	return validated.User, nil
}

func (h *passkeyHandlers) BeginAuthentication(w http.ResponseWriter, r *http.Request) {
	credentialAssertion, cookie, err := h.plugin.BeginAuthentication(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	if err := h.plugin.setChallengeCookie(w, cookie); err != nil {
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
	h.plugin.deleteChallengeCookie(w)
	h.responder.SessionResponse(w, r, h.plugin.core, authResult, sessionResult)
}

func (h *passkeyHandlers) ListPasskeys(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	page, err := h.plugin.ListPasskeys(r.Context(), session.User, limen.ParsePagination(r))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, limen.SerializePage(h.plugin.core, h.plugin.passkeySchema, page))
}

func (h *passkeyHandlers) UpdatePasskey(w http.ResponseWriter, r *http.Request) {
	body := limen.BindAndValidate[UpdatePasskeyRequest](w, r, h.responder, func(v *limen.Validator) {
		h.validatePasskeyIDParam(v)
		v.Field("name").Required().String().MinLength(1).MaxLength(100)
	})

	if body == nil {
		return
	}

	user, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	passkey, err := h.plugin.UpdatePasskey(r.Context(), user.User, limen.GetParam(r, "id"), body)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, h.plugin.core.SerializeModel(h.plugin.passkeySchema, passkey))
}

func (h *passkeyHandlers) DeletePasskey(w http.ResponseWriter, r *http.Request) {
	if limen.ValidateRequest(w, r, h.responder, h.validatePasskeyIDParam) == nil {
		return
	}

	user, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	if err := h.plugin.DeletePasskey(r.Context(), user.User, limen.GetParam(r, "id")); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}
