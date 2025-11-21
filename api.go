package aegis

import (
	"net/http"
)

type aegisAPI struct {
	core         *AegisCore
	responder    *Responder
	authInstance *Aegis
	config       *HTTPConfig
}

func NewAegisAPI(httpCore *AegisHTTPCore, core *AegisCore) *aegisAPI {
	return &aegisAPI{
		core:         core,
		responder:    httpCore.Responder,
		authInstance: httpCore.AuthInstance,
		config:       httpCore.Config,
	}
}

func (api *aegisAPI) RegisterRoutes(routeBuilder *RouteBuilder) {
	routeBuilder.ProtectedGET("/me", "me", api.GetSession)
	routeBuilder.ProtectedPOST("/signout", "signout", api.SignOut)
}

func (api *aegisAPI) GetSession(w http.ResponseWriter, r *http.Request) {
	session, err := api.authInstance.GetSession(r)
	if err != nil {
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

	err = api.authInstance.sessionManager.Revoke(r.Context(), r, session.Session.Token)
	if err != nil {
		api.responder.Error(w, r, NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	api.authInstance.sessionManager.RevokeAllCookies(w)

	api.responder.JSON(w, r, http.StatusOK, "OK")
}
