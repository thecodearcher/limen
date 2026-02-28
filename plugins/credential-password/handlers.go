package credentialpassword

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/thecodearcher/aegis"
)

type credentialPasswordHandlers struct {
	plugin    *credentialPasswordPlugin
	builder   *aegis.RouteBuilder
	responder *aegis.Responder
}

func NewCredentialPasswordAPI(emailPasswordPlugin *credentialPasswordPlugin, httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) *credentialPasswordHandlers {
	return &credentialPasswordHandlers{
		plugin:    emailPasswordPlugin,
		builder:   routeBuilder,
		responder: httpCore.Responder,
	}
}

// PluginHTTPConfig returns the HTTP configuration for the credential password plugin,
// including rate limiting rules for all authentication endpoints.
func (p *credentialPasswordPlugin) PluginHTTPConfig() aegis.PluginHTTPConfig {
	return aegis.PluginHTTPConfig{
		Middleware: []aegis.Middleware{},
		RateLimitRules: []*aegis.RateLimitRule{
			aegis.NewRateLimitRule("/signin/credential", 5, 10*time.Second),
			aegis.NewRateLimitRule("/signup/credential", 5, 10*time.Second),
			aegis.NewRateLimitRule("/passwords/request-reset", 5, 10*time.Minute),
			aegis.NewRateLimitRule("/verify-email", 5, 10*time.Minute),
			aegis.NewRateLimitRule("/passwords/reset", 5, 10*time.Minute),
			aegis.NewRateLimitRule("/passwords/change", 5, 10*time.Minute),
			aegis.NewRateLimitRule("/passwords", 5, 10*time.Minute),
			aegis.NewRateLimitRule("/usernames/check", 10, 1*time.Minute),
		},
	}
}

func (p *credentialPasswordPlugin) RegisterRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
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
	e.builder.ProtectedPUT("/passwords", "passwords-set", e.SetPassword)
	e.builder.POST("/usernames/check", "usernames-check", e.CheckUsernameAvailability)
}

// SignInWithCredentialAndPassword handles user sign-in with either email or username (if enabled) and password.
// The credential can be either an email address or a username.
func (p *credentialPasswordHandlers) SignInWithCredentialAndPassword(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, p.responder,
		func(v *aegis.Validator, data map[string]any) *aegis.Validator {
			return v.RequiredString("credential", data["credential"]).
				RequiredString("password", data["password"])
		})

	if body == nil {
		return
	}

	result, err := p.plugin.SignInWithCredentialAndPassword(r.Context(), body["credential"].(string), body["password"].(string))
	if err != nil {
		p.responder.Error(w, r, aegis.NewAegisError(ErrInvalidCredential.Error(), ErrInvalidCredential.Status(), nil))
		return
	}

	shortSession := false
	if val, ok := body["remember_me"].(bool); ok {
		shortSession = !val
	}
	sessionResult, err := p.plugin.core.CreateSession(r.Context(), r, w, result, aegis.WithShortSession(shortSession))
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	p.responder.SessionResponse(w, r, p.plugin.core, result, sessionResult)
}

// SignUpWithCredentialAndPassword handles user registration with email and password.
// If username support is enabled, a username can be provided in the request body.
func (p *credentialPasswordHandlers) SignUpWithCredentialAndPassword(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, p.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.RequiredString("email", data["email"]).
			RequiredString("password", data["password"]).
			Email("email", data["email"]).
			Custom("username", func() error {
				username, ok := data["username"].(string)
				if !ok {
					username = ""
				}
				return p.plugin.validateUsername(username)
			}, false).
			Custom("password", func() error {
				password, ok := data["password"].(string)
				if !ok {
					password = ""
				}
				return p.plugin.validatePassword(password)
			}, false)
	})

	if body == nil {
		return
	}

	additionalFields := map[string]any{}
	if p.plugin.config.enableUsername && body["username"] != nil {
		additionalFields["username"] = strings.TrimSpace(body["username"].(string))
	}

	password := body["password"].(string)
	result, err := p.plugin.SignUpWithCredentialAndPassword(r.Context(), &aegis.User{
		Email:    body["email"].(string),
		Password: &password,
	}, additionalFields)

	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	if !p.plugin.config.autoSignInOnSignUp {
		p.responder.SessionResponse(w, r, p.plugin.core, result, nil)
		return
	}

	sessionResult, err := p.plugin.core.CreateSession(r.Context(), r, w, result)
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	p.responder.SessionResponse(w, r, p.plugin.core, result, sessionResult)
}

