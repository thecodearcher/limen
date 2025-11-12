package emailpassword

import (
	"context"
	"errors"
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
	router.AddRoute(httpx.MethodPOST, "/signup", api.SignUpWithEmailAndPassword, "signup", "Sign up with email and password")
	router.AddRoute(httpx.MethodPOST, "/verify-email", api.VerifyEmail, "verify-email", "Verify email")
	return router
}

func (p *emailPasswordAPI) SignInWithEmailAndPassword(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder,
		func(v *validator.Validator, data map[string]any) *validator.Validator {
			return v.Required("email", data["email"]).
				Email("email", data["email"]).
				Required("password", data["password"])
		})

	if body == nil {
		return
	}

	result, err := p.feature.SignInWithEmailAndPassword(r.Context(), body["email"].(string), body["password"].(string))
	if err != nil {
		p.responder.Error(w, r, ErrAPIInvalidCredentials)
		return
	}

	sessionManager := session.NewSessionManager(p.feature.core)
	sessionResult, err := sessionManager.CreateSession(context.Background(), r, result)
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusInternalServerError, nil))
		return
	}

	p.responder.SessionResponse(w, r, p.feature.core, result, sessionResult)
}

func (p *emailPasswordAPI) SignUpWithEmailAndPassword(w http.ResponseWriter, r *http.Request) {
	additionalFields, err := aegis.GetSchemaAdditionalFieldsForRequest(w, r, p.feature.userSchema)
	if err != nil {
		p.responder.Error(w, r, err.(*aegis.AegisError))
		return
	}

	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.
			Required("email", data["email"]).
			Required("password", data["password"]).
			Email("email", data["email"])
	})

	if body == nil {
		return
	}

	result, err := p.feature.SignUpWithEmailAndPassword(r.Context(), &aegis.User{
		Email:    body["email"].(string),
		Password: body["password"].(string),
	}, additionalFields)

	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusConflict, nil))
			return
		}
		if errors.Is(err, ErrInvalidPassword) {
			p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusUnprocessableEntity, nil))
			return
		}
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	if !p.feature.config.autoSignInOnSignUp {
		p.responder.SessionResponse(w, r, p.feature.core, result, &aegis.SessionResult{})
		return
	}

	sessionManager := session.NewSessionManager(p.feature.core)
	sessionResult, err := sessionManager.CreateSession(r.Context(), r, result)
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusInternalServerError, nil))
		return
	}

	p.responder.SessionResponse(w, r, p.feature.core, result, sessionResult)
}

func (p *emailPasswordAPI) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.Required("token", data["token"])
	})

	if body == nil {
		return
	}

	err := p.feature.VerifyEmail(r.Context(), body["token"].(string))
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	p.responder.JSON(w, r, http.StatusOK, map[string]any{
		"message": "email verified successfully",
	})
}
