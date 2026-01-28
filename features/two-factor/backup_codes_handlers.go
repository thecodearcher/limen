package twofactor

import (
	"net/http"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/pkg/validator"
)

type backupCodesHandlers struct {
	backupCodes *backupCodes
	responder   *aegis.Responder
}

func newBackupCodesHandlers(backupCodes *backupCodes, responder *aegis.Responder) *backupCodesHandlers {
	return &backupCodesHandlers{backupCodes: backupCodes, responder: responder}
}

func (b *backupCodesHandlers) UpdateBackupCodes(w http.ResponseWriter, r *http.Request) {
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		b.responder.Error(w, r, err)
		return
	}

	backupCodes, err := b.backupCodes.UpdateBackupCodes(r.Context(), session.User.ID)
	if err != nil {
		b.responder.Error(w, r, err)
		return
	}
	b.responder.JSON(w, r, http.StatusOK, backupCodes)
}

func (b *backupCodesHandlers) GetBackupCodes(w http.ResponseWriter, r *http.Request) {
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		b.responder.Error(w, r, err)
		return
	}
	backupCodes, err := b.backupCodes.GetBackupCodes(r.Context(), session.User.ID)
	if err != nil {
		b.responder.Error(w, r, err)
		return
	}
	b.responder.JSON(w, r, http.StatusOK, backupCodes)
}

func (b *backupCodesHandlers) VerifyBackupCode(w http.ResponseWriter, r *http.Request) {
	body := validator.ValidateJSON(w, r, b.responder, func(v *validator.Validator, data map[string]any) *validator.Validator {
		return v.RequiredString("code", data["code"])
	})
	if body == nil {
		return
	}
	session, err := aegis.GetCurrentSessionFromCtx(r)
	if err != nil {
		b.responder.Error(w, r, err)
		return
	}
	if err := b.backupCodes.VerifyBackupCode(r.Context(), session.User.ID, body["code"].(string)); err != nil {
		b.responder.Error(w, r, err)
		return
	}
	b.responder.JSON(w, r, http.StatusOK, "backup code verified successfully")
}
