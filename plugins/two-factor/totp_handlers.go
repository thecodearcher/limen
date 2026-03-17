package twofactor

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

type totpHandlers struct {
	totp      *totp
	responder *limen.Responder
}

func newTOTPHandlers(totp *totp, responder *limen.Responder) *totpHandlers {
	return &totpHandlers{totp: totp, responder: responder}
}

func (t *totpHandlers) GetSetupURI(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
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
	body := limen.ValidateJSON(w, r, t.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("code", data["code"])
	})

	if body == nil {
		return
	}

	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		t.responder.Error(w, r, err)
		return
	}

	if err := t.totp.VerifyCode(r.Context(), session.User.ID, body["code"].(string)); err != nil {
		t.responder.Error(w, r, err)
		return
	}

	t.responder.JSON(w, r, http.StatusOK, map[string]any{
		"valid": true,
	})
}
