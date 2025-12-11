package usernamepassword

import (
	"errors"
	"net/http"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/pkg/validator"
)

type usernamePasswordAPI struct {
	feature   *usernamePasswordFeature
	builder   *aegis.RouteBuilder
	responder *aegis.Responder
}

func NewUsernamePasswordAPI(usernamePasswordFeature *usernamePasswordFeature, httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) *usernamePasswordAPI {
	return &usernamePasswordAPI{
		feature:   usernamePasswordFeature,
		builder:   routeBuilder,
		responder: httpCore.Responder,
	}
}

func routes(u *usernamePasswordAPI) {
	u.builder.POST("/signin/username", "signin-username", u.SignInWithUsernameAndPassword)
	u.builder.POST("/signup/username", "signup-username", u.SignUpWithUsernameAndPassword)
}

func (u *usernamePasswordAPI) SignInWithUsernameAndPassword(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, u.responder,
		func(v *validator.Validator, data map[string]any) *validator.Validator {
			return v.Required("username", data["username"]).
				Required("password", data["password"])
		})

	if body == nil {
		return
	}

	result, err := u.feature.SignInWithUsernameAndPassword(r.Context(), body["username"].(string), body["password"].(string))
	if err != nil {
		if errors.Is(err, ErrUsernameNotFound) || errors.Is(err, ErrAPIInvalidCredentials) {
			u.responder.Error(w, r, ErrAPIInvalidCredentials)
			return
		}
		u.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusInternalServerError, nil))
		return
	}

	sessionResult, err := u.feature.core.SessionManager.CreateSession(r.Context(), r, result)
	if err != nil {
		u.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusInternalServerError, nil))
		return
	}

	u.responder.SessionResponse(w, r, u.feature.core, result, sessionResult)
}

func (u *usernamePasswordAPI) SignUpWithUsernameAndPassword(w http.ResponseWriter, r *http.Request) {
	additionalFields, err := aegis.GetSchemaAdditionalFieldsForRequest(w, r, u.feature.userSchema)
	if err != nil {
		u.responder.Error(w, r, err.(*aegis.AegisError))
		return
	}

	body := validator.ValidateJSON(w, r, u.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.
			Required("username", data["username"]).
			Required("email", data["email"]).
			Required("password", data["password"]).
			Email("email", data["email"])
	})

	if body == nil {
		return
	}

	username := body["username"].(string)

	// Add username to additionalFields so it gets stored in the database
	if additionalFields == nil {
		additionalFields = make(map[string]any)
	}

	user := &aegis.User{
		Email:    body["email"].(string),
		Password: body["password"].(string),
	}

	result, err := u.feature.SignUpWithUsernameAndPassword(r.Context(), user, username, additionalFields)

	if err != nil {
		if errors.Is(err, ErrUsernameAlreadyExists) {
			u.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusConflict, nil))
			return
		}
		if errors.Is(err, ErrUsernameTooShort) || errors.Is(err, ErrUsernameTooLong) || errors.Is(err, ErrUsernameInvalidFormat) {
			u.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusUnprocessableEntity, nil))
			return
		}
		u.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	// Always create session on signup (similar to email-password's autoSignInOnSignUp behavior)
	sessionResult, err := u.feature.core.SessionManager.CreateSession(r.Context(), r, result)
	if err != nil {
		u.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusInternalServerError, nil))
		return
	}

	u.responder.SessionResponse(w, r, u.feature.core, result, sessionResult)
}
