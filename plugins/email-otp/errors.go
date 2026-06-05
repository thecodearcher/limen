package emailotp

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrEmailRequired   = limen.NewLimenError("email is required", http.StatusUnprocessableEntity, nil)
	ErrOTPRequired     = limen.NewLimenError("otp is required", http.StatusUnprocessableEntity, nil)
	ErrInvalidOTPType  = limen.NewLimenError("invalid otp type", http.StatusUnprocessableEntity, nil)
	ErrInvalidOTP      = limen.NewLimenError("invalid otp", http.StatusBadRequest, nil)
	ErrOTPExpired      = limen.NewLimenError("otp has expired", http.StatusBadRequest, nil)
	ErrTooManyAttempts = limen.NewLimenError("too many attempts", http.StatusForbidden, nil)
	ErrUserNotFound    = limen.NewLimenError("user not found", http.StatusBadRequest, nil)
	ErrSignUpDisabled  = limen.NewLimenError("sign up is disabled", http.StatusBadRequest, nil)
)
