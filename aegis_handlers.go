package aegis

import (
	"net/http"
)

type aegisHandlers struct {
	core      *AegisCore
	responder *Responder
	config    *httpConfig
}

func registerBaseRoutes(router *Router, httpCore *AegisHTTPCore, core *AegisCore, basePath string) {
	routeBuilder := &RouteBuilder{
		group: router.Group(basePath),
		core:  httpCore,
	}
	handlers := newAegisHandlers(httpCore, core)
	handlers.RegisterRoutes(routeBuilder)
}

func newAegisHandlers(httpCore *AegisHTTPCore, core *AegisCore) *aegisHandlers {
	return &aegisHandlers{
		core:      core,
		responder: httpCore.Responder,
		config:    httpCore.config,
	}
}

func (h *aegisHandlers) RegisterRoutes(routeBuilder *RouteBuilder) {
	routeBuilder.ProtectedGET("/me", "me", h.GetSession)
	routeBuilder.ProtectedPOST("/signout", "signout", h.SignOut)
}

func (h *aegisHandlers) GetSession(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.core.Cookies().ClearSessionCookie(w)
		h.responder.Error(w, r, NewAegisError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	h.responder.SessionResponse(w, r, h.core, &AuthenticationResult{User: session.User}, nil)
}

func (h *aegisHandlers) SignOut(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, NewAegisError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	err = h.core.SessionManager.RevokeSession(r.Context(), session.Session.Token)
	if err != nil {
		h.responder.Error(w, r, NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	h.core.Cookies().ClearSessionCookie(w)

	h.responder.JSON(w, r, http.StatusOK, "OK")
}
