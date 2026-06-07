package passkey

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrSessionRequired      = limen.NewLimenError("passkey registration requires an authenticated session", http.StatusUnauthorized, nil)
	ErrChallengeNotFound    = limen.NewLimenError("passkey challenge not found", http.StatusBadRequest, nil)
	ErrChallengeMismatch    = limen.NewLimenError("passkey challenge does not match", http.StatusBadRequest, nil)
	ErrInvalidResponse      = limen.NewLimenError("invalid passkey response", http.StatusBadRequest, nil)
	ErrRegistrationFailed   = limen.NewLimenError("failed to verify passkey registration", http.StatusBadRequest, nil)
	ErrAuthenticationFailed = limen.NewLimenError("passkey authentication failed", http.StatusUnauthorized, nil)
	ErrPasskeyNotFound      = limen.NewLimenError("passkey not found", http.StatusNotFound, nil)
	ErrPasskeyAlreadyExists = limen.NewLimenError("passkey already registered", http.StatusConflict, nil)
	ErrOriginRequired       = limen.NewLimenError("at least one allowed origin is required", http.StatusInternalServerError, nil)
	ErrIDRequired           = limen.NewLimenError("id is required", http.StatusUnprocessableEntity, nil)
	ErrNameRequired         = limen.NewLimenError("name is required", http.StatusUnprocessableEntity, nil)
	ErrForbidden            = limen.NewLimenError("you are not allowed to perform this action on this passkey", http.StatusForbidden, nil)
	ErrUnauthorized         = limen.NewLimenError("authentication required", http.StatusUnauthorized, nil)
)
