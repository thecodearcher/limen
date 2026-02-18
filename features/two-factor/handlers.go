package twofactor

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/thecodearcher/aegis"
)

type twoFactorHandlers struct {
	feature   *twoFactorFeature
	responder *aegis.Responder
	httpCore  *aegis.AegisHTTPCore
}

func newTwoFactorHandlers(feature *twoFactorFeature, responder *aegis.Responder, httpCore *aegis.AegisHTTPCore) *twoFactorHandlers {
	return &twoFactorHandlers{
		feature:   feature,
		responder: responder,
		httpCore:  httpCore,
	}
}

func (a *twoFactorHandlers) InitiateTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, a.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.RequiredString("password", data["password"])
	})

	if body == nil {
		return
	}

	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	user := a.feature.userSchema.UserToUserWithTwoFactor(session.User)
	result, err := a.feature.InitiateTwoFactorSetup(r.Context(), user, body["password"].(string))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.JSON(w, r, http.StatusOK, result)
}

func (a *twoFactorHandlers) FinalizeTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, a.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.RequiredString("code", data["code"])
	})

	if body == nil {
		return
	}

	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	user := a.feature.userSchema.UserToUserWithTwoFactor(session.User)
	err = a.feature.FinalizeTwoFactorSetup(r.Context(), user, body["code"].(string))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.JSON(w, r, http.StatusOK, map[string]any{
		"message": "2FA setup finalized successfully",
	})
}

// Disable disables 2FA for the current user
func (a *twoFactorHandlers) Disable(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, a.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
		return v.RequiredString("password", data["password"])
	})

	if body == nil {
		return
	}

	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	err = a.feature.DisableTwoFactor(r.Context(), session.User.ID, body["password"].(string))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.JSON(w, r, http.StatusOK, map[string]any{
		"message": "2FA disabled successfully",
	})
}

// VerifyLoginWithTwoFactor verifies the 2FA code and completes the login process
func (a *twoFactorHandlers) VerifyLoginWithTwoFactor(w http.ResponseWriter, r *http.Request) {
	body := aegis.ValidateJSON(w, r, a.responder, func(v *aegis.Validator, data map[string]any) *aegis.Validator {
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

	authResult, sessionResult, err := a.feature.VerifyLoginWithTwoFactor(r, w, body["code"].(string), TwoFactorMethod(body["method"].(string)))
	if err != nil {
		a.responder.Error(w, r, err)
		return
	}

	a.responder.SessionResponse(w, r, a.feature.core, authResult, sessionResult)
}
