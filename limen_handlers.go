package limen

import (
	"errors"
	"net/http"
)

type limenHandlers struct {
	core      *LimenCore
	responder *Responder
	config    *httpConfig
}

func registerBaseRoutes(router *router, httpCore *LimenHTTPCore, core *LimenCore, basePath string) {
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
	routeBuilder.ProtectedGET("/sessions", "list-sessions", h.ListSessions)
	routeBuilder.ProtectedPOST("/signout", "signout", h.SignOut)
	routeBuilder.ProtectedPOST("/revoke-sessions", "revoke-sessions", h.RevokeAllSessions)

	if h.core.EmailVerificationEnabled() {
		routeBuilder.POST("/verify-email", "verify-email", h.VerifyEmail)
		routeBuilder.POST("/email-verifications", "email-verifications", h.RequestEmailVerification)
	}
}

func (h *limenHandlers) GetSession(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.core.Cookies().DeleteSessionCookie(w)
		h.responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	h.responder.SessionResponse(w, r, h.core, &AuthenticationResult{User: session.User}, nil)
}

func (h *limenHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	sessions, err := h.core.SessionManager.ListSessions(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, sessions)
}

func (h *limenHandlers) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	err = h.core.SessionManager.RevokeAllSessions(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *limenHandlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	body := ValidateRequest(w, r, h.responder, func(v *Validator) {
		v.Field("token").Required()
	})
	if body == nil {
		return
	}

	email, err := h.core.VerifyEmail(r.Context(), body["token"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.core.ClearEmailVerificationChallenge(w)
	if !h.core.config.Email.verification.autoSignInAfterVerification {
		h.responder.JSON(w, r, http.StatusOK, "email verified successfully")
		return
	}

	user, err := h.core.DBAction.FindUserByEmail(r.Context(), email)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	sessionResult, err := h.core.CreateSession(r.Context(), r, w, &AuthenticationResult{User: user})
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.SessionResponse(w, r, h.core, &AuthenticationResult{User: user}, sessionResult)
}

func (h *limenHandlers) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	body := ValidateRequest(w, r, h.responder, func(v *Validator) {
		v.Field("email").Optional().Email()
	})
	if body == nil {
		return
	}

	user, refreshed := h.provenEmailVerificationUser(r)
	if user != nil {
		_, err := h.core.RequestEmailVerification(r.Context(), &User{Email: user.Email}, true)
		if err != nil {
			h.responder.ErrorWithSession(w, r, err, refreshed)
			return
		}
		h.responder.JSONWithSession(w, r, http.StatusOK, "email verification requested successfully", refreshed)
		return
	}

	if !h.core.config.Email.verification.requestByEmailEnabled {
		h.responder.Error(w, r, ErrUnauthorized)
		return
	}

	if email, _ := body["email"].(string); email != "" {
		_, err := h.core.RequestEmailVerification(r.Context(), &User{Email: email}, true)
		if err != nil && !errors.Is(err, ErrRecordNotFound) && !errors.Is(err, ErrEmailAlreadyVerified) {
			h.responder.Error(w, r, err)
			return
		}
	}

	h.responder.JSON(w, r, http.StatusOK, "if the email address is associated with an account, "+
		"you will receive an email with instructions to verify it")
}

func (h *limenHandlers) provenEmailVerificationUser(r *http.Request) (*User, *SessionResult) {
	session, err := h.core.SessionManager.ValidateSession(r.Context(), r)
	if err == nil && session != nil && session.User != nil {
		return session.User, session.Refreshed
	}

	user, err := h.core.ResolveEmailVerificationChallenge(r)
	if err != nil {
		return nil, nil
	}
	return user, nil
}

func (h *limenHandlers) SignOut(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r.Context())
	if err != nil {
		h.responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	err = h.core.SessionManager.RevokeSession(r.Context(), session.Session.Token)
	if err != nil {
		h.responder.Error(w, r, NewLimenError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	h.core.Cookies().DeleteSessionCookie(w)

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}
