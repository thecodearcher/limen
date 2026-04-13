package limen

import (
	"net/http"
)

type limenHandlers struct {
	core      *LimenCore
	responder *Responder
	config    *httpConfig
}

func registerBaseRoutes(router *router, httpCore *LimenHTTPCore, core *LimenCore, basePath string) {
	routeBuilder := &RouteBuilder{
		group: router.Group(basePath),
		core:  httpCore,
	}
	handlers := newLimenHandlers(httpCore, core)
	handlers.RegisterRoutes(routeBuilder)
}

func newLimenHandlers(httpCore *LimenHTTPCore, core *LimenCore) *limenHandlers {
	return &limenHandlers{
		core:      core,
		responder: httpCore.Responder,
		config:    httpCore.config,
	}
}

func (h *limenHandlers) RegisterRoutes(routeBuilder *RouteBuilder) {
	routeBuilder.ProtectedGET("/me", "me", h.GetSession)
	routeBuilder.ProtectedGET("/sessions", "list-sessions", h.ListSessions)
	routeBuilder.ProtectedPOST("/signout", "signout", h.SignOut)
	routeBuilder.ProtectedPOST("/revoke-sessions", "revoke-sessions", h.RevokeAllSessions)

	if h.core.EmailVerificationEnabled() {
		routeBuilder.POST("/verify-email", "verify-email", h.VerifyEmail)
		routeBuilder.ProtectedPOST("/email-verifications", "email-verifications", h.RequestEmailVerification)
	}
}

func (h *limenHandlers) GetSession(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.core.Cookies().ClearSessionCookie(w)
		h.responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	h.responder.SessionResponse(w, r, h.core, &AuthenticationResult{User: session.User}, nil)
}

func (h *limenHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	sessions, err := h.core.SessionManager.ListSessions(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, sessions)
}

func (h *limenHandlers) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	err = h.core.SessionManager.RevokeAllSessions(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *limenHandlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	body := ValidateJSON(w, r, h.responder, func(v *Validator, data map[string]any) *Validator {
		return v.RequiredString("token", data["token"])
	})
	if body == nil {
		return
	}

	err := h.core.VerifyEmail(r.Context(), body["token"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, "email verified successfully")
}

func (h *limenHandlers) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	_, err = h.core.RequestEmailVerification(r.Context(), &User{
		Email: session.User.Email,
	}, true)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, "email verification requested successfully")
}

func (h *limenHandlers) SignOut(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	err = h.core.SessionManager.RevokeSession(r.Context(), session.Session.Token)
	if err != nil {
		h.responder.Error(w, r, NewLimenError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	h.core.Cookies().ClearSessionCookie(w)

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}
