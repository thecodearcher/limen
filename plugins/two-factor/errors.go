package twofactor

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrTwoFactorAlreadyEnabled              = limen.NewLimenError("two factor already enabled", http.StatusConflict, nil)
	ErrInvalidCode                          = limen.NewLimenError("invalid code", http.StatusUnauthorized, nil)
	ErrTwoFactorNotEnabled                  = limen.NewLimenError("two factor not enabled", http.StatusUnauthorized, nil)
	ErrInvalidPassword                      = limen.NewLimenError("invalid password", http.StatusUnauthorized, nil)
	ErrPasswordRequired                     = limen.NewLimenError("password is required", http.StatusUnprocessableEntity, nil)
	ErrCredentialPasswordPluginNotAvailable = limen.NewLimenError("credential password plugin not available", http.StatusServiceUnavailable, nil)
	ErrInvalidBackupCode                    = limen.NewLimenError("invalid backup code", http.StatusUnauthorized, nil)
	ErrInvalidOTPCode                       = limen.NewLimenError("invalid OTP code", http.StatusUnauthorized, nil)
	ErrInvalidChallenge                     = limen.NewLimenError("invalid 2FA challenge", http.StatusUnauthorized, nil)
	ErrChallengeExpired                     = limen.NewLimenError("2FA challenge has expired", http.StatusUnauthorized, nil)
	ErrChallengeMissing                     = limen.NewLimenError("2FA challenge is missing", http.StatusUnauthorized, nil)
)
