package twofactor

import (
	"net/http"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/pkg/validator"
)

type totpHandlers struct {
	totp      *totp
	responder *aegis.Responder
}

func newTOTPHandlers(totp *totp, responder *aegis.Responder) *totpHandlers {
	return &totpHandlers{totp: totp, responder: responder}
}

func (t *totpHandlers) GetSetupURI(w http.ResponseWriter, r *http.Request) {
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		t.responder.Error(w, r, err)
		return
	}

	user := t.totp.plugin.userSchema.UserToUserWithTwoFactor(session.User)
	result, err := t.totp.GetSetupURI(r.Context(), user)
	if err != nil {
		t.responder.Error(w, r, err)
		return
	}

	t.responder.JSON(w, r, http.StatusOK, result)
}

func (t *totpHandlers) VerifyCode(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, t.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.RequiredString("code", data["code"])
	})

	if body == nil {
		return
	}

	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		t.responder.Error(w, r, err)
		return
	}

	valid := t.totp.VerifyCode(r.Context(), session.User.ID, body["code"].(string))
	if !valid {
		t.responder.Error(w, r, aegis.NewAegisError("invalid code", http.StatusUnauthorized, nil))
		return
	}

	t.responder.JSON(w, r, http.StatusOK, map[string]any{
		"valid": true,
	})
}
