package credentialpassword

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/pkg/httpx"
	"github.com/thecodearcher/aegis/pkg/validator"
)

type credentialPasswordHandlers struct {
	feature   *credentialPasswordFeature
	builder   *aegis.RouteBuilder
	responder *aegis.Responder
}

func NewCredentialPasswordAPI(emailPasswordFeature *credentialPasswordFeature, httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) *credentialPasswordHandlers {
	return &credentialPasswordHandlers{
		feature:   emailPasswordFeature,
		builder:   routeBuilder,
		responder: httpCore.Responder,
	}
}

// PluginHTTPConfig returns the HTTP configuration for the credential password feature,
// including rate limiting rules for all authentication endpoints.
func (p *credentialPasswordFeature) PluginHTTPConfig() aegis.PluginHTTPConfig {
	return aegis.PluginHTTPConfig{
		Middleware: []httpx.Middleware{},
		RateLimitRules: []*aegis.RateLimitRule{
			aegis.NewRateLimitRule("/signin/credential", 5, 10*time.Second),
			aegis.NewRateLimitRule("/signup/credential", 5, 10*time.Second),
			aegis.NewRateLimitRule("/passwords/request-reset", 5, 10*time.Minute),
			aegis.NewRateLimitRule("/verify-email", 5, 10*time.Minute),
			aegis.NewRateLimitRule("/passwords/reset", 5, 10*time.Minute),
			aegis.NewRateLimitRule("/passwords/change", 5, 10*time.Minute),
			aegis.NewRateLimitRule("/usernames/check", 10, 1*time.Minute),
		},
	}
}

func (p *credentialPasswordFeature) RegisterRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
	api := NewCredentialPasswordAPI(p, httpCore, routeBuilder)
	routes(api)
}

func routes(e *credentialPasswordHandlers) {
	e.builder.POST("/signin/credential", "signin", e.SignInWithCredentialAndPassword)
	e.builder.POST("/signup/credential", "signup", e.SignUpWithCredentialAndPassword)
	e.builder.POST("/verify-email", "verify-email", e.VerifyEmail)
	e.builder.POST("/passwords/request-reset", "passwords-request-reset", e.RequestPasswordReset)
	e.builder.POST("/passwords/reset", "passwords-reset", e.ResetPassword)
	e.builder.ProtectedPOST("/email-verifications", "email-verifications", e.RequestEmailVerification)
	e.builder.ProtectedPOST("/passwords/change", "passwords-change", e.ChangePassword)
	e.builder.POST("/usernames/check", "usernames-check", e.CheckUsernameAvailability)
}

// SignInWithCredentialAndPassword handles user sign-in with either email or username (if enabled) and password.
// The credential can be either an email address or a username.
func (p *credentialPasswordHandlers) SignInWithCredentialAndPassword(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder,
		func(v *validator.Validator, data map[string]any) *validator.Validator {
			return v.RequiredString("credential", data["credential"]).
				RequiredString("password", data["password"])
		})

	if body == nil {
		return
	}

	result, err := p.feature.SignInWithCredentialAndPassword(r.Context(), body["credential"].(string), body["password"].(string))
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(ErrInvalidCredential.Error(), errorStatus(ErrInvalidCredential), nil))
		return
	}

	sessionResult, err := p.feature.core.SessionManager.CreateSession(r.Context(), r, result)
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusInternalServerError, nil))
		return
	}

	p.responder.SessionResponse(w, r, p.feature.core, result, sessionResult)
}

// SignUpWithCredentialAndPassword handles user registration with email and password.
// If username support is enabled, a username can be provided in the request body.
func (p *credentialPasswordHandlers) SignUpWithCredentialAndPassword(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.RequiredString("email", data["email"]).
			RequiredString("password", data["password"]).
			Email("email", data["email"]).
			Custom("username", func() error {
				username, ok := data["username"].(string)
				if !ok {
					username = ""
				}
				return p.feature.validateUsername(username)
			}, false).
			Custom("password", func() error {
				password, ok := data["password"].(string)
				if !ok {
					password = ""
				}
				return p.feature.validatePassword(password)
			}, false)
	})

	if body == nil {
		return
	}

	additionalFields := map[string]any{}
	if p.feature.config.enableUsername && body["username"] != nil {
		additionalFields["username"] = strings.TrimSpace(body["username"].(string))
	}

	result, err := p.feature.SignUpWithCredentialAndPassword(r.Context(), &aegis.User{
		Email:    body["email"].(string),
		Password: body["password"].(string),
	}, additionalFields)

	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), errorStatus(err), nil))
		return
	}

	if !p.feature.config.autoSignInOnSignUp {
		p.responder.SessionResponse(w, r, p.feature.core, result, nil)
		return
	}

	sessionResult, err := p.feature.core.SessionManager.CreateSession(r.Context(), r, result)
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusInternalServerError, nil))
		return
	}

	p.responder.SessionResponse(w, r, p.feature.core, result, sessionResult)
}

// VerifyEmail handles email verification using a verification token.
func (p *credentialPasswordHandlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.RequiredString("token", data["token"])
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

// RequestEmailVerification handles requests for email verification.
func (p *credentialPasswordHandlers) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	_, err = p.feature.RequestEmailVerification(r.Context(), &aegis.User{
		Email: session.User.Email,
	}, true)

	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	p.responder.JSON(w, r, http.StatusOK, "email verification requested successfully")
}

// RequestPasswordReset handles password reset requests.
// To prevent email enumeration, always returns success regardless of whether the email exists.
func (p *credentialPasswordHandlers) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.
			RequiredString("email", data["email"]).
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

// ResetPassword handles password reset using a valid reset token.
func (p *credentialPasswordHandlers) ResetPassword(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.
			RequiredString("token", data["token"]).
			RequiredString("new_password", data["new_password"]).
			Custom("new_password", func() error {
				newPassword, ok := data["new_password"].(string)
				if !ok {
					newPassword = ""
				}
				return p.feature.validatePassword(newPassword)
			}, false)
	})

	if body == nil {
		return
	}

	err := p.feature.ResetPassword(r.Context(), body["token"].(string), body["new_password"].(string))
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), errorStatus(err), nil))
		return
	}

	p.responder.JSON(w, r, http.StatusOK, "password reset successfully")
}

// ChangePassword handles password changes for authenticated users.
// Optionally revokes all other sessions when revoke_other_sessions is true.
func (p *credentialPasswordHandlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.
			RequiredString("current_password", data["current_password"]).
			RequiredString("new_password", data["new_password"]).
			Custom("new_password", func() error {
				newPassword, ok := data["new_password"].(string)
				if !ok {
					newPassword = ""
				}
				return p.feature.validatePassword(newPassword)
			}, false)
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

// CheckUsernameAvailability handles username availability checks.
func (p *credentialPasswordHandlers) CheckUsernameAvailability(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, p.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.RequiredString("username", data["username"]).
			Custom("username", func() error {
				username, ok := data["username"].(string)
				if !ok {
					username = ""
				}
				return p.feature.validateUsername(username)
			}, false)
	})

	if body == nil {
		return
	}

	available, err := p.feature.CheckUsernameAvailability(r.Context(), body["username"].(string))
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(err.Error(), errorStatus(err), nil))
		return
	}

	p.responder.JSON(w, r, http.StatusOK, map[string]any{
		"available": available,
	})
}
