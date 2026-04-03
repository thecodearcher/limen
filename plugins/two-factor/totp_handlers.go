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
