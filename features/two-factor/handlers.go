package twofactor

import (
	"net/http"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/pkg/validator"
)

type twoFactorAPI struct {
	feature   *twoFactorFeature
	responder *aegis.Responder
}

func newTwoFactorAPI(feature *twoFactorFeature, responder *aegis.Responder) *twoFactorAPI {
	return &twoFactorAPI{
		feature:   feature,
		responder: responder,
	}
}

func (a *twoFactorAPI) InitiateTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, a.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
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

func (a *twoFactorAPI) FinalizeTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, a.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
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
func (a *twoFactorAPI) Disable(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, a.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
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
