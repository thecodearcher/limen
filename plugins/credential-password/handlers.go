package credentialpassword

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/thecodearcher/limen"
)

type credentialPasswordHandlers struct {
	plugin    *credentialPasswordPlugin
	builder   *limen.RouteBuilder
	responder *limen.Responder
}

func NewCredentialPasswordAPI(emailPasswordPlugin *credentialPasswordPlugin, httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) *credentialPasswordHandlers {
	return &credentialPasswordHandlers{
		plugin:    emailPasswordPlugin,
		builder:   routeBuilder,
		responder: httpCore.Responder,
	}
}

// PluginHTTPConfig returns the HTTP configuration for the credential password plugin,
// including rate limiting rules for all authentication endpoints.
func (p *credentialPasswordPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		Middleware: []limen.Middleware{},
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRule("/signin/credential", 5, 10*time.Second),
			limen.NewRateLimitRule("/signup/credential", 5, 10*time.Second),
			limen.NewRateLimitRule("/passwords/request-reset", 5, 10*time.Minute),
			limen.NewRateLimitRule("/passwords/reset", 5, 10*time.Minute),
			limen.NewRateLimitRule("/passwords/change", 5, 10*time.Minute),
			limen.NewRateLimitRule("/passwords", 5, 10*time.Minute),
			limen.NewRateLimitRule("/usernames/check", 10, 1*time.Minute),
		},
	}
}

func (p *credentialPasswordPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	api := NewCredentialPasswordAPI(p, httpCore, routeBuilder)
	routes(api)
}

func routes(e *credentialPasswordHandlers) {
	e.builder.POST("/signin/credential", "signin", e.SignInWithCredentialAndPassword)
	e.builder.POST("/signup/credential", "signup", e.SignUpWithCredentialAndPassword)
	e.builder.POST("/passwords/request-reset", "passwords-request-reset", e.RequestPasswordReset)
	e.builder.POST("/passwords/reset", "passwords-reset", e.ResetPassword)
	e.builder.ProtectedPOST("/passwords/change", "passwords-change", e.ChangePassword)
	e.builder.ProtectedPUT("/passwords", "passwords-set", e.SetPassword)
	e.builder.POST("/usernames/check", "usernames-check", e.CheckUsernameAvailability)
}

// SignInWithCredentialAndPassword handles user sign-in with either email or username (if enabled) and password.
// The credential can be either an email address or a username.
func (p *credentialPasswordHandlers) SignInWithCredentialAndPassword(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, p.responder, func(v *limen.Validator) {
		v.Field("credential").Required()
		v.Field("password").Required()
	})

	if body == nil {
		return
	}

	result, err := p.plugin.SignInWithCredentialAndPassword(r.Context(), body["credential"].(string), body["password"].(string))
	if err != nil {
		var unverified emailNotVerifiedError
		if errors.As(err, &unverified) {
			p.handleUnverifiedSignIn(w, r, unverified.email)
			return
		}

		p.responder.Error(w, r, limen.NewLimenError(ErrInvalidCredential.Error(), ErrInvalidCredential.Status(), nil))
		return
	}

	shortSession := false
	if val, ok := body["remember_me"].(bool); ok {
		shortSession = !val
	}
	sessionResult, err := p.plugin.core.CreateSession(r.Context(), r, w, result, limen.WithShortSession(shortSession))
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	p.responder.SessionResponse(w, r, p.plugin.core, result, sessionResult)
}

// handleUnverifiedSignIn keeps the 403 and, when configured, issues a waiting-room
// cookie and resends the verification email.
func (p *credentialPasswordHandlers) handleUnverifiedSignIn(w http.ResponseWriter, r *http.Request, email string) {
	if err := p.issueVerificationChallenge(w, email); err != nil {
		p.responder.Error(w, r, err)
		return
	}

	if p.plugin.config.sendEmailVerificationOnSignIn {
		_, err := p.plugin.core.RequestEmailVerification(r.Context(), &limen.User{Email: email}, true)
		if err != nil && !errors.Is(err, limen.ErrEmailAlreadyVerified) {
			p.responder.Error(w, r, err)
			return
		}
	}

	p.responder.Error(w, r, ErrEmailNotVerified)
}

