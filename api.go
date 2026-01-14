package aegis

import (
	"net/http"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type aegisAPI struct {
	core      *AegisCore
	responder *Responder
	config    *httpConfig
}

func registerBaseRoutes(router *httpx.Router, httpCore *AegisHTTPCore, core *AegisCore, basePath string) {
	routeBuilder := &RouteBuilder{
		group: router.Group(basePath),
		core:  httpCore,
	}
	api := newAegisAPI(httpCore, core)
	api.RegisterRoutes(routeBuilder)
}

func newAegisAPI(httpCore *AegisHTTPCore, core *AegisCore) *aegisAPI {
	return &aegisAPI{
		core:      core,
		responder: httpCore.Responder,
		config:    httpCore.config,
	}
}

func (api *aegisAPI) RegisterRoutes(routeBuilder *RouteBuilder) {
	routeBuilder.ProtectedGET("/me", "me", api.GetSession)
	routeBuilder.ProtectedPOST("/signout", "signout", api.SignOut)
}

func (api *aegisAPI) GetSession(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		api.core.SessionManager.RevokeAllCookies(w)
		api.responder.Error(w, r, NewAegisError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	api.responder.SessionResponse(w, r, api.core, &AuthenticationResult{User: session.User}, nil)
}

func (api *aegisAPI) SignOut(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		api.responder.Error(w, r, NewAegisError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	err = api.core.SessionManager.Revoke(r.Context(), r, session.Session.Token)
	if err != nil {
		api.responder.Error(w, r, NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	api.core.SessionManager.RevokeAllCookies(w)

	api.responder.JSON(w, r, http.StatusOK, "OK")
}
