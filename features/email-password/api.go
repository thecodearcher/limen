package emailpassword

import (
	"errors"
	"net/http"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/pkg/httpx"
	"github.com/thecodearcher/aegis/pkg/validator"
)

type emailPasswordAPI struct {
	feature   *emailPasswordFeature
	httpCore  *aegis.AegisHTTPCore
	responder *aegis.Responder
}

func NewEmailPasswordAPI(emailPasswordFeature *emailPasswordFeature, httpCore *aegis.AegisHTTPCore) *emailPasswordAPI {
	return &emailPasswordAPI{feature: emailPasswordFeature, httpCore: httpCore, responder: httpCore.Responder}
}

func (p *emailPasswordFeature) HTTPMount(httpCore *aegis.AegisHTTPCore) aegis.HTTPMount {
	api := NewEmailPasswordAPI(p, httpCore)
	return aegis.HTTPMount{
		Handler: routes(api),
	}
}

func routes(e *emailPasswordAPI) *httpx.Router {
	router := httpx.NewRouter()
	router.AddRoute(httpx.MethodPOST, "/signin/email", e.SignInWithEmailAndPassword, "signin")
	router.AddRoute(httpx.MethodPOST, "/signup/email", e.SignUpWithEmailAndPassword, "signup")
	router.AddRoute(httpx.MethodPOST, "/verify-email", e.VerifyEmail, "verify-email")
	router.AddRoute(httpx.MethodPOST, "/email-verifications", e.RequestEmailVerification, "email-verifications")
	router.AddRoute(httpx.MethodPOST, "/passwords/request-reset", e.RequestPasswordReset, "passwords-request-reset")
	router.AddRoute(httpx.MethodPOST, "/passwords/reset", e.ResetPassword, "passwords-reset")
	router.AddRoute(httpx.MethodPOST, "/passwords/change", e.ChangePassword, "passwords-change", e.httpCore.MiddlewareRequireSession())
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

	sessionResult, err := p.feature.core.SessionManager.CreateSession(r.Context(), r, result)
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

	sessionResult, err := p.feature.core.SessionManager.CreateSession(r.Context(), r, result)
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

	p.responder.JSON(w, r, http.StatusOK, "email verified successfully")
}

// TODO: update the request email verification to use the user from the request context instead of the email
func (p *emailPasswordAPI) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.Required("email", data["email"])
	})

	if body == nil {
		return
	}

	_, err := p.feature.RequestEmailVerification(r.Context(), &aegis.User{
		Email: body["email"].(string),
	})

	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	p.responder.JSON(w, r, http.StatusOK, "email verification requested successfully")
}

func (p *emailPasswordAPI) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.
			Required("email", data["email"]).
			Email("email", data["email"])
	})

	if body == nil {
		return
	}

	message := "if the email address is associated with an account, you will receive an email with instructions to reset your password"
	_, err := p.feature.RequestPasswordReset(r.Context(), body["email"].(string))
	if err != nil {
		if errors.Is(err, ErrEmailNotFound) {
			// we don't want to leak the existence of the email address
			p.responder.JSON(w, r, http.StatusOK, message)
			return
		}

		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	p.responder.JSON(w, r, http.StatusOK, message)
}

func (p *emailPasswordAPI) ResetPassword(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.
			Required("token", data["token"]).
			Required("new_password", data["new_password"])
	})

	if body == nil {
		return
	}

	err := p.feature.ResetPassword(r.Context(), body["token"].(string), body["new_password"].(string))
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	p.responder.JSON(w, r, http.StatusOK, "password reset successfully")
}

func (p *emailPasswordAPI) ChangePassword(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.
			Required("current_password", data["current_password"]).
			Required("new_password", data["new_password"])
	})

	if body == nil {
		return
	}

	revokeOtherSessions := true
	if value, ok := body["revoke_other_sessions"].(bool); ok {
		revokeOtherSessions = value
	}

	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	err = p.feature.UpdatePassword(r.Context(), session.User, body["current_password"].(string), body["new_password"].(string), revokeOtherSessions)
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	authResult := &aegis.AuthenticationResult{
		User: session.User,
	}

	if revokeOtherSessions {
		sessionResult, err := p.feature.core.SessionManager.CreateSession(r.Context(), r, authResult)
		if err != nil {
			p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusInternalServerError, nil))
			return
		}
		p.responder.SessionResponse(w, r, p.feature.core, authResult, sessionResult)
		return
	}

	p.responder.SessionResponse(w, r, p.feature.core, authResult, nil)
}