// issueVerificationChallenge sets the waiting-room cookie when the session is
// withheld until the email is verified.
func (p *credentialPasswordHandlers) issueVerificationChallenge(w http.ResponseWriter, email string) error {
	if !p.plugin.config.requireEmailVerification || !p.plugin.core.EmailVerificationChallengeEnabled() {
		return nil
	}
	return p.plugin.core.IssueEmailVerificationChallenge(w, email)
}

func (p *credentialPasswordHandlers) isUsernameRequiredOnSignup() bool {
	return p.plugin.config.usernameRequiredOnSignup && p.plugin.config.enableUsername
}

// SignUpWithCredentialAndPassword handles user registration with email and password.
// If username support is enabled, a username can be provided in the request body.
func (p *credentialPasswordHandlers) SignUpWithCredentialAndPassword(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, p.responder, func(v *limen.Validator) {
		v.Field("email").Required().Email()
		v.Field("password").Required().String().Custom(func(value any, _ map[string]any) error {
			password, _ := value.(string)
			return p.plugin.validatePassword(password)
		})
		v.Field("username").RequiredWhen(p.isUsernameRequiredOnSignup).String().Custom(func(value any, _ map[string]any) error {
			username, _ := value.(string)
			return p.plugin.validateUsername(username)
		})
	})

	if body == nil {
		return
	}

	additionalFields := map[string]any{}
	if p.plugin.config.enableUsername && body["username"] != nil {
		additionalFields["username"] = strings.TrimSpace(body["username"].(string))
	}

	password := body["password"].(string)
	result, err := p.plugin.SignUpWithCredentialAndPassword(r.Context(), &limen.User{
		Email:    body["email"].(string),
		Password: &password,
	}, additionalFields)

	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	if !p.plugin.config.autoSignInOnSignUp || p.plugin.config.requireEmailVerification {
		if err := p.issueVerificationChallenge(w, result.User.Email); err != nil {
			p.responder.Error(w, r, err)
			return
		}
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

// RequestPasswordReset handles password reset requests.
// To prevent email enumeration, always returns success regardless of whether the email exists.
func (p *credentialPasswordHandlers) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateRequest(w, r, p.responder, func(v *limen.Validator) {
		v.Field("email").Required().Email()
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
	body := limen.ValidateRequest(w, r, p.responder, func(v *limen.Validator) {
		v.Field("token").Required()
		v.Field("new_password").Required().Custom(func(value any, _ map[string]any) error {
			newPassword, _ := value.(string)
			return p.plugin.validatePassword(newPassword)
		})
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
	body := limen.ValidateRequest(w, r, p.responder, func(v *limen.Validator) {
		v.Field("current_password").Required()
		v.Field("new_password").Required().Custom(func(value any, _ map[string]any) error {
			newPassword, _ := value.(string)
			return p.plugin.validatePassword(newPassword)
		})
	})

	if body == nil {
		return
	}

	revokeOtherSessions := true
	if value, ok := body["revoke_other_sessions"].(bool); ok {
		revokeOtherSessions = value
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	err = p.plugin.UpdatePassword(r.Context(), session.User, body["current_password"].(string), body["new_password"].(string), revokeOtherSessions)
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	authResult := &limen.AuthenticationResult{
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
	body := limen.ValidateRequest(w, r, p.responder, func(v *limen.Validator) {
		v.Field("new_password").Required().Custom(func(value any, _ map[string]any) error {
			newPassword, _ := value.(string)
			return p.plugin.validatePassword(newPassword)
		})
	})

	if body == nil {
		return
	}

	revokeOtherSessions := true
	if value, ok := body["revoke_other_sessions"].(bool); ok {
		revokeOtherSessions = value
	}

	session, err := limen.GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	err = p.plugin.SetPassword(r.Context(), session.User, body["new_password"].(string), revokeOtherSessions)
	if err != nil {
		p.responder.Error(w, r, err)
		return
	}

	authResult := &limen.AuthenticationResult{
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
	body := limen.ValidateRequest(w, r, p.responder, func(v *limen.Validator) {
		v.Field("username").Required().Custom(func(value any, _ map[string]any) error {
			username, _ := value.(string)
			return p.plugin.validateUsername(username)
		})
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