// VerifyEmail handles email verification using a verification token.
func (p *credentialPasswordHandlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, p.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.RequiredString("token", data["token"])
	})

	if body == nil {
		return
	}

	err := p.plugin.VerifyEmail(r.Context(), body["token"].(string))
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	p.responder.JSON(w, r, http.StatusOK, "email verified successfully")
}

// RequestEmailVerification handles requests for email verification.
func (p *credentialPasswordHandlers) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	_, err = p.plugin.RequestEmailVerification(r.Context(), &aegis.User{
		Email: session.User.Email,
	}, true)

	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	p.responder.JSON(w, r, http.StatusOK, "email verification requested successfully")
}

// RequestPasswordReset handles password reset requests.
// To prevent email enumeration, always returns success regardless of whether the email exists.
func (p *credentialPasswordHandlers) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, p.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.
			RequiredString("email", data["email"]).
			Email("email", data["email"])
	})

	if body == nil {
		return
	}

	message := "if the email address is associated with an account, you will receive an email with instructions to reset your password"
	_, err := p.plugin.RequestPasswordReset(r.Context(), body["email"].(string))
	if err != nil {
		if errors.Is(err, ErrEmailNotFound) {
			// we don't want to leak the existence of the email address
			p.responder.JSON(w, r, http.StatusOK, message)
			return
		}

		p.responder.Error(w, r, err)
		return
	}

	p.responder.JSON(w, r, http.StatusOK, message)
}

// ResetPassword handles password reset using a valid reset token.
func (p *credentialPasswordHandlers) ResetPassword(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, p.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.
			RequiredString("token", data["token"]).
			RequiredString("new_password", data["new_password"]).
			Custom("new_password", func() error {
				newPassword, ok := data["new_password"].(string)
				if !ok {
					newPassword = ""
				}
				return p.plugin.validatePassword(newPassword)
			}, false)
	})

	if body == nil {
		return
	}

	err := p.plugin.ResetPassword(r.Context(), body["token"].(string), body["new_password"].(string))
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	p.responder.JSON(w, r, http.StatusOK, "password reset successfully")
}

// ChangePassword handles password changes for authenticated users.
// Optionally revokes all other sessions when revoke_other_sessions is true.
func (p *credentialPasswordHandlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, p.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.
			RequiredString("current_password", data["current_password"]).
			RequiredString("new_password", data["new_password"]).
			Custom("new_password", func() error {
				newPassword, ok := data["new_password"].(string)
				if !ok {
					newPassword = ""
				}
				return p.plugin.validatePassword(newPassword)
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
		p.responder.Error(w, r, err)
		return
	}

	err = p.plugin.UpdatePassword(r.Context(), session.User, body["current_password"].(string), body["new_password"].(string), revokeOtherSessions)
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	authResult := &aegis.AuthenticationResult{
		User: session.User,
	}

	if revokeOtherSessions {
		sessionResult, err := p.plugin.core.CreateSession(r.Context(), r, w, authResult)
		if err != nil {
			p.responder.Error(w, r, err)
			return
		}
		p.responder.SessionResponse(w, r, p.plugin.core, authResult, sessionResult)
		return
	}

	p.responder.SessionResponse(w, r, p.plugin.core, authResult, nil)
}

func (p *credentialPasswordHandlers) SetPassword(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, p.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.
			RequiredString("new_password", data["new_password"]).
			Custom("new_password", func() error {
				newPassword, ok := data["new_password"].(string)
				if !ok {
					newPassword = ""
				}
				return p.plugin.validatePassword(newPassword)
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
		p.responder.Error(w, r, err)
		return
	}

	err = p.plugin.SetPassword(r.Context(), session.User, body["new_password"].(string), revokeOtherSessions)
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	authResult := &aegis.AuthenticationResult{
		User: session.User,
	}

	if revokeOtherSessions {
		sessionResult, err := p.plugin.core.CreateSession(r.Context(), r, w, authResult)
		if err != nil {
			p.responder.Error(w, r, err)
			return
		}
		p.responder.SessionResponse(w, r, p.plugin.core, authResult, sessionResult)
		return
	}

	p.responder.SessionResponse(w, r, p.plugin.core, authResult, nil)
}

// CheckUsernameAvailability handles username availability checks.
func (p *credentialPasswordHandlers) CheckUsernameAvailability(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, p.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.RequiredString("username", data["username"]).
			Custom("username", func() error {
				username, ok := data["username"].(string)
				if !ok {
					username = ""
				}
				return p.plugin.validateUsername(username)
			}, false)
	})

	if body == nil {
		return
	}

	available, err := p.plugin.CheckUsernameAvailability(r.Context(), body["username"].(string))
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	p.responder.JSON(w, r, http.StatusOK, map[string]any{
		"available": available,
	})
}
