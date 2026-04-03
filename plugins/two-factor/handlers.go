package twofactor

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/thecodearcher/limen"
)

type twoFactorHandlers struct {
	plugin    *twoFactorPlugin
	responder *limen.Responder
	httpCore  *limen.LimenHTTPCore
}

func newTwoFactorHandlers(plugin *twoFactorPlugin, responder *limen.Responder, httpCore *limen.LimenHTTPCore) *twoFactorHandlers {
	return &twoFactorHandlers{
		plugin:    plugin,
		responder: responder,
		httpCore:  httpCore,
	}
}

func (a *twoFactorHandlers) InitiateTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, a.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("password", data["password"])
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	user := a.plugin.userSchema.UserToUserWithTwoFactor(session.User)
	result, err := a.plugin.InitiateTwoFactorSetup(r.Context(), user, body["password"].(string))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.JSON(w, r, http.StatusOK, result)
}

func (a *twoFactorHandlers) FinalizeTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, a.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("code", data["code"])
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	user := a.plugin.userSchema.UserToUserWithTwoFactor(session.User)
	err = a.plugin.FinalizeTwoFactorSetup(r.Context(), user, body["code"].(string))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	authResult, sessionResult, err := a.plugin.rotateSession(r, w, session)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.SessionResponse(w, r, a.plugin.core, authResult, sessionResult)
}

// Disable disables 2FA for the current user
func (a *twoFactorHandlers) Disable(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, a.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("password", data["password"])
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	err = a.plugin.DisableTwoFactor(r.Context(), session.User.ID, body["password"].(string))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	authResult, sessionResult, err := a.plugin.rotateSession(r, w, session)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.SessionResponse(w, r, a.plugin.core, authResult, sessionResult)
}

// VerifyLoginWithTwoFactor verifies the 2FA code and completes the login process
func (a *twoFactorHandlers) VerifyLoginWithTwoFactor(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, a.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("code", data["code"]).
			Custom("method", func() error {
				allowedMethods := []string{string(TwoFactorMethodOTP), string(TwoFactorMethodTOTP)}
				method := data["method"]
				if method == nil || method == "" {
					method = string(TwoFactorMethodTOTP)
					data["method"] = method
				}

				if method, ok := method.(string); ok && slices.Contains(allowedMethods, method) {
					return nil
				}

				return fmt.Errorf("invalid 2FA method must be one of: %s", strings.Join(allowedMethods, ", "))
			}, false)
	})

	if body == nil {
		return
	}

	authResult, sessionResult, err := a.plugin.VerifyLoginWithTwoFactor(r, w, body["code"].(string), TwoFactorMethod(body["method"].(string)))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.SessionResponse(w, r, a.plugin.core, authResult, sessionResult)
}
