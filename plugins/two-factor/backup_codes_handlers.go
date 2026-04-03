package twofactor

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

type backupCodesHandlers struct {
	backupCodes *backupCodes
	responder   *limen.Responder
}

func newBackupCodesHandlers(backupCodes *backupCodes, responder *limen.Responder) *backupCodesHandlers {
	return &backupCodesHandlers{backupCodes: backupCodes, responder: responder}
}

func (b *backupCodesHandlers) UpdateBackupCodes(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
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
	session, err := limen.GetCurrentSessionFromCtx(r)
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
