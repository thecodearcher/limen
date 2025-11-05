package emailpassword

import (
	"context"
	"net/http"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/internal/session"
	"github.com/thecodearcher/aegis/pkg/httpx"
	"github.com/thecodearcher/aegis/pkg/validator"
)

type emailPasswordAPI struct {
	feature   *emailPasswordFeature
	responder *aegis.Responder
	config    *aegis.HTTPConfig
}

func NewEmailPasswordAPI(emailPasswordFeature *emailPasswordFeature, responder *aegis.Responder, config *aegis.HTTPConfig) *emailPasswordAPI {
	return &emailPasswordAPI{feature: emailPasswordFeature, responder: responder, config: config}
}

func (p *emailPasswordFeature) HTTPMount(c *aegis.Aegis, responder *aegis.Responder, config *aegis.HTTPConfig) aegis.HTTPMount {
	api := NewEmailPasswordAPI(p, responder, config)
	return aegis.HTTPMount{
		Handler: routes(api),
	}
}

func routes(api *emailPasswordAPI) *httpx.Router {
	router := httpx.NewRouter()
	router.AddRoute(httpx.MethodPOST, "/signin", api.SignInWithEmailAndPassword, "signin", "Sign in with email and password")
	return router
}

func (p *emailPasswordAPI) SignInWithEmailAndPassword(w http.ResponseWriter, r *http.Request) {
	type SignInRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	body := validator.DecodeJSONAndValidate(w, r, p.responder,
		func(v *validator.Validator, data *SignInRequest) *validator.Validator {
			return v.Required("email", data.Email).
				Email("email", data.Email).
				Required("password", data.Password)
		})

	if body == nil {
		return
	}

	result, err := p.feature.SignInWithEmailAndPassword(context.Background(), body.Email, body.Password)
	if err != nil {
		p.responder.Error(w, r, ErrInvalidCredentials)
		return
	}

	sessionManager := session.NewSessionManager(p.feature.core)
	sessionResult, err := sessionManager.CreateSession(context.Background(), result.User, r)
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusInternalServerError, nil))
		return
	}

	p.responder.SessionResponse(w, r, p.feature.core, result, sessionResult)
}
