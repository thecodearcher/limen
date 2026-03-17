package limen

import (
	"net/http"
)

type limenHandlers struct {
	core      *LimenCore
	responder *Responder
	config    *httpConfig
}

func registerBaseRoutes(router *Router, httpCore *LimenHTTPCore, core *LimenCore, basePath string) {
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
	routeBuilder.ProtectedPOST("/signout", "signout", h.SignOut)
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

	h.responder.JSON(w, r, http.StatusOK, "OK")
}
